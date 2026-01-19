//go:build integration
// +build integration

package webhook

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/keeper-security/keeper-k8s-injector/pkg/config"
	"github.com/keeper-security/keeper-k8s-injector/pkg/ksm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	// Test vault records (actual titles from vault)
	testRecordSalesDB       = "salesdb"
	testRecordStripeAPI     = "Stripe POS API"
	testRecordDemoSecret    = "demo-secret"
	testRecordPostgres      = "postgres-credentials"
	testFolderProdDBs       = "Keeper K8s Injector Test/prod/dbs"
	testFolderProdAPIs      = "Keeper K8s Injector Test/prod/apis"
)

// getConfigFilePath returns the config file path from environment or default
func getConfigFilePath() string {
	// Allow override via environment variable
	if path := os.Getenv("KEEPER_CONFIG_FILE"); path != "" {
		return path
	}

	// Try multiple locations
	candidates := []string{
		"config.base64",                     // Relative to test directory
		"../../config.base64",               // From pkg/webhook to root
		"/app/config.base64",                // Docker absolute path
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback
	return "config.base64"
}

// loadRealConfig loads the real KSM config from config.base64
func loadRealConfig(t *testing.T) string {
	t.Helper()

	configFilePath := getConfigFilePath()

	// Check if config file exists
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: config file not found at %s (set KEEPER_CONFIG_FILE env var to override)", configFilePath)
	}

	// Read base64 config
	configBase64, err := os.ReadFile(configFilePath)
	require.NoError(t, err, "failed to read config file")

	// Decode base64
	configJSON, err := base64.StdEncoding.DecodeString(string(configBase64))
	require.NoError(t, err, "failed to decode base64 config")

	return string(configJSON)
}

// createRealKSMClient creates a KSM client with real credentials
func createRealKSMClient(t *testing.T) *ksm.Client {
	t.Helper()

	configJSON := loadRealConfig(t)

	client, err := ksm.NewClient(context.Background(), ksm.Config{
		ConfigJSON:  configJSON,
		AuthMethod:  ksm.AuthMethodSecret,
		StrictMatch: false,
		Logger:      zap.NewNop(),
	})
	require.NoError(t, err, "failed to create KSM client")

	return client
}

// TestIntegration_CreateK8sSecret_SingleRecord tests creating a K8s Secret from a single record
func TestIntegration_CreateK8sSecret_SingleRecord(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordSalesDB)
	require.NoError(t, err, "failed to fetch secret from Keeper")
	require.NotNil(t, secretData)
	require.NotEmpty(t, secretData.Fields, "secret has no fields")

	t.Logf("Fetched secret '%s' with %d fields", testRecordSalesDB, len(secretData.Fields))

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordSalesDB,
		K8sSecretName: "salesdb-creds",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Verify K8s Secret structure
	assert.Equal(t, "salesdb-creds", k8sSecret.Name)
	assert.Equal(t, "default", k8sSecret.Namespace)
	assert.Equal(t, corev1.SecretTypeOpaque, k8sSecret.Type)
	assert.NotEmpty(t, k8sSecret.Data, "K8s Secret has no data")

	// Verify expected fields exist (based on vault structure)
	expectedFields := []string{"type", "host", "login", "password"}
	for _, field := range expectedFields {
		assert.Contains(t, k8sSecret.Data, field, "missing field: %s", field)
	}

	// Verify owner reference
	require.Len(t, k8sSecret.OwnerReferences, 1)
	assert.Equal(t, "test-pod", k8sSecret.OwnerReferences[0].Name)

	// Create Secret in fake cluster
	err = mutator.createOrUpdateSecret(context.Background(), k8sSecret, "overwrite", true)
	require.NoError(t, err)

	// Verify Secret was created
	created := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "salesdb-creds", Namespace: "default"}, created)
	require.NoError(t, err)
	assert.Equal(t, k8sSecret.Data, created.Data)

	t.Logf("✅ Successfully created K8s Secret with %d fields", len(created.Data))
}

