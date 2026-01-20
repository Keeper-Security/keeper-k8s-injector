// Package config handles parsing and validation of Keeper injection annotations.
package config

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
)

const (
	// AnnotationPrefix is the prefix for all Keeper annotations
	AnnotationPrefix = "keeper.security/"

	// Core annotations
	AnnotationInject     = AnnotationPrefix + "inject"
	AnnotationSecret     = AnnotationPrefix + "secret"
	AnnotationSecrets    = AnnotationPrefix + "secrets"
	AnnotationConfig     = AnnotationPrefix + "config"
	AnnotationKSMConfig = AnnotationPrefix + "ksm-config"
	AnnotationAuthMethod = AnnotationPrefix + "auth-method"

	// Folder annotations
	AnnotationFolder     = AnnotationPrefix + "folder"      // Folder path (e.g., "Production/Databases")
	AnnotationFolderUID  = AnnotationPrefix + "folder-uid"  // Folder UID (direct reference)
	AnnotationFolderPath = AnnotationPrefix + "folder-path" // Output path for folder secrets

	// Behavior annotations
	AnnotationFailOnError     = AnnotationPrefix + "fail-on-error"
	AnnotationRefreshInterval = AnnotationPrefix + "refresh-interval"
	AnnotationInitOnly        = AnnotationPrefix + "init-only"
	AnnotationSignal          = AnnotationPrefix + "signal"
	AnnotationStrictLookup    = AnnotationPrefix + "strict-lookup"

	// Environment variable injection annotations
	AnnotationInjectEnvVars = AnnotationPrefix + "inject-env-vars" // Inject secrets as env vars instead of files
	AnnotationEnvPrefix     = AnnotationPrefix + "env-prefix"      // Optional prefix for env var names (e.g., "DB_")

	// Kubernetes Secret injection annotations (v0.9.0)
	AnnotationInjectAsK8sSecret  = AnnotationPrefix + "inject-as-k8s-secret" // Enable K8s Secret injection
	AnnotationK8sSecretName      = AnnotationPrefix + "k8s-secret-name"      // K8s Secret name (single secret mode)
	AnnotationK8sSecretNamespace = AnnotationPrefix + "k8s-secret-namespace" // Namespace override (defaults to pod namespace)
	AnnotationK8sSecretMode      = AnnotationPrefix + "k8s-secret-mode"      // Conflict resolution mode (overwrite|merge|skip-if-exists|fail)
	AnnotationK8sSecretType      = AnnotationPrefix + "k8s-secret-type"      // Secret type (Opaque, kubernetes.io/tls, etc.)
	AnnotationK8sSecretRotation  = AnnotationPrefix + "k8s-secret-rotation"  // Enable sidecar rotation (default: false)
	AnnotationK8sSecretOwnerRef  = AnnotationPrefix + "k8s-secret-owner-ref" // Add owner reference for auto-cleanup (default: true)

	// CA Certificate annotations (for corporate proxies/SSL inspection)
	AnnotationCACertSecret    = AnnotationPrefix + "ca-cert-secret"     // K8s Secret name with CA cert
	AnnotationCACertConfigMap = AnnotationPrefix + "ca-cert-configmap"  // K8s ConfigMap name with CA cert
	AnnotationCACertKey       = AnnotationPrefix + "ca-cert-key"        // Key in Secret/ConfigMap (default: "ca.crt")

	// Cloud Secrets Provider annotations (AWS/GCP/Azure)
	AnnotationAWSSecretID     = AnnotationPrefix + "aws-secret-id"      // AWS Secrets Manager secret ID
	AnnotationAWSRegion       = AnnotationPrefix + "aws-region"         // AWS region (optional, auto-detect)
	AnnotationGCPSecretID     = AnnotationPrefix + "gcp-secret-id"      // GCP Secret Manager resource name
	AnnotationAzureVaultName  = AnnotationPrefix + "azure-vault-name"   // Azure Key Vault name
	AnnotationAzureSecretName = AnnotationPrefix + "azure-secret-name"  // Azure secret name

	// Default values
	DefaultSecretsPath     = "/keeper/secrets"
	DefaultRefreshInterval = "5m"
	DefaultFailOnError     = "true"
	DefaultInitOnly        = "false"
	DefaultStrictLookup    = "false"

	// KeeperNotationPrefix is the URI scheme for Keeper notation
	KeeperNotationPrefix = "keeper://"
)

