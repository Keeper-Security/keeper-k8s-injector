package sidecar

import (
	"strings"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		template string
		want     string
		wantErr  bool
	}{
		{
			name: "simple field substitution",
			data: map[string]interface{}{
				"login":    "testuser",
				"password": "testpass",
			},
			template: `DB_USER={{ .login }}
DB_PASS={{ .password }}`,
			want:    "DB_USER=testuser\nDB_PASS=testpass",
			wantErr: false,
		},
		{
			name: "connection string building",
			data: map[string]interface{}{
				"login":    "admin",
				"password": "secret123",
				"hostname": "postgres.example.com",
			},
			template: `postgresql://{{ .login }}:{{ .password }}@{{ .hostname }}:5432/mydb`,
			want:    "postgresql://admin:secret123@postgres.example.com:5432/mydb",
			wantErr: false,
		},
		{
			name: "template with base64enc function",
			data: map[string]interface{}{
				"password": "secret",
			},
			template: `{{ .password | base64enc }}`,
			want:    "c2VjcmV0",
			wantErr: false,
		},
		{
			name: "template with upper function",
			data: map[string]interface{}{
				"username": "admin",
			},
			template: `{{ .username | upper }}`,
			want:    "ADMIN",
			wantErr: false,
		},
		{
			name: "template with lower function",
			data: map[string]interface{}{
				"database": "MyDatabase",
			},
			template: `{{ .database | lower }}`,
			want:    "mydatabase",
			wantErr: false,
		},
		{
			name: "template with trim function",
			data: map[string]interface{}{
				"value": "  trimme  ",
			},
			template: `{{ .value | trim }}`,
			want:    "trimme",
			wantErr: false,
		},
		{
			name: "properties file template",
			data: map[string]interface{}{
				"login":    "dbuser",
				"password": "dbpass",
				"hostname": "db.local",
			},
			template: `db.username={{ .login }}
db.password={{ .password }}
db.host={{ .hostname }}`,
			want:    "db.username=dbuser\ndb.password=dbpass\ndb.host=db.local",
			wantErr: false,
		},
		{
			name:     "empty template",
			data:     map[string]interface{}{"key": "value"},
			template: "",
			wantErr:  true,
		},
		{
			name: "invalid template syntax",
			data: map[string]interface{}{"key": "value"},
			template: "{{ .key",
			wantErr:  true,
		},
		{
			name: "missing field in data",
			data: map[string]interface{}{"key": "value"},
			template: "{{ .missingfield }}",
			want: "<no value>",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTemplate(tt.data, tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("renderTemplate() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestTemplateFunc_Base64Enc(t *testing.T) {
	data := map[string]interface{}{
		"password": "secret",
	}
	tmpl := `{{ .password | base64enc }}`

	result, err := renderTemplate(data, tmpl)
	if err != nil {
		t.Fatalf("renderTemplate() error = %v", err)
	}

	want := "c2VjcmV0"
	if string(result) != want {
		t.Errorf("base64enc result = %q, want %q", string(result), want)
	}
}

func TestTemplateFunc_Base64Dec(t *testing.T) {
	data := map[string]interface{}{
		"encoded": "c2VjcmV0",
	}
	tmpl := `{{ .encoded | base64dec }}`

	result, err := renderTemplate(data, tmpl)
	if err != nil {
		t.Fatalf("renderTemplate() error = %v", err)
	}

	want := "secret"
	if string(result) != want {
		t.Errorf("base64dec result = %q, want %q", string(result), want)
	}
}

func TestTemplateFunc_StringTransform(t *testing.T) {
	data := map[string]interface{}{
		"value": "  Test Value  ",
	}

	tests := []struct {
		name     string
		template string
		want     string
	}{
		{"upper", `{{ .value | upper }}`, "  TEST VALUE  "},
		{"lower", `{{ .value | lower }}`, "  test value  "},
		{"trim", `{{ .value | trim }}`, "Test Value"},
		{"chained", `{{ .value | trim | upper }}`, "TEST VALUE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderTemplate(data, tt.template)
			if err != nil {
				t.Fatalf("renderTemplate() error = %v", err)
			}
			if string(result) != tt.want {
				t.Errorf("%s result = %q, want %q", tt.name, string(result), tt.want)
			}
		})
	}
}

func TestTemplateMultiline(t *testing.T) {
	data := map[string]interface{}{
		"username": "admin",
		"password": "p@ssw0rd",
		"database": "production",
	}

	tmpl := `# Database Configuration
	username={{ .username }}
	password={{ .password | base64enc }}
	database={{ .database | upper }}`

	result, err := renderTemplate(data, tmpl)
	if err != nil {
		t.Fatalf("renderTemplate() error = %v", err)
	}

	// Verify key elements are present
	output := string(result)
	if !strings.Contains(output, "username=admin") {
		t.Errorf("missing username in output: %s", output)
	}
	if !strings.Contains(output, "database=PRODUCTION") {
		t.Errorf("missing uppercased database in output: %s", output)
	}
	if !strings.Contains(output, "password=cEBzc3cwcmQ=") {
		t.Errorf("missing base64 encoded password in output: %s", output)
	}
}
