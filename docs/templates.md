# Template Guide

Go templates provide flexible secret formatting beyond the built-in json, env, and raw formats.

## Basic Usage

```yaml
annotations:
  keeper.security/config: |
    secrets:
      - record: postgres-credentials
        path: /app/config/database.sh
        template: |
          export DB_USER="{{ .login }}"
          export DB_PASS="{{ .password }}"
```

Output (`/app/config/database.sh`):
```sh
export DB_USER="admin"
export DB_PASS="secret123"
```

## Template Syntax

Templates use Go's `text/template` package with Sprig functions.

### Field Access

```yaml
template: |
  {{ .login }}        # Access field
  {{ .password }}     # Access another field
```

### Piping

Chain functions using pipes:

```yaml
template: |
  {{ .password | base64enc }}
  {{ .username | upper }}
  {{ .value | trim | lower }}
```

### Conditionals

```yaml
template: |
  {{- if .hostname -}}
  host={{ .hostname }}
  {{- else -}}
  host=localhost
  {{- end -}}
```

## Common Patterns

### Connection Strings

**PostgreSQL:**
```yaml
template: |
  postgresql://{{ .login }}:{{ .password }}@{{ .hostname }}:5432/{{ .database }}
```

**MySQL:**
```yaml
template: |
  mysql://{{ .username }}:{{ .password }}@tcp({{ .host }}:3306)/{{ .dbname }}
```

**Redis:**
```yaml
template: |
  redis://:{{ .password }}@{{ .hostname }}:6379/0
```

### Configuration Files

**Properties file:**
```yaml
template: |
  db.url=jdbc:postgresql://{{ .hostname }}:5432/mydb
  db.username={{ .login }}
  db.password={{ .password }}
  db.pool.size=10
```

**INI file:**
```yaml
template: |
  [database]
  host={{ .hostname }}
  port=5432
  user={{ .login }}
  password={{ .password }}

  [redis]
  url=redis://{{ .redisHost }}:6379
```

**YAML file:**
```yaml
template: |
  database:
    host: {{ .hostname }}
    port: 5432
    username: {{ .login }}
    password: {{ .password }}
```

### Environment Scripts

**Bash:**
```yaml
template: |
  #!/bin/bash
  export DB_HOST="{{ .hostname }}"
  export DB_USER="{{ .login }}"
  export DB_PASS="{{ .password | base64enc }}"
  export DB_URL="postgresql://${DB_USER}:$(echo ${DB_PASS} | base64 -d)@${DB_HOST}:5432/mydb"
```

## Template Functions

### Encoding

| Function | Description | Example |
|----------|-------------|---------|
| `base64enc` | Base64 encode | `{{ .password | base64enc }}` |
| `base64dec` | Base64 decode | `{{ .encoded | base64dec }}` |
| `b64enc` | Alias for base64enc | `{{ .value | b64enc }}` |
| `b64dec` | Alias for base64dec | `{{ .value | b64dec }}` |

### Hashing

| Function | Description | Example |
|----------|-------------|---------|
| `sha256sum` | SHA-256 hash | `{{ .password | sha256sum }}` |
| `sha512sum` | SHA-512 hash | `{{ .password | sha512sum }}` |
| `sha1sum` | SHA-1 hash | `{{ .value | sha1sum }}` |
| `md5sum` | MD5 hash | `{{ .value | md5sum }}` |

### String Manipulation

| Function | Description | Example |
|----------|-------------|---------|
| `upper` | Uppercase | `{{ .value | upper }}` |
| `lower` | Lowercase | `{{ .value | lower }}` |
| `title` | Title case | `{{ .value | title }}` |
| `trim` | Trim whitespace | `{{ .value | trim }}` |
| `trimPrefix` | Remove prefix | `{{ .value | trimPrefix "http://" }}` |
| `trimSuffix` | Remove suffix | `{{ .value | trimSuffix ".com" }}` |
| `replace` | Replace string | `{{ .value | replace "old" "new" }}` |
| `split` | Split string | `{{ .value | split "," }}` |
| `join` | Join array | `{{ .items | join "," }}` |
| `contains` | Check substring | `{{ .value | contains "test" }}` |
| `hasPrefix` | Check prefix | `{{ .value | hasPrefix "http" }}` |
| `hasSuffix` | Check suffix | `{{ .value | hasSuffix ".com" }}` |

