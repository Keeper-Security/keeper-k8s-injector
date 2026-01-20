// Package ksm provides a wrapper around the Keeper Secrets Manager Go SDK.
package ksm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	"go.uber.org/zap"
)

// Client wraps the KSM SDK client with additional functionality
type Client struct {
	sm          *ksm.SecretsManager
	logger      *zap.Logger
	mu          sync.RWMutex
	strictMatch bool
}

// AuthMethod represents the authentication method to use
type AuthMethod string

const (
	// AuthMethodSecret uses a K8s Secret containing KSM config
	AuthMethodSecret AuthMethod = "secret"
	// AuthMethodOIDC uses OIDC token exchange with ServiceAccount tokens
	AuthMethodOIDC AuthMethod = "oidc"
)

// Config holds the configuration for creating a KSM client
type Config struct {
	// ConfigJSON is the base64-decoded KSM configuration JSON (for AuthMethodSecret)
	ConfigJSON string
	// AuthMethod specifies how to authenticate with KSM
	AuthMethod AuthMethod
	// OIDCConfig holds OIDC-specific configuration (for AuthMethodOIDC)
	OIDCConfig *OIDCConfig
	// StrictMatch if true, fails when multiple records match a title
	StrictMatch bool
	// Logger for client operations
	Logger *zap.Logger
}

// OIDCConfig holds OIDC authentication configuration
type OIDCConfig struct {
	// TokenPath is the path to the ServiceAccount token (default: /var/run/secrets/kubernetes.io/serviceaccount/token)
	TokenPath string
	// Audience is the expected audience for the token
	Audience string
	// IssuerURL is the OIDC issuer URL (if different from default)
	IssuerURL string
}

// SecretData represents retrieved secret data
type SecretData struct {
	// RecordUID is the unique identifier of the record
	RecordUID string `json:"uid"`
	// Title is the record title
	Title string `json:"title"`
	// Type is the record type (login, password, etc.)
	Type string `json:"type"`
	// Fields contains all record fields as key-value pairs
	Fields map[string]interface{} `json:"fields"`
	// Files contains file attachment metadata
	Files []FileInfo `json:"files,omitempty"`
}

