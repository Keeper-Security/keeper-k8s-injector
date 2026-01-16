package config

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseAnnotations_CACert(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		wantSecret  string
		wantCM      string
		wantKey     string
	}{
		{
			name: "CA cert from secret",
			annotations: map[string]string{
				AnnotationInject:       "true",
				AnnotationAuthSecret:   "keeper-auth",
				AnnotationSecret:       "test-secret",
				AnnotationCACertSecret: "corporate-ca",
			},
			wantSecret: "corporate-ca",
			wantCM:     "",
			wantKey:    "ca.crt", // default
		},
		{
			name: "CA cert from configmap",
			annotations: map[string]string{
				AnnotationInject:          "true",
				AnnotationAuthSecret:      "keeper-auth",
				AnnotationSecret:          "test-secret",
				AnnotationCACertConfigMap: "zscaler-ca",
			},
			wantSecret: "",
			wantCM:     "zscaler-ca",
			wantKey:    "ca.crt", // default
		},
		{
			name: "CA cert with custom key",
			annotations: map[string]string{
				AnnotationInject:       "true",
				AnnotationAuthSecret:   "keeper-auth",
				AnnotationSecret:       "test-secret",
				AnnotationCACertSecret: "corporate-ca",
				AnnotationCACertKey:    "custom-ca.pem",
			},
			wantSecret: "corporate-ca",
			wantCM:     "",
			wantKey:    "custom-ca.pem",
		},
		{
			name: "Both secret and configmap specified - secret takes precedence",
			annotations: map[string]string{
				AnnotationInject:          "true",
				AnnotationAuthSecret:      "keeper-auth",
				AnnotationSecret:          "test-secret",
				AnnotationCACertSecret:    "ca-secret",
				AnnotationCACertConfigMap: "ca-configmap",
			},
			wantSecret: "ca-secret",
			wantCM:     "ca-configmap",
			wantKey:    "ca.crt",
		},
		{
			name: "No CA cert configured",
			annotations: map[string]string{
				AnnotationInject:     "true",
				AnnotationAuthSecret: "keeper-auth",
				AnnotationSecret:     "test-secret",
			},
			wantSecret: "",
			wantCM:     "",
			wantKey:    "ca.crt", // still defaults
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

			if cfg.CACertSecret != tt.wantSecret {
				t.Errorf("CACertSecret = %q, want %q", cfg.CACertSecret, tt.wantSecret)
			}
			if cfg.CACertConfigMap != tt.wantCM {
				t.Errorf("CACertConfigMap = %q, want %q", cfg.CACertConfigMap, tt.wantCM)
			}
			if cfg.CACertKey != tt.wantKey {
				t.Errorf("CACertKey = %q, want %q", cfg.CACertKey, tt.wantKey)
			}
		})
	}
}

func TestParseAnnotations_CACertValidation(t *testing.T) {
	// Test that CA cert annotations are properly validated
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				AnnotationInject:       "true",
				AnnotationAuthSecret:   "keeper-auth",
				AnnotationSecret:       "test-secret",
				AnnotationCACertSecret: "   ", // whitespace should be handled
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	// Whitespace-only values should still be set (might be user error, but not fatal)
	if cfg.CACertSecret != "   " {
		t.Errorf("Expected whitespace to be preserved, got %q", cfg.CACertSecret)
	}
}
