// Package sidecar implements the secrets agent that runs as init container or sidecar.
package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/keeper-security/keeper-k8s-injector/pkg/ksm"
	"github.com/keeper-security/keeper-k8s-injector/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Mode represents the agent operating mode
type Mode string

const (
	ModeInit    Mode = "init"
	ModeSidecar Mode = "sidecar"
)

// SecretConfig represents a single secret to fetch
type SecretConfig struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Format   string   `json:"format"`
	Fields   []string `json:"fields,omitempty"`
	Notation string   `json:"notation,omitempty"` // Keeper notation (e.g., keeper://UID/field/password)
	FileName string   `json:"fileName,omitempty"` // For file attachments
	IsFile   bool     `json:"isFile,omitempty"`   // Whether this is a file attachment
}

// FolderConfig represents a folder to fetch all secrets from
type FolderConfig struct {
	FolderUID  string `json:"folderUid,omitempty"`
	FolderPath string `json:"folderPath,omitempty"`
	OutputPath string `json:"outputPath"`
}

// AgentConfig holds the agent configuration
type AgentConfig struct {
	Mode            Mode
	Secrets         []SecretConfig
	Folders         []FolderConfig
	RefreshInterval time.Duration
	FailOnError     bool
	StrictLookup    bool
	RefreshSignal   string
	KSMConfig       string // Base64-encoded KSM config (for secret auth)
	AuthMethod      string // Auth method: "secret" (default) or "oidc"
	Logger          *zap.Logger
}

// Agent manages secret fetching and rotation
type Agent struct {
	config    *AgentConfig
	ksmClient *ksm.Client
	logger    *zap.Logger
	mu        sync.RWMutex
	lastFetch map[string]time.Time
	healthy   bool
	ready     bool
}

// NewAgent creates a new secrets agent
func NewAgent(cfg *AgentConfig) (*Agent, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	return &Agent{
		config:    cfg,
		logger:    cfg.Logger,
		lastFetch: make(map[string]time.Time),
		healthy:   true,
		ready:     false,
	}, nil
}

// Run starts the agent in the configured mode
func (a *Agent) Run(ctx context.Context) error {
	// Determine auth method
	authMethod := ksm.AuthMethodSecret
	if a.config.AuthMethod == "oidc" {
		authMethod = ksm.AuthMethodOIDC
	}

	// Initialize KSM client
	ksmCfg := ksm.Config{
		ConfigJSON:  a.config.KSMConfig,
		AuthMethod:  authMethod,
		StrictMatch: a.config.StrictLookup,
		Logger:      a.logger,
	}

	client, err := ksm.NewClient(ctx, ksmCfg)
	if err != nil {
		return fmt.Errorf("failed to create KSM client: %w", err)
	}
	a.ksmClient = client
	defer client.Close()

	// Initial fetch
	if err := a.fetchAllSecrets(ctx); err != nil {
		if a.config.FailOnError {
			return fmt.Errorf("initial secret fetch failed: %w", err)
		}
		a.logger.Error("initial secret fetch failed, continuing anyway", zap.Error(err))
	}
	a.ready = true

	// If init mode, we're done
	if a.config.Mode == ModeInit {
		a.logger.Info("init mode complete, secrets written successfully")
		return nil
	}

	// Sidecar mode: start health server and refresh loop
	return a.runSidecarMode(ctx)
}

// runSidecarMode runs the continuous refresh loop
func (a *Agent) runSidecarMode(ctx context.Context) error {
	// Start health server
	go a.startHealthServer()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	// Refresh ticker
	ticker := time.NewTicker(a.config.RefreshInterval)
	defer ticker.Stop()

	a.logger.Info("starting sidecar mode",
		zap.Duration("refreshInterval", a.config.RefreshInterval))

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("context cancelled, shutting down")
			return nil

		case sig := <-sigChan:
			a.logger.Info("received signal, shutting down", zap.String("signal", sig.String()))
			return nil

		case <-ticker.C:
			if err := a.fetchAllSecrets(ctx); err != nil {
				a.logger.Error("secret refresh failed", zap.Error(err))
				// Don't mark unhealthy on refresh failure - keep last good values
			} else {
				a.logger.Debug("secrets refreshed successfully")
			}
		}
	}
}

