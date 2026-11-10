package webhook

import (
	"encoding/json"
	"testing"

	"github.com/keeper-security/keeper-k8s-injector/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

// sidecarSecrets round-trips buildSidecarConfig through JSON, exactly as KEEPER_CONFIG
// reaches the sidecar, and returns its "secrets" entries.
func sidecarSecrets(t *testing.T, cfg *config.InjectionConfig) (map[string]interface{}, []map[string]interface{}) {
	t.Helper()
	m := newTestMutator(t)
	raw, err := json.Marshal(m.buildSidecarConfig(cfg))
	require.NoError(t, err)
	var out struct {
		Secrets []map[string]interface{} `json:"secrets"`
	}
	require.NoError(t, json.Unmarshal(raw, &out))
	var all map[string]interface{}
	require.NoError(t, json.Unmarshal(raw, &all))
	return all, out.Secrets
}

// Regression for #27: the pod-level inject-as-k8s-secret / k8s-secret-name annotations
// must reach the sidecar, otherwise k8s-secret-rotation never updates the Secret.
func TestBuildSidecarConfig_PodLevelK8sSecretPropagates(t *testing.T) {
	cfg := &config.InjectionConfig{
		InjectAsK8sSecret: true,
		K8sSecretName:     "app-secrets",
		K8sSecretMode:     "overwrite",
		K8sSecretRotation: true,
		Secrets: []config.SecretRef{
			{Name: "db", Path: "/keeper/secrets/db.json", Format: "json"},
			{Name: "api", Path: "/keeper/secrets/api.json", Format: "json"},
		},
	}

	all, secrets := sidecarSecrets(t, cfg)

	require.Len(t, secrets, 2)
	for _, s := range secrets {
		assert.Equal(t, true, s["injectAsK8sSecret"], "entry %v", s["name"])
		assert.Equal(t, "app-secrets", s["k8sSecretName"], "entry %v", s["name"])
	}
	assert.Equal(t, true, all["k8sSecretRotation"])
	assert.Equal(t, "overwrite", all["k8sSecretMode"])
}

func TestBuildSidecarConfig_PerSecretNameOverridesPodLevel(t *testing.T) {
	cfg := &config.InjectionConfig{
		InjectAsK8sSecret: true,
		K8sSecretName:     "pod-level",
		Secrets: []config.SecretRef{
			{Name: "db", Path: "/keeper/secrets/db.json", Format: "json", K8sSecretName: "db-secret"},
			{Name: "api", Path: "/keeper/secrets/api.json", Format: "json"},
		},
	}

	_, secrets := sidecarSecrets(t, cfg)

	require.Len(t, secrets, 2)
	assert.Equal(t, "db-secret", secrets[0]["k8sSecretName"])
	assert.Equal(t, "pod-level", secrets[1]["k8sSecretName"])
}

func TestBuildSidecarConfig_PerSecretFlagWithoutPodLevel(t *testing.T) {
	cfg := &config.InjectionConfig{
		Secrets: []config.SecretRef{
			{Name: "db", Path: "/keeper/secrets/db.json", Format: "json", InjectAsK8sSecret: true, K8sSecretName: "db-secret",
				K8sSecretKeys: map[string]string{"password": "DB_PASSWORD"}},
			{Name: "api", Path: "/keeper/secrets/api.json", Format: "json"},
		},
	}

	_, secrets := sidecarSecrets(t, cfg)

	require.Len(t, secrets, 2)
	assert.Equal(t, true, secrets[0]["injectAsK8sSecret"])
	assert.Equal(t, "db-secret", secrets[0]["k8sSecretName"])
	assert.Equal(t, map[string]interface{}{"password": "DB_PASSWORD"}, secrets[0]["k8sSecretKeys"])
	assert.NotContains(t, secrets[1], "injectAsK8sSecret")
	assert.NotContains(t, secrets[1], "k8sSecretName")
}

// The sidecar resolves the Secret's namespace from POD_NAMESPACE during rotation.
func TestBuildAgentEnv_SetsPodNamespaceFromDownwardAPI(t *testing.T) {
	m := newTestMutator(t)
	env := m.buildAgentEnv(&config.InjectionConfig{AuthMethod: "secret", AuthSecretName: "ksm"}, "{}")

	var found *corev1.EnvVar
	for i := range env {
		if env[i].Name == "POD_NAMESPACE" {
			found = &env[i]
		}
	}
	require.NotNil(t, found)
	require.NotNil(t, found.ValueFrom)
	require.NotNil(t, found.ValueFrom.FieldRef)
	assert.Equal(t, "metadata.namespace", found.ValueFrom.FieldRef.FieldPath)
}