// SecretRef represents a reference to a secret in Keeper
type SecretRef struct {
	// Name is the title or UID of the secret in Keeper
	Name string
	// Path is where to write the secret (default: /keeper/secrets/{name}.json)
	Path string
	// Fields to extract (empty = all fields)
	Fields []string
	// Format: json, env, raw, properties, yaml, ini
	Format string
	// Template is a Go template string for custom formatting
	Template string
	// Notation is the full Keeper notation string (e.g., keeper://UID/field/password)
	// If set, this takes precedence over Name/Fields
	Notation string
	// FileName is for file attachment downloads (e.g., "cert.pem")
	FileName string
	// IsFile indicates this is a file attachment download
	IsFile bool

	// InjectAsEnvVars if true, inject as environment variables instead of files
	InjectAsEnvVars bool
	// EnvVarPrefix is an optional prefix for env var names (e.g., "DB_" → DB_PASSWORD)
	EnvVarPrefix string

	// K8s Secret injection (per-secret, v0.9.0)
	InjectAsK8sSecret bool              // Enable K8s Secret injection for this secret
	K8sSecretName     string            // K8s Secret name for this secret
	K8sSecretKeys     map[string]string // Keeper field → Secret key mapping
	K8sSecretType     string            // Secret type (Opaque, kubernetes.io/tls, etc.)
}

// FolderRef represents a reference to a folder in Keeper
type FolderRef struct {
	// FolderUID is the folder UID to fetch secrets from
	FolderUID string
	// FolderPath is the folder path (e.g., "Production/Databases")
	FolderPath string
	// OutputPath is where to write the secrets (default: /keeper/secrets)
	OutputPath string

	// K8s Secret injection (per-folder, v0.9.0)
	InjectAsK8sSecret   bool   // Enable K8s Secret injection for this folder
	K8sSecretNamePrefix string // Prefix for generated Secret names (e.g., "api-" → "api-stripe")
}

// InjectionConfig holds the parsed injection configuration for a pod
type InjectionConfig struct {
	// Enabled indicates if injection should occur
	Enabled bool
	// Secrets to inject
	Secrets []SecretRef
	// Folders to fetch all secrets from
	Folders []FolderRef
	// AuthSecretName is the name of the K8s secret containing KSM credentials
	AuthSecretName string
	// AuthSecretNamespace is the namespace of the auth secret (defaults to pod namespace)
	AuthSecretNamespace string
	// AuthMethod: "secret" or "oidc"
	AuthMethod string
	// RefreshInterval for sidecar rotation (e.g., "5m", "1h")
	RefreshInterval string
	// InitOnly if true, only use init container (no sidecar)
	InitOnly bool
	// FailOnError if true, pod fails to start if secrets can't be fetched
	FailOnError bool
	// Signal to send to app container on secret refresh (e.g., "SIGHUP")
	Signal string
	// CACertSecret is the name of the K8s Secret containing custom CA certificate
	CACertSecret string
	// CACertConfigMap is the name of the K8s ConfigMap containing custom CA certificate
	CACertConfigMap string
	// CACertKey is the key in the Secret/ConfigMap (default: "ca.crt")
	CACertKey string
	// StrictLookup if true, fail on duplicate title matches
	StrictLookup bool

	// Cloud Secrets Provider configuration
	AWSSecretID     string // AWS Secrets Manager secret ID/ARN
	AWSRegion       string // AWS region
	GCPSecretID     string // GCP Secret Manager resource name
	AzureVaultName  string // Azure Key Vault name
	AzureSecretName string // Azure Key Vault secret name

	// Environment variable injection configuration
	InjectEnvVars bool   // Global flag to inject all secrets as env vars
	EnvPrefix     string // Global prefix for all env var names

	// Kubernetes Secret injection configuration (v0.9.0)
	InjectAsK8sSecret  bool   // Global flag to enable K8s Secret injection
	K8sSecretName      string // K8s Secret name (single secret mode)
	K8sSecretNamespace string // Namespace override (defaults to pod namespace)
	K8sSecretMode      string // Conflict resolution mode (overwrite|merge|skip-if-exists|fail)
	K8sSecretType      string // Secret type (Opaque, kubernetes.io/tls, etc.)
	K8sSecretRotation  bool   // Enable sidecar rotation (default: false)
	K8sSecretOwnerRef  bool   // Add owner reference for auto-cleanup (default: true)
}