// fetchAllSecrets fetches all configured secrets and folders
func (a *Agent) fetchAllSecrets(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	var errors []error
	totalSecrets := 0

	// Fetch individual secrets
	for _, secretCfg := range a.config.Secrets {
		startTime := time.Now()
		if err := a.fetchSecret(ctx, secretCfg); err != nil {
			a.logger.Error("failed to fetch secret",
				zap.String("name", secretCfg.Name),
				zap.Error(err))
			errors = append(errors, err)
			metrics.RecordSecretFetch(secretCfg.Name, false, time.Since(startTime).Seconds())
		} else {
			a.lastFetch[secretCfg.Name] = time.Now()
			metrics.RecordSecretFetch(secretCfg.Name, true, time.Since(startTime).Seconds())
			totalSecrets++
		}
	}

	// Fetch secrets from folders
	for _, folderCfg := range a.config.Folders {
		startTime := time.Now()
		count, err := a.fetchSecretsFromFolder(ctx, folderCfg)
		if err != nil {
			a.logger.Error("failed to fetch secrets from folder",
				zap.String("folder", folderCfg.FolderUID),
				zap.Error(err))
			errors = append(errors, err)
			metrics.RecordSecretFetch("folder:"+folderCfg.FolderUID, false, time.Since(startTime).Seconds())
		} else {
			a.lastFetch["folder:"+folderCfg.FolderUID] = time.Now()
			metrics.RecordSecretFetch("folder:"+folderCfg.FolderUID, true, time.Since(startTime).Seconds())
			totalSecrets += count
		}
	}

	// Update metrics
	metrics.SecretsActive.Set(float64(totalSecrets))
	if len(errors) == 0 {
		metrics.LastRefreshTimestamp.SetToCurrentTime()
		metrics.RecordRefreshCycle(true)
	} else {
		metrics.RecordRefreshCycle(false)
	}

	if len(errors) > 0 && a.config.FailOnError {
		return fmt.Errorf("failed to fetch %d secrets", len(errors))
	}

	return nil
}

// fetchSecret fetches a single secret and writes it to disk
func (a *Agent) fetchSecret(ctx context.Context, cfg SecretConfig) error {
	a.logger.Debug("fetching secret",
		zap.String("name", cfg.Name),
		zap.String("path", cfg.Path),
		zap.String("notation", cfg.Notation),
		zap.Bool("isFile", cfg.IsFile))

	var data []byte
	var err error

	// Handle different fetch modes
	switch {
	case cfg.Notation != "":
		// Use Keeper notation for fetching
		data, err = a.ksmClient.GetNotation(ctx, cfg.Notation)
		if err != nil {
			return fmt.Errorf("notation query failed: %w", err)
		}

	case cfg.IsFile:
		// Fetch file attachment
		data, err = a.ksmClient.GetFileContent(ctx, cfg.Name, cfg.FileName)
		if err != nil {
			return fmt.Errorf("failed to fetch file %s from %s: %w", cfg.FileName, cfg.Name, err)
		}

	case len(cfg.Fields) == 1:
		// Single field extraction
		data, err = a.ksmClient.GetSecretField(ctx, cfg.Name, cfg.Fields[0])
		if err != nil {
			return fmt.Errorf("failed to fetch field %s: %w", cfg.Fields[0], err)
		}

	default:
		// Full record or multiple fields
		secret, fetchErr := a.ksmClient.GetSecret(ctx, cfg.Name)
		if fetchErr != nil {
			return fetchErr
		}

		// Filter fields if specified
		if len(cfg.Fields) > 0 {
			filtered := make(map[string]interface{})
			for _, f := range cfg.Fields {
				if v, ok := secret.Fields[f]; ok {
					filtered[f] = v
				}
			}
			data, err = formatSecret(filtered, cfg.Format)
		} else {
			data, err = formatSecret(secret.Fields, cfg.Format)
		}

		if err != nil {
			return fmt.Errorf("failed to format secret: %w", err)
		}
	}

	// Write to file
	return a.writeSecretFile(cfg.Path, data)
}

