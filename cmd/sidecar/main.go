// Package main is the entry point for the Keeper sidecar agent.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/keeper-security/keeper-k8s-injector/pkg/sidecar"
	"github.com/keeper-security/keeper-k8s-injector/pkg/sidecar/cloud"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// secretsConfig is the JSON configuration structure passed via environment
type secretsConfig struct {
	Secrets       []secretEntry `json:"secrets"`
	Folders       []folderEntry `json:"folders,omitempty"`
	FailOnError   bool          `json:"failOnError"`
	StrictLookup  bool          `json:"strictLookup"`
	RefreshSignal string        `json:"refreshSignal"`
	AuthMethod    string        `json:"authMethod,omitempty"` // "secret", "aws-secrets-manager", "gcp-secret-manager", "azure-key-vault"

	// Cloud provider configuration
	AWSSecretID     string `json:"awsSecretId,omitempty"`
	AWSRegion       string `json:"awsRegion,omitempty"`
	GCPSecretID     string `json:"gcpSecretId,omitempty"`
	AzureVaultName  string `json:"azureVaultName,omitempty"`
	AzureSecretName string `json:"azureSecretName,omitempty"`
}

type secretEntry struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Format   string   `json:"format"`
	Fields   []string `json:"fields,omitempty"`
	Notation string   `json:"notation,omitempty"`
	FileName string   `json:"fileName,omitempty"`
	IsFile   bool     `json:"isFile,omitempty"`
}

type folderEntry struct {
	FolderUID  string `json:"folderUid,omitempty"`
	FolderPath string `json:"folderPath,omitempty"`
	OutputPath string `json:"outputPath"`
}

func main() {
	var (
		mode            string
		refreshInterval time.Duration
		logLevel        string
		logFormat       string
	)

	flag.StringVar(&mode, "mode", "sidecar", "Operating mode: init or sidecar")
	flag.DurationVar(&refreshInterval, "refresh-interval", 5*time.Minute, "Secret refresh interval (sidecar mode only)")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.StringVar(&logFormat, "log-format", "json", "Log format (json, console)")
	flag.Parse()

	// Set up logger
	logger := setupLogger(logLevel, logFormat)

	logger.Info("starting Keeper sidecar agent",
		zap.String("mode", mode),
		zap.Duration("refreshInterval", refreshInterval))

	// Load custom CA certificate if present (for corporate proxies)
	if err := loadCustomCACert(logger); err != nil {
		logger.Warn("failed to load custom CA certificate", zap.Error(err))
	}

	// Parse configuration from environment
	configJSON := os.Getenv("KEEPER_CONFIG")
	if configJSON == "" {
		logger.Fatal("KEEPER_CONFIG environment variable not set")
	}

	var cfg secretsConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		logger.Fatal("failed to parse KEEPER_CONFIG", zap.Error(err))
	}

	// Create context for cloud SDK calls
	ctx := context.Background()

	// Get KSM auth config from cloud provider or K8s Secret
	var ksmConfig string
	var err error

	switch cfg.AuthMethod {
	case "aws-secrets-manager":
		logger.Info("fetching KSM config from AWS Secrets Manager",
			zap.String("secretId", cfg.AWSSecretID),
			zap.String("region", cfg.AWSRegion))

		ksmConfig, err = cloud.FetchKSMConfigFromAWS(ctx, cfg.AWSSecretID, cfg.AWSRegion)
		if err != nil {
			logger.Fatal("failed to fetch KSM config from AWS", zap.Error(err))
		}
		logger.Info("successfully fetched KSM config from AWS Secrets Manager")

	case "gcp-secret-manager":
		logger.Info("fetching KSM config from GCP Secret Manager",
			zap.String("secretId", cfg.GCPSecretID))

		ksmConfig, err = cloud.FetchKSMConfigFromGCP(ctx, cfg.GCPSecretID)
		if err != nil {
			logger.Fatal("failed to fetch KSM config from GCP", zap.Error(err))
		}
		logger.Info("successfully fetched KSM config from GCP Secret Manager")

	case "azure-key-vault":
		logger.Info("fetching KSM config from Azure Key Vault",
			zap.String("vaultName", cfg.AzureVaultName),
			zap.String("secretName", cfg.AzureSecretName))

		ksmConfig, err = cloud.FetchKSMConfigFromAzure(ctx, cfg.AzureVaultName, cfg.AzureSecretName)
		if err != nil {
			logger.Fatal("failed to fetch KSM config from Azure", zap.Error(err))
		}
		logger.Info("successfully fetched KSM config from Azure Key Vault")

	case "secret", "":
		// Default: read from K8s Secret via environment variable
		ksmConfig = os.Getenv("KEEPER_AUTH_CONFIG")
		if ksmConfig == "" {
			logger.Fatal("KEEPER_AUTH_CONFIG environment variable not set")
		}
		logger.Debug("using KSM config from Kubernetes Secret")

	default:
		logger.Fatal("unknown auth method",
			zap.String("authMethod", cfg.AuthMethod),
			zap.Strings("supported", []string{"secret", "aws-secrets-manager", "gcp-secret-manager", "azure-key-vault"}))
	}

	// Validate KSM config format before use
	if err := validateKSMConfig(ksmConfig); err != nil {
		logger.Fatal("invalid KSM configuration", zap.Error(err))
	}
	logger.Debug("KSM configuration validated successfully")

	// Convert to agent config
	secrets := make([]sidecar.SecretConfig, len(cfg.Secrets))
	for i, s := range cfg.Secrets {
		secrets[i] = sidecar.SecretConfig{
			Name:     s.Name,
			Path:     s.Path,
			Format:   s.Format,
			Fields:   s.Fields,
			Notation: s.Notation,
			FileName: s.FileName,
			IsFile:   s.IsFile,
		}
	}

	// Convert folders
	folders := make([]sidecar.FolderConfig, len(cfg.Folders))
	for i, f := range cfg.Folders {
		folders[i] = sidecar.FolderConfig{
			FolderUID:  f.FolderUID,
			FolderPath: f.FolderPath,
			OutputPath: f.OutputPath,
		}
	}

	agentMode := sidecar.ModeSidecar
	if mode == "init" {
		agentMode = sidecar.ModeInit
	}

	agentCfg := &sidecar.AgentConfig{
		Mode:            agentMode,
		Secrets:         secrets,
		Folders:         folders,
		RefreshInterval: refreshInterval,
		FailOnError:     cfg.FailOnError,
		StrictLookup:    cfg.StrictLookup,
		RefreshSignal:   cfg.RefreshSignal,
		KSMConfig:       ksmConfig,
		AuthMethod:      cfg.AuthMethod,
		Logger:          logger,
	}

	// Create and run agent
	agent, err := sidecar.NewAgent(agentCfg)
	if err != nil {
		logger.Fatal("failed to create agent", zap.Error(err))
	}

	if err := agent.Run(ctx); err != nil {
		logger.Fatal("agent failed", zap.Error(err))
	}

	logger.Info("agent completed successfully")
}