// ParseAnnotations extracts injection configuration from pod annotations
func ParseAnnotations(pod *corev1.Pod) (*InjectionConfig, error) {
	annotations := pod.Annotations
	if annotations == nil {
		return &InjectionConfig{Enabled: false}, nil
	}

	// Check if injection is enabled
	inject, ok := annotations[AnnotationInject]
	if !ok || strings.ToLower(inject) != "true" {
		return &InjectionConfig{Enabled: false}, nil
	}

	config := &InjectionConfig{
		Enabled:         true,
		AuthMethod:      "secret",
		RefreshInterval: DefaultRefreshInterval,
		InitOnly:        false,
		FailOnError:     true,
		StrictLookup:    false,
	}

	// Parse auth configuration
	if authSecret, ok := annotations[AnnotationKSMConfig]; ok {
		config.AuthSecretName = authSecret
	}
	if authMethod, ok := annotations[AnnotationAuthMethod]; ok {
		config.AuthMethod = authMethod
	}

	// Parse behavior annotations
	if failOnError, ok := annotations[AnnotationFailOnError]; ok {
		config.FailOnError = strings.ToLower(failOnError) == "true"
	}
	if refreshInterval, ok := annotations[AnnotationRefreshInterval]; ok {
		config.RefreshInterval = refreshInterval
	}
	if initOnly, ok := annotations[AnnotationInitOnly]; ok {
		config.InitOnly = strings.ToLower(initOnly) == "true"
	}
	if signal, ok := annotations[AnnotationSignal]; ok {
		config.Signal = signal
	}
	if strictLookup, ok := annotations[AnnotationStrictLookup]; ok {
		config.StrictLookup = strings.ToLower(strictLookup) == "true"
	}

	// Parse environment variable injection annotations
	if injectEnvVars, ok := annotations[AnnotationInjectEnvVars]; ok {
		config.InjectEnvVars = strings.ToLower(injectEnvVars) == "true"
	}
	if envPrefix, ok := annotations[AnnotationEnvPrefix]; ok {
		config.EnvPrefix = envPrefix
	}

	// Parse Kubernetes Secret injection annotations (v0.9.0)
	if injectAsK8sSecret, ok := annotations[AnnotationInjectAsK8sSecret]; ok {
		config.InjectAsK8sSecret = strings.ToLower(injectAsK8sSecret) == "true"
	}
	if k8sSecretName, ok := annotations[AnnotationK8sSecretName]; ok {
		config.K8sSecretName = k8sSecretName
	}
	if k8sSecretNamespace, ok := annotations[AnnotationK8sSecretNamespace]; ok {
		config.K8sSecretNamespace = k8sSecretNamespace
	}
	if k8sSecretMode, ok := annotations[AnnotationK8sSecretMode]; ok {
		config.K8sSecretMode = k8sSecretMode
	} else {
		config.K8sSecretMode = "overwrite" // Default mode
	}
	if k8sSecretType, ok := annotations[AnnotationK8sSecretType]; ok {
		config.K8sSecretType = k8sSecretType
	}
	if k8sSecretRotation, ok := annotations[AnnotationK8sSecretRotation]; ok {
		config.K8sSecretRotation = strings.ToLower(k8sSecretRotation) == "true"
	}
	// Owner reference defaults to true
	config.K8sSecretOwnerRef = true
	if k8sSecretOwnerRef, ok := annotations[AnnotationK8sSecretOwnerRef]; ok && strings.ToLower(k8sSecretOwnerRef) == "false" {
		config.K8sSecretOwnerRef = false
	}

	// Parse CA certificate configuration
	if caCertSecret, ok := annotations[AnnotationCACertSecret]; ok {
		config.CACertSecret = caCertSecret
	}
	if caCertConfigMap, ok := annotations[AnnotationCACertConfigMap]; ok {
		config.CACertConfigMap = caCertConfigMap
	}
	if caCertKey, ok := annotations[AnnotationCACertKey]; ok {
		config.CACertKey = caCertKey
	} else {
		config.CACertKey = "ca.crt" // Default key
	}

	// Parse cloud secrets provider configuration
	if awsSecretID, ok := annotations[AnnotationAWSSecretID]; ok {
		config.AWSSecretID = awsSecretID
	}
	if awsRegion, ok := annotations[AnnotationAWSRegion]; ok {
		config.AWSRegion = awsRegion
	}
	if gcpSecretID, ok := annotations[AnnotationGCPSecretID]; ok {
		config.GCPSecretID = gcpSecretID
	}
	if azureVaultName, ok := annotations[AnnotationAzureVaultName]; ok {
		config.AzureVaultName = azureVaultName
	}
	if azureSecretName, ok := annotations[AnnotationAzureSecretName]; ok {
		config.AzureSecretName = azureSecretName
	}

	// Parse secrets - Level 1: Single secret
	if secret, ok := annotations[AnnotationSecret]; ok {
		config.Secrets = append(config.Secrets, SecretRef{
			Name:            strings.TrimSpace(secret),
			Path:            fmt.Sprintf("%s/%s.json", DefaultSecretsPath, sanitizeName(secret)),
			Format:          "json",
			InjectAsEnvVars: config.InjectEnvVars,
			EnvVarPrefix:    config.EnvPrefix,
		})
	}

	// Parse secrets - Level 2: Multiple secrets (comma-separated)
	if secrets, ok := annotations[AnnotationSecrets]; ok {
		for _, s := range strings.Split(secrets, ",") {
			name := strings.TrimSpace(s)
			if name != "" {
				config.Secrets = append(config.Secrets, SecretRef{
					Name:            name,
					Path:            fmt.Sprintf("%s/%s.json", DefaultSecretsPath, sanitizeName(name)),
					Format:          "json",
					InjectAsEnvVars: config.InjectEnvVars,
					EnvVarPrefix:    config.EnvPrefix,
				})
			}
		}
	}

	// Parse secrets - Level 3: Custom paths (keeper.security/secret-{name} = path)
	// Also supports Keeper notation: keeper://UID/field/password:/path
	for key, value := range annotations {
		if strings.HasPrefix(key, AnnotationPrefix+"secret-") && key != AnnotationKSMConfig {
			name := strings.TrimPrefix(key, AnnotationPrefix+"secret-")
			secretRef := parseSecretAnnotation(name, value)
			config.Secrets = append(config.Secrets, secretRef)
		}
	}

	// Parse file attachments (keeper.security/file-{name} = record:filename:/path)
	for key, value := range annotations {
		if strings.HasPrefix(key, AnnotationPrefix+"file-") {
			name := strings.TrimPrefix(key, AnnotationPrefix+"file-")
			fileRef := parseFileAnnotation(name, value)
			config.Secrets = append(config.Secrets, fileRef)
		}
	}

	// Parse folder annotations (folder path or folder UID)
	folderPath, hasFolderPath := annotations[AnnotationFolder]
	folderUID, hasFolderUID := annotations[AnnotationFolderUID]
	if hasFolderPath || hasFolderUID {
		outputPath := DefaultSecretsPath
		if fp, ok := annotations[AnnotationFolderPath]; ok {
			outputPath = fp
		}
		config.Folders = append(config.Folders, FolderRef{
			FolderUID:  strings.TrimSpace(folderUID),
			FolderPath: strings.TrimSpace(folderPath),
			OutputPath: outputPath,
		})
	}

	// Parse secrets - Level 5: Full YAML config (escape hatch)
	if fullConfig, ok := annotations[AnnotationConfig]; ok {
		refs, folders, err := parseFullConfig(fullConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", AnnotationConfig, err)
		}
		config.Secrets = append(config.Secrets, refs...)
		config.Folders = append(config.Folders, folders...)
	}

	// Validate configuration
	if len(config.Secrets) == 0 && len(config.Folders) == 0 {
		return nil, fmt.Errorf("injection enabled but no secrets or folders specified")
	}
	if config.AuthSecretName == "" && config.AuthMethod == "secret" {
		return nil, fmt.Errorf("ksm-config annotation required when using secret auth method")
	}

	return config, nil
}

