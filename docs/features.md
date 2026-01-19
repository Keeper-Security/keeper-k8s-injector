# Complete Feature Reference

Comprehensive guide to all Keeper K8s Injector features.

## Secret Injection Methods

### 1. Single Secret (Simple)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/secret: "database-credentials"
```

Output: `/keeper/secrets/database-credentials.json`

### 2. Multiple Secrets

```yaml
annotations:
  keeper.security/secrets: "db-creds, api-keys, tls-cert"
```

Output: Three JSON files in `/keeper/secrets/`

### 3. Custom Paths

```yaml
annotations:
  keeper.security/secret-db: "/app/config/database.json"
  keeper.security/secret-api: "/etc/app/api-keys.json"
```

### 4. Keeper Notation (Field Extraction)

```yaml
annotations:
  keeper.security/secret-password: "keeper://ABC123/field/password:/app/secrets/db-pass"
```

Extracts only the password field, writes as raw text.

### 5. File Attachments

```yaml
annotations:
  keeper.security/file-cert: "TLS Certificate:server.crt:/app/certs/server.crt"
  keeper.security/file-key: "TLS Certificate:server.key:/app/certs/server.key"
```

Downloads file attachments from Keeper records.

### 6. Folder Support

Fetch all secrets from a Keeper folder using either UID or path:

**By Folder UID:**
```yaml
annotations:
  keeper.security/folder-uid: "FOLDER_UID_HERE"
  keeper.security/folder-path: "/app/folder-secrets"
```

**By Folder Path:**
```yaml
annotations:
  keeper.security/folder: "Production/Databases"
  keeper.security/folder-path: "/app/db-secrets"
```

Result: All secrets in the folder are written as JSON files to the specified output directory.

### 7. Folder Path Notation

Reference individual secrets using their folder path location:

**Format:** `keeper://FOLDER_PATH/RECORD_NAME/TYPE/SELECTOR:OUTPUT_PATH`

**Examples:**

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"

  # Extract field from record in folder
  keeper.security/secret-db-pass: "Production/Databases/mysql-prod/field/password:/app/secrets/db-pass"

  # Get entire record from folder
  keeper.security/secret-api: "Dev/APIs/stripe-api:/app/secrets/stripe.json"

  # Nested folder path
  keeper.security/secret-cert: "Production/Region/US-East/Database/postgres/field/certificate:/app/certs/cert.pem"
```

**Benefits:**
- **Readable**: Folder paths are more intuitive than 22-character UIDs
- **Maintainable**: Easier to understand secret locations
- **Contextual**: Folder structure provides organizational context
- **Precise**: Case-sensitive matching ensures accuracy

**Supported selectors:**
- `field/<fieldname>` - Extract specific field
- `custom_field/<label>` - Extract custom field
- `file/<filename>` - Download file attachment
- `type` - Get record type
- `title` - Get record title
- `notes` - Get notes field
- (no selector) - Get entire record as JSON

---

## Injection Targets

Keeper K8s Injector supports two methods for injecting secrets into pods:

### Method 1: Files (Recommended)

Secrets are written to tmpfs-backed files in `/keeper/secrets/`. This is the default and recommended approach.

**Advantages**:
- ✅ Not visible in pod metadata or process listings
- ✅ Can be rotated without pod restart (via sidecar)
- ✅ More secure for sensitive data
- ✅ Supports all output formats (JSON, env, YAML, etc.)

**Example**:
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/secret: "database-credentials"
```

Result: `/keeper/secrets/database-credentials.json`

### Method 2: Environment Variables (Optional)

Secrets can be injected directly as environment variables in all containers. Use this for legacy applications that only support env vars.

**⚠️ Security Notice**: Environment variables are visible in `kubectl describe pod` and process listings. Cannot be rotated without pod restart.

**Simple Usage**:
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/inject-env-vars: "true"
  keeper.security/secret: "database-credentials"
```

**Result**: Environment variables injected into all containers:
```bash
LOGIN=demouser
PASSWORD=secret123
HOSTNAME=db.example.com
```

**With Prefix**:
```yaml
annotations:
  keeper.security/inject-env-vars: "true"
  keeper.security/env-prefix: "DB_"
  keeper.security/secret: "database-credentials"
