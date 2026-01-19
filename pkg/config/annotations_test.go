package config

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestShouldInject(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        bool
	}{
		{
			name:        "no annotations",
			annotations: nil,
			want:        false,
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			want:        false,
		},
		{
			name: "inject true",
			annotations: map[string]string{
				"keeper.security/inject": "true",
			},
			want: true,
		},
		{
			name: "inject TRUE (uppercase)",
			annotations: map[string]string{
				"keeper.security/inject": "TRUE",
			},
			want: true,
		},
		{
			name: "inject false",
			annotations: map[string]string{
				"keeper.security/inject": "false",
			},
			want: false,
		},
		{
			name: "inject empty",
			annotations: map[string]string{
				"keeper.security/inject": "",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}
			if got := ShouldInject(pod); got != tt.want {
				t.Errorf("ShouldInject() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseAnnotations_Level1_SingleSecret(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				"keeper.security/secret":      "database-credentials",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled = true")
	}
	if cfg.AuthSecretName != "keeper-auth" {
		t.Errorf("AuthSecretName = %v, want keeper-auth", cfg.AuthSecretName)
	}
	if len(cfg.Secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(cfg.Secrets))
	}
	if cfg.Secrets[0].Name != "database-credentials" {
		t.Errorf("Secret name = %v, want database-credentials", cfg.Secrets[0].Name)
	}
	if cfg.Secrets[0].Path != "/keeper/secrets/database-credentials.json" {
		t.Errorf("Secret path = %v, want /keeper/secrets/database-credentials.json", cfg.Secrets[0].Path)
	}
}

func TestParseAnnotations_Level2_MultipleSecrets(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				"keeper.security/secrets":     "database-creds, api-keys, tls-cert",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 3 {
		t.Fatalf("Expected 3 secrets, got %d", len(cfg.Secrets))
	}

	expectedNames := []string{"database-creds", "api-keys", "tls-cert"}
	for i, expected := range expectedNames {
		if cfg.Secrets[i].Name != expected {
			t.Errorf("Secret[%d].Name = %v, want %v", i, cfg.Secrets[i].Name, expected)
		}
	}
}

func TestParseAnnotations_Level3_CustomPaths(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":          "true",
				"keeper.security/auth-secret":     "keeper-auth",
				"keeper.security/secret-database": "/app/config/db.json",
				"keeper.security/secret-api":      "/etc/myapp/api.json",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 2 {
		t.Fatalf("Expected 2 secrets, got %d", len(cfg.Secrets))
	}

	// Find database secret
	var dbSecret, apiSecret *SecretRef
	for i := range cfg.Secrets {
		if cfg.Secrets[i].Name == "database" {
			dbSecret = &cfg.Secrets[i]
		}
		if cfg.Secrets[i].Name == "api" {
			apiSecret = &cfg.Secrets[i]
		}
	}

	if dbSecret == nil {
		t.Error("database secret not found")
	} else if dbSecret.Path != "/app/config/db.json" {
		t.Errorf("database path = %v, want /app/config/db.json", dbSecret.Path)
	}

	if apiSecret == nil {
		t.Error("api secret not found")
	} else if apiSecret.Path != "/etc/myapp/api.json" {
		t.Errorf("api path = %v, want /etc/myapp/api.json", apiSecret.Path)
	}
}

func TestParseAnnotations_BehaviorAnnotations(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":           "true",
				"keeper.security/auth-secret":      "keeper-auth",
				"keeper.security/secret":           "test-secret",
				"keeper.security/refresh-interval": "10m",
				"keeper.security/fail-on-error":    "false",
				"keeper.security/init-only":        "true",
				"keeper.security/signal":           "SIGHUP",
				"keeper.security/strict-lookup":    "true",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if cfg.RefreshInterval != "10m" {
		t.Errorf("RefreshInterval = %v, want 10m", cfg.RefreshInterval)
	}
	if cfg.FailOnError != false {
		t.Errorf("FailOnError = %v, want false", cfg.FailOnError)
	}
	if cfg.InitOnly != true {
		t.Errorf("InitOnly = %v, want true", cfg.InitOnly)
	}
	if cfg.Signal != "SIGHUP" {
		t.Errorf("Signal = %v, want SIGHUP", cfg.Signal)
	}
	if cfg.StrictLookup != true {
		t.Errorf("StrictLookup = %v, want true", cfg.StrictLookup)
	}
}