// TestIntegration_CreateK8sSecret_CustomKeys tests custom key mapping
func TestIntegration_CreateK8sSecret_CustomKeys(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-456"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordSalesDB)
	require.NoError(t, err)

	// Build K8s Secret with custom key mapping
	secretRef := config.SecretRef{
		Name:          testRecordSalesDB,
		K8sSecretName: "db-credentials",
		K8sSecretKeys: map[string]string{
			"login":    "DB_USER",
			"password": "DB_PASS",
			"host":     "DB_HOST",
		},
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Verify custom keys exist
	assert.Contains(t, k8sSecret.Data, "DB_USER")
	assert.Contains(t, k8sSecret.Data, "DB_PASS")
	assert.Contains(t, k8sSecret.Data, "DB_HOST")

	// Verify original keys don't exist
	assert.NotContains(t, k8sSecret.Data, "login")
	assert.NotContains(t, k8sSecret.Data, "password")
	assert.NotContains(t, k8sSecret.Data, "host")

	t.Logf("✅ Custom key mapping successful: %d mapped keys", len(k8sSecret.Data))
}

// TestIntegration_ConflictMode_Overwrite tests overwrite mode
func TestIntegration_ConflictMode_Overwrite(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"oldkey": []byte("oldvalue"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-789"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "test-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Overwrite existing Secret
	err = mutator.createOrUpdateSecret(context.Background(), k8sSecret, "overwrite", true)
	require.NoError(t, err)

	// Verify Secret was overwritten
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)

	// Old key should be gone
	assert.NotContains(t, result.Data, "oldkey")

	// New keys should exist
	assert.NotEmpty(t, result.Data)

	t.Logf("✅ Overwrite mode successful: old keys removed, %d new keys added", len(result.Data))
}

// TestIntegration_ConflictMode_Merge tests merge mode
func TestIntegration_ConflictMode_Merge(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"existingkey": []byte("existingvalue"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-abc"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "test-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Merge with existing Secret
	err = mutator.createOrUpdateSecret(context.Background(), k8sSecret, "merge", true)
	require.NoError(t, err)

	// Verify Secret was merged
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)

	// Old key should still exist
	assert.Contains(t, result.Data, "existingkey")
	assert.Equal(t, []byte("existingvalue"), result.Data["existingkey"])

	// New keys should also exist
	assert.Greater(t, len(result.Data), 1, "should have both old and new keys")

	t.Logf("✅ Merge mode successful: %d total keys (existing + new)", len(result.Data))
}

// TestIntegration_ConflictMode_SkipIfExists tests skip-if-exists mode
func TestIntegration_ConflictMode_SkipIfExists(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"preserved": []byte("originalvalue"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-def"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "test-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Skip if exists
	err = mutator.createOrUpdateSecret(context.Background(), k8sSecret, "skip-if-exists", true)
	require.NoError(t, err)

	// Verify Secret was NOT modified
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)

	// Original data should be unchanged
	assert.Len(t, result.Data, 1)
	assert.Equal(t, []byte("originalvalue"), result.Data["preserved"])

	t.Logf("✅ Skip-if-exists mode successful: existing Secret preserved")
}

// TestIntegration_ConflictMode_Fail tests fail mode
func TestIntegration_ConflictMode_Fail(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"existing": []byte("data"),
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existing).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-ghi"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "test-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Should fail because Secret exists
	err = mutator.createOrUpdateSecret(context.Background(), k8sSecret, "fail", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	t.Logf("✅ Fail mode successful: error returned as expected")
}

// TestIntegration_OwnerReference_Enabled tests owner reference creation
func TestIntegration_OwnerReference_Enabled(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-owner"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true, // Enabled
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "owned-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Verify owner reference exists
	require.Len(t, k8sSecret.OwnerReferences, 1)
	assert.Equal(t, "v1", k8sSecret.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pod", k8sSecret.OwnerReferences[0].Kind)
	assert.Equal(t, "test-pod", k8sSecret.OwnerReferences[0].Name)
	assert.Equal(t, types.UID("test-uid-owner"), k8sSecret.OwnerReferences[0].UID)
	assert.True(t, *k8sSecret.OwnerReferences[0].Controller)

	t.Logf("✅ Owner reference enabled: Secret will be deleted with pod")
}

// TestIntegration_OwnerReference_Disabled tests owner reference disabled
func TestIntegration_OwnerReference_Disabled(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-noowner"),
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: false, // Disabled
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Fetch real secret
	secretData, err := ksmClient.GetSecret(context.Background(), testRecordDemoSecret)
	require.NoError(t, err)

	// Build K8s Secret
	secretRef := config.SecretRef{
		Name:          testRecordDemoSecret,
		K8sSecretName: "standalone-secret",
	}

	k8sSecret, err := mutator.buildK8sSecret(pod, secretRef, secretData, cfg)
	require.NoError(t, err)

	// Verify NO owner reference
	assert.Len(t, k8sSecret.OwnerReferences, 0)

	t.Logf("✅ Owner reference disabled: Secret will persist after pod deletion")
}

// TestIntegration_BatchFetch_MultipleSecrets tests efficient batching
func TestIntegration_BatchFetch_MultipleSecrets(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Multiple secrets to fetch
	secrets := []config.SecretRef{
		{Name: testRecordSalesDB, K8sSecretName: "salesdb"},
		{Name: testRecordDemoSecret, K8sSecretName: "demo"},
	}

	cfg := &config.InjectionConfig{
		FailOnError: true,
	}

	// Batch fetch (should be ONE API call)
	secretsData, err := mutator.batchFetchSecrets(context.Background(), ksmClient, secrets, cfg)
	require.NoError(t, err)
	require.Len(t, secretsData, 2, "should fetch 2 secrets")

	// Verify both secrets were fetched
	for i, secret := range secrets {
		data, ok := secretsData[i]
		require.True(t, ok, "secret %s not found in batch result", secret.Name)
		assert.NotEmpty(t, data.Fields, "secret %s has no fields", secret.Name)
		t.Logf("✅ Fetched '%s' with %d fields", secret.Name, len(data.Fields))
	}

	t.Logf("✅ Batch fetch successful: 2 secrets fetched efficiently")
}

// TestIntegration_SecretNotFound_Error tests error handling for missing secrets
func TestIntegration_SecretNotFound_Error(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	// Create real KSM client
	ksmClient := createRealKSMClient(t)
	defer ksmClient.Close()

	// Non-existent secret
	secrets := []config.SecretRef{
		{Name: "NonExistent-Secret-12345", K8sSecretName: "missing"},
	}

	cfg := &config.InjectionConfig{
		FailOnError: true,
	}

	// Should fail with error
	_, err := mutator.batchFetchSecrets(context.Background(), ksmClient, secrets, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	t.Logf("✅ Error handling successful: missing secret detected")
}

// TestIntegration_SizeLimit_Validation tests size validation
func TestIntegration_SizeLimit_Validation(t *testing.T) {
	// Create a secret that exceeds 1MB
	largeData := make(map[string][]byte)
	largeData["huge"] = make([]byte, MaxSecretSize+1)

	secret := &corev1.Secret{
		Data: largeData,
	}

	err := validateSecretSize(secret)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")

	t.Logf("✅ Size validation successful: oversized Secret rejected")
}
