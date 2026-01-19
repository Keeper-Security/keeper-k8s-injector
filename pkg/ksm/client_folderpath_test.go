package ksm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNotationPath_FolderPathWithField(t *testing.T) {
	tests := []struct {
		name             string
		notation         string
		expectedFolder   string
		expectedRecord   string
		expectedSelector string
		expectedParam    string
		expectedPrefix   bool
		shouldBeNil      bool
	}{
		{
			name:             "folder path with field",
			notation:         "keeper://Production/Databases/mysql/field/password",
			expectedFolder:   "Production/Databases",
			expectedRecord:   "mysql",
			expectedSelector: "field",
			expectedParam:    "password",
			expectedPrefix:   true,
		},
		{
			name:             "folder path without keeper prefix",
			notation:         "Production/Databases/mysql/field/password",
			expectedFolder:   "Production/Databases",
			expectedRecord:   "mysql",
			expectedSelector: "field",
			expectedParam:    "password",
			expectedPrefix:   false,
		},
		{
			name:             "nested folder path",
			notation:         "Production/Region/US-East/Databases/postgres/field/username",
			expectedFolder:   "Production/Region/US-East/Databases",
			expectedRecord:   "postgres",
			expectedSelector: "field",
			expectedParam:    "username",
			expectedPrefix:   false,
		},
		{
			name:             "single folder with field",
			notation:         "keeper://Production/mysql/field/password",
			expectedFolder:   "Production",
			expectedRecord:   "mysql",
			expectedSelector: "field",
			expectedParam:    "password",
			expectedPrefix:   true,
		},
		{
			name:             "folder path with custom_field",
			notation:         "Dev/Apps/api-key/custom_field/token",
			expectedFolder:   "Dev/Apps",
			expectedRecord:   "api-key",
			expectedSelector: "custom_field",
			expectedParam:    "token",
			expectedPrefix:   false,
		},
		{
			name:             "folder path with file",
			notation:         "Production/Certs/ssl-cert/file/certificate.pem",
			expectedFolder:   "Production/Certs",
			expectedRecord:   "ssl-cert",
			expectedSelector: "file",
			expectedParam:    "certificate.pem",
			expectedPrefix:   false,
		},
		{
			name:             "folder path with type selector",
			notation:         "keeper://QA/test-secret/type",
			expectedFolder:   "QA",
			expectedRecord:   "test-secret",
			expectedSelector: "type",
			expectedParam:    "",
			expectedPrefix:   true,
		},
		{
			name:             "folder path with title selector",
			notation:         "Production/DB/mysql/title",
			expectedFolder:   "Production/DB",
			expectedRecord:   "mysql",
			expectedSelector: "title",
			expectedParam:    "",
			expectedPrefix:   false,
		},
		{
			name:             "folder path without selector (whole record)",
			notation:         "Production/Databases/mysql",
			expectedFolder:   "Production/Databases",
			expectedRecord:   "mysql",
			expectedSelector: "",
			expectedParam:    "",
			expectedPrefix:   false,
		},
		{
			name:           "UID-only notation (no folder path)",
			notation:       "keeper://ABC123XYZ456789012/field/password",
			expectedFolder: "",
			expectedRecord: "ABC123XYZ456789012",
			shouldBeNil:    true, // Not enough parts before selector for folder path
		},
		{
			name:           "title-only notation (no folder path)",
			notation:       "my-secret/field/password",
			expectedFolder: "",
			shouldBeNil:    true, // Not enough parts before selector for folder path
		},
		{
			name:        "single part notation",
			notation:    "ABC123XYZ456789012",
			shouldBeNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := parseNotationPath(tt.notation)

			if tt.shouldBeNil {
				assert.Nil(t, np, "expected nil result")
				return
			}

			require.NotNil(t, np, "expected non-nil result")
			assert.Equal(t, tt.expectedFolder, np.folderPath, "folder path mismatch")
			assert.Equal(t, tt.expectedRecord, np.recordName, "record name mismatch")
			assert.Equal(t, tt.expectedSelector, np.selector, "selector mismatch")
			assert.Equal(t, tt.expectedParam, np.parameter, "parameter mismatch")
			assert.Equal(t, tt.expectedPrefix, np.hasKeeperURI, "keeper:// prefix mismatch")
		})
	}
}