### Logic

| Function | Description | Example |
|----------|-------------|---------|
| `default` | Default value | `{{ .value | default "fallback" }}` |
| `empty` | Check if empty | `{{ if empty .value }}empty{{ end }}` |
| `coalesce` | First non-empty | `{{ coalesce .val1 .val2 "default" }}` |
| `ternary` | Conditional | `{{ .isProd | ternary "prod" "dev" }}` |

### Comparison

| Function | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `{{ if eq .env "prod" }}...{{ end }}` |
| `ne` | Not equal | `{{ if ne .type "test" }}...{{ end }}` |
| `lt` | Less than | `{{ if lt .port 1024 }}...{{ end }}` |
| `le` | Less or equal | `{{ if le .count 10 }}...{{ end }}` |
| `gt` | Greater than | `{{ if gt .value 0 }}...{{ end }}` |
| `ge` | Greater or equal | `{{ if ge .version 2 }}...{{ end }}` |

### Date/Time (Sprig)

| Function | Description | Example |
|----------|-------------|---------|
| `now` | Current time | `{{ now | date "2006-01-02" }}` |
| `date` | Format date | `{{ now | date "15:04:05" }}` |
| `dateModify` | Modify date | `{{ now | dateModify "+24h" }}` |
| `ago` | Time ago | `{{ .timestamp | ago }}` |

### Lists (Sprig)

| Function | Description | Example |
|----------|-------------|---------|
| `list` | Create list | `{{ list "a" "b" "c" }}` |
| `first` | First element | `{{ .items | first }}` |
| `last` | Last element | `{{ .items | last }}` |
| `reverse` | Reverse list | `{{ .items | reverse }}` |
| `uniq` | Unique elements | `{{ .items | uniq }}` |
| `compact` | Remove empties | `{{ .items | compact }}` |

### JSON/YAML (Sprig)

| Function | Description | Example |
|----------|-------------|---------|
| `toJson` | Convert to JSON | `{{ .data | toJson }}` |
| `toPrettyJson` | Pretty JSON | `{{ .data | toPrettyJson }}` |
| `toYaml` | Convert to YAML | `{{ .data | toYaml }}` |
| `fromJson` | Parse JSON | `{{ .jsonString | fromJson }}` |

### Crypto (Sprig)

| Function | Description | Example |
|----------|-------------|---------|
| `bcrypt` | Bcrypt hash | `{{ .password | bcrypt }}` |
| `htpasswd` | Apache htpasswd | `{{ htpasswd "user" .password }}` |
| `randAlpha` | Random string | `{{ randAlpha 16 }}` |
| `uuidv4` | Generate UUID | `{{ uuidv4 }}` |

## Complete Function Reference

