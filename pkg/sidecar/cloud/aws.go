package cloud

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// FetchKSMConfigFromAWS retrieves Keeper Secrets Manager configuration
// from AWS Secrets Manager using IRSA (IAM Roles for Service Accounts).
//
// The pod must have a ServiceAccount with eks.amazonaws.com/role-arn annotation
// pointing to an IAM role with secretsmanager:GetSecretValue permission.
//
// Parameters:
//   - secretID: AWS Secrets Manager secret ID or ARN
//   - region: AWS region (if empty, uses default from environment/metadata)
//
// Returns base64-encoded KSM configuration string.
func FetchKSMConfigFromAWS(ctx context.Context, secretID, region string) (string, error) {
	// Validation
	if secretID == "" {
		return "", fmt.Errorf("AWS secret ID cannot be empty")
	}
	// Load AWS SDK configuration
	// Automatically uses IRSA credentials from environment variables:
	// - AWS_ROLE_ARN
	// - AWS_WEB_IDENTITY_TOKEN_FILE
	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config (ensure IRSA is configured): %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Fetch secret value
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get secret from AWS Secrets Manager: %w", err)
	}

	// Return secret string (should be base64-encoded KSM config)
	if result.SecretString == nil {
		return "", fmt.Errorf("secret %s has no string value (binary secrets not supported)", secretID)
	}

	return *result.SecretString, nil
}
