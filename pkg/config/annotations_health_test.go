package config

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWith(annotations map[string]string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "ns", Annotations: annotations}}
}

func TestParseAnnotations_AuthSecretLegacyName(t *testing.T) {
	base := map[string]string{"keeper.security/inject": "true", "keeper.security/secret": "rec"}
	with := func(extra map[string]string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}

	cfg, err := ParseAnnotations(podWith(with(map[string]string{"keeper.security/auth-secret": "legacy-auth"})))
	if err != nil {
		t.Fatalf("legacy name only: %v", err)
	}
	if cfg.AuthSecretName != "legacy-auth" {
		t.Errorf("legacy name only: got %q", cfg.AuthSecretName)
	}

	cfg, err = ParseAnnotations(podWith(with(map[string]string{"keeper.security/auth-secret": "legacy-auth", "keeper.security/ksm-config": "current-auth"})))
	if err != nil {
		t.Fatalf("both names: %v", err)
	}
	if cfg.AuthSecretName != "current-auth" {
		t.Errorf("ksm-config must win over the legacy name: got %q", cfg.AuthSecretName)
	}
}

func TestParseAnnotations_HealthPort(t *testing.T) {
	base := map[string]string{"keeper.security/inject": "true", "keeper.security/secret": "rec", "keeper.security/ksm-config": "auth"}
	cfg, err := ParseAnnotations(podWith(base))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HealthPort != 0 {
		t.Errorf("unset: got %d, want 0 (choose automatically)", cfg.HealthPort)
	}

	for _, tc := range []struct {
		value string
		want  int
		ok    bool
	}{{"9090", 9090, true}, {" 8181 ", 8181, true}, {"0", 0, false}, {"70000", 0, false}, {"http", 0, false}} {
		a := map[string]string{"keeper.security/health-port": tc.value}
		for k, v := range base {
			a[k] = v
		}
		cfg, err := ParseAnnotations(podWith(a))
		if tc.ok && (err != nil || cfg.HealthPort != tc.want) {
			t.Errorf("%q: got %v, %v", tc.value, cfg, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("%q: expected an error", tc.value)
		}
	}
}
