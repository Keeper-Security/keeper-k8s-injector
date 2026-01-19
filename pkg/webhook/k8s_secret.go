// Package webhook implements Kubernetes Secret injection for secrets.
package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keeper-security/keeper-k8s-injector/pkg/config"
	"github.com/keeper-security/keeper-k8s-injector/pkg/ksm"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// MaxSecretSize is the maximum size of a K8s Secret (1MB)
	MaxSecretSize = 1024 * 1024
)

// injectK8sSecrets creates K8s Secret objects from Keeper secrets.
// This is called during pod admission (webhook time) to create/update Secrets.
func (m *PodMutator) injectK8sSecrets(ctx context.Context, pod *corev1.Pod, cfg *config.InjectionConfig) error {
	if !cfg.InjectAsK8sSecret {
		return nil
	}

	// Filter secrets that should become K8s Secrets
	k8sSecrets := filterK8sSecretConfigs(cfg)
	if len(k8sSecrets) == 0 {
		m.logger.Debug("no secrets configured for K8s Secret injection")
		return nil
	}

	m.logger.Info("injecting Kubernetes Secrets",
		zap.Int("secretCount", len(k8sSecrets)),
		zap.String("pod", pod.Name))

	// Create KSM client (reuse from envvar.go pattern)
	ksmClient, err := m.createKSMClient(ctx, pod.Namespace, cfg)
	if err != nil {
		if cfg.FailOnError {
			return fmt.Errorf("failed to create KSM client: %w", err)
		}
		m.logger.Warn("failed to create KSM client, skipping K8s Secret injection", zap.Error(err))
		return nil
	}
	defer func() {
		if closeErr := ksmClient.Close(); closeErr != nil {
			m.logger.Warn("failed to close KSM client", zap.Error(closeErr))
		}
	}()

	// Fetch all secrets in ONE call (efficient batching)
	secretsData, err := m.batchFetchSecrets(ctx, ksmClient, k8sSecrets, cfg)
	if err != nil {
		if cfg.FailOnError {
			return fmt.Errorf("failed to fetch secrets: %w", err)
		}
		m.logger.Error("failed to fetch secrets, continuing", zap.Error(err))
	}

	// Create/update K8s Secrets
	for i, secretRef := range k8sSecrets {
		data, ok := secretsData[i]
		if !ok {
			continue // Secret was skipped (e.g., not found and fail-on-error=false)
		}

		k8sSecret, err := m.buildK8sSecret(pod, secretRef, data, cfg)
		if err != nil {
			return fmt.Errorf("failed to build K8s Secret: %w", err)
		}

		// Validate size
		if err := validateSecretSize(k8sSecret); err != nil {
			return fmt.Errorf("K8s Secret %s exceeds size limit: %w", k8sSecret.Name, err)
		}

		if err := m.createOrUpdateSecret(ctx, k8sSecret, cfg.K8sSecretMode, cfg.K8sSecretOwnerRef); err != nil {
			return fmt.Errorf("failed to create/update K8s Secret %s: %w", k8sSecret.Name, err)
		}

		m.logger.Info("created/updated K8s Secret",
			zap.String("name", k8sSecret.Name),
			zap.String("namespace", k8sSecret.Namespace))
	}

	return nil
}

// filterK8sSecretConfigs returns only secrets that should become K8s Secrets
func filterK8sSecretConfigs(cfg *config.InjectionConfig) []config.SecretRef {
	var k8sSecrets []config.SecretRef
	for _, secret := range cfg.Secrets {
		// Check if this secret should be injected as K8s Secret
		if secret.InjectAsK8sSecret || cfg.InjectAsK8sSecret {
			k8sSecrets = append(k8sSecrets, secret)
		}
	}
	return k8sSecrets
}

