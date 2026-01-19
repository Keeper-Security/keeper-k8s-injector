package webhook

import (
	"context"
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

// TestBuildK8sSecret_AllFields tests building a K8s Secret with all fields
func TestBuildK8sSecret_AllFields(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "app-secrets",
	}

	data := &ksm.SecretData{
		RecordUID: "test-uid",
		Title:     "test-secret",
		Fields: map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
			"host":     "localhost",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Equal(t, "app-secrets", secret.Name)
	assert.Equal(t, "default", secret.Namespace)
	assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
	assert.Equal(t, []byte("testuser"), secret.Data["username"])
	assert.Equal(t, []byte("testpass"), secret.Data["password"])
	assert.Equal(t, []byte("localhost"), secret.Data["host"])

	// Verify labels
	assert.Equal(t, "keeper-injector", secret.Labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "true", secret.Labels["keeper.security/injected"])

	// Verify annotations
	assert.Equal(t, "test-pod", secret.Annotations["keeper.security/source-pod"])
	assert.Equal(t, "test-secret", secret.Annotations["keeper.security/source-record"])

	// Verify owner reference
	require.Len(t, secret.OwnerReferences, 1)
	assert.Equal(t, "v1", secret.OwnerReferences[0].APIVersion)
	assert.Equal(t, "Pod", secret.OwnerReferences[0].Kind)
	assert.Equal(t, "test-pod", secret.OwnerReferences[0].Name)
	assert.Equal(t, types.UID("test-uid-123"), secret.OwnerReferences[0].UID)
}

// TestBuildK8sSecret_SelectedFields tests field filtering
func TestBuildK8sSecret_SelectedFields(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "app-secrets",
		Fields:        []string{"password", "host"},
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
			"host":     "localhost",
			"port":     "5432",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	// Only selected fields should be present
	assert.Len(t, secret.Data, 2)
	assert.Equal(t, []byte("testpass"), secret.Data["password"])
	assert.Equal(t, []byte("localhost"), secret.Data["host"])
	assert.NotContains(t, secret.Data, "username")
	assert.NotContains(t, secret.Data, "port")
}

// TestBuildK8sSecret_CustomKeyMapping tests custom key mapping
func TestBuildK8sSecret_CustomKeyMapping(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "db-credentials",
		K8sSecretKeys: map[string]string{
			"username": "DB_USER",
			"password": "DB_PASS",
			"host":     "DB_HOST",
		},
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"username": "admin",
			"password": "secret123",
			"host":     "db.example.com",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	// Verify custom key mapping
	assert.Equal(t, []byte("admin"), secret.Data["DB_USER"])
	assert.Equal(t, []byte("secret123"), secret.Data["DB_PASS"])
	assert.Equal(t, []byte("db.example.com"), secret.Data["DB_HOST"])
	assert.NotContains(t, secret.Data, "username")
	assert.NotContains(t, secret.Data, "password")
	assert.NotContains(t, secret.Data, "host")
}

// TestBuildK8sSecret_TLSType tests TLS Secret type
func TestBuildK8sSecret_TLSType(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "tls-cert",
		K8sSecretName: "tls-secret",
		K8sSecretType: "kubernetes.io/tls",
		K8sSecretKeys: map[string]string{
			"cert": "tls.crt",
			"key":  "tls.key",
		},
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"cert": "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
			"key":  "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	assert.Equal(t, corev1.SecretType("kubernetes.io/tls"), secret.Type)
	assert.Contains(t, secret.Data, "tls.crt")
	assert.Contains(t, secret.Data, "tls.key")
}

// TestBuildK8sSecret_OwnerRefDisabled tests owner reference disabled
func TestBuildK8sSecret_OwnerRefDisabled(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "app-secrets",
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"password": "testpass",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: false, // Disabled
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	// No owner references should be set
	assert.Len(t, secret.OwnerReferences, 0)
}

// TestBuildK8sSecret_CustomNamespace tests custom namespace
func TestBuildK8sSecret_CustomNamespace(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "app-secrets",
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"password": "testpass",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretNamespace: "production", // Custom namespace
		K8sSecretOwnerRef:  true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	assert.Equal(t, "production", secret.Namespace)
}

// TestCreateOrUpdateSecret_Create tests creating a new Secret
func TestCreateOrUpdateSecret_Create(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("secret123"),
		},
	}

	err := mutator.createOrUpdateSecret(context.Background(), secret, "overwrite", true)
	require.NoError(t, err)

	// Verify secret was created
	created := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, created)
	require.NoError(t, err)
	assert.Equal(t, []byte("secret123"), created.Data["password"])
}