Sprig provides 100+ functions. See [Sprig Documentation](http://masterminds.github.io/sprig/) for the complete list.

## Examples

### Database Connection String

```yaml
keeper.security/config: |
  secrets:
    - record: postgres-credentials
      path: /app/database-url.txt
      template: |
        postgresql://{{ .login }}:{{ .password | urlquery }}@{{ .hostname }}:{{ .port | default "5432" }}/{{ .database }}
```

### Properties File with Defaults

```yaml
keeper.security/config: |
  secrets:
    - record: app-config
      path: /app/application.properties
      template: |
        app.name={{ .appName | default "myapp" }}
        app.env={{ .environment | default "dev" }}
        db.host={{ .dbHost }}
        db.user={{ .dbUser }}
        db.password={{ .dbPassword | base64enc }}
        debug={{ .debug | default "false" }}
```

### Environment-Specific Configuration

```yaml
keeper.security/config: |
  secrets:
    - record: app-config
      path: /app/config.sh
      template: |
        {{- if eq .environment "production" -}}
        export API_URL="https://api.prod.example.com"
        export LOG_LEVEL="error"
        {{- else -}}
        export API_URL="https://api.dev.example.com"
        export LOG_LEVEL="debug"
        {{- end -}}
        export API_KEY="{{ .apiKey }}"
```

### TLS Configuration

```yaml
keeper.security/config: |
  secrets:
    - record: tls-config
      path: /app/nginx.conf
      template: |
        server {
          listen 443 ssl;
          server_name {{ .domain }};
          ssl_certificate /certs/{{ .domain }}.crt;
          ssl_certificate_key /certs/{{ .domain }}.key;
          ssl_protocols TLSv1.2 TLSv1.3;
        }
```

### Multi-Line Scripts

```yaml
keeper.security/config: |
  secrets:
    - record: deployment-credentials
      path: /app/deploy.sh
      template: |
        #!/bin/bash
        set -e

        echo "Deploying to {{ .environment | upper }}"

        # Database migration
        export DB_URL="postgresql://{{ .dbUser }}:{{ .dbPass }}@{{ .dbHost }}/{{ .dbName }}"
        ./migrate up

        # Deploy application
        kubectl set image deployment/myapp \
          app={{ .imageRegistry }}/myapp:{{ .imageTag }}
```

## Error Handling

### Missing Fields

Go templates output `<no value>` for missing fields. Use `default` for fallbacks:

```yaml
template: |
  host={{ .hostname | default "localhost" }}
```

### Invalid Template Syntax

Template parse errors prevent secret injection:

```
level=error msg="template parse failed"
  secret=my-secret
  error="template: secret:1: unexpected '}' in operand"
```

Fix: Validate template syntax before deploying.

### Execution Errors

Runtime errors during template execution:

```
level=error msg="template execution failed"
  secret=my-secret
  error="error calling base64dec: illegal base64 data"
```

Fix: Ensure data types match function expectations.

## Best Practices

### 1. Use Default Values

Avoid failures from missing fields:

```yaml
template: |
  port={{ .port | default "5432" }}
  host={{ .hostname | default "localhost" }}
```

### 2. Escape Special Characters

For shell scripts, escape quotes:

```yaml
template: |
  export PASSWORD='{{ .password | replace "'" "'\\''" }}'
```

### 3. Format for Target Application

Match the expected format:

```yaml
# For Java
template: |
  spring.datasource.url=jdbc:{{ .dbUrl }}

# For Python
template: |
  DATABASE_URL = "{{ .dbUrl }}"

# For environment variables
template: |
  export DATABASE_URL="{{ .dbUrl }}"
```

### 4. Test Templates

Test template rendering before production:

```bash
# Create test secret with template
kubectl apply -f test-template.yaml

# Verify output
kubectl exec deploy/myapp -- cat /app/config.txt
```

## Troubleshooting

### Template not rendering

1. Check sidecar logs:
   ```bash
   kubectl logs deploy/myapp -c keeper-sidecar
   ```

2. Verify template syntax is valid

3. Check all referenced fields exist in the Keeper record

### Output is empty

1. Verify the record has the expected fields
2. Check for template syntax errors (silent failures)
3. Use `default` for optional fields

### Special characters broken

Use Sprig encoding functions:

```yaml
template: |
  # URL encoding
  url={{ .value | urlquery }}

  # HTML escaping
  html={{ .value | htmlEscape }}

  # JSON escaping
  json={{ .value | toJson }}
```

## Advanced Usage

### Conditional Rendering

```yaml
template: |
  {{- if .tlsEnabled -}}
  ssl_mode=require
  ssl_cert=/certs/{{ .tlsCertFile }}
  {{- else -}}
  ssl_mode=disable
  {{- end -}}
```

### Loops (when fields are arrays)

```yaml
template: |
  {{- range .allowedHosts -}}
  allowed_host={{ . }}
  {{- end -}}
```

### Comments

```yaml
template: |
  # Generated from Keeper at {{ now | date "2006-01-02 15:04:05" }}
  username={{ .login }}
  password={{ .password }}
```

## See Also

- [Configuration Guide](configuration.md) - All annotation options
- [Sprig Documentation](http://masterminds.github.io/sprig/) - Complete function reference
- [Go Templates](https://pkg.go.dev/text/template) - Go template syntax guide

---

**[← Back to Documentation Index](INDEX.md)**