func TestParseAnnotations_MissingAuthSecret(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject": "true",
				"keeper.security/secret": "test-secret",
				// Missing auth-secret
			},
		},
	}

	_, err := ParseAnnotations(pod)
	if err == nil {
		t.Error("Expected error for missing auth-secret")
	}
}

func TestParseAnnotations_NoSecrets(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				// No secrets specified
			},
		},
	}

	_, err := ParseAnnotations(pod)
	if err == nil {
		t.Error("Expected error when no secrets specified")
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"With Spaces", "with-spaces"},
		{"With/Slashes", "with-slashes"},
		{"UPPERCASE", "uppercase"},
		{"Mixed Case Name", "mixed-case-name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeName(tt.input); got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseAnnotations_KeeperNotation(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":         "true",
				"keeper.security/auth-secret":    "keeper-auth",
				"keeper.security/secret-db-pass": "keeper://ABC123def456GHI789jkl/field/password:/app/secrets/db-pass",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(cfg.Secrets))
	}

	secret := cfg.Secrets[0]
	if secret.Notation != "keeper://ABC123def456GHI789jkl/field/password" {
		t.Errorf("Notation = %v, want keeper://ABC123def456GHI789jkl/field/password", secret.Notation)
	}
	if secret.Path != "/app/secrets/db-pass" {
		t.Errorf("Path = %v, want /app/secrets/db-pass", secret.Path)
	}
}

func TestParseAnnotations_FileAttachment(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				"keeper.security/file-cert":   "Database Credentials:cert.pem:/app/certs/cert.pem",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(cfg.Secrets))
	}

	secret := cfg.Secrets[0]
	if !secret.IsFile {
		t.Error("Expected IsFile = true")
	}
	if secret.Name != "Database Credentials" {
		t.Errorf("Name = %v, want 'Database Credentials'", secret.Name)
	}
	if secret.FileName != "cert.pem" {
		t.Errorf("FileName = %v, want cert.pem", secret.FileName)
	}
	if secret.Path != "/app/certs/cert.pem" {
		t.Errorf("Path = %v, want /app/certs/cert.pem", secret.Path)
	}
}

func TestParseAnnotations_FolderSupport(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				"keeper.security/folder":      "Production/Databases",
				"keeper.security/folder-path": "/app/db-secrets",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(cfg.Folders))
	}

	folder := cfg.Folders[0]
	if folder.FolderPath != "Production/Databases" {
		t.Errorf("FolderPath = %v, want Production/Databases", folder.FolderPath)
	}
	if folder.OutputPath != "/app/db-secrets" {
		t.Errorf("OutputPath = %v, want /app/db-secrets", folder.OutputPath)
	}
}

func TestParseAnnotations_FullYAMLConfig(t *testing.T) {
	configYAML := `
secrets:
  - record: "Database Credentials"
    path: /app/config/db.json
    format: json
  - notation: "keeper://ABC123/field/password"
    path: /app/secrets/password
    format: raw
  - record: "Certificate Store"
    file: cert.pem
    path: /app/certs/cert.pem
folders:
  - uid: "FOLDER123"
    outputPath: /app/folder-secrets
`
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth",
				"keeper.security/config":      configYAML,
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 3 {
		t.Fatalf("Expected 3 secrets, got %d", len(cfg.Secrets))
	}

	// Check first secret (record with path)
	if cfg.Secrets[0].Name != "Database Credentials" {
		t.Errorf("Secrets[0].Name = %v, want 'Database Credentials'", cfg.Secrets[0].Name)
	}
	if cfg.Secrets[0].Path != "/app/config/db.json" {
		t.Errorf("Secrets[0].Path = %v, want /app/config/db.json", cfg.Secrets[0].Path)
	}

	// Check second secret (notation)
	if cfg.Secrets[1].Notation != "keeper://ABC123/field/password" {
		t.Errorf("Secrets[1].Notation = %v, want keeper://ABC123/field/password", cfg.Secrets[1].Notation)
	}

	// Check third secret (file)
	if !cfg.Secrets[2].IsFile {
		t.Error("Secrets[2].IsFile should be true")
	}
	if cfg.Secrets[2].FileName != "cert.pem" {
		t.Errorf("Secrets[2].FileName = %v, want cert.pem", cfg.Secrets[2].FileName)
	}

	// Check folder
	if len(cfg.Folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(cfg.Folders))
	}
	if cfg.Folders[0].FolderUID != "FOLDER123" {
		t.Errorf("Folders[0].FolderUID = %v, want FOLDER123", cfg.Folders[0].FolderUID)
	}
}

