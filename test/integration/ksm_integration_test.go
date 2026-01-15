//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/keeper-security/keeper-k8s-injector/pkg/ksm"
	"github.com/keeper-security/keeper-k8s-injector/pkg/sidecar"
	"go.uber.org/zap"
)

func getKSMConfig(t *testing.T) string {
	// First try environment variable
	config := os.Getenv("KSM_CONFIG")
	if config != "" {
		return config
	}

	// Try reading from file
	configB64, err := os.ReadFile("../ksm-config.b64")
	if err != nil {
		t.Skipf("No KSM config available: %v", err)
	}

	// Decode base64 - trim any whitespace
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(configB64)))
	if err != nil {
		t.Fatalf("Failed to decode KSM config: %v", err)
	}

	return string(decoded)
}

func TestKSMClientConnection(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	client, err := ksm.NewClient(ctx, ksm.Config{
		ConfigJSON:  config,
		StrictMatch: false,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create KSM client: %v", err)
	}
	defer client.Close()

	t.Log("Successfully created KSM client connection")
}

func TestKSMListSecrets(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	client, err := ksm.NewClient(ctx, ksm.Config{
		ConfigJSON:  config,
		StrictMatch: false,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create KSM client: %v", err)
	}
	defer client.Close()

	// List all available secrets
	secrets, err := client.ListSecrets(ctx)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	t.Logf("Found %d secrets:", len(secrets))
	for _, secret := range secrets {
		fileInfo := ""
		if len(secret.Files) > 0 {
			fileInfo = fmt.Sprintf(", Files: %d", len(secret.Files))
			for _, f := range secret.Files {
				fileInfo += fmt.Sprintf(" [%s: %s, %d bytes]", f.Name, f.MimeType, f.Size)
			}
		}
		t.Logf("  - Title: '%s', UID: %s, Type: %s, Fields: %v%s",
			secret.Title, secret.RecordUID, secret.Type, getFieldNames(secret.Fields), fileInfo)
	}
}

func TestSidecarAgentInit(t *testing.T) {
	config := getKSMConfig(t)

	logger, _ := zap.NewDevelopment()

	// Create a temp directory for secrets
	tmpDir, err := os.MkdirTemp("", "keeper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create agent with minimal config
	agent, err := sidecar.NewAgent(&sidecar.AgentConfig{
		Mode:            sidecar.ModeInit,
		Secrets:         []sidecar.SecretConfig{},
		RefreshInterval: 0,
		FailOnError:     false, // Don't fail on missing secrets
		StrictLookup:    false,
		KSMConfig:       config,
		Logger:          logger,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	t.Logf("Successfully created sidecar agent")
	_ = agent // Agent created successfully
}

func TestFetchAndWriteSecret(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	// Create a temp directory for secrets
	tmpDir, err := os.MkdirTemp("", "keeper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Define secret path
	secretPath := tmpDir + "/database-credentials.json"

	// Create agent that fetches "Database Credentials" secret
	agent, err := sidecar.NewAgent(&sidecar.AgentConfig{
		Mode: sidecar.ModeInit,
		Secrets: []sidecar.SecretConfig{
			{
				Name:   "Database Credentials",
				Path:   secretPath,
				Format: "json",
			},
		},
		RefreshInterval: 0,
		FailOnError:     true,
		StrictLookup:    false,
		KSMConfig:       config,
		Logger:          logger,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Run the agent in init mode
	err = agent.Run(ctx)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Verify the secret file was created
	content, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read secret file: %v", err)
	}

	t.Logf("Secret file content:\n%s", string(content))

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		t.Fatalf("Secret file is not valid JSON: %v", err)
	}

	t.Logf("Successfully wrote secret with fields: %v", getFieldNames(data))
}

func TestFetchMultipleFormats(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	// Create a temp directory for secrets
	tmpDir, err := os.MkdirTemp("", "keeper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test different formats
	agent, err := sidecar.NewAgent(&sidecar.AgentConfig{
		Mode: sidecar.ModeInit,
		Secrets: []sidecar.SecretConfig{
			{
				Name:   "Database Credentials",
				Path:   tmpDir + "/db.json",
				Format: "json",
			},
			{
				Name:   "Database Credentials",
				Path:   tmpDir + "/db.env",
				Format: "env",
			},
		},
		RefreshInterval: 0,
		FailOnError:     true,
		StrictLookup:    false,
		KSMConfig:       config,
		Logger:          logger,
	})
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	err = agent.Run(ctx)
	if err != nil {
		t.Fatalf("Agent run failed: %v", err)
	}

	// Check JSON format
	jsonContent, err := os.ReadFile(tmpDir + "/db.json")
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}
	t.Logf("JSON format:\n%s", string(jsonContent))

	// Check ENV format
	envContent, err := os.ReadFile(tmpDir + "/db.env")
	if err != nil {
		t.Fatalf("Failed to read ENV file: %v", err)
	}
	t.Logf("ENV format:\n%s", string(envContent))
}

func getFieldNames(fields map[string]interface{}) []string {
	names := make([]string, 0, len(fields))
	for k := range fields {
		names = append(names, k)
	}
	return names
}

func TestFileAttachments(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	client, err := ksm.NewClient(ctx, ksm.Config{
		ConfigJSON:  config,
		StrictMatch: false,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create KSM client: %v", err)
	}
	defer client.Close()

	// Test fetching file from "ExcelPassword" record which has multiple files
	fileContent, err := client.GetFileContent(ctx, "ExcelPassword", "mykey.pem")
	if err != nil {
		t.Fatalf("Failed to get file content: %v", err)
	}

	t.Logf("File content (mykey.pem, %d bytes):\n%s", len(fileContent), string(fileContent[:min(200, len(fileContent))]))

	// Verify it looks like a PEM certificate
	if !strings.Contains(string(fileContent), "-----BEGIN") {
		t.Error("Expected PEM file to contain -----BEGIN")
	}

	// Test fetching command.txt from PAM record
	cmdContent, err := client.GetFileContent(ctx, "GitHub Action: Keeper Demo - PAM Machine Record", "command.txt")
	if err != nil {
		t.Fatalf("Failed to get command.txt: %v", err)
	}
	t.Logf("File content (command.txt, %d bytes):\n%s...", len(cmdContent), string(cmdContent[:min(200, len(cmdContent))]))
}

func TestDifferentRecordTypes(t *testing.T) {
	config := getKSMConfig(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger, _ := zap.NewDevelopment()

	client, err := ksm.NewClient(ctx, ksm.Config{
		ConfigJSON:  config,
		StrictMatch: false,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("Failed to create KSM client: %v", err)
	}
	defer client.Close()

	// Test different record types
	testCases := []struct {
		title      string
		recordType string
	}{
		{"Database Credentials", "login"},
		{"GitHub Action: Keeper Demo - PAM Machine Record", "pamMachine"},
		{"customtyperecord1", "CustomDemoType"},
	}

	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			secret, err := client.GetSecretByTitle(ctx, tc.title)
			if err != nil {
				t.Fatalf("Failed to get secret '%s': %v", tc.title, err)
			}

			t.Logf("Record: %s", tc.title)
			t.Logf("  Type: %s (expected: %s)", secret.Type, tc.recordType)
			t.Logf("  UID: %s", secret.RecordUID)
			t.Logf("  Fields: %v", secret.Fields)
			t.Logf("  Files: %d", len(secret.Files))

			if secret.Type != tc.recordType {
				t.Errorf("Expected type %s, got %s", tc.recordType, secret.Type)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