```

**Result**:
```bash
DB_LOGIN=demouser
DB_PASSWORD=secret123
DB_HOSTNAME=db.example.com
```

**Mixed Mode** (some secrets as files, some as env vars):
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/config: |
    secrets:
      - record: database-credentials
        injectAsEnvVars: true
        envPrefix: "DB_"
      - record: tls-certificate
        path: /keeper/secrets/tls.json
```

**When to use environment variables**:
- Legacy applications that only support env vars
- Simple read-once patterns (not frequently rotated secrets)
- Development/testing environments

**When to use files**:
- Production environments (recommended)
- Secrets that rotate frequently
- Sensitive credentials (database passwords, API keys)
- Compliance requirements (SOC2, PCI-DSS)

**Limitations**:
- Environment variables cannot be rotated without pod restart
- Visible in `kubectl describe pod` output
- Visible in process listings
- May be captured in logs or debugging output

---

## Output Formats

### JSON (Default)

```yaml
format: json
```

Output:
```json
{
  "login": "admin",
  "password": "secret123"
}
```

### Environment Variables (.env)

```yaml
format: env
```

Output:
```
LOGIN=admin
PASSWORD=secret123
```

Usage: `export $(cat /keeper/secrets/db.env | xargs)`

### Properties (Java)

```yaml
format: properties
```

Output:
```
login=admin
password=secret123
```

### YAML

```yaml
format: yaml
```

Output:
```yaml
login: admin
password: secret123
```

### INI

```yaml
format: ini
```

Output:
```
[secret]
login=admin
password=secret123
```

### Go Templates

```yaml
template: |
  export DB_URL="postgresql://{{ .login }}:{{ .password }}@postgres:5432/mydb"
```

Output:
```bash
export DB_URL="postgresql://admin:secret123@postgres:5432/mydb"
```

See [Template Guide](templates.md) for complete reference.

---

## Secret Rotation

### Automatic Rotation

```yaml
annotations:
  keeper.security/refresh-interval: "5m"
```

Sidecar checks for updates every 5 minutes and rewrites files.

### Signal on Update

```yaml
annotations:
  keeper.security/signal: "SIGHUP"
```

Sidecar sends SIGHUP to app container when secrets change.

### Init-Only Mode (No Rotation)

```yaml
annotations:
  keeper.security/init-only: "true"
```

Only fetches secrets once at pod startup, no sidecar.

---

## Resilience Features

### Retry with Exponential Backoff

Automatic retry on transient failures:
- 3 attempts
- Delays: 200ms, 400ms, 800ms (exponential)
- Respects context cancellation

### In-Memory Secret Caching

Secrets cached after successful fetch:
- 24-hour maximum age
- Thread-safe concurrent access
- Cleared on pod restart
- Memory-only (no disk persistence)

### Cache Fallback

When Keeper API is unavailable after retry:
```yaml
keeper.security/fail-on-error: "true"   # Fail if no cache (default)
keeper.security/fail-on-error: "false"  # Use cache, or start without secrets
```

**Behavior:**
- Keeper up → Fetch and cache
- Keeper down, cache exists → Use cached value (warn in logs)
- Keeper down, no cache + fail-on-error=true → Pod fails
- Keeper down, no cache + fail-on-error=false → Pod starts, no secrets

---

## Corporate Proxy Support

### Custom CA Certificate

For SSL inspection environments (Zscaler, Palo Alto, Cisco Umbrella):

```yaml
annotations:
  keeper.security/ca-cert-configmap: "corporate-ca"
```

Or from Secret:

```yaml
annotations:
  keeper.security/ca-cert-secret: "zscaler-ca"
  keeper.security/ca-cert-key: "root-ca.pem"  # optional, default: ca.crt
```

**Setup:**
```bash
# Create ConfigMap from CA cert file
kubectl create configmap corporate-ca --from-file=ca.crt=zscaler-root.pem

# Use in pod
annotations:
  keeper.security/ca-cert-configmap: "corporate-ca"
```

The sidecar automatically loads the CA cert and adds it to the system trust store.

---

## Authentication Methods

### Method 1: Kubernetes Secret (Default)

```yaml
annotations:
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/auth-method: "secret"  # default, can be omitted
```

Works in any Kubernetes cluster.

### Method 2: AWS Secrets Manager (EKS with IRSA)

