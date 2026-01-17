package cloud

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
)

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
