package sidecar

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// scalarToString renders a field value as a single-line string, matching formatAsEnv's
// handling of string/[]byte and JSON-encoding anything else.
func scalarToString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// escapePropertiesValue makes a value safe for a single .properties / .ini line: a raw
// newline in a Keeper field value would otherwise start a spurious key (or, in INI, a
// fake [section]) and break java.util.Properties / INI parsers. Escape the line-significant
// characters per the .properties convention.
func escapePropertiesValue(v interface{}) string {
	s := scalarToString(v)
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

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
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, escapePropertiesValue(data[k])))
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
		buf.WriteString(fmt.Sprintf("%s=%s\n", k, escapePropertiesValue(data[k])))
	}

	return buf.Bytes()
}
