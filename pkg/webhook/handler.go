// Package webhook implements the Kubernetes mutating admission webhook for secret injection.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/keeper-security/keeper-k8s-injector/pkg/config"
	"github.com/keeper-security/keeper-k8s-injector/pkg/metrics"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// PodMutator handles pod mutation for secret injection
type PodMutator struct {
	Client  client.Client
	decoder admission.Decoder
	logger  *zap.Logger
	config  *WebhookConfig
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	// SidecarImage is the image to use for the sidecar container
	SidecarImage string
	// SidecarImagePullPolicy is the pull policy for the sidecar image
	SidecarImagePullPolicy corev1.PullPolicy
	// DefaultAuthSecretName is the default auth secret if not specified per-pod
	DefaultAuthSecretName string
	// DefaultRefreshInterval is the default refresh interval
	DefaultRefreshInterval string
	// ExcludedNamespaces are namespaces where injection is disabled
	ExcludedNamespaces []string
	// CPURequest for sidecar container
	CPURequest string
	// MemoryRequest for sidecar container
	MemoryRequest string
	// CPULimit for sidecar container
	CPULimit string
	// MemoryLimit for sidecar container
	MemoryLimit string
}

// DefaultWebhookConfig returns sensible defaults
func DefaultWebhookConfig() *WebhookConfig {
	return &WebhookConfig{
		SidecarImage:           "keeper/injector-sidecar:latest",
		SidecarImagePullPolicy: corev1.PullIfNotPresent,
		DefaultRefreshInterval: "5m",
		ExcludedNamespaces:     []string{"kube-system", "kube-public", "kube-node-lease"},
		CPURequest:             "10m",
		MemoryRequest:          "32Mi",
		CPULimit:               "50m",
		MemoryLimit:            "64Mi",
	}
}

// NewPodMutator creates a new pod mutator
func NewPodMutator(client client.Client, logger *zap.Logger, cfg *WebhookConfig) *PodMutator {
	if cfg == nil {
		cfg = DefaultWebhookConfig()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PodMutator{
		Client: client,
		logger: logger,
		config: cfg,
	}
}

// Handle implements admission.Handler
func (m *PodMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	startTime := time.Now()
	pod := &corev1.Pod{}

	err := m.decoder.Decode(req, pod)
	if err != nil {
		m.logger.Error("failed to decode pod", zap.Error(err))
		metrics.RecordMutation(req.Namespace, false, time.Since(startTime).Seconds(), 0)
		return admission.Errored(http.StatusBadRequest, err)
	}

	m.logger.Debug("handling pod mutation",
		zap.String("name", pod.Name),
		zap.String("namespace", pod.Namespace),
		zap.String("generateName", pod.GenerateName))

	// Check if namespace is excluded
	for _, ns := range m.config.ExcludedNamespaces {
		if req.Namespace == ns {
			m.logger.Debug("namespace excluded from injection", zap.String("namespace", ns))
			return admission.Allowed("namespace excluded from injection")
		}
	}

	// Check if injection is requested
	if !config.ShouldInject(pod) {
		m.logger.Debug("injection not requested for pod")
		return admission.Allowed("injection not requested")
	}

	// Parse injection configuration
	injectionConfig, err := config.ParseAnnotations(pod)
	if err != nil {
		m.logger.Error("failed to parse annotations", zap.Error(err))
		metrics.RecordMutation(req.Namespace, false, time.Since(startTime).Seconds(), 0)
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("invalid injection configuration: %w", err))
	}

	// Mutate the pod
	mutatedPod := pod.DeepCopy()
	if err := m.mutatePod(ctx, mutatedPod, injectionConfig); err != nil {
		m.logger.Error("failed to mutate pod", zap.Error(err))
		metrics.RecordMutation(req.Namespace, false, time.Since(startTime).Seconds(), 0)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Create patch
	marshaledPod, err := json.Marshal(mutatedPod)
	if err != nil {
		m.logger.Error("failed to marshal mutated pod", zap.Error(err))
		metrics.RecordMutation(req.Namespace, false, time.Since(startTime).Seconds(), 0)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Record successful mutation metrics
	metrics.RecordMutation(req.Namespace, true, time.Since(startTime).Seconds(), len(injectionConfig.Secrets))

	m.logger.Info("pod mutation successful",
		zap.String("name", pod.Name),
		zap.String("namespace", req.Namespace),
		zap.Int("secretCount", len(injectionConfig.Secrets)))

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// mutatePod adds the init container and/or sidecar to the pod
func (m *PodMutator) mutatePod(ctx context.Context, pod *corev1.Pod, cfg *config.InjectionConfig) error {
	// Add shared volume for secrets
	secretsVolume := corev1.Volume{
		Name: "keeper-secrets",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory, // tmpfs - memory backed
			},
		},
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, secretsVolume)

	// Add CA certificate volume if specified (for corporate proxies)
	if cfg.CACertSecret != "" || cfg.CACertConfigMap != "" {
		caCertVolume := corev1.Volume{
			Name: "keeper-ca-cert",
		}
		if cfg.CACertSecret != "" {
			caCertVolume.VolumeSource = corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: cfg.CACertSecret,
					Items: []corev1.KeyToPath{
						{
							Key:  cfg.CACertKey,
							Path: "ca.crt",
						},
					},
				},
			}
		} else {
			caCertVolume.VolumeSource = corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cfg.CACertConfigMap,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  cfg.CACertKey,
							Path: "ca.crt",
						},
					},
				},
			}
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, caCertVolume)
	}

	// Add volume mount to all existing containers
	secretsVolumeMount := corev1.VolumeMount{
		Name:      "keeper-secrets",
		MountPath: config.DefaultSecretsPath,
		ReadOnly:  true,
	}
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].VolumeMounts = append(
			pod.Spec.Containers[i].VolumeMounts,
			secretsVolumeMount,
		)
	}

	// Build sidecar config JSON
	sidecarConfig := m.buildSidecarConfig(cfg)
	sidecarConfigJSON, err := json.Marshal(sidecarConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal sidecar config: %w", err)
	}

	// Create init container (always runs first to ensure secrets exist at startup)
	initContainer := m.buildInitContainer(cfg, string(sidecarConfigJSON))
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, initContainer)

	// Create sidecar container (for rotation, unless init-only)
	if !cfg.InitOnly {
		sidecarContainer := m.buildSidecarContainer(cfg, string(sidecarConfigJSON))
		pod.Spec.Containers = append(pod.Spec.Containers, sidecarContainer)
	}

	// Inject environment variables (if enabled)
	if err := m.injectEnvironmentVariables(ctx, pod, cfg); err != nil {
		return fmt.Errorf("failed to inject environment variables: %w", err)
	}

	// Add annotation to indicate injection occurred (for GitOps compatibility)
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	pod.Annotations["keeper.security/injected"] = "true"

	return nil
}