// loadCustomCACert loads custom CA certificate for corporate proxies/SSL inspection.
// Supports environments like Zscaler, Palo Alto, Cisco Umbrella, etc.
func loadCustomCACert(logger *zap.Logger) error {
	const caCertPath = "/usr/local/share/ca-certificates/keeper-ca.crt"

	// Check if custom CA cert exists
	certPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No custom CA cert configured - this is normal
			return nil
		}
		return err
	}

	logger.Info("loading custom CA certificate", zap.String("path", caCertPath))

	// Get system cert pool
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		// If system pool fails, create new pool
		rootCAs = x509.NewCertPool()
	}

	// Append custom CA cert
	if !rootCAs.AppendCertsFromPEM(certPEM) {
		return err
	}

	// Configure default HTTP transport with custom cert pool
	http.DefaultTransport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    rootCAs,
			MinVersion: tls.VersionTLS12,
		},
	}

	logger.Info("custom CA certificate loaded successfully")
	return nil
}

// validateKSMConfig validates the KSM configuration format.
// Accepts either base64-encoded or plain JSON config.
// Returns error if config is invalid or missing required fields.
func validateKSMConfig(config string) error {
	if config == "" {
		return fmt.Errorf("KSM config is empty")
	}

	trimmed := strings.TrimSpace(config)

	// Try to parse as JSON first
	var test map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &test); err == nil {
		// Valid JSON - check for required fields
		if _, ok := test["clientId"]; !ok {
			return fmt.Errorf("KSM config missing clientId field")
		}
		return nil
	}

	// Not JSON - should be base64
	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return fmt.Errorf("KSM config is neither valid JSON nor base64: %w", err)
	}

	// Parse decoded as JSON
	if err := json.Unmarshal(decoded, &test); err != nil {
		return fmt.Errorf("decoded KSM config is not valid JSON: %w", err)
	}

	// Check for required fields
	if _, ok := test["clientId"]; !ok {
		return fmt.Errorf("KSM config missing clientId field")
	}

	return nil
}

func setupLogger(level, format string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	var config zap.Config
	if format == "console" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	return logger
}