// TestCreateOrUpdateSecret_Overwrite tests overwrite mode
func TestCreateOrUpdateSecret_Overwrite(t *testing.T) {
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

	updated := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Labels: map[string]string{
				"updated": "true",
			},
		},
		Data: map[string][]byte{
			"newkey": []byte("newvalue"),
		},
	}

	err := mutator.createOrUpdateSecret(context.Background(), updated, "overwrite", true)
	require.NoError(t, err)

	// Verify secret was overwritten
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)
	assert.Equal(t, []byte("newvalue"), result.Data["newkey"])
	assert.NotContains(t, result.Data, "oldkey")
	assert.Equal(t, "true", result.Labels["updated"])
}

// TestCreateOrUpdateSecret_Merge tests merge mode
func TestCreateOrUpdateSecret_Merge(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	existing := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Labels: map[string]string{
				"existing": "label",
			},
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

	updated := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
			Labels: map[string]string{
				"new": "label",
			},
		},
		Data: map[string][]byte{
			"newkey": []byte("newvalue"),
		},
	}

	err := mutator.createOrUpdateSecret(context.Background(), updated, "merge", true)
	require.NoError(t, err)

	// Verify secret was merged
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)

	// Both old and new keys should exist
	assert.Equal(t, []byte("oldvalue"), result.Data["oldkey"])
	assert.Equal(t, []byte("newvalue"), result.Data["newkey"])

	// Both old and new labels should exist
	assert.Equal(t, "label", result.Labels["existing"])
	assert.Equal(t, "label", result.Labels["new"])
}

// TestCreateOrUpdateSecret_SkipIfExists tests skip-if-exists mode
func TestCreateOrUpdateSecret_SkipIfExists(t *testing.T) {
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

	updated := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"newkey": []byte("newvalue"),
		},
	}

	err := mutator.createOrUpdateSecret(context.Background(), updated, "skip-if-exists", true)
	require.NoError(t, err)

	// Verify secret was NOT updated
	result := &corev1.Secret{}
	err = fakeClient.Get(context.Background(), client.ObjectKey{Name: "test-secret", Namespace: "default"}, result)
	require.NoError(t, err)
	assert.Equal(t, []byte("oldvalue"), result.Data["oldkey"])
	assert.NotContains(t, result.Data, "newkey")
}

// TestCreateOrUpdateSecret_Fail tests fail mode
func TestCreateOrUpdateSecret_Fail(t *testing.T) {
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

	updated := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"newkey": []byte("newvalue"),
		},
	}

	err := mutator.createOrUpdateSecret(context.Background(), updated, "fail", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// TestValidateSecretSize tests size validation
func TestValidateSecretSize(t *testing.T) {
	t.Run("valid size", func(t *testing.T) {
		secret := &corev1.Secret{
			Data: map[string][]byte{
				"key": []byte("value"),
			},
		}
		err := validateSecretSize(secret)
		assert.NoError(t, err)
	})

	t.Run("exceeds size limit", func(t *testing.T) {
		// Create a secret larger than 1MB
		largeData := make([]byte, MaxSecretSize+1)
		secret := &corev1.Secret{
			Data: map[string][]byte{
				"large": largeData,
			},
		}
		err := validateSecretSize(secret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum")
	})
}

// TestLooksLikeUID tests UID detection
func TestLooksLikeUID(t *testing.T) {
	assert.True(t, looksLikeUID("abcd1234efgh5678ijkl90"))
	assert.False(t, looksLikeUID("short"))
	assert.False(t, looksLikeUID("has spaces in it 1234567"))
	assert.False(t, looksLikeUID("My Database Secret"))
}

// TestValueToBytes tests value conversion
func TestValueToBytes(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		result := valueToBytes("test string")
		assert.Equal(t, []byte("test string"), result)
	})

	t.Run("byte slice", func(t *testing.T) {
		input := []byte("test bytes")
		result := valueToBytes(input)
		assert.Equal(t, input, result)
	})

	t.Run("complex type", func(t *testing.T) {
		input := map[string]string{"key": "value"}
		result := valueToBytes(input)
		assert.Contains(t, string(result), "key")
		assert.Contains(t, string(result), "value")
	})
}

// TestBuildK8sSecret_MissingSecretName tests error when k8sSecretName not provided
func TestBuildK8sSecret_MissingSecretName(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name: "test-secret",
		// K8sSecretName is missing
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"password": "testpass",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
		// K8sSecretName also missing at config level
	}

	_, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "k8s secret name not specified")
}

// TestBuildK8sSecret_EmptyData tests handling of secrets with no fields
func TestBuildK8sSecret_EmptyData(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "empty-secret",
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{}, // No fields
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Empty(t, secret.Data, "Secret should have no data")
}

