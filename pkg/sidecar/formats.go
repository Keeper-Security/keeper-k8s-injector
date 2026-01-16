package sidecar

import (
	"bytes"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// formatAsProperties converts secret data to Java .properties format.
// Keys are sorted alphabetically for consistent output.
func formatAsProperties(data map[string]interface{}) []byte {
	var buf bytes.Buffer

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		buf.WriteString(fmt.Sprintf("%s=%v\n", k, v))
	}

	return buf.Bytes()
}

// formatAsYAML converts secret data to YAML format.
func formatAsYAML(data map[string]interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}

// formatAsINI converts secret data to INI format.
// All fields are placed under a [secret] section.
func formatAsINI(data map[string]interface{}) []byte {
	var buf bytes.Buffer
	buf.WriteString("[secret]\n")

	// Sort keys for consistent output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := data[k]
		buf.WriteString(fmt.Sprintf("%s=%v\n", k, v))
	}

	return buf.Bytes()
}
