package sidecar

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// renderTemplate executes a Go template with secret data.
// Uses Sprig library for comprehensive template functions.
func renderTemplate(data map[string]interface{}, templateStr string) ([]byte, error) {
	if templateStr == "" {
		return nil, fmt.Errorf("template string is empty")
	}

	// Create template with Sprig functions
	tmpl, err := template.New("secret").Funcs(templateFuncs()).Parse(templateStr)
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("template execute error: %w", err)
	}

	return buf.Bytes(), nil
}

// templateFuncs returns the template function map.
// Uses Sprig as base (100+ functions) and adds Keeper-specific overrides.
func templateFuncs() template.FuncMap {
	// Start with Sprig's comprehensive function library
	// Provides: date/time, crypto, string, math, encoding, and more
	funcs := sprig.TxtFuncMap()

	// Add or override with Keeper-specific functions
	funcs["base64enc"] = base64Encode
	funcs["base64dec"] = base64Decode
	funcs["sha256sum"] = sha256Hash
	funcs["sha512sum"] = sha512Hash

	return funcs
}

// base64Encode encodes a string to base64.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// base64Decode decodes a base64 string.
func base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}
	return string(data), nil
}

// sha256Hash computes SHA-256 hash of a string.
func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// sha512Hash computes SHA-512 hash of a string.
func sha512Hash(s string) string {
	h := sha512.Sum512([]byte(s))
	return hex.EncodeToString(h[:])
}