// buildInitContainer creates the init container spec
func (m *PodMutator) buildInitContainer(cfg *config.InjectionConfig, configJSON string) corev1.Container {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "keeper-secrets",
			MountPath: config.DefaultSecretsPath,
		},
	}

	// Add CA cert volume mount if configured
	if cfg.CACertSecret != "" || cfg.CACertConfigMap != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "keeper-ca-cert",
			MountPath: "/usr/local/share/ca-certificates/keeper-ca.crt",
			SubPath:   "ca.crt",
			ReadOnly:  true,
		})
	}

	return corev1.Container{
		Name:            "keeper-secrets-init",
		Image:           m.config.SidecarImage,
		ImagePullPolicy: m.config.SidecarImagePullPolicy,
		Args:            []string{"--mode=init"},
		Env: []corev1.EnvVar{
			{
				Name:  "KEEPER_CONFIG",
				Value: configJSON,
			},
			{
				Name: "KEEPER_AUTH_CONFIG",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cfg.AuthSecretName,
						},
						Key: "config",
					},
				},
			},
		},
		VolumeMounts: volumeMounts,
		Resources: m.buildResourceRequirements(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}
}

// buildSidecarContainer creates the sidecar container spec
func (m *PodMutator) buildSidecarContainer(cfg *config.InjectionConfig, configJSON string) corev1.Container {
	args := []string{
		"--mode=sidecar",
		fmt.Sprintf("--refresh-interval=%s", cfg.RefreshInterval),
	}
	if cfg.Signal != "" {
		args = append(args, fmt.Sprintf("--signal=%s", cfg.Signal))
	}

	return corev1.Container{
		Name:            "keeper-secrets-sidecar",
		Image:           m.config.SidecarImage,
		ImagePullPolicy: m.config.SidecarImagePullPolicy,
		Args:            args,
		Env: []corev1.EnvVar{
			{
				Name:  "KEEPER_CONFIG",
				Value: configJSON,
			},
			{
				Name: "KEEPER_AUTH_CONFIG",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: cfg.AuthSecretName,
						},
						Key: "config",
					},
				},
			},
		},
		VolumeMounts: m.buildVolumeMounts(cfg),
		Resources: m.buildResourceRequirements(),
		SecurityContext: &corev1.SecurityContext{
			RunAsNonRoot:             boolPtr(true),
			ReadOnlyRootFilesystem:   boolPtr(true),
			AllowPrivilegeEscalation: boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/healthz",
					Port: intStrFromInt(8080),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/readyz",
					Port: intStrFromInt(8080),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       10,
		},
	}
}

