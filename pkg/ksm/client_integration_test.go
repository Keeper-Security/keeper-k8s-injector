// +build integration

package ksm

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// This integration test requires:
// 1. A real Keeper vault with folder structure
// 2. KSM config in /Users/mustinov/Source/keeper-k8s-injector/config.base64
//
// Vault structure expected:
// - Keeper K8s Injector Test (root folder)
//   - dev/apis/Stripe POS API
//   - dev/dbs/salesdb
//   - prod/apis/Stripe POS API
//   - prod/dbs/salesdb
//   - (root level) demo-secret
//   - (root level) postgres-credentials

func loadKSMConfig(t *testing.T) string {
	// Try mounted path first (for Docker), then local path
	configPaths := []string{
		"/keeper/config.base64",
		"/Users/mustinov/Source/keeper-k8s-injector/config.base64",
	}

	var data []byte
	var err error
	var configPath string

	for _, path := range configPaths {
		data, err = os.ReadFile(path)
		if err == nil {
			configPath = path
			break
		}
	}

	if data == nil {
		t.Skipf("Skipping integration test: config file not found at any of: %v", configPaths)
		return ""
	}

	t.Logf("Using config from: %s", configPath)

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		t.Fatalf("Failed to decode base64 config: %v", err)
	}

	return string(decoded)
}