// parseSecretAnnotation parses Level 3/4 annotation values
// Supports multiple formats:
//   - "record[field]:path" - Field extraction with custom path
//   - "keeper://UID/field/password:/path" - Full Keeper notation with path
//   - "keeper://UID/field/password" - Keeper notation (auto-generate path)
//   - "/path/to/file" - Just a path (uses annotation key name as record)
func parseSecretAnnotation(name, value string) SecretRef {
	ref := SecretRef{
		Name:   name,
		Format: "json",
	}

	// Check if value is Keeper notation
	if strings.HasPrefix(value, KeeperNotationPrefix) || strings.HasPrefix(value, "/") && strings.Contains(value, "/field/") {
		return parseKeeperNotation(name, value)
	}

	// Check for field extraction syntax: "record[field]:path"
	if strings.Contains(value, "[") && strings.Contains(value, "]:") {
		// Level 4: Field extraction
		parts := strings.SplitN(value, "]:", 2)
		if len(parts) == 2 {
			recordAndField := parts[0]
			ref.Path = parts[1]

			// Parse record[field]
			bracketIdx := strings.Index(recordAndField, "[")
			if bracketIdx > 0 {
				ref.Name = recordAndField[:bracketIdx]
				ref.Fields = []string{recordAndField[bracketIdx+1:]}
				ref.Format = "raw" // Single field = raw value
			}
		}
	} else if strings.HasPrefix(value, "/") {
		// Level 3: Just a custom path
		ref.Path = value
	} else if strings.Contains(value, ":") {
		// Format: "record:path" or "record[field]:path"
		colonIdx := strings.LastIndex(value, ":")
		if colonIdx > 0 && strings.HasPrefix(value[colonIdx:], ":/") {
			ref.Name = value[:colonIdx]
			ref.Path = value[colonIdx+1:]
		} else {
			ref.Path = value
		}
	} else {
		ref.Path = value
	}

	// Default path if not set
	if ref.Path == "" {
		ref.Path = fmt.Sprintf("%s/%s.json", DefaultSecretsPath, sanitizeName(name))
	}

	return ref
}