// batchFetchSecrets fetches all secrets efficiently with ONE Keeper API call.
// This is a key optimization: instead of N API calls, we make 1 call to list all records.
func (m *PodMutator) batchFetchSecrets(ctx context.Context, ksmClient *ksm.Client, secrets []config.SecretRef, cfg *config.InjectionConfig) (map[int]*ksm.SecretData, error) {
	result := make(map[int]*ksm.SecretData)

	// OPTIMIZATION: Fetch all records in ONE API call
	allRecords, err := ksmClient.ListSecrets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to batch fetch secrets: %w", err)
	}

	// Build lookup maps
	recordsByUID := make(map[string]*ksm.SecretData)
	recordsByTitle := make(map[string]*ksm.SecretData)
	for _, record := range allRecords {
		recordsByUID[record.RecordUID] = record
		recordsByTitle[record.Title] = record
	}

	// Map fetched records to SecretRefs
	notationCount := 0
	fileCount := 0

	for i, secretRef := range secrets {
		var data *ksm.SecretData

		switch {
		case secretRef.Notation != "":
			// Individual call for notation
			notationData, err := ksmClient.GetNotation(ctx, secretRef.Notation)
			if err != nil {
				return nil, fmt.Errorf("notation %s failed: %w", secretRef.Notation, err)
			}
			data = &ksm.SecretData{
				Fields: map[string]interface{}{
					"value": string(notationData),
				},
			}
			notationCount++

		case secretRef.IsFile:
			// Individual call for file
			fileData, err := ksmClient.GetFileContent(ctx, secretRef.Name, secretRef.FileName)
			if err != nil {
				return nil, fmt.Errorf("file %s fetch failed: %w", secretRef.FileName, err)
			}
			data = &ksm.SecretData{
				Fields: map[string]interface{}{
					secretRef.FileName: fileData,
				},
			}
			fileCount++

		default:
			// Use batched data
			if looksLikeUID(secretRef.Name) {
				data = recordsByUID[secretRef.Name]
			} else {
				data = recordsByTitle[secretRef.Name]
			}

			if data == nil {
				if cfg.FailOnError {
					return nil, fmt.Errorf("record %s not found", secretRef.Name)
				}
				m.logger.Warn("record not found, skipping",
					zap.String("name", secretRef.Name))
				continue
			}
		}

		result[i] = data
	}

	m.logger.Info("batch fetched secrets",
		zap.Int("total", len(secrets)),
		zap.Int("batched", len(secrets)-notationCount-fileCount),
		zap.Int("notation_calls", notationCount),
		zap.Int("file_calls", fileCount))

	return result, nil
}

// buildK8sSecret constructs a K8s Secret from Keeper data
func (m *PodMutator) buildK8sSecret(pod *corev1.Pod, secretRef config.SecretRef, data *ksm.SecretData, cfg *config.InjectionConfig) (*corev1.Secret, error) {
	namespace := cfg.K8sSecretNamespace
	if namespace == "" {
		namespace = pod.Namespace
	}

	secretName := secretRef.K8sSecretName
	if secretName == "" {
		secretName = cfg.K8sSecretName
	}
	if secretName == "" {
		return nil, fmt.Errorf("k8s secret name not specified for secret %s", secretRef.Name)
	}

	// Build Secret data
	secretData := make(map[string][]byte)

	if len(secretRef.K8sSecretKeys) > 0 {
		// Custom key mapping
		for keeperField, k8sKey := range secretRef.K8sSecretKeys {
			if value, ok := data.Fields[keeperField]; ok {
				secretData[k8sKey] = valueToBytes(value)
			}
		}
	} else if len(secretRef.Fields) > 0 {
		// Selected fields (use field names as keys)
		for _, field := range secretRef.Fields {
			if value, ok := data.Fields[field]; ok {
				secretData[field] = valueToBytes(value)
			}
		}
	} else {
		// All fields as individual keys
		for field, value := range data.Fields {
			secretData[field] = valueToBytes(value)
		}
	}

	// Determine Secret type
	secretType := corev1.SecretTypeOpaque
	if secretRef.K8sSecretType != "" {
		secretType = corev1.SecretType(secretRef.K8sSecretType)
	} else if cfg.K8sSecretType != "" {
		secretType = corev1.SecretType(cfg.K8sSecretType)
	}

	// Build owner references
	var ownerRefs []metav1.OwnerReference
	if cfg.K8sSecretOwnerRef {
		ownerRefs = []metav1.OwnerReference{
			{
				APIVersion: "v1",
				Kind:       "Pod",
				Name:       pod.Name,
				UID:        pod.UID,
				Controller: boolPtrK8s(true),
			},
		}
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "keeper-injector",
				"keeper.security/injected":     "true",
			},
			Annotations: map[string]string{
				"keeper.security/source-pod":    pod.Name,
				"keeper.security/source-record": secretRef.Name,
			},
			OwnerReferences: ownerRefs,
		},
		Type: secretType,
		Data: secretData,
	}, nil
}

