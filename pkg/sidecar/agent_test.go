package sidecar

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestFormatAsEnv(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
		"port":     5432,
	}

	result := formatAsEnv(data)
	resultStr := string(result)

	// Check that all keys are present (order may vary)
	if !contains(resultStr, "USERNAME=admin") {
		t.Error("Expected USERNAME=admin in output")
	}
	if !contains(resultStr, "PASSWORD=secret123") {
		t.Error("Expected PASSWORD=secret123 in output")
	}
	if !contains(resultStr, "PORT=5432") {
		t.Error("Expected PORT=5432 in output")
	}
}

func TestToEnvKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"username", "USERNAME"},
		{"userName", "USERNAME"},
		{"user_name", "USER_NAME"},
		{"user-name", "USER_NAME"},
		{"USER", "USER"},
		{"port123", "PORT123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := toEnvKey(tt.input); got != tt.want {
				t.Errorf("toEnvKey(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeEnvValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"with space", "'with space'"},
		{"with\nnewline", "'with\nnewline'"},
		{"with=equals", "'with=equals'"},
		{"with'quote", "'with'\\''quote'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := escapeEnvValue(tt.input); got != tt.want {
				t.Errorf("escapeEnvValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSecret(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "secret",
	}

	// Test JSON format
	jsonResult, err := formatSecret(data, SecretConfig{Format: "json"})
	if err != nil {
		t.Fatalf("formatSecret(json) error = %v", err)
	}
	if !contains(string(jsonResult), `"username"`) {
		t.Error("JSON output should contain username key")
	}

	// Test env format
	envResult, err := formatSecret(data, SecretConfig{Format: "env"})
	if err != nil {
		t.Fatalf("formatSecret(env) error = %v", err)
	}
	if !contains(string(envResult), "USERNAME=admin") {
		t.Error("ENV output should contain USERNAME=admin")
	}

	// Test raw format with single value
	singleData := map[string]interface{}{
		"password": "secret123",
	}
	rawResult, err := formatSecret(singleData, SecretConfig{Format: "raw"})
	if err != nil {
		t.Fatalf("formatSecret(raw) error = %v", err)
	}
	if string(rawResult) != "secret123" {
		t.Errorf("Raw output = %q, want %q", string(rawResult), "secret123")
	}
}

func TestWriteSecretFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "keeper-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tmpDir) // Ignore cleanup errors in tests
	}()

	agent := &Agent{
		logger: zap.NewNop(), // Use nop logger
	}

	// Test writing a file
	testPath := filepath.Join(tmpDir, "secrets", "test.json")
	testData := []byte(`{"key": "value"}`)

	err = agent.writeSecretFile(testPath, testData)
	if err != nil {
		t.Fatalf("writeSecretFile() error = %v", err)
	}

	// Verify file exists and has correct content
	content, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("Failed to read written file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("File content = %q, want %q", string(content), string(testData))
	}

	// Verify file permissions
	info, err := os.Stat(testPath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0400 {
		t.Errorf("File permissions = %o, want 0400", info.Mode().Perm())
	}
}

func TestNewAgent(t *testing.T) {
	cfg := &AgentConfig{
		Mode:            ModeInit,
		RefreshInterval: 0,
		FailOnError:     true,
		Secrets: []SecretConfig{
			{Name: "test", Path: "/tmp/test.json", Format: "json"},
		},
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent == nil {
		t.Fatal("NewAgent() returned nil")
	}
	if agent.config.Mode != ModeInit {
		t.Errorf("agent.config.Mode = %v, want %v", agent.config.Mode, ModeInit)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