// parseKeeperNotation parses Keeper notation format
// Format: keeper://UID/field/password:/path or keeper://UID/field/password
func parseKeeperNotation(name, value string) SecretRef {
	ref := SecretRef{
		Name:   name,
		Format: "raw", // Keeper notation typically returns single values
	}

	// Split notation from output path (separated by last ":" that's followed by "/")
	notation := value
	outputPath := ""

	// Find the output path - look for ":/" pattern after the notation
	// Keeper notation format: keeper://record/selector[/parameter][index]
	// We need to be careful because notation itself contains ":"
	if strings.HasPrefix(value, KeeperNotationPrefix) {
		// Remove prefix, find path
		rest := value[len(KeeperNotationPrefix):]
		// Look for :/ that indicates output path (not part of notation)
		lastColonSlash := strings.LastIndex(rest, ":/")
		if lastColonSlash > 0 {
			notation = KeeperNotationPrefix + rest[:lastColonSlash]
			outputPath = rest[lastColonSlash+1:]
		}
	} else {
		// Without keeper:// prefix (e.g., UID/field/password:/path)
		lastColonSlash := strings.LastIndex(value, ":/")
		if lastColonSlash > 0 {
			notation = value[:lastColonSlash]
			outputPath = value[lastColonSlash+1:]
		}
	}

	ref.Notation = notation

	// Extract record name from notation for default path
	recordName := extractRecordFromNotation(notation)
	if recordName != "" {
		ref.Name = recordName
	}

	// Determine format based on notation type
	if strings.Contains(notation, "/file/") {
		ref.IsFile = true
		// Extract filename from notation
		if idx := strings.Index(notation, "/file/"); idx >= 0 {
			ref.FileName = notation[idx+6:]
			// Remove any trailing index like [0]
			if bracketIdx := strings.Index(ref.FileName, "["); bracketIdx > 0 {
				ref.FileName = ref.FileName[:bracketIdx]
			}
		}
	}

	// Set output path
	if outputPath != "" {
		ref.Path = outputPath
	} else {
		// Generate default path based on notation
		ref.Path = fmt.Sprintf("%s/%s", DefaultSecretsPath, sanitizeName(name))
	}

	return ref
}

