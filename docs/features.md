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

```yaml
annotations:
  keeper.security/folder-uid: "FOLDER_UID_HERE"
  keeper.security/folder-path: "/app/folder-secrets"
```

Fetches all secrets from a Keeper folder.

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
