# Annotations Reference

All annotations use the `keeper.security/` prefix.

## Required Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/inject` | Enable injection | `"true"` |
| `keeper.security/auth-secret` | K8s secret with KSM config | `"keeper-auth"` |

## Secret Selection

### Level 1: Single Secret (Simplest)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret: "database-credentials"
```

Result: `/keeper/secrets/database-credentials.json`

### Level 2: Multiple Secrets

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secrets: "database-creds, api-keys, tls-cert"
```

Result:
- `/keeper/secrets/database-creds.json`
- `/keeper/secrets/api-keys.json`
- `/keeper/secrets/tls-cert.json`

### Level 3: Custom Paths

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret-database: "/app/config/db.json"
  keeper.security/secret-api: "/etc/myapp/api-keys.json"
```

### Level 4: Field Extraction

Extract specific fields from a record:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret-password: "database-creds[password]:/app/secrets/db-pass"
```

Result: `/app/secrets/db-pass` contains only the password value (raw, not JSON)

### Level 5: Full Configuration

For complex scenarios, use YAML configuration:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/config: |
    secrets:
      - record: database-credentials
        path: /app/config/db.json
        fields: [login, password]
        format: env
      - record: api-keys
        path: /app/config/api.json
        format: json
```

## Behavior Annotations

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/refresh-interval` | `"5m"` | How often to refresh secrets |
| `keeper.security/init-only` | `"false"` | Only use init container (no sidecar) |
| `keeper.security/fail-on-error` | `"true"` | Fail pod startup if secrets can't be fetched |
| `keeper.security/signal` | `""` | Signal to send on refresh (e.g., `"SIGHUP"`) |
| `keeper.security/strict-lookup` | `"false"` | Fail if multiple records match title |

## Authentication Annotations

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/auth-secret` | Required | K8s secret name with KSM config |
| `keeper.security/auth-method` | `"secret"` | Auth method: `"secret"` or `"oidc"` |

## Output Formats

When using Level 5 configuration, you can specify output format:

| Format | Description | Example Output |
|--------|-------------|----------------|
| `json` | JSON object (default) | `{"login": "user", "password": "pass"}` |
| `env` | Environment file | `LOGIN=user\nPASSWORD=pass` |
| `raw` | Raw value (single field only) | `mypassword123` |

## Examples

### Minimal Configuration
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret: "my-secret"
```

### Production Configuration
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secrets: "database, redis, api-keys"
  keeper.security/refresh-interval: "10m"
  keeper.security/fail-on-error: "true"
  keeper.security/signal: "SIGHUP"
```

### Init-Only Mode (No Rotation)
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret: "static-config"
  keeper.security/init-only: "true"
```

## Secrets by UID

If you prefer using record UIDs instead of titles:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret: "OqPt3Vd37My7G8rTb-8Q"  # 22-char UID
```

The injector auto-detects UIDs vs titles based on format.

## Keeper Notation

Use Keeper notation for precise field extraction with custom output paths:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  # Format: keeper://UID/field/FIELDNAME:OUTPUT_PATH
  keeper.security/secret-password: "keeper://QabbPIdM8Unw4hwVM-F8VQ/field/password:/app/secrets/db-pass"
  keeper.security/secret-login: "keeper://QabbPIdM8Unw4hwVM-F8VQ/field/login:/app/secrets/db-user"
```

Result:
- `/app/secrets/db-pass` contains the password value (raw text)
- `/app/secrets/db-user` contains the login value (raw text)

### Notation Syntax

The Keeper notation format is: `keeper://RECORD_UID/TYPE/SELECTOR`

| Type | Selector | Description |
|------|----------|-------------|
| `field` | field name | Extract a specific field value |
| `file` | filename | Download a file attachment |
| `custom_field` | field label | Extract custom field by label |

## File Attachments

Download file attachments from Keeper records:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  # Format: RECORD_TITLE:FILENAME:OUTPUT_PATH
  keeper.security/file-cert: "Database Credentials:cert.pem:/app/certs/server.pem"
  keeper.security/file-key: "Database Credentials:key.pem:/app/certs/server.key"
```

Result:
- `/app/certs/server.pem` contains the `cert.pem` file from "Database Credentials" record
- `/app/certs/server.key` contains the `key.pem` file from "Database Credentials" record

## Folder Support

Fetch all secrets from a Keeper folder:

### By Folder UID (Recommended)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/folder-uid: "FOLDER_UID_HERE"
  keeper.security/folder-path: "/app/folder-secrets"  # Output directory
```

Result: All secrets in the folder are written as JSON files to `/app/folder-secrets/`

### By Folder Path (Coming Soon)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/folder: "Production/Databases"
  keeper.security/folder-path: "/app/db-secrets"
```

> **Note:** Folder path lookup is not yet implemented. Use `folder-uid` for now.

## Complete Example

This example demonstrates multiple annotation types:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-auth"

    # Standard secret (full record as JSON)
    keeper.security/secret: "Application Config"

    # Keeper notation (specific field)
    keeper.security/secret-password: "keeper://ABC123def456/field/password:/app/secrets/db-pass"

    # File attachment
    keeper.security/file-cert: "TLS Certificates:server.pem:/app/certs/server.pem"

    # Behavior
    keeper.security/refresh-interval: "10m"
    keeper.security/fail-on-error: "true"
    keeper.security/init-only: "false"
spec:
  containers:
    - name: app
      image: myapp:latest
```
