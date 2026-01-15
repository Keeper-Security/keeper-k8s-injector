// Package main is the entry point for the Keeper webhook controller.
package main

import (
	"flag"
	"os"

	"github.com/keeper-security/keeper-k8s-injector/pkg/webhook"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	ctrlwebhook "sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = corev1.AddToScheme(scheme)
}

func main() {
	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		certDir              string
		sidecarImage         string
		logLevel             string
		logFormat            string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/etc/webhook/certs", "Directory containing TLS certificates.")
	flag.StringVar(&sidecarImage, "sidecar-image", "keeper/injector-sidecar:latest", "Image for the sidecar container.")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error).")
	flag.StringVar(&logFormat, "log-format", "json", "Log format (json, console).")
	flag.Parse()

	// Set up logger
	logger := setupLogger(logLevel, logFormat)
	ctrl.SetLogger(ctrlzap.New(ctrlzap.UseDevMode(logFormat == "console")))

	logger.Info("starting Keeper webhook controller",
		zap.String("sidecarImage", sidecarImage),
		zap.String("certDir", certDir))

	// Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "keeper-injector-leader",
		WebhookServer: ctrlwebhook.NewServer(ctrlwebhook.Options{
			Port:    9443,
			CertDir: certDir,
		}),
	})
	if err != nil {
		logger.Fatal("unable to create manager", zap.Error(err))
	}

	// Configure webhook
	webhookCfg := &webhook.WebhookConfig{
		SidecarImage:           sidecarImage,
		SidecarImagePullPolicy: corev1.PullIfNotPresent,
		DefaultRefreshInterval: getEnvOrDefault("KEEPER_REFRESH_INTERVAL", "5m"),
		ExcludedNamespaces:     []string{"kube-system", "kube-public", "kube-node-lease"},
		CPURequest:             getEnvOrDefault("KEEPER_SIDECAR_CPU_REQUEST", "10m"),
		MemoryRequest:          getEnvOrDefault("KEEPER_SIDECAR_MEMORY_REQUEST", "32Mi"),
		CPULimit:               getEnvOrDefault("KEEPER_SIDECAR_CPU_LIMIT", "50m"),
		MemoryLimit:            getEnvOrDefault("KEEPER_SIDECAR_MEMORY_LIMIT", "64Mi"),
	}

	// Create decoder for webhook
	decoder := admission.NewDecoder(scheme)

	// Create and register mutating webhook
	mutator := webhook.NewPodMutator(mgr.GetClient(), logger, webhookCfg)
	if err := mutator.InjectDecoder(decoder); err != nil {
		logger.Fatal("failed to inject decoder", zap.Error(err))
	}
	mgr.GetWebhookServer().Register("/mutate-pods", &ctrlwebhook.Admission{Handler: mutator})

	// Add health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up health check", zap.Error(err))
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		logger.Fatal("unable to set up ready check", zap.Error(err))
	}

	logger.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logger.Fatal("problem running manager", zap.Error(err))
	}
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

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
