package cloud

import (
	"context"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// FetchKSMConfigFromGCP retrieves Keeper Secrets Manager configuration
// from GCP Secret Manager using Workload Identity.
//
// The pod must have a ServiceAccount with iam.gke.io/gcp-service-account annotation
// pointing to a GCP service account with secretmanager.versions.access permission.
//
// Parameters:
//   - secretID: GCP Secret Manager resource name
//     Format: "projects/PROJECT_ID/secrets/SECRET_NAME/versions/VERSION"
//     Example: "projects/my-project/secrets/ksm-config/versions/latest"
//
// Returns base64-encoded KSM configuration string.
func FetchKSMConfigFromGCP(ctx context.Context, secretID string) (string, error) {
	// Validation
	if secretID == "" {
		return "", fmt.Errorf("GCP secret ID cannot be empty")
	}

	// Validate format: must start with "projects/"
	if !strings.HasPrefix(secretID, "projects/") {
		return "", fmt.Errorf("invalid GCP secret ID format, must start with 'projects/', got: %s", secretID)
	}
	// Create Secret Manager client
	// Automatically uses Workload Identity credentials
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create GCP Secret Manager client (ensure Workload Identity is configured): %w", err)
	}
	defer func() {
		_ = client.Close()
	}()

	// Access secret version
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access GCP secret: %w", err)
	}

	// Return secret data (should be base64-encoded KSM config)
	if result.Payload == nil || result.Payload.Data == nil {
		return "", fmt.Errorf("secret %s has no data", secretID)
	}

	return string(result.Payload.Data), nil
}
