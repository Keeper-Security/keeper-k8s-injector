// Package main is the entry point for the Keeper sidecar agent.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"time"

	"github.com/keeper-security/keeper-k8s-injector/pkg/sidecar"
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
	AuthMethod    string        `json:"authMethod,omitempty"` // "secret" (default) or "oidc"
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

	// Parse configuration from environment
	configJSON := os.Getenv("KEEPER_CONFIG")
	if configJSON == "" {
		logger.Fatal("KEEPER_CONFIG environment variable not set")
	}

	var cfg secretsConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		logger.Fatal("failed to parse KEEPER_CONFIG", zap.Error(err))
	}

	// Get KSM auth config
	ksmConfig := os.Getenv("KEEPER_AUTH_CONFIG")
	if ksmConfig == "" {
		logger.Fatal("KEEPER_AUTH_CONFIG environment variable not set")
	}

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

	ctx := context.Background()
	if err := agent.Run(ctx); err != nil {
		logger.Fatal("agent failed", zap.Error(err))
	}

	logger.Info("agent completed successfully")
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
