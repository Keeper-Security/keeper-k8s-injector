package main

import (
	"encoding/json"
	"testing"
)

// TestSecretEntryRoundTripsAllFields guards against the class of bug where a field is emitted
// by the webhook into KEEPER_CONFIG but silently dropped here because secretEntry lacks it.
// Regression test for the Go-template drop (template was missing from secretEntry) and the
// per-secret K8s-Secret fields / rotation flags needed for sidecar K8s-Secret refresh.
func TestSecretEntryRoundTripsAllFields(t *testing.T) {
	js := `{
		"secrets": [{
			"name": "demo",
			"path": "/keeper/secrets/x.txt",
			"format": "json",
			"template": "{{ .login | upper }}",
			"fields": ["password"],
			"notation": "keeper://demo/field/password",
			"fileName": "cert.pem",
			"isFile": true,
			"injectAsK8sSecret": true,
			"k8sSecretName": "app-secrets",
			"k8sSecretKeys": {"login": "user"}
		}],
		"failOnError": true,
		"k8sSecretRotation": true,
		"k8sSecretNamespace": "ns1"
	}`

	var cfg secretsConfig
	if err := json.Unmarshal([]byte(js), &cfg); err != nil {
		t.Fatalf("unmarshal KEEPER_CONFIG: %v", err)
	}
	if len(cfg.Secrets) != 1 {
		t.Fatalf("want 1 secret, got %d", len(cfg.Secrets))
	}
	s := cfg.Secrets[0]
	if s.Template != "{{ .login | upper }}" {
		t.Errorf("Template dropped at deserialization: %q", s.Template)
	}
	if !s.InjectAsK8sSecret {
		t.Error("InjectAsK8sSecret dropped at deserialization")
	}
	if s.K8sSecretName != "app-secrets" {
		t.Errorf("K8sSecretName dropped: %q", s.K8sSecretName)
	}
	if s.K8sSecretKeys["login"] != "user" {
		t.Errorf("K8sSecretKeys dropped: %v", s.K8sSecretKeys)
	}
	if !cfg.K8sSecretRotation {
		t.Error("K8sSecretRotation dropped at deserialization")
	}
	if cfg.K8sSecretNamespace != "ns1" {
		t.Errorf("K8sSecretNamespace dropped: %q", cfg.K8sSecretNamespace)
	}
}