func TestExtractRecordFromNotation(t *testing.T) {
	tests := []struct {
		notation string
		want     string
	}{
		{"keeper://ABC123/field/password", "ABC123"},
		{"keeper://My Record Title/field/login", "My Record Title"},
		{"ABC123/field/password", "ABC123"},
		{"keeper://uid123/file/cert.pem", "uid123"},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			if got := extractRecordFromNotation(tt.notation); got != tt.want {
				t.Errorf("extractRecordFromNotation(%q) = %q, want %q", tt.notation, got, tt.want)
			}
		})
	}
}

// ============================================================================
// K8s Secret Injection Tests (v0.9.0)
// ============================================================================

func TestParseAnnotations_K8sSecretInjection(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":              "true",
				"keeper.security/auth-secret":         "keeper-creds",
				"keeper.security/secret":              "database-credentials",
				"keeper.security/inject-as-k8s-secret": "true",
				"keeper.security/k8s-secret-name":     "app-secrets",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if !cfg.Enabled {
		t.Error("Expected Enabled = true")
	}
	if !cfg.InjectAsK8sSecret {
		t.Error("Expected InjectAsK8sSecret = true")
	}
	if cfg.K8sSecretName != "app-secrets" {
		t.Errorf("K8sSecretName = %v, want app-secrets", cfg.K8sSecretName)
	}
	if cfg.K8sSecretMode != "overwrite" {
		t.Errorf("K8sSecretMode = %v, want overwrite (default)", cfg.K8sSecretMode)
	}
	if !cfg.K8sSecretOwnerRef {
		t.Error("Expected K8sSecretOwnerRef = true (default)")
	}
}

func TestParseAnnotations_K8sSecretMode(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		expectedMode string
	}{
		{"default", "", "overwrite"},
		{"overwrite explicit", "overwrite", "overwrite"},
		{"merge", "merge", "merge"},
		{"skip-if-exists", "skip-if-exists", "skip-if-exists"},
		{"fail", "fail", "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotations := map[string]string{
				"keeper.security/inject":              "true",
				"keeper.security/auth-secret":         "keeper-creds",
				"keeper.security/secret":              "test-secret",
				"keeper.security/inject-as-k8s-secret": "true",
				"keeper.security/k8s-secret-name":     "test",
			}
			if tt.mode != "" {
				annotations["keeper.security/k8s-secret-mode"] = tt.mode
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
			}

			cfg, err := ParseAnnotations(pod)
			if err != nil {
				t.Fatalf("ParseAnnotations() error = %v", err)
			}
			if cfg.K8sSecretMode != tt.expectedMode {
				t.Errorf("K8sSecretMode = %v, want %v", cfg.K8sSecretMode, tt.expectedMode)
			}
		})
	}
}

func TestParseAnnotations_K8sSecretOwnerRef(t *testing.T) {
	t.Run("default enabled", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"keeper.security/inject":              "true",
					"keeper.security/auth-secret":         "keeper-creds",
					"keeper.security/secret":              "test-secret",
					"keeper.security/inject-as-k8s-secret": "true",
					"keeper.security/k8s-secret-name":     "test",
				},
			},
		}

		cfg, err := ParseAnnotations(pod)
		if err != nil {
			t.Fatalf("ParseAnnotations() error = %v", err)
		}
		if !cfg.K8sSecretOwnerRef {
			t.Error("Expected K8sSecretOwnerRef = true (default)")
		}
	})

	t.Run("explicitly disabled", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"keeper.security/inject":              "true",
					"keeper.security/auth-secret":         "keeper-creds",
					"keeper.security/secret":              "test-secret",
					"keeper.security/inject-as-k8s-secret": "true",
					"keeper.security/k8s-secret-name":     "test",
					"keeper.security/k8s-secret-owner-ref": "false",
				},
			},
		}

		cfg, err := ParseAnnotations(pod)
		if err != nil {
			t.Fatalf("ParseAnnotations() error = %v", err)
		}
		if cfg.K8sSecretOwnerRef {
			t.Error("Expected K8sSecretOwnerRef = false")
		}
	})
}