// createOrUpdateSecret handles Secret creation with conflict resolution
func (m *PodMutator) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret, mode string, ownerRefEnabled bool) error {
	existing := &corev1.Secret{}
	err := m.Client.Get(ctx, client.ObjectKeyFromObject(secret), existing)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// Secret doesn't exist, create it
			return m.Client.Create(ctx, secret)
		}
		return fmt.Errorf("failed to check existing secret: %w", err)
	}

	// Secret exists, handle based on mode
	switch mode {
	case "fail":
		return fmt.Errorf("secret %s already exists (mode: fail)", secret.Name)

	case "skip-if-exists":
		m.logger.Info("skipping existing secret", zap.String("name", secret.Name))
		return nil

	case "merge":
		// Merge data (new keys override, existing keys preserved)
		if existing.Data == nil {
			existing.Data = make(map[string][]byte)
		}
		for k, v := range secret.Data {
			existing.Data[k] = v
		}
		// Update labels and annotations
		if existing.Labels == nil {
			existing.Labels = make(map[string]string)
		}
		for k, v := range secret.Labels {
			existing.Labels[k] = v
		}
		if existing.Annotations == nil {
			existing.Annotations = make(map[string]string)
		}
		for k, v := range secret.Annotations {
			existing.Annotations[k] = v
		}
		// Update owner reference based on setting
		if ownerRefEnabled && len(secret.OwnerReferences) > 0 {
			existing.OwnerReferences = secret.OwnerReferences
		}
		return m.Client.Update(ctx, existing)

	case "overwrite", "":
		// Replace all data
		existing.Data = secret.Data
		existing.Labels = secret.Labels
		existing.Annotations = secret.Annotations
		// Update owner reference based on setting
		if ownerRefEnabled {
			existing.OwnerReferences = secret.OwnerReferences
		}
		return m.Client.Update(ctx, existing)

	default:
		return fmt.Errorf("unknown k8s-secret-mode: %s (valid: overwrite, merge, skip-if-exists, fail)", mode)
	}
}

// validateSecretSize checks if Secret exceeds K8s size limit (1MB)
func validateSecretSize(secret *corev1.Secret) error {
	totalSize := 0
	for k, v := range secret.Data {
		totalSize += len(k) + len(v)
	}
	if totalSize > MaxSecretSize {
		return fmt.Errorf("secret size %d bytes exceeds maximum %d bytes", totalSize, MaxSecretSize)
	}
	return nil
}

// looksLikeUID returns true if the string looks like a Keeper UID (22 chars)
func looksLikeUID(s string) bool {
	return len(s) == 22 && !strings.Contains(s, " ")
}

// valueToBytes converts a value to []byte for K8s Secret
func valueToBytes(value interface{}) []byte {
	switch v := value.(type) {
	case string:
		return []byte(v)
	case []byte:
		return v
	default:
		// JSON encode complex types
		data, _ := json.Marshal(v)
		return data
	}
}

// boolPtrK8s returns a pointer to a bool
func boolPtrK8s(b bool) *bool {
	return &b
}