// fetchSecretsFromFolder fetches all secrets from a folder
func (a *Agent) fetchSecretsFromFolder(ctx context.Context, cfg FolderConfig) (int, error) {
	a.logger.Debug("fetching secrets from folder",
		zap.String("folderUID", cfg.FolderUID),
		zap.String("folderPath", cfg.FolderPath),
		zap.String("outputPath", cfg.OutputPath))

	// Get secrets from folder by UID
	if cfg.FolderUID == "" {
		return 0, fmt.Errorf("folder UID is required (folder path lookup not yet implemented)")
	}

	secrets, err := a.ksmClient.GetSecretsInFolder(ctx, cfg.FolderUID)
	if err != nil {
		return 0, fmt.Errorf("failed to get secrets from folder: %w", err)
	}

	// Write each secret to a file
	count := 0
	for _, secret := range secrets {
		// Generate filename from title
		filename := sanitizeFilename(secret.Title) + ".json"
		path := filepath.Join(cfg.OutputPath, filename)

		data, err := json.MarshalIndent(secret.Fields, "", "  ")
		if err != nil {
			a.logger.Warn("failed to marshal secret from folder",
				zap.String("title", secret.Title),
				zap.Error(err))
			continue
		}

		if err := a.writeSecretFile(path, data); err != nil {
			a.logger.Warn("failed to write secret from folder",
				zap.String("title", secret.Title),
				zap.String("path", path),
				zap.Error(err))
			continue
		}
		count++
	}

	a.logger.Debug("fetched secrets from folder",
		zap.String("folderUID", cfg.FolderUID),
		zap.Int("count", count))

	return count, nil
}

// sanitizeFilename converts a string to a safe filename
func sanitizeFilename(name string) string {
	// Replace unsafe characters
	safe := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			safe = append(safe, c)
		} else if c == ' ' {
			safe = append(safe, '-')
		}
	}
	return string(safe)
}

// writeSecretFile writes secret data to the specified path
func (a *Agent) writeSecretFile(path string, data []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to temp file first, then rename (atomic)
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0400); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	a.logger.Debug("secret written", zap.String("path", path), zap.Int("bytes", len(data)))
	return nil
}

// formatSecret formats secret data according to the specified format
func formatSecret(data map[string]interface{}, format string) ([]byte, error) {
	switch format {
	case "json", "":
		return json.MarshalIndent(data, "", "  ")
	case "env":
		return formatAsEnv(data), nil
	case "raw":
		// For single values, return raw
		if len(data) == 1 {
			for _, v := range data {
				switch val := v.(type) {
				case string:
					return []byte(val), nil
				case []byte:
					return val, nil
				default:
					return json.Marshal(v)
				}
			}
		}
		return json.Marshal(data)
	default:
		return json.MarshalIndent(data, "", "  ")
	}
}

// formatAsEnv formats data as environment variable file
func formatAsEnv(data map[string]interface{}) []byte {
	var result []byte
	for k, v := range data {
		var value string
		switch val := v.(type) {
		case string:
			value = val
		case []byte:
			value = string(val)
		default:
			jsonBytes, _ := json.Marshal(v)
			value = string(jsonBytes)
		}
		// Escape quotes and newlines for env format
		value = escapeEnvValue(value)
		result = append(result, []byte(fmt.Sprintf("%s=%s\n", toEnvKey(k), value))...)
	}
	return result
}

// toEnvKey converts a field name to environment variable style
func toEnvKey(key string) string {
	// Convert to uppercase and replace non-alphanumeric with underscore
	result := make([]byte, 0, len(key))
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' {
			result = append(result, c-32) // To uppercase
		} else if c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {
			result = append(result, c)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// escapeEnvValue escapes a value for use in an env file
func escapeEnvValue(value string) string {
	// Simple quoting for values with special characters
	needsQuotes := false
	for _, c := range value {
		if c == ' ' || c == '\n' || c == '\t' || c == '"' || c == '\'' || c == '=' {
			needsQuotes = true
			break
		}
	}
	if needsQuotes {
		// Use single quotes and escape existing single quotes
		escaped := ""
		for _, c := range value {
			if c == '\'' {
				escaped += "'\\''"
			} else {
				escaped += string(c)
			}
		}
		return "'" + escaped + "'"
	}
	return value
}

// startHealthServer starts the health check HTTP server
func (a *Agent) startHealthServer() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if a.healthy {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("unhealthy"))
		}
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if a.ready {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready"))
		}
	})

	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	a.logger.Info("starting health server", zap.String("addr", ":8080"))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.logger.Error("health server error", zap.Error(err))
	}
}