func TestParseAnnotations_K8sSecretRotation(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":              "true",
				"keeper.security/auth-secret":         "keeper-creds",
				"keeper.security/secret":              "test-secret",
				"keeper.security/inject-as-k8s-secret": "true",
				"keeper.security/k8s-secret-name":     "test",
				"keeper.security/k8s-secret-rotation": "true",
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}
	if !cfg.K8sSecretRotation {
		t.Error("Expected K8sSecretRotation = true")
	}
}

func TestParseYAMLConfig_K8sSecretKeys(t *testing.T) {
	yamlConfig := `
secrets:
  - record: "database-credentials"
    injectAsK8sSecret: true
    k8sSecretName: "db-creds"
    k8sSecretKeys:
      username: "DB_USER"
      password: "DB_PASS"
      host: "DB_HOST"
`

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-creds",
				"keeper.security/config":      yamlConfig,
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(cfg.Secrets))
	}

	secret := cfg.Secrets[0]
	if secret.Name != "database-credentials" {
		t.Errorf("Name = %v, want database-credentials", secret.Name)
	}
	if !secret.InjectAsK8sSecret {
		t.Error("Expected InjectAsK8sSecret = true")
	}
	if secret.K8sSecretName != "db-creds" {
		t.Errorf("K8sSecretName = %v, want db-creds", secret.K8sSecretName)
	}
	if secret.K8sSecretKeys == nil {
		t.Fatal("Expected K8sSecretKeys to be set")
	}
	if secret.K8sSecretKeys["username"] != "DB_USER" {
		t.Errorf("K8sSecretKeys[username] = %v, want DB_USER", secret.K8sSecretKeys["username"])
	}
	if secret.K8sSecretKeys["password"] != "DB_PASS" {
		t.Errorf("K8sSecretKeys[password] = %v, want DB_PASS", secret.K8sSecretKeys["password"])
	}
	if secret.K8sSecretKeys["host"] != "DB_HOST" {
		t.Errorf("K8sSecretKeys[host] = %v, want DB_HOST", secret.K8sSecretKeys["host"])
	}
}

func TestParseYAMLConfig_K8sSecretType(t *testing.T) {
	yamlConfig := `
secrets:
  - record: "tls-certificate"
    injectAsK8sSecret: true
    k8sSecretName: "tls-cert"
    k8sSecretType: "kubernetes.io/tls"
    k8sSecretKeys:
      cert: "tls.crt"
      key: "tls.key"
`

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-creds",
				"keeper.security/config":      yamlConfig,
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(cfg.Secrets))
	}

	secret := cfg.Secrets[0]
	if secret.K8sSecretType != "kubernetes.io/tls" {
		t.Errorf("K8sSecretType = %v, want kubernetes.io/tls", secret.K8sSecretType)
	}
}

func TestParseYAMLConfig_FolderWithK8sSecrets(t *testing.T) {
	yamlConfig := `
folders:
  - folderPath: "Production/APIs"
    injectAsK8sSecret: true
    k8sSecretNamePrefix: "api-"
`

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-creds",
				"keeper.security/config":      yamlConfig,
			},
		},
	}

	cfg, err := ParseAnnotations(pod)
	if err != nil {
		t.Fatalf("ParseAnnotations() error = %v", err)
	}

	if len(cfg.Folders) != 1 {
		t.Fatalf("Expected 1 folder, got %d", len(cfg.Folders))
	}

	folder := cfg.Folders[0]
	if folder.FolderPath != "Production/APIs" {
		t.Errorf("FolderPath = %v, want Production/APIs", folder.FolderPath)
	}
	if !folder.InjectAsK8sSecret {
		t.Error("Expected InjectAsK8sSecret = true")
	}
	if folder.K8sSecretNamePrefix != "api-" {
		t.Errorf("K8sSecretNamePrefix = %v, want api-", folder.K8sSecretNamePrefix)
	}
}