func TestIntegration_FolderTree(t *testing.T) {
	configJSON := loadKSMConfig(t)
	if configJSON == "" {
		return
	}

	ctx := context.Background()
	logger := zap.NewNop()

	client, err := NewClient(ctx, Config{
		ConfigJSON:  configJSON,
		AuthMethod:  AuthMethodSecret,
		StrictMatch: false,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Build folder tree
	tree, err := client.BuildFolderTree(ctx)
	require.NoError(t, err)
	require.NotNil(t, tree)

	// List all folder paths
	paths := tree.ListPaths()
	t.Logf("Found %d folders:", len(paths))
	for _, path := range paths {
		t.Logf("  - %s", path)
	}

	// Verify expected folder paths exist
	// Note: Folder paths include the root shared folder name
	expectedPaths := []string{
		"Keeper K8s Injector Test",
		"Keeper K8s Injector Test/dev",
		"Keeper K8s Injector Test/dev/apis",
		"Keeper K8s Injector Test/dev/dbs",
		"Keeper K8s Injector Test/prod",
		"Keeper K8s Injector Test/prod/apis",
		"Keeper K8s Injector Test/prod/dbs",
	}

	for _, expected := range expectedPaths {
		uid, err := tree.ResolvePath(expected)
		assert.NoError(t, err, "Failed to resolve path: %s", expected)
		assert.NotEmpty(t, uid, "Path %s resolved to empty UID", expected)
		t.Logf("Path '%s' resolved to UID: %s", expected, uid)
	}
}

func TestIntegration_GetSecretByPath(t *testing.T) {
	configJSON := loadKSMConfig(t)
	if configJSON == "" {
		return
	}

	ctx := context.Background()
	logger := zap.NewNop()

	client, err := NewClient(ctx, Config{
		ConfigJSON:  configJSON,
		AuthMethod:  AuthMethodSecret,
		StrictMatch: false,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name       string
		folderPath string
		recordName string
	}{
		{
			name:       "dev apis Stripe POS API",
			folderPath: "Keeper K8s Injector Test/dev/apis",
			recordName: "Stripe POS API",
		},
		{
			name:       "dev dbs salesdb",
			folderPath: "Keeper K8s Injector Test/dev/dbs",
			recordName: "salesdb",
		},
		{
			name:       "prod apis Stripe POS API",
			folderPath: "Keeper K8s Injector Test/prod/apis",
			recordName: "Stripe POS API",
		},
		{
			name:       "prod dbs salesdb",
			folderPath: "Keeper K8s Injector Test/prod/dbs",
			recordName: "salesdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := client.GetSecretByPath(ctx, tt.folderPath, tt.recordName)
			require.NoError(t, err)
			require.NotNil(t, secret)
			assert.Equal(t, tt.recordName, secret.Title)
			assert.NotEmpty(t, secret.RecordUID)
			assert.NotEmpty(t, secret.Fields)

			t.Logf("Found secret '%s' in folder '%s':", tt.recordName, tt.folderPath)
			t.Logf("  UID: %s", secret.RecordUID)
			t.Logf("  Type: %s", secret.Type)
			t.Logf("  Fields: %d", len(secret.Fields))
			for key := range secret.Fields {
				t.Logf("    - %s", key)
			}
		})
	}
}

func TestIntegration_NotationWithFolderPath(t *testing.T) {
	configJSON := loadKSMConfig(t)
	if configJSON == "" {
		return
	}

	ctx := context.Background()
	logger := zap.NewNop()

	client, err := NewClient(ctx, Config{
		ConfigJSON:  configJSON,
		AuthMethod:  AuthMethodSecret,
		StrictMatch: false,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer client.Close()

	tests := []struct {
		name     string
		notation string
	}{
		{
			name:     "dev apis stripe - whole record",
			notation: "Keeper K8s Injector Test/dev/apis/Stripe POS API",
		},
		{
			name:     "dev dbs salesdb - whole record",
			notation: "Keeper K8s Injector Test/dev/dbs/salesdb",
		},
		{
			name:     "prod apis stripe - whole record",
			notation: "keeper://Keeper K8s Injector Test/prod/apis/Stripe POS API",
		},
		{
			name:     "prod dbs salesdb - title",
			notation: "Keeper K8s Injector Test/prod/dbs/salesdb/title",
		},
		{
			name:     "prod dbs salesdb - type",
			notation: "keeper://Keeper K8s Injector Test/prod/dbs/salesdb/type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := client.GetNotation(ctx, tt.notation)
			require.NoError(t, err)
			require.NotNil(t, data)
			assert.NotEmpty(t, data)

			t.Logf("Notation '%s' returned: %s", tt.notation, string(data))
		})
	}
}

func TestIntegration_NotationWithFolderPath_FieldSelector(t *testing.T) {
	configJSON := loadKSMConfig(t)
	if configJSON == "" {
		return
	}

	ctx := context.Background()
	logger := zap.NewNop()

	client, err := NewClient(ctx, Config{
		ConfigJSON:  configJSON,
		AuthMethod:  AuthMethodSecret,
		StrictMatch: false,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer client.Close()

	// First, get the record to see what fields are available
	secret, err := client.GetSecretByPath(ctx, "Keeper K8s Injector Test/dev/dbs", "salesdb")
	require.NoError(t, err)

	t.Logf("salesdb fields:")
	for key := range secret.Fields {
		t.Logf("  - %s", key)
	}

	// Test field extraction if password field exists
	if _, ok := secret.Fields["password"]; ok {
		notation := "Keeper K8s Injector Test/dev/dbs/salesdb/field/password"
		data, err := client.GetNotation(ctx, notation)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		t.Logf("Password from notation: %s", string(data))
	} else {
		t.Log("No password field found, skipping field extraction test")
	}
}

func TestIntegration_FolderPathNotFound(t *testing.T) {
	configJSON := loadKSMConfig(t)
	if configJSON == "" {
		return
	}

	ctx := context.Background()
	logger := zap.NewNop()

	client, err := NewClient(ctx, Config{
		ConfigJSON:  configJSON,
		AuthMethod:  AuthMethodSecret,
		StrictMatch: false,
		Logger:      logger,
	})
	require.NoError(t, err)
	defer client.Close()

	// Test non-existent folder path
	_, err = client.GetSecretByPath(ctx, "nonexistent/folder", "record")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve folder path")

	// Test non-existent record in valid folder
	_, err = client.GetSecretByPath(ctx, "Keeper K8s Injector Test/dev/apis", "nonexistent-record")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no record found")
}