// extractRecordFromNotation extracts the record UID or title from notation
func extractRecordFromNotation(notation string) string {
	// Remove keeper:// prefix if present
	notation = strings.TrimPrefix(notation, KeeperNotationPrefix)

	// Find first "/" which separates record from selector
	slashIdx := strings.Index(notation, "/")
	if slashIdx > 0 {
		return notation[:slashIdx]
	}
	return notation
}

// parseFileAnnotation parses file attachment annotation values
// Format: "record:filename:/path" or "keeper://UID/file/filename:/path"
func parseFileAnnotation(name, value string) SecretRef {
	ref := SecretRef{
		Name:   name,
		Format: "raw",
		IsFile: true,
	}

	// Check if value is Keeper notation
	if strings.HasPrefix(value, KeeperNotationPrefix) {
		return parseKeeperNotation(name, value)
	}

	// Parse format: "record:filename:/path"
	parts := strings.SplitN(value, ":", 3)
	if len(parts) >= 2 {
		ref.Name = strings.TrimSpace(parts[0])
		ref.FileName = strings.TrimSpace(parts[1])
		if len(parts) == 3 {
			ref.Path = strings.TrimSpace(parts[2])
		}
	} else {
		// Just filename, use annotation name as record
		ref.FileName = value
	}

	// Default path
	if ref.Path == "" {
		if ref.FileName != "" {
			ref.Path = fmt.Sprintf("%s/%s", DefaultSecretsPath, ref.FileName)
		} else {
			ref.Path = fmt.Sprintf("%s/%s", DefaultSecretsPath, sanitizeName(name))
		}
	}

	return ref
}

// FullConfig represents the Level 5 YAML configuration structure
type FullConfig struct {
	Secrets []SecretYAMLConfig `yaml:"secrets,omitempty"`
	Folders []FolderYAMLConfig `yaml:"folders,omitempty"`
}

// SecretYAMLConfig represents a secret in YAML config
type SecretYAMLConfig struct {
	// Record is the record title or UID
	Record string `yaml:"record,omitempty"`
	// Notation is the full Keeper notation (alternative to Record)
	Notation string `yaml:"notation,omitempty"`
	// Path is where to write the secret
	Path string `yaml:"path,omitempty"`
	// Fields to extract (empty = all)
	Fields []string `yaml:"fields,omitempty"`
	// Format: json, env, raw, properties, yaml, ini
	Format string `yaml:"format,omitempty"`
	// Template is a Go template string for custom formatting
	Template string `yaml:"template,omitempty"`
	// File is for file attachment downloads
	File string `yaml:"file,omitempty"`
	// FileName is for file attachment downloads (alias for File, for clarity)
	FileName string `yaml:"fileName,omitempty"`
	// InjectAsEnvVars if true, inject this secret as env vars instead of file
	InjectAsEnvVars bool `yaml:"injectAsEnvVars,omitempty"`
	// EnvPrefix is an optional prefix for env var names
	EnvPrefix string `yaml:"envPrefix,omitempty"`
	// InjectAsK8sSecret if true, inject as K8s Secret object (v0.9.0)
	InjectAsK8sSecret bool `yaml:"injectAsK8sSecret,omitempty"`
	// K8sSecretName is the name of the K8s Secret to create
	K8sSecretName string `yaml:"k8sSecretName,omitempty"`
	// K8sSecretKeys maps Keeper fields to Secret keys
	K8sSecretKeys map[string]string `yaml:"k8sSecretKeys,omitempty"`
	// K8sSecretType is the Secret type (Opaque, kubernetes.io/tls, etc.)
	K8sSecretType string `yaml:"k8sSecretType,omitempty"`
}

