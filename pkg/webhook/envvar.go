// Package webhook implements environment variable injection for secrets.
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// injectEnvironmentVariables adds env vars to all containers for secrets marked with InjectAsEnvVars
func (m *PodMutator) injectEnvironmentVariables(ctx context.Context, pod *corev1.Pod, cfg *config.InjectionConfig) error {
	// Filter secrets that should be injected as env vars
	envSecrets := filterEnvVarSecrets(cfg)
	if len(envSecrets) == 0 {
		m.logger.Debug("no secrets configured for env var injection")
		return nil
	}

	m.logger.Info("injecting environment variables",
		zap.Int("secretCount", len(envSecrets)),
		zap.String("pod", pod.Name))

	// Create KSM client to fetch secrets
	ksmClient, err := m.createKSMClient(ctx, pod.Namespace, cfg)
	if err != nil {
		if cfg.FailOnError {
			return fmt.Errorf("failed to create KSM client: %w", err)
		}
		m.logger.Warn("failed to create KSM client, skipping env var injection", zap.Error(err))
		return nil
	}
	defer func() {
		if closeErr := ksmClient.Close(); closeErr != nil {
			m.logger.Warn("failed to close KSM client", zap.Error(closeErr))
		}
	}()

	// Fetch and convert each secret to env vars
	for _, secret := range envSecrets {
		envVars, err := m.buildEnvVarsFromSecret(ctx, ksmClient, secret, cfg)
		if err != nil {
			if cfg.FailOnError {
				return fmt.Errorf("failed to build env vars for secret %s: %w", secret.Name, err)
			}
			m.logger.Warn("failed to build env vars, skipping secret",
				zap.String("secret", secret.Name),
				zap.Error(err))
			continue
		}

		// Inject into all containers
		for i := range pod.Spec.Containers {
			pod.Spec.Containers[i].Env = append(pod.Spec.Containers[i].Env, envVars...)
		}

		m.logger.Debug("injected env vars from secret",
			zap.String("secret", secret.Name),
			zap.Int("varCount", len(envVars)))
	}

	return nil
}

// filterEnvVarSecrets returns only secrets that should be injected as env vars
func filterEnvVarSecrets(cfg *config.InjectionConfig) []config.SecretRef {
	var envSecrets []config.SecretRef
	for _, secret := range cfg.Secrets {
		// Skip file attachments (can't inject as env vars)
		if secret.IsFile {
			continue
		}
		// Check if this secret should be injected as env vars
		if secret.InjectAsEnvVars || cfg.InjectEnvVars {
			envSecrets = append(envSecrets, secret)
		}
	}
	return envSecrets
}

// createKSMClient creates a KSM client using credentials from K8s secret
func (m *PodMutator) createKSMClient(ctx context.Context, namespace string, cfg *config.InjectionConfig) (*ksm.Client, error) {
	// Fetch auth secret from K8s
	authSecret := &corev1.Secret{}
	secretKey := client.ObjectKey{
		Name:      cfg.AuthSecretName,
		Namespace: namespace,
	}
	if err := m.Client.Get(ctx, secretKey, authSecret); err != nil {
		return nil, fmt.Errorf("failed to fetch auth secret %s: %w", cfg.AuthSecretName, err)
	}

	// Extract KSM config from secret
	configData, ok := authSecret.Data["config"]
	if !ok {
		return nil, fmt.Errorf("auth secret %s does not contain 'config' key", cfg.AuthSecretName)
	}

	// Create KSM client
	ksmConfig := ksm.Config{
		ConfigJSON:  string(configData),
		AuthMethod:  ksm.AuthMethod(cfg.AuthMethod),
		StrictMatch: cfg.StrictLookup,
		Logger:      m.logger,
	}

	client, err := ksm.NewClient(ctx, ksmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create KSM client: %w", err)
	}

	return client, nil
}

// buildEnvVarsFromSecret fetches a secret and converts it to []EnvVar
func (m *PodMutator) buildEnvVarsFromSecret(ctx context.Context, ksmClient *ksm.Client, secret config.SecretRef, cfg *config.InjectionConfig) ([]corev1.EnvVar, error) {
	// Fetch secret data from KSM
	var secretData *ksm.SecretData
	var err error

	if secret.Notation != "" {
		// Handle Keeper notation (returns raw value)
		data, notationErr := ksmClient.GetNotation(ctx, secret.Notation)
		if notationErr != nil {
			return nil, fmt.Errorf("notation query failed: %w", notationErr)
		}
		// For notation, create a simple key-value env var
		key := secret.Name
		if secret.EnvVarPrefix != "" {
			key = secret.EnvVarPrefix + key
		} else if cfg.EnvPrefix != "" {
			key = cfg.EnvPrefix + key
		}
		return []corev1.EnvVar{
			{
				Name:  toEnvKey(key),
				Value: string(data),
			},
		}, nil
	}

	// Fetch by title or UID
	secretData, err = ksmClient.GetSecret(ctx, secret.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch secret: %w", err)
	}

	// Handle field extraction
	if len(secret.Fields) > 0 {
		var envVars []corev1.EnvVar
		for _, field := range secret.Fields {
			value, ok := secretData.Fields[field]
			if !ok {
				if cfg.FailOnError {
					return nil, fmt.Errorf("field %s not found in secret %s", field, secret.Name)
				}
				m.logger.Warn("field not found in secret",
					zap.String("field", field),
					zap.String("secret", secret.Name))
				continue
			}

			key := field
			prefix := secret.EnvVarPrefix
			if prefix == "" {
				prefix = cfg.EnvPrefix
			}
			if prefix != "" {
				key = prefix + key
			}

			envVars = append(envVars, corev1.EnvVar{
				Name:  toEnvKey(key),
				Value: valueToString(value),
			})
		}
		return envVars, nil
	}

	// Convert all fields to env vars
	return convertFieldsToEnvVars(secretData.Fields, secret.EnvVarPrefix, cfg.EnvPrefix), nil
}

// convertFieldsToEnvVars converts secret fields map to []EnvVar
func convertFieldsToEnvVars(fields map[string]interface{}, secretPrefix, globalPrefix string) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	for key, value := range fields {
		envKey := key
		prefix := secretPrefix
		if prefix == "" {
			prefix = globalPrefix
		}
		if prefix != "" {
			envKey = prefix + envKey
		}

		envVars = append(envVars, corev1.EnvVar{
			Name:  toEnvKey(envKey),
			Value: valueToString(value),
		})
	}

	return envVars
}

// toEnvKey converts a field name to environment variable style (UPPER_CASE)
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

// valueToString converts a value to a string for env var
func valueToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		// JSON encode complex types
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		// Remove quotes from JSON strings
		str := string(data)
		if strings.HasPrefix(str, "\"") && strings.HasSuffix(str, "\"") {
			return str[1 : len(str)-1]
		}
		return str
	}
}