// buildVolumeMounts creates volume mounts for init and sidecar containers
func (m *PodMutator) buildVolumeMounts(cfg *config.InjectionConfig) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      "keeper-secrets",
			MountPath: config.DefaultSecretsPath,
		},
	}

	// Add CA cert mount if configured (for corporate proxies/SSL inspection)
	if cfg.CACertSecret != "" || cfg.CACertConfigMap != "" {
		mounts = append(mounts, corev1.VolumeMount{
			Name:      "keeper-ca-cert",
			MountPath: "/usr/local/share/ca-certificates/keeper-ca.crt",
			SubPath:   "ca.crt",
			ReadOnly:  true,
		})
	}

	return mounts
}

// buildResourceRequirements creates resource requirements for containers
func (m *PodMutator) buildResourceRequirements() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    mustParseQuantity(m.config.CPURequest),
			corev1.ResourceMemory: mustParseQuantity(m.config.MemoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    mustParseQuantity(m.config.CPULimit),
			corev1.ResourceMemory: mustParseQuantity(m.config.MemoryLimit),
		},
	}
}

// buildSidecarConfig creates the configuration passed to the sidecar
func (m *PodMutator) buildSidecarConfig(cfg *config.InjectionConfig) map[string]interface{} {
	secrets := make([]map[string]interface{}, 0, len(cfg.Secrets))
	for _, s := range cfg.Secrets {
		secret := map[string]interface{}{
			"name":   s.Name,
			"path":   s.Path,
			"format": s.Format,
		}
		if len(s.Fields) > 0 {
			secret["fields"] = s.Fields
		}
		if s.Notation != "" {
			secret["notation"] = s.Notation
		}
		if s.Template != "" {
			secret["template"] = s.Template
		}
		if s.FileName != "" {
			secret["fileName"] = s.FileName
		}
		if s.IsFile {
			secret["isFile"] = s.IsFile
		}
		secrets = append(secrets, secret)
	}

	// Build folder configs
	folders := make([]map[string]interface{}, 0, len(cfg.Folders))
	for _, f := range cfg.Folders {
		folder := map[string]interface{}{
			"outputPath": f.OutputPath,
		}
		if f.FolderUID != "" {
			folder["folderUid"] = f.FolderUID
		}
		if f.FolderPath != "" {
			folder["folderPath"] = f.FolderPath
		}
		folders = append(folders, folder)
	}

	result := map[string]interface{}{
		"secrets":       secrets,
		"failOnError":   cfg.FailOnError,
		"strictLookup":  cfg.StrictLookup,
		"refreshSignal": cfg.Signal,
		"authMethod":    cfg.AuthMethod,
	}

	if len(folders) > 0 {
		result["folders"] = folders
	}

	// Add cloud provider configuration if present
	if cfg.AWSSecretID != "" {
		result["awsSecretId"] = cfg.AWSSecretID
		if cfg.AWSRegion != "" {
			result["awsRegion"] = cfg.AWSRegion
		}
	}
	if cfg.GCPSecretID != "" {
		result["gcpSecretId"] = cfg.GCPSecretID
	}
	if cfg.AzureVaultName != "" && cfg.AzureSecretName != "" {
		result["azureVaultName"] = cfg.AzureVaultName
		result["azureSecretName"] = cfg.AzureSecretName
	}

	return result
}

// InjectDecoder injects the decoder
func (m *PodMutator) InjectDecoder(d admission.Decoder) error {
	m.decoder = d
	return nil
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intStrFromInt(i int) intstr.IntOrString {
	return intstr.FromInt32(int32(i))
}

func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("invalid quantity %s: %v", s, err))
	}
	return q
}