// FolderYAMLConfig represents a folder in YAML config
type FolderYAMLConfig struct {
	// UID is the folder UID
	UID string `yaml:"uid,omitempty"`
	// Path is the folder path (e.g., "Production/Databases")
	Path string `yaml:"path,omitempty"`
	// FolderPath is an alias for Path (for consistency)
	FolderPath string `yaml:"folderPath,omitempty"`
	// OutputPath is where to write secrets
	OutputPath string `yaml:"outputPath,omitempty"`
	// InjectAsK8sSecret if true, inject as K8s Secret objects (v0.9.0)
	InjectAsK8sSecret bool `yaml:"injectAsK8sSecret,omitempty"`
	// K8sSecretNamePrefix is the prefix for generated Secret names
	K8sSecretNamePrefix string `yaml:"k8sSecretNamePrefix,omitempty"`
}

// parseFullConfig parses Level 5 YAML configuration
func parseFullConfig(configYAML string) ([]SecretRef, []FolderRef, error) {
	var cfg FullConfig
	if err := yaml.Unmarshal([]byte(configYAML), &cfg); err != nil {
		return nil, nil, fmt.Errorf("invalid YAML: %w", err)
	}

	var secrets []SecretRef
	for _, s := range cfg.Secrets {
		ref := SecretRef{
			Format:            s.Format,
			Template:          s.Template,
			InjectAsEnvVars:   s.InjectAsEnvVars,
			EnvVarPrefix:      s.EnvPrefix,
			InjectAsK8sSecret: s.InjectAsK8sSecret,
			K8sSecretName:     s.K8sSecretName,
			K8sSecretKeys:     s.K8sSecretKeys,
			K8sSecretType:     s.K8sSecretType,
		}

		// Set default format
		if ref.Format == "" {
			ref.Format = "json"
		}

		// Handle notation vs record
		if s.Notation != "" {
			ref.Notation = s.Notation
			ref.Name = extractRecordFromNotation(s.Notation)
			if strings.Contains(s.Notation, "/file/") {
				ref.IsFile = true
			}
		} else if s.Record != "" {
			ref.Name = s.Record
		}

		// Handle file attachment (support both "file" and "fileName" fields)
		fileName := s.File
		if fileName == "" {
			fileName = s.FileName
		}
		if fileName != "" {
			ref.IsFile = true
			ref.FileName = fileName
			ref.Format = "raw"
		}

		// Set fields
		ref.Fields = s.Fields
		if len(ref.Fields) == 1 {
			ref.Format = "raw"
		}

		// Set path
		if s.Path != "" {
			ref.Path = s.Path
		} else if ref.Name != "" {
			ref.Path = fmt.Sprintf("%s/%s.json", DefaultSecretsPath, sanitizeName(ref.Name))
		}

		secrets = append(secrets, ref)
	}

	var folders []FolderRef
	for _, f := range cfg.Folders {
		// Support both "path" and "folderPath" fields
		folderPath := f.Path
		if folderPath == "" {
			folderPath = f.FolderPath
		}

		ref := FolderRef{
			FolderUID:           f.UID,
			FolderPath:          folderPath,
			OutputPath:          f.OutputPath,
			InjectAsK8sSecret:   f.InjectAsK8sSecret,
			K8sSecretNamePrefix: f.K8sSecretNamePrefix,
		}
		if ref.OutputPath == "" {
			ref.OutputPath = DefaultSecretsPath
		}
		folders = append(folders, ref)
	}

	return secrets, folders, nil
}

// sanitizeName converts a secret name to a safe filename
func sanitizeName(name string) string {
	// Replace spaces and special chars with dashes
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "/", "-")
	return name
}

// ShouldInject returns true if the pod should have secrets injected
func ShouldInject(pod *corev1.Pod) bool {
	if pod.Annotations == nil {
		return false
	}
	inject, ok := pod.Annotations[AnnotationInject]
	return ok && strings.ToLower(inject) == "true"
}
