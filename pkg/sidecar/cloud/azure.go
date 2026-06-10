package cloud

import (
	"context"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

// azureVaultNameRe matches valid Azure Key Vault names: 3-24 chars, alphanumeric and hyphens.
var azureVaultNameRe = regexp.MustCompile(`^[A-Za-z0-9-]{3,24}$`)

// FetchKSMConfigFromAzure retrieves Keeper Secrets Manager configuration
// from Azure Key Vault using Workload Identity.
//
// The pod must have a ServiceAccount with azure.workload.identity/client-id annotation
// pointing to a managed identity with Key Vault Secrets User role.
//
// Parameters:
//   - vaultName: Azure Key Vault name (not the full URL)
//   - secretName: Secret name in the Key Vault
//
// Returns base64-encoded KSM configuration string.
func FetchKSMConfigFromAzure(ctx context.Context, vaultName, secretName string) (string, error) {
	// Validation
	if vaultName == "" {
		return "", fmt.Errorf("azure vault name cannot be empty")
	}
	if secretName == "" {
		return "", fmt.Errorf("azure secret name cannot be empty")
	}

	// Validate vault name format (prevent SSRF): Azure vault names must be 3-24 chars and
	// contain only alphanumerics and hyphens. Enforce the charset too (not just length) —
	// the value is interpolated into the vault URL below, so a stray character could redirect
	// the request to an attacker-controlled host.
	if !azureVaultNameRe.MatchString(vaultName) {
		return "", fmt.Errorf("invalid vault name %q (must be 3-24 chars, alphanumeric and hyphens only)", vaultName)
	}
	// Create Azure credential
	// Automatically uses Workload Identity from environment variables:
	// - AZURE_CLIENT_ID
	// - AZURE_TENANT_ID
	// - AZURE_FEDERATED_TOKEN_FILE
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure credential (ensure Workload Identity is configured): %w", err)
	}

	// Construct vault URL
	vaultURL := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)

	// Create Key Vault client
	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Azure Key Vault client: %w", err)
	}

	// Get secret value (latest version)
	result, err := client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret from Azure Key Vault: %w", err)
	}

	// Return secret value (should be base64-encoded KSM config)
	if result.Value == nil {
		return "", fmt.Errorf("secret %s in vault %s has no value", secretName, vaultName)
	}

	return *result.Value, nil
}