// FileInfo represents a file attachment
type FileInfo struct {
	UID      string `json:"uid"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	MimeType string `json:"mimeType"`
	Size     int64  `json:"size"`
}

// NewClient creates a new KSM client from configuration
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	// Handle different auth methods
	switch cfg.AuthMethod {
	case AuthMethodOIDC:
		return nil, fmt.Errorf("OIDC authentication is not yet supported - requires Keeper backend support for OIDC token exchange. Use ksm-config with a K8s Secret containing your KSM configuration instead")

	case AuthMethodSecret, "":
		// Default: use config JSON from K8s Secret
		if cfg.ConfigJSON == "" {
			return nil, fmt.Errorf("ConfigJSON is required for secret auth method")
		}

		// Initialize KSM client from config JSON
		options := &ksm.ClientOptions{
			Config: ksm.NewMemoryKeyValueStorage(cfg.ConfigJSON),
		}

		sm := ksm.NewSecretsManager(options)

		return &Client{
			sm:          sm,
			logger:      cfg.Logger,
			strictMatch: cfg.StrictMatch,
		}, nil

	default:
		return nil, fmt.Errorf("unknown auth method: %s", cfg.AuthMethod)
	}
}

// GetSecretByTitle retrieves a secret by its title
// Returns the first match if multiple records have the same title (unless strict mode)
func (c *Client) GetSecretByTitle(ctx context.Context, title string) (*SecretData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching secret by title", zap.String("title", title))

	// Get all records and filter by title
	records, err := c.sm.GetSecrets([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	var matches []*ksm.Record
	for _, record := range records {
		if record.Title() == title {
			matches = append(matches, record)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no record found with title: %s", title)
	}

	if len(matches) > 1 && c.strictMatch {
		return nil, fmt.Errorf("multiple records (%d) found with title: %s (strict mode enabled)", len(matches), title)
	}

	if len(matches) > 1 {
		c.logger.Warn("multiple records match title, using first match",
			zap.String("title", title),
			zap.Int("count", len(matches)))
	}

	return c.recordToSecretData(matches[0])
}

// GetSecretByUID retrieves a secret by its UID
func (c *Client) GetSecretByUID(ctx context.Context, uid string) (*SecretData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching secret by UID", zap.String("uid", uid))

	records, err := c.sm.GetSecrets([]string{uid})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", uid, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no record found with UID: %s", uid)
	}

	return c.recordToSecretData(records[0])
}

// GetSecret retrieves a secret by title or UID (auto-detects)
func (c *Client) GetSecret(ctx context.Context, nameOrUID string) (*SecretData, error) {
	// If it looks like a UID (22 chars, alphanumeric with dashes), treat as UID
	if looksLikeUID(nameOrUID) {
		return c.GetSecretByUID(ctx, nameOrUID)
	}
	return c.GetSecretByTitle(ctx, nameOrUID)
}

// GetSecretField retrieves a specific field from a secret
func (c *Client) GetSecretField(ctx context.Context, nameOrUID, field string) ([]byte, error) {
	secret, err := c.GetSecret(ctx, nameOrUID)
	if err != nil {
		return nil, err
	}

	value, ok := secret.Fields[field]
	if !ok {
		return nil, fmt.Errorf("field %s not found in record %s", field, nameOrUID)
	}

	// Convert to bytes based on type
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	default:
		// JSON encode complex types
		return json.Marshal(v)
	}
}

// GetFileContent retrieves file content from a record
func (c *Client) GetFileContent(ctx context.Context, nameOrUID, fileName string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var records []*ksm.Record
	var err error

	if looksLikeUID(nameOrUID) {
		records, err = c.sm.GetSecrets([]string{nameOrUID})
	} else {
		// Get by title - need to fetch all and filter
		allRecords, fetchErr := c.sm.GetSecrets([]string{})
		if fetchErr != nil {
			return nil, fetchErr
		}
		for _, r := range allRecords {
			if r.Title() == nameOrUID {
				records = append(records, r)
				break
			}
		}
		err = nil
	}

	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("record not found: %s", nameOrUID)
	}

	record := records[0]
	// Use Files field directly, not as a method
	for _, f := range record.Files {
		if f.Name == fileName || f.Title == fileName {
			data := f.GetFileData()
			if data == nil {
				return nil, fmt.Errorf("failed to get file data for %s", fileName)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf("file %s not found in record %s", fileName, nameOrUID)
}

// recordToSecretData converts a KSM record to our SecretData format
// Extracts ALL fields (standard and custom) from the record
func (c *Client) recordToSecretData(record *ksm.Record) (*SecretData, error) {
	data := &SecretData{
		RecordUID: record.Uid,
		Title:     record.Title(),
		Type:      record.Type(),
		Fields:    make(map[string]interface{}),
	}

	// Extract ALL standard fields from RecordDict["fields"]
	if fieldsRaw, ok := record.RecordDict["fields"]; ok {
		if fields, ok := fieldsRaw.([]interface{}); ok {
			for _, f := range fields {
				if field, ok := f.(map[string]interface{}); ok {
					fieldType, _ := field["type"].(string)
					fieldLabel, _ := field["label"].(string)
					if value, ok := field["value"].([]interface{}); ok && len(value) > 0 {
						// Use label if available, otherwise use type
						key := fieldLabel
						if key == "" {
							key = fieldType
						}
						if key != "" {
							// If single value and it's a string, extract it
							if len(value) == 1 {
								if strVal, ok := value[0].(string); ok {
									data.Fields[key] = strVal
								} else {
									data.Fields[key] = value[0]
								}
							} else {
								data.Fields[key] = value
							}
						}
					}
				}
			}
		}
	}

	// Extract ALL custom fields from RecordDict["custom"]
	if customRaw, ok := record.RecordDict["custom"]; ok {
		if customs, ok := customRaw.([]interface{}); ok {
			for _, c := range customs {
				if custom, ok := c.(map[string]interface{}); ok {
					fieldType, _ := custom["type"].(string)
					fieldLabel, _ := custom["label"].(string)
					if value, ok := custom["value"].([]interface{}); ok && len(value) > 0 {
						// Use label if available, otherwise use type
						key := fieldLabel
						if key == "" {
							key = fieldType
						}
						if key != "" {
							// If single value and it's a string, extract it
							if len(value) == 1 {
								if strVal, ok := value[0].(string); ok {
									data.Fields[key] = strVal
								} else {
									data.Fields[key] = value[0]
								}
							} else {
								data.Fields[key] = value
							}
						}
					}
				}
			}
		}
	}

	// Extract notes if present
	if notes := record.GetFieldValueByType("note"); notes != "" {
		data.Fields["notes"] = notes
	}

	// Extract file metadata
	for _, f := range record.Files {
		data.Files = append(data.Files, FileInfo{
			UID:      f.Uid,
			Name:     f.Name,
			Title:    f.Title,
			MimeType: f.Type,
			Size:     int64(f.Size),
		})
	}

	return data, nil
}

// looksLikeUID checks if a string looks like a KSM record UID
func looksLikeUID(s string) bool {
	// KSM UIDs are typically 22 characters, base64-like
	if len(s) != 22 {
		return false
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

// ListSecrets returns all available secrets
func (c *Client) ListSecrets(ctx context.Context) ([]*SecretData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("listing all secrets")

	records, err := c.sm.GetSecrets([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var secrets []*SecretData
	for _, record := range records {
		secret, err := c.recordToSecretData(record)
		if err != nil {
			c.logger.Warn("failed to convert record", zap.String("uid", record.Uid), zap.Error(err))
			continue
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// GetNotation retrieves data using Keeper notation format
// Notation format: keeper://UID/field/password or UID/field/password
// Now supports folder paths: keeper://Production/Databases/mysql/field/password
// Supports: field, custom_field, file, type, title, notes
func (c *Client) GetNotation(ctx context.Context, notation string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching by notation", zap.String("notation", notation))

	// Try to parse as folder path notation
	np := parseNotationPath(notation)
	if np != nil && np.folderPath != "" {
		// This is a folder path notation - handle it specially
		c.logger.Debug("detected folder path in notation",
			zap.String("folderPath", np.folderPath),
			zap.String("recordName", np.recordName),
			zap.String("selector", np.selector),
			zap.String("parameter", np.parameter))

		// Get record by folder path
		folders, err := c.sm.GetFolders()
		if err != nil {
			return nil, fmt.Errorf("failed to get folders: %w", err)
		}
		tree := BuildFolderTree(folders)

		folderUID, err := tree.ResolvePath(np.folderPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve folder path '%s': %w", np.folderPath, err)
		}

		// Find record in folder
		records, err := c.sm.GetSecrets([]string{})
		if err != nil {
			return nil, fmt.Errorf("failed to get secrets: %w", err)
		}

		var matchedRecord *ksm.Record
		for _, record := range records {
			if (record.FolderUid() == folderUID || record.InnerFolderUid() == folderUID) &&
				(record.Title() == np.recordName || record.Uid == np.recordName) {
				matchedRecord = record
				break
			}
		}

		if matchedRecord == nil {
			return nil, fmt.Errorf("no record found with name '%s' in folder path '%s'", np.recordName, np.folderPath)
		}

		// If no selector, return entire record as JSON
		if np.selector == "" {
			secretData, err := c.recordToSecretData(matchedRecord)
			if err != nil {
				return nil, err
			}
			return json.Marshal(secretData)
		}

		// Apply selector to extract specific field
		switch np.selector {
		case "field":
			if np.parameter == "" {
				return nil, fmt.Errorf("field selector requires parameter (e.g., /field/password)")
			}
			// Use SDK's GetValue method on the record directly
			if val := matchedRecord.GetFieldValueByType(np.parameter); val != "" {
				return []byte(val), nil
			}
			// Try custom fields
			secretData, err := c.recordToSecretData(matchedRecord)
			if err != nil {
				return nil, err
			}
			if val, ok := secretData.Fields[np.parameter]; ok {
				if strVal, ok := val.(string); ok {
					return []byte(strVal), nil
				}
				return json.Marshal(val)
			}
			return nil, fmt.Errorf("field '%s' not found in record", np.parameter)

		case "custom_field":
			if np.parameter == "" {
				return nil, fmt.Errorf("custom_field selector requires parameter")
			}
			secretData, err := c.recordToSecretData(matchedRecord)
			if err != nil {
				return nil, err
			}
			if val, ok := secretData.Fields[np.parameter]; ok {
				if strVal, ok := val.(string); ok {
					return []byte(strVal), nil
				}
				return json.Marshal(val)
			}
			return nil, fmt.Errorf("custom field '%s' not found in record", np.parameter)

		case "file":
			if np.parameter == "" {
				return nil, fmt.Errorf("file selector requires parameter (filename)")
			}
			for _, f := range matchedRecord.Files {
				if f.Name == np.parameter || f.Title == np.parameter {
					data := f.GetFileData()
					if data == nil {
						return nil, fmt.Errorf("failed to get file data for %s", np.parameter)
					}
					return data, nil
				}
			}
			return nil, fmt.Errorf("file '%s' not found in record", np.parameter)

		case "type":
			return []byte(matchedRecord.Type()), nil

		case "title":
			return []byte(matchedRecord.Title()), nil

		case "notes":
			if notes := matchedRecord.GetFieldValueByType("note"); notes != "" {
				return []byte(notes), nil
			}
			return []byte{}, nil

		default:
			return nil, fmt.Errorf("unknown selector: %s", np.selector)
		}
	}

	// Not a folder path notation - use original SDK method
	results, err := c.sm.GetNotationResults(notation)
	if err != nil {
		return nil, fmt.Errorf("notation query failed: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("notation query returned no results: %s", notation)
	}

	// If single result, return as-is
	if len(results) == 1 {
		return []byte(results[0]), nil
	}

	// Multiple results - return as JSON array
	return json.Marshal(results)
}

// GetNotationValue is a convenience method that returns the first string result
func (c *Client) GetNotationValue(ctx context.Context, notation string) (string, error) {
	data, err := c.GetNotation(ctx, notation)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FolderInfo represents a Keeper folder
type FolderInfo struct {
	UID       string `json:"uid"`
	ParentUID string `json:"parentUid,omitempty"`
	Name      string `json:"name,omitempty"`
}

// GetFolders returns all accessible folders
func (c *Client) GetFolders(ctx context.Context) ([]FolderInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching folders")

	folders, err := c.sm.GetFolders()
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}

	var result []FolderInfo
	for _, f := range folders {
		result = append(result, FolderInfo{
			UID:       f.FolderUid,
			ParentUID: f.ParentUid,
			Name:      f.Name,
		})
	}

	return result, nil
}

// GetSecretsInFolder returns all secrets in a folder by folder UID
func (c *Client) GetSecretsInFolder(ctx context.Context, folderUID string) ([]*SecretData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching secrets in folder", zap.String("folderUID", folderUID))

	// Get all records and filter by folder
	records, err := c.sm.GetSecrets([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	var secrets []*SecretData
	for _, record := range records {
		// Check if record is in the specified folder
		if record.FolderUid() == folderUID || record.InnerFolderUid() == folderUID {
			secret, err := c.recordToSecretData(record)
			if err != nil {
				c.logger.Warn("failed to convert record", zap.String("uid", record.Uid), zap.Error(err))
				continue
			}
			secrets = append(secrets, secret)
		}
	}

	return secrets, nil
}

// BuildFolderTree fetches all folders and builds a hierarchical tree
func (c *Client) BuildFolderTree(ctx context.Context) (*FolderTree, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("building folder tree")

	folders, err := c.sm.GetFolders()
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}

	return BuildFolderTree(folders), nil
}

// GetSecretByPath retrieves a secret using folder path and record name
// Example: GetSecretByPath(ctx, "Production/Databases", "mysql-creds")
func (c *Client) GetSecretByPath(ctx context.Context, folderPath, recordName string) (*SecretData, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.logger.Debug("fetching secret by path",
		zap.String("folderPath", folderPath),
		zap.String("recordName", recordName))

	// Build folder tree
	folders, err := c.sm.GetFolders()
	if err != nil {
		return nil, fmt.Errorf("failed to get folders: %w", err)
	}
	tree := BuildFolderTree(folders)

	// Resolve folder path to UID
	folderUID, err := tree.ResolvePath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve folder path: %w", err)
	}

	// Get all secrets in the folder
	records, err := c.sm.GetSecrets([]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}

	// Filter records by folder and find matching record
	var matches []*ksm.Record
	for _, record := range records {
		// Check if record is in the target folder
		if record.FolderUid() == folderUID || record.InnerFolderUid() == folderUID {
			// Check if record matches by title or UID
			if record.Title() == recordName || record.Uid == recordName {
				matches = append(matches, record)
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no record found with name '%s' in folder path '%s'", recordName, folderPath)
	}

	if len(matches) > 1 && c.strictMatch {
		return nil, fmt.Errorf("multiple records (%d) found with name '%s' in folder '%s' (strict mode enabled)", len(matches), recordName, folderPath)
	}

	if len(matches) > 1 {
		c.logger.Warn("multiple records match in folder, using first match",
			zap.String("recordName", recordName),
			zap.String("folderPath", folderPath),
			zap.Int("count", len(matches)))
	}

	return c.recordToSecretData(matches[0])
}

// notationParts represents parsed notation components
type notationParts struct {
	folderPath   string // Folder path (e.g., "Production/Databases")
	recordName   string // Record title or UID
	selector     string // "field", "custom_field", "file", "type", "title", "notes"
	parameter    string // Parameter for selector (e.g., "password" for field)
	hasKeeperURI bool   // Whether notation starts with "keeper://"
}

// parseNotationPath parses a notation string and detects folder paths
// Returns nil if the notation doesn't contain a folder path
func parseNotationPath(notation string) *notationParts {
	// Strip keeper:// prefix if present
	hasPrefix := false
	if strings.HasPrefix(notation, "keeper://") {
		notation = strings.TrimPrefix(notation, "keeper://")
		hasPrefix = true
	}

	// Trim leading and trailing slashes
	notation = strings.Trim(notation, "/")

	// Split by / and filter out empty parts
	rawParts := strings.Split(notation, "/")
	var parts []string
	for _, p := range rawParts {
		if p != "" {
			parts = append(parts, p)
		}
	}

	if len(parts) < 2 {
		// Not enough parts for folder path + record
		return nil
	}

	// Look for selector keywords to identify where record name ends
	selectorKeywords := []string{"field", "custom_field", "file", "type", "title", "notes"}
	selectorIndex := -1
	for i, part := range parts {
		for _, keyword := range selectorKeywords {
			if part == keyword {
				selectorIndex = i
				break
			}
		}
		if selectorIndex != -1 {
			break
		}
	}

	// If no selector found, check if last part looks like a UID
	// In that case, the entire path might be folder path + UID
	if selectorIndex == -1 {
		// Notation like "Production/Databases/ABC123XYZ" (folder path + UID)
		// or "Production/Databases/record-title" (folder path + title)
		if len(parts) >= 2 {
			// Last part is record name, everything before is folder path
			recordName := parts[len(parts)-1]
			folderPath := strings.Join(parts[:len(parts)-1], "/")
			return &notationParts{
				folderPath:   folderPath,
				recordName:   recordName,
				selector:     "",
				parameter:    "",
				hasKeeperURI: hasPrefix,
			}
		}
		return nil
	}

	// Selector found - everything before selector is folder path + record
	if selectorIndex < 2 {
		// Not enough parts for folder path (need at least folder + record + selector)
		return nil
	}

	// Parse: [folder.../record]/selector[/parameter]
	recordName := parts[selectorIndex-1]
	folderPath := ""
	if selectorIndex > 1 {
		folderPath = strings.Join(parts[:selectorIndex-1], "/")
	}

	np := &notationParts{
		folderPath:   folderPath,
		recordName:   recordName,
		selector:     parts[selectorIndex],
		hasKeeperURI: hasPrefix,
	}

	// Get parameter if present
	if len(parts) > selectorIndex+1 {
		np.parameter = parts[selectorIndex+1]
	}

	return np
}

// Close releases any resources held by the client
func (c *Client) Close() error {
	// KSM SDK doesn't require explicit cleanup
	return nil
}
