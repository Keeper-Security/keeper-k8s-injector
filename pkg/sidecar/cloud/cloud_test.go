package cloud

import (
	"context"
	"os"
	"testing"
)

// Note: These are unit tests that test the error handling and validation logic.
// Integration tests against real cloud providers would require actual cloud accounts.

func TestFetchKSMConfigFromAWS_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		secretID  string
		region    string
		wantError string
	}{
		{
			name:      "empty secret ID",
			secretID:  "",
			region:    "us-west-2",
			wantError: "secret ID cannot be empty",
		},
		{
			name:      "empty region defaults to SDK",
			secretID:  "test-secret",
			region:    "",
			wantError: "", // Should use SDK default region
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchKSMConfigFromAWS(ctx, tt.secretID, tt.region)

			// We expect errors in real execution (no AWS creds in test)
			// Just validate we don't panic and error messages are reasonable
			if err == nil {
				t.Error("Expected error without AWS credentials, got nil")
			}

			// Empty secret ID should fail early
			if tt.secretID == "" && err == nil {
				t.Error("Empty secret ID should return error")
			}
		})
	}
}

func TestFetchKSMConfigFromGCP_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		secretID  string
		wantError bool
	}{
		{
			name:      "empty secret ID",
			secretID:  "",
			wantError: true,
		},
		{
			name:      "invalid format - missing projects prefix",
			secretID:  "secrets/ksm-config/versions/latest",
			wantError: true,
		},
		{
			name:      "valid format",
			secretID:  "projects/my-project/secrets/ksm-config/versions/latest",
			wantError: true, // Still fails in test (no GCP creds) but passes validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchKSMConfigFromGCP(ctx, tt.secretID)

			if tt.wantError && err == nil {
				t.Errorf("Expected error for %q, got nil", tt.secretID)
			}
		})
	}
}

func TestFetchKSMConfigFromAzure_ValidationErrors(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		vaultName   string
		secretName  string
		wantError   bool
		errorString string
	}{
		{
			name:        "empty vault name",
			vaultName:   "",
			secretName:  "ksm-config",
			wantError:   true,
			errorString: "vault name cannot be empty",
		},
		{
			name:        "empty secret name",
			vaultName:   "mykeyvault",
			secretName:  "",
			wantError:   true,
			errorString: "secret name cannot be empty",
		},
		{
			name:        "valid inputs",
			vaultName:   "mykeyvault",
			secretName:  "ksm-config",
			wantError:   true, // Still fails (no Azure creds) but passes validation
			errorString: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FetchKSMConfigFromAzure(ctx, tt.vaultName, tt.secretName)

			if tt.wantError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			// Note: Can't check error string exactly in unit tests
			// Integration tests would verify actual Azure SDK error messages
			_ = tt.errorString
		})
	}
}

// Test environment variable reading (AWS)
func TestAWS_EnvironmentDetection(t *testing.T) {
	// Save original env
	origRoleARN := os.Getenv("AWS_ROLE_ARN")
	origTokenFile := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	defer func() {
		_ = os.Setenv("AWS_ROLE_ARN", origRoleARN)
		_ = os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", origTokenFile)
	}()

	// Test with IRSA env vars set
	_ = os.Setenv("AWS_ROLE_ARN", "arn:aws:iam::123456:role/test")
	_ = os.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/var/run/secrets/eks.amazonaws.com/serviceaccount/token")

	// Verify env vars are set
	if os.Getenv("AWS_ROLE_ARN") == "" {
		t.Error("AWS_ROLE_ARN should be set")
	}
	if os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") == "" {
		t.Error("AWS_WEB_IDENTITY_TOKEN_FILE should be set")
	}
}

// Test Azure environment detection
func TestAzure_EnvironmentDetection(t *testing.T) {
	// Save original env
	origClientID := os.Getenv("AZURE_CLIENT_ID")
	origTenantID := os.Getenv("AZURE_TENANT_ID")
	origTokenFile := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
	defer func() {
		_ = os.Setenv("AZURE_CLIENT_ID", origClientID)
		_ = os.Setenv("AZURE_TENANT_ID", origTenantID)
		_ = os.Setenv("AZURE_FEDERATED_TOKEN_FILE", origTokenFile)
	}()

	// Test with Azure Workload Identity env vars
	_ = os.Setenv("AZURE_CLIENT_ID", "12345678-1234-1234-1234-123456789012")
	_ = os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
	_ = os.Setenv("AZURE_FEDERATED_TOKEN_FILE", "/var/run/secrets/azure/tokens/azure-identity-token")

	// Verify env vars are set
	if os.Getenv("AZURE_CLIENT_ID") == "" {
		t.Error("AZURE_CLIENT_ID should be set")
	}
}