```yaml
annotations:
  keeper.security/auth-method: "aws-secrets-manager"
  keeper.security/aws-secret-id: "prod/keeper/ksm-config"
  keeper.security/aws-region: "us-west-2"
```

Requires EKS cluster with OIDC and ServiceAccount with IAM role annotation.

**Security benefit:** No KSM config stored in Kubernetes cluster.

### Method 3: GCP Secret Manager (GKE with Workload Identity)

```yaml
annotations:
  keeper.security/auth-method: "gcp-secret-manager"
  keeper.security/gcp-secret-id: "projects/PROJECT/secrets/ksm-config/versions/latest"
```

Requires GKE with Workload Identity and ServiceAccount with GCP SA annotation.

### Method 4: Azure Key Vault (AKS with Workload Identity)

```yaml
annotations:
  keeper.security/auth-method: "azure-key-vault"
  keeper.security/azure-vault-name: "mykeyvault"
  keeper.security/azure-secret-name: "ksm-config"
```

Requires AKS with Workload Identity and ServiceAccount with Azure client-id annotation.

See [Cloud Secrets Guide](cloud-secrets.md) for detailed setup instructions.

---

## Error Handling

### Fail on Error (Default)

```yaml
annotations:
  keeper.security/fail-on-error: "true"
```

Pod fails to start if secrets can't be fetched.

### Graceful Degradation

```yaml
annotations:
  keeper.security/fail-on-error: "false"
```

Pod starts even if secret fetch fails (keeps last good values).

---

## Advanced Features

### Strict Lookup

```yaml
annotations:
  keeper.security/strict-lookup: "true"
```

Fails if multiple records match the same title.

### Template Functions (100+)

Available via Sprig library:
- **Encoding:** base64enc, base64dec, urlquery
- **Hashing:** sha256sum, sha512sum, md5sum
- **Strings:** upper, lower, trim, replace, split, join
- **Date/Time:** now, date, ago
- **Crypto:** bcrypt, htpasswd, uuidv4
- **Logic:** default, empty, coalesce, ternary
- **Lists:** first, last, reverse, uniq

See [Template Guide](templates.md) for examples.

---

## Production Configuration

### High Availability

```yaml
# Helm values
replicaCount: 3
podDisruptionBudget:
  enabled: true
  minAvailable: 1
```

### Resource Limits

```yaml
# Helm values
sidecar:
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 100m
      memory: 128Mi
```

### Metrics

```yaml
# Helm values
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
```

Prometheus metrics at `:8080/metrics`:
- `keeper_secrets_fetched_total`
- `keeper_secrets_fetch_errors_total`
- `keeper_secrets_fetch_duration_seconds`

---

## Security Features

### tmpfs Storage

Secrets stored in memory-only tmpfs volumes, never written to disk.

### Read-Only Root Filesystem

All containers run with read-only root filesystems.

### Non-Root Execution

Containers run as non-root user (UID 65534).

### Dropped Capabilities

All Linux capabilities dropped from containers.

### Pod-Scoped Lifetime

Secrets removed automatically when pod terminates.

---

## Complete Example

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: production-app
  annotations:
    # Core configuration
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-credentials"

    # CA certificate for corporate proxy
    keeper.security/ca-cert-configmap: "corporate-ca"

    # Full YAML configuration with templates
    keeper.security/config: |
      secrets:
        # Database connection string via template
        - record: postgres-credentials
          path: /app/config/database.sh
          template: |
            export DB_URL="postgresql://{{ .login }}:{{ .password }}@{{ .hostname }}:5432/mydb"

        # API keys in properties format
        - record: api-config
          path: /app/config/api.properties
          format: properties

        # TLS certificates from file attachments
        - record: tls-certificate
          file: server.crt
          path: /app/certs/server.crt

        - record: tls-certificate
          file: server.key
          path: /app/certs/server.key

    # Behavior configuration
    keeper.security/refresh-interval: "10m"
    keeper.security/signal: "SIGHUP"
    keeper.security/fail-on-error: "true"
    keeper.security/strict-lookup: "true"
spec:
  containers:
    - name: app
      image: myapp:latest
```

---

## See Also

- [Annotations Reference](annotations.md) - All annotations with examples
- [Template Guide](templates.md) - Template syntax and functions
- [Quick Start](quickstart.md) - Getting started guide
- [Troubleshooting](troubleshooting.md) - Common issues
