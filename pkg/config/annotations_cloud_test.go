package config

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseAnnotations_AWSSecretsManager(t *testing.T) {
	tests := []struct {
		name           string
		annotations    map[string]string
		wantAuthMethod string
		wantSecretID   string
		wantRegion     string
	}{
		{
			name: "AWS with secret ID and region",
			annotations: map[string]string{
				AnnotationInject:      "true",
				AnnotationAuthMethod:  "aws-secrets-manager",
				AnnotationAWSSecretID: "prod/keeper/ksm-config",
				AnnotationAWSRegion:   "us-west-2",
				AnnotationSecret:      "test-secret",
			},
			wantAuthMethod: "aws-secrets-manager",
			wantSecretID:   "prod/keeper/ksm-config",
			wantRegion:     "us-west-2",
		},
		{
			name: "AWS with ARN and no region",
			annotations: map[string]string{
				AnnotationInject:      "true",
				AnnotationAuthMethod:  "aws-secrets-manager",
				AnnotationAWSSecretID: "arn:aws:secretsmanager:us-east-1:123456789:secret:ksm-abc123",
				AnnotationSecret:      "test-secret",
			},
			wantAuthMethod: "aws-secrets-manager",
			wantSecretID:   "arn:aws:secretsmanager:us-east-1:123456789:secret:ksm-abc123",
			wantRegion:     "", // empty, will auto-detect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}

			cfg, err := ParseAnnotations(pod)
			if err != nil {
				t.Fatalf("ParseAnnotations() error = %v", err)
			}

			if cfg.AuthMethod != tt.wantAuthMethod {
				t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, tt.wantAuthMethod)
			}
			if cfg.AWSSecretID != tt.wantSecretID {
				t.Errorf("AWSSecretID = %q, want %q", cfg.AWSSecretID, tt.wantSecretID)
			}
			if cfg.AWSRegion != tt.wantRegion {
				t.Errorf("AWSRegion = %q, want %q", cfg.AWSRegion, tt.wantRegion)
			}
		})
	}
}

func TestParseAnnotations_GCPSecretManager(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationInject:      "true",
				AnnotationAuthMethod:  "gcp-secret-manager",
				AnnotationGCPSecretID: "projects/my-project/secrets/ksm-config/versions/latest",
				AnnotationSecret:      "test-secret",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if cfg.AuthMethod != "gcp-secret-manager" {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, "gcp-secret-manager")
	}
	if cfg.GCPSecretID != "projects/my-project/secrets/ksm-config/versions/latest" {
		t.Errorf("GCPSecretID = %q", cfg.GCPSecretID)
	}
}

func TestParseAnnotations_AzureKeyVault(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationInject:          "true",
				AnnotationAuthMethod:      "azure-key-vault",
				AnnotationAzureVaultName:  "mykeyvault",
				AnnotationAzureSecretName: "ksm-config",
				AnnotationSecret:          "test-secret",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if cfg.AuthMethod != "azure-key-vault" {
		t.Errorf("AuthMethod = %q, want %q", cfg.AuthMethod, "azure-key-vault")
	}
	if cfg.AzureVaultName != "mykeyvault" {
		t.Errorf("AzureVaultName = %q, want %q", cfg.AzureVaultName, "mykeyvault")
	}
	if cfg.AzureSecretName != "ksm-config" {
		t.Errorf("AzureSecretName = %q, want %q", cfg.AzureSecretName, "ksm-config")
	}
}

func TestParseAnnotations_CloudAuthBackwardCompatibility(t *testing.T) {
	// Test that existing secret-based auth still works
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationInject:     "true",
				AnnotationAuthSecret: "keeper-credentials",
				AnnotationSecret:     "test-secret",
				// No auth-method specified = defaults to "secret"
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if cfg.AuthMethod != "secret" {
		t.Errorf("AuthMethod should default to 'secret', got %q", cfg.AuthMethod)
	}
	if cfg.AuthSecretName != "keeper-credentials" {
		t.Errorf("AuthSecretName = %q, want 'keeper-credentials'", cfg.AuthSecretName)
	}

	// Cloud fields should be empty
	if cfg.AWSSecretID != "" || cfg.GCPSecretID != "" || cfg.AzureVaultName != "" {
		t.Error("Cloud config fields should be empty for secret-based auth")
	}
}