// TestBuildK8sSecret_MissingFieldInCustomMapping tests missing field in custom mapping
func TestBuildK8sSecret_MissingFieldInCustomMapping(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "app-secrets",
		K8sSecretKeys: map[string]string{
			"username":    "DB_USER",
			"password":    "DB_PASS",
			"nonexistent": "MISSING_FIELD", // This field doesn't exist
		},
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"username": "admin",
			"password": "secret123",
			// "nonexistent" field missing
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)

	// Only existing fields should be mapped
	assert.Contains(t, secret.Data, "DB_USER")
	assert.Contains(t, secret.Data, "DB_PASS")
	assert.NotContains(t, secret.Data, "MISSING_FIELD")
}

// TestBuildK8sSecret_NotationFormat tests notation with K8s Secrets
func TestBuildK8sSecret_NotationFormat(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "password",
		Notation:      "keeper://ABC123/field/password",
		K8sSecretName: "password-secret",
	}

	// Notation returns single value in "value" field
	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"value": "secretpassword123",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Equal(t, []byte("secretpassword123"), secret.Data["value"])
}

// TestBuildK8sSecret_FileAttachment tests file attachment with K8s Secrets
func TestBuildK8sSecret_FileAttachment(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "TLS Certificate",
		FileName:      "cert.pem",
		K8sSecretName: "tls-files",
		K8sSecretKeys: map[string]string{
			"cert.pem": "tls.crt",
		},
	}

	// File attachment data
	certData := []byte("-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----")
	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"cert.pem": certData,
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Equal(t, certData, secret.Data["tls.crt"])
}

// TestCreateOrUpdateSecret_InvalidMode tests invalid conflict mode
func TestCreateOrUpdateSecret_InvalidMode(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	mutator := &PodMutator{
		Client: fakeClient,
		logger: zap.NewNop(),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("value"),
		},
	}

	// Create the secret first
	err := mutator.createOrUpdateSecret(context.Background(), secret, "overwrite", true)
	require.NoError(t, err)

	// Try to update with invalid mode
	err = mutator.createOrUpdateSecret(context.Background(), secret, "invalid-mode", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown k8s-secret-mode")
}

// TestBuildK8sSecret_GlobalSecretName tests using global k8sSecretName
func TestBuildK8sSecret_GlobalSecretName(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name: "test-secret",
		// No K8sSecretName at secret level
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			"password": "testpass",
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretName:     "global-secret", // Global name
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Equal(t, "global-secret", secret.Name)
}

// TestBuildK8sSecret_SecretTypeInheritance tests type inheritance from config
func TestBuildK8sSecret_SecretTypeInheritance(t *testing.T) {
	mutator := &PodMutator{
		logger: zap.NewNop(),
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       types.UID("test-uid-123"),
		},
	}

	secretRef := config.SecretRef{
		Name:          "test-secret",
		K8sSecretName: "docker-secret",
		// No K8sSecretType at secret level
	}

	data := &ksm.SecretData{
		Fields: map[string]interface{}{
			".dockerconfigjson": `{"auths":{}}`,
		},
	}

	cfg := &config.InjectionConfig{
		K8sSecretType:     "kubernetes.io/dockerconfigjson", // Global type
		K8sSecretOwnerRef: true,
	}

	secret, err := mutator.buildK8sSecret(pod, secretRef, data, cfg)
	require.NoError(t, err)
	assert.Equal(t, corev1.SecretType("kubernetes.io/dockerconfigjson"), secret.Type)
}

// TestFilterK8sSecretConfigs tests filtering logic
func TestFilterK8sSecretConfigs(t *testing.T) {
	t.Run("global flag enabled", func(t *testing.T) {
		cfg := &config.InjectionConfig{
			InjectAsK8sSecret: true,
			Secrets: []config.SecretRef{
				{Name: "secret1"},
				{Name: "secret2"},
			},
		}
		filtered := filterK8sSecretConfigs(cfg)
		assert.Len(t, filtered, 2)
	})

	t.Run("per-secret flag", func(t *testing.T) {
		cfg := &config.InjectionConfig{
			InjectAsK8sSecret: false,
			Secrets: []config.SecretRef{
				{Name: "secret1", InjectAsK8sSecret: true},
				{Name: "secret2", InjectAsK8sSecret: false},
				{Name: "secret3"},
			},
		}
		filtered := filterK8sSecretConfigs(cfg)
		assert.Len(t, filtered, 1)
		assert.Equal(t, "secret1", filtered[0].Name)
	})

	t.Run("no secrets enabled", func(t *testing.T) {
		cfg := &config.InjectionConfig{
			InjectAsK8sSecret: false,
			Secrets: []config.SecretRef{
				{Name: "secret1"},
				{Name: "secret2"},
			},
		}
		filtered := filterK8sSecretConfigs(cfg)
		assert.Len(t, filtered, 0)
	})
}