func TestParseNotationPath_EdgeCases(t *testing.T) {
	t.Run("folder and record with special characters", func(t *testing.T) {
		np := parseNotationPath("Prod-2024/DB_MySQL/my-secret-123/field/password")
		require.NotNil(t, np)
		assert.Equal(t, "Prod-2024/DB_MySQL", np.folderPath)
		assert.Equal(t, "my-secret-123", np.recordName)
		assert.Equal(t, "field", np.selector)
		assert.Equal(t, "password", np.parameter)
	})

	t.Run("folder path with spaces and parentheses", func(t *testing.T) {
		np := parseNotationPath("Production/API (v2)/stripe-key/field/token")
		require.NotNil(t, np)
		assert.Equal(t, "Production/API (v2)", np.folderPath)
		assert.Equal(t, "stripe-key", np.recordName)
	})

	t.Run("UID in folder path", func(t *testing.T) {
		// Even if folder/record looks like UID, it should be parsed as folder path
		np := parseNotationPath("Production/Databases/ABC123XYZ456789012/field/password")
		require.NotNil(t, np)
		assert.Equal(t, "Production/Databases", np.folderPath)
		assert.Equal(t, "ABC123XYZ456789012", np.recordName)
		assert.Equal(t, "field", np.selector)
	})
}

func TestParseNotationPath_BackwardCompatibility(t *testing.T) {
	t.Run("UID with field selector (no folder path)", func(t *testing.T) {
		// This should return nil because there's not enough parts for folder path
		np := parseNotationPath("ABC123XYZ456789012/field/password")
		assert.Nil(t, np, "UID-based notation without folder path should return nil")
	})

	t.Run("title with field selector (no folder path)", func(t *testing.T) {
		// This should return nil because there's not enough parts for folder path
		np := parseNotationPath("my-record/field/password")
		assert.Nil(t, np, "title-based notation without folder path should return nil")
	})

	t.Run("just UID (no selector)", func(t *testing.T) {
		np := parseNotationPath("ABC123XYZ456789012")
		assert.Nil(t, np, "single UID should return nil")
	})
}

func TestParseNotationPath_AllSelectors(t *testing.T) {
	selectors := []string{"field", "custom_field", "file", "type", "title", "notes"}

	for _, selector := range selectors {
		t.Run(selector, func(t *testing.T) {
			notation := "Folder1/Folder2/record/" + selector
			if selector == "field" || selector == "custom_field" || selector == "file" {
				notation += "/param"
			}

			np := parseNotationPath(notation)
			require.NotNil(t, np)
			assert.Equal(t, "Folder1/Folder2", np.folderPath)
			assert.Equal(t, "record", np.recordName)
			assert.Equal(t, selector, np.selector)
		})
	}
}

// Note: Integration tests for GetSecretByPath and GetNotation with folder paths
// would require a real KSM connection and are better suited for E2E tests.
// The parseNotationPath function is fully unit-tested above.

func TestParseNotationPath_LeadingTrailingSlashes(t *testing.T) {
	tests := []struct {
		name     string
		notation string
	}{
		{"leading slash", "/Production/Databases/mysql/field/password"},
		{"trailing slash", "Production/Databases/mysql/field/password/"},
		{"both slashes", "/Production/Databases/mysql/field/password/"},
		{"multiple leading", "//Production/Databases/mysql/field/password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := parseNotationPath(tt.notation)
			require.NotNil(t, np)
			assert.Equal(t, "Production/Databases", np.folderPath)
			assert.Equal(t, "mysql", np.recordName)
			assert.Equal(t, "field", np.selector)
			assert.Equal(t, "password", np.parameter)
		})
	}
}
