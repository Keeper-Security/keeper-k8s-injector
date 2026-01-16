# Template Examples

Demonstrates Go template rendering for flexible secret formatting. Build connection strings, config files, and environment scripts without JSON parsing.

**Time to complete: ~10 minutes**

## What This Demonstrates

- Go template rendering with Keeper secrets
- Connection string building
- Multiple output formats (properties, INI, shell scripts)
- Template functions (base64enc, upper, lower, sha256sum, etc.)
- Conditional logic
- Default values for missing fields

## Prerequisites

1. Keeper K8s Injector installed in your cluster
2. A Keeper Secrets Manager application with a config file

## Quick Start

### 1. Create Your KSM Auth Secret

```bash
kubectl create secret generic keeper-credentials \
  --from-file=config=path/to/your/ksm-config.json
```

### 2. Create Secrets in Keeper

#### Secret 1: postgres-credentials

- Title: `postgres-credentials`
- Login: `dbuser`
- Password: `mysecretpassword`
- Custom field `hostname`: `postgres.example.com`
- Custom field `database`: `production_db`

#### Secret 2: app-config

- Title: `app-config`
- Custom field `environment`: `production` (or `development`)
- Custom field `appName`: `myapp`
- Custom field `apiKey`: `sk_live_abc123xyz`

### 3. Deploy the Example

```bash
kubectl apply -f deployment.yaml
```

### 4. Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=template-demo --timeout=120s
```

### 5. View the Demo

```bash
kubectl port-forward svc/template-demo 8080:80
```

Open http://localhost:8080 in your browser.

You'll see all six template examples rendered with your Keeper data.

## Template Examples Explained

### Example 1: Connection String

**Template:**
```
postgresql://{{ .login }}:{{ .password }}@{{ .hostname | default "postgres" }}:5432/{{ .database | default "mydb" }}
```

**Output:**
```
postgresql://dbuser:mysecretpassword@postgres.example.com:5432/production_db
```

**Use case:** Build database URLs without shell string manipulation.

### Example 2: Properties File

**Template:**
```
app.database.url=jdbc:postgresql://{{ .hostname | default "localhost" }}:5432/mydb
app.database.username={{ .login }}
app.database.password={{ .password | base64enc }}
```

**Output:**
```
app.database.url=jdbc:postgresql://postgres.example.com:5432/mydb
app.database.username=dbuser
app.database.password=bXlzZWNyZXRwYXNzd29yZA==
```

**Use case:** Java/Spring Boot configuration.

### Example 3: Shell Script

**Template:**
```bash
export DB_HOST="{{ .hostname | default "localhost" }}"
export DB_USER="{{ .login }}"
export DB_PASS="{{ .password }}"
```

**Output:**
```bash
export DB_HOST="postgres.example.com"
export DB_USER="dbuser"
export DB_PASS="mysecretpassword"
```

**Use case:** Source in shell scripts: `source /app/examples/03-database.sh`

### Example 4: INI Format

**Template:**
```
[database]
host={{ .hostname | default "localhost" }}
port=5432
username={{ .login }}
password={{ .password }}
```

**Use case:** Python ConfigParser, legacy apps.

### Example 5: Template Functions

Demonstrates transformation functions:

```
Uppercase: {{ .login | upper }}
SHA256: {{ .password | sha256sum }}
Base64: {{ .password | base64enc }}
```

### Example 6: Conditional Logic

Different configs for production vs development:

```
{{- if eq .environment "production" -}}
log_level=error
api_url=https://api.prod.example.com
{{- else -}}
log_level=debug
api_url=https://api.dev.example.com
{{- end -}}
```

## Available Template Functions

### Basic Functions
- `upper`, `lower`, `title`, `trim`
- `base64enc`, `base64dec`
- `sha256sum`, `sha512sum`
- `replace`, `split`, `join`

### Sprig Functions (100+)
- Date/time: `now`, `date`, `ago`
- Crypto: `bcrypt`, `htpasswd`, `uuidv4`
- Encoding: `urlquery`, `htmlEscape`, `toJson`
- Logic: `default`, `empty`, `coalesce`, `ternary`
- Lists: `first`, `last`, `reverse`, `uniq`

See [docs/templates.md](../../docs/templates.md) for complete reference.

## Benefits Over JSON Parsing

**Before (with jq):**
```bash
DB_USER=$(cat /keeper/secrets/db.json | jq -r '.login')
DB_PASS=$(cat /keeper/secrets/db.json | jq -r '.password')
DB_URL="postgresql://${DB_USER}:${DB_PASS}@postgres:5432/mydb"
```

**After (with templates):**
```yaml
template: |
  export DB_URL="postgresql://{{ .login }}:{{ .password }}@postgres:5432/mydb"
```

No jq dependency, no shell parsing, works under corporate proxies.

## Cleanup

```bash
kubectl delete -f deployment.yaml
```

## Troubleshooting

### Template not rendering

Check sidecar logs:
```bash
kubectl logs deployment/template-demo -c keeper-sidecar
```

### Missing field error

Use `default` for optional fields:
```yaml
template: |
  host={{ .hostname | default "localhost" }}
```

### Special characters issues

Use Sprig encoding functions:
```yaml
template: |
  {{ .value | urlquery }}    # URL encoding
  {{ .value | htmlEscape }}  # HTML escaping
```

## Next Steps

- [Documentation](../../docs/templates.md) - Complete template guide
- [Annotations Reference](../../docs/annotations.md) - All configuration options
