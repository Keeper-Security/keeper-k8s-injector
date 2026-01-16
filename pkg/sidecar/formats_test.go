package sidecar

import (
	"strings"
	"testing"
)

func TestFormatAsProperties(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
		"hostname": "db.example.com",
	}

	result := formatAsProperties(data)
	output := string(result)

	// Check all key-value pairs are present
	if !strings.Contains(output, "username=admin") {
		t.Errorf("missing username in properties output: %s", output)
	}
	if !strings.Contains(output, "password=secret123") {
		t.Errorf("missing password in properties output: %s", output)
	}
	if !strings.Contains(output, "hostname=db.example.com") {
		t.Errorf("missing hostname in properties output: %s", output)
	}

	// Check format is correct (key=value\n)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	for _, line := range lines {
		if !strings.Contains(line, "=") {
			t.Errorf("line doesn't contain '=': %s", line)
		}
	}
}

func TestFormatAsYAML(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
	}

	result, err := formatAsYAML(data)
	if err != nil {
		t.Fatalf("formatAsYAML() error = %v", err)
	}

	output := string(result)

	// Check YAML format
	if !strings.Contains(output, "username: admin") {
		t.Errorf("missing username in YAML output: %s", output)
	}
	if !strings.Contains(output, "password: secret123") {
		t.Errorf("missing password in YAML output: %s", output)
	}
}

func TestFormatAsINI(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "secret123",
	}

	result := formatAsINI(data)
	output := string(result)

	// Check INI format with [secret] section
	if !strings.HasPrefix(output, "[secret]\n") {
		t.Errorf("INI output should start with [secret]: %s", output)
	}

	if !strings.Contains(output, "username=admin") {
		t.Errorf("missing username in INI output: %s", output)
	}
	if !strings.Contains(output, "password=secret123") {
		t.Errorf("missing password in INI output: %s", output)
	}
}

func TestFormatAsProperties_EmptyData(t *testing.T) {
	data := map[string]interface{}{}
	result := formatAsProperties(data)

	if len(result) != 0 {
		t.Errorf("expected empty output for empty data, got: %s", string(result))
	}
}

func TestFormatAsYAML_ComplexTypes(t *testing.T) {
	data := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nested": map[string]interface{}{
			"key": "nested-value",
		},
	}

	result, err := formatAsYAML(data)
	if err != nil {
		t.Fatalf("formatAsYAML() error = %v", err)
	}

	output := string(result)

	// Verify complex types are handled
	if !strings.Contains(output, "string: value") {
		t.Errorf("missing string in YAML")
	}
	if !strings.Contains(output, "number: 42") {
		t.Errorf("missing number in YAML")
	}
	if !strings.Contains(output, "bool: true") {
		t.Errorf("missing bool in YAML")
	}
}
