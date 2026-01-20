# Configuration Guide

Complete reference for configuring Keeper K8s Injector through annotations and Helm values.

## Quick Reference

**Minimal Configuration:**
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secret: "my-secret"
```

**For detailed examples, see [Examples](../examples/) folder.**

---

## Table of Contents

1. [Authentication Setup](#authentication-setup)
2. [Annotation Reference](#annotation-reference)
3. [Helm Chart Values](#helm-chart-values)

---

## Authentication Setup

### Method 1: Kubernetes Secret (Default)

Store KSM configuration in a Kubernetes Secret:

```bash
# Create from base64 config (recommended)
kubectl create secret generic keeper-credentials \
  --from-literal=config='<paste-base64-config-here>' \
  -n default

# Or from JSON file
kubectl create secret generic keeper-credentials \
  --from-file=config=ksm-config.json \
  -n default
```

Use in pod:
```yaml
annotations:
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/auth-method: "secret"  # default, can be omitted
```

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

**See [Cloud Authentication Guide](cloud-auth.md) for detailed setup instructions.**

---

## Annotation Reference

All annotations use the `keeper.security/` prefix.

### Required Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/inject` | Enable injection | `"true"` |
| `keeper.security/ksm-config` | K8s secret with KSM config | `"keeper-auth"` |

### Secret Selection

#### Level 1: Single Secret (Simplest)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secret: "database-credentials"
```

Result: `/keeper/secrets/database-credentials.json`

#### Level 2: Multiple Secrets

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secrets: "database-creds, api-keys, tls-cert"
```

Result:
- `/keeper/secrets/database-creds.json`
- `/keeper/secrets/api-keys.json`
- `/keeper/secrets/tls-cert.json`

#### Level 3: Custom Paths

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secret-database: "/app/config/db.json"
  keeper.security/secret-api: "/etc/myapp/api-keys.json"
```

#### Level 4: Field Extraction

Extract specific fields from a record:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secret-password: "database-creds[password]:/app/secrets/db-pass"
```

Result: `/app/secrets/db-pass` contains only the password value (raw, not JSON)

#### Level 5: Full Configuration

For complex scenarios, use YAML configuration:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
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

#### Level 6: Templates (Advanced)

Use Go templates for custom formatting:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/config: |
    secrets:
      - record: postgres-credentials
        path: /app/config/database.sh
        template: |
          export DB_USER="{{ .login }}"
          export DB_PASS="{{ .password }}"
          export DB_URL="postgresql://{{ .login }}:{{ .password }}@{{ .hostname }}:5432/mydb"
```

Result: Shell script with connection string built from Keeper fields.

Templates support 100+ functions from [Sprig](http://masterminds.github.io/sprig/). See [Template Guide](templates.md) for details.

### Behavior Annotations

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/refresh-interval` | `"5m"` | How often to refresh secrets |
| `keeper.security/init-only` | `"false"` | Only use init container (no sidecar) |
| `keeper.security/fail-on-error` | `"true"` | Fail pod startup if secrets can't be fetched |
| `keeper.security/signal` | `""` | Signal to send on refresh (e.g., `"SIGHUP"`) |
| `keeper.security/strict-lookup` | `"false"` | Fail if multiple records match title |

### Environment Variable Injection Annotations

**⚠️ Security Notice**: Environment variables are less secure than file-based injection. See [Security Trade-offs](#security-trade-offs) below.

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/inject-env-vars` | `"false"` | Inject secrets as environment variables instead of files |
| `keeper.security/env-prefix` | `""` | Optional prefix for all env var names (e.g., `"DB_"`) |

#### Simple Usage (All Secrets as Env Vars)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/inject-env-vars: "true"
  keeper.security/secret: "database-credentials"
```

**Result**: Environment variables injected into all containers:
```
LOGIN=demouser
PASSWORD=secret123
HOSTNAME=db.example.com
```

#### With Prefix (Namespace Env Vars)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/inject-env-vars: "true"
  keeper.security/env-prefix: "DB_"
  keeper.security/secret: "database-credentials"
```

**Result**:
```
DB_LOGIN=demouser
DB_PASSWORD=secret123
DB_HOSTNAME=db.example.com
```

#### Mixed Mode (Some as Files, Some as Env Vars)

For fine-grained control, use YAML configuration:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/config: |
    secrets:
      - record: database-credentials
        injectAsEnvVars: true
        envPrefix: "DB_"
      - record: tls-certificate
        path: /keeper/secrets/tls.json
        # File-based, not env vars
```

**Result**:
- Env vars: `DB_LOGIN`, `DB_PASSWORD`, `DB_HOSTNAME`
- File: `/keeper/secrets/tls.json`

#### Security Trade-offs

**When to use environment variables**:
- Legacy applications that only support env vars
- Simple read-once patterns (not frequently rotated secrets)
- Development/testing environments

**When to use files (recommended)**:
- Production environments
- Secrets that rotate frequently
- Sensitive credentials (database passwords, API keys)
- Compliance requirements (SOC2, PCI-DSS)

**Environment variable limitations**:
- ❌ Visible in `kubectl describe pod` output
- ❌ Visible in process listings inside containers
- ❌ May be captured in logs or debugging output
- ❌ Cannot be rotated without pod restart
- ✅ Secrets never stored in etcd (not K8s Secrets)

**File-based advantages**:
- ✅ Not visible in pod metadata
- ✅ Can be rotated without pod restart (via sidecar)
- ✅ More secure for sensitive data
- ✅ tmpfs storage prevents disk persistence

### Kubernetes Secret Injection (v0.9.0)

**⚠️ Security Notice**: K8s Secrets are less secure than file-based injection. Use for GitOps workflows or apps requiring K8s Secret mounts.

#### Overview

Create Kubernetes Secret objects directly from Keeper secrets, enabling native K8s integration while maintaining efficient Keeper backend calls.

#### When to Use

- ✅ Apps that mount K8s Secrets as volumes
- ✅ GitOps workflows requiring K8s Secret manifests
- ✅ CSI driver compatibility
- ✅ Sharing secrets across multiple pods
- ❌ **Not recommended for maximum security** (use file-based tmpfs instead)

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/inject-as-k8s-secret` | `"false"` | Enable K8s Secret injection |
| `keeper.security/k8s-secret-name` | Required | K8s Secret name to create |
| `keeper.security/k8s-secret-namespace` | Pod namespace | Namespace for Secret (optional) |
| `keeper.security/k8s-secret-mode` | `"overwrite"` | Conflict resolution (`overwrite`, `merge`, `skip-if-exists`, `fail`) |
| `keeper.security/k8s-secret-type` | `"Opaque"` | Secret type (`Opaque`, `kubernetes.io/tls`, etc.) |
| `keeper.security/k8s-secret-rotation` | `"false"` | Enable sidecar rotation (updates Secret on refresh) |
| `keeper.security/k8s-secret-owner-ref` | `"true"` | Auto-delete Secret when pod terminates |

#### Basic Usage

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-credentials"
    keeper.security/inject-as-k8s-secret: "true"
    keeper.security/k8s-secret-name: "app-secrets"
    keeper.security/secret: "database-credentials"
spec:
  containers:
    - name: app
      image: myapp:latest
      env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: password
```

**Result**: K8s Secret `app-secrets` created with all fields from `database-credentials`.

#### Custom Key Mapping

Map Keeper fields to specific Secret keys:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/config: |
    secrets:
      - record: "postgres-prod"
        injectAsK8sSecret: true
        k8sSecretName: "db-credentials"
        k8sSecretKeys:
          username: "POSTGRES_USER"
          password: "POSTGRES_PASSWORD"
          host: "POSTGRES_HOST"
```

**Result**: K8s Secret with custom key names:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-credentials
data:
  POSTGRES_USER: <base64>
  POSTGRES_PASSWORD: <base64>
  POSTGRES_HOST: <base64>
```

#### Rotation via Sidecar

Enable automatic Secret updates when secrets change in Keeper:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/inject-as-k8s-secret: "true"
  keeper.security/k8s-secret-name: "app-secrets"
  keeper.security/k8s-secret-rotation: "true"
  keeper.security/refresh-interval: "5m"
  keeper.security/init-only: "false"  # Required for rotation
  keeper.security/secret: "database-credentials"
```

**Result**: Sidecar updates K8s Secret every 5 minutes with latest values from Keeper.

#### Security Comparison

| Aspect | Files (tmpfs) | Env Vars | K8s Secrets |
|--------|--------------|----------|-------------|
| **Storage** | RAM (tmpfs) | Pod spec | etcd (disk) |
| **Persistence** | Pod lifetime | Pod lifetime | Survives pod deletion |
| **Backups** | Not included | Not included | ✅ Included in backups |
| **Encryption** | N/A (RAM) | N/A | Requires etcd encryption |
| **Audit** | Container logs | Pod metadata | K8s audit logs |
| **Visibility** | Hidden | `kubectl describe` | `kubectl get secret` |
| **Rotation** | ✅ Yes (sidecar) | ❌ No | ✅ Yes (sidecar) |
| **Best For** | Production | Legacy apps | K8s-native apps |

### Authentication Annotations

#### Basic Authentication (K8s Secret)

| Annotation | Default | Description |
|------------|---------|-------------|
| `keeper.security/ksm-config` | Required | K8s secret name with KSM config |
| `keeper.security/auth-method` | `"secret"` | Auth method (see below) |

#### Cloud Provider Authentication

| Auth Method | Description | Cloud |
|-------------|-------------|-------|
| `"secret"` | K8s Secret (default) | Any |
| `"aws-secrets-manager"` | AWS Secrets Manager via IRSA | EKS |
| `"gcp-secret-manager"` | GCP Secret Manager via Workload Identity | GKE |
| `"azure-key-vault"` | Azure Key Vault via Workload Identity | AKS |

#### AWS Secrets Manager Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/aws-secret-id` | AWS Secrets Manager secret ID or ARN | `"prod/keeper/ksm-config"` |
| `keeper.security/aws-region` | AWS region (optional, auto-detect) | `"us-west-2"` |

**Example:**
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-method: "aws-secrets-manager"
  keeper.security/aws-secret-id: "prod/keeper/ksm-config"
  keeper.security/aws-region: "us-west-2"
```

#### GCP Secret Manager Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/gcp-secret-id` | GCP Secret Manager resource name | `"projects/my-project/secrets/ksm-config/versions/latest"` |

**Example:**
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-method: "gcp-secret-manager"
  keeper.security/gcp-secret-id: "projects/my-project/secrets/ksm-config/versions/latest"
```

#### Azure Key Vault Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/azure-vault-name` | Azure Key Vault name | `"mykeyvault"` |
| `keeper.security/azure-secret-name` | Secret name in Key Vault | `"ksm-config"` |

**Example:**
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-method: "azure-key-vault"
  keeper.security/azure-vault-name: "mykeyvault"
  keeper.security/azure-secret-name: "ksm-config"
```

See [Cloud Authentication Guide](cloud-auth.md) for complete setup instructions.

### CA Certificate Annotations (Corporate Proxies)

For environments with SSL inspection (Zscaler, Palo Alto, Cisco Umbrella, etc.):

| Annotation | Description | Example |
|------------|-------------|---------|
| `keeper.security/ca-cert-secret` | K8s Secret with custom CA certificate | `"corporate-ca"` |
| `keeper.security/ca-cert-configmap` | K8s ConfigMap with custom CA certificate | `"zscaler-ca"` |
| `keeper.security/ca-cert-key` | Key in Secret/ConfigMap | `"ca.crt"` (default) |

**Example:**
```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/secret: "my-secret"
  keeper.security/ca-cert-configmap: "corporate-ca"  # For SSL inspection
```

This loads the custom CA certificate and adds it to the system trust store, allowing the sidecar to connect through corporate proxies.

See [Corporate Proxy Guide](corporate-proxy.md) for detailed setup.

### Output Formats

When using Level 5+ configuration, you can specify output format:

| Format | Description | Example Output |
|--------|-------------|----------------|
| `json` | JSON object (default) | `{"login": "user", "password": "pass"}` |
| `env` | Environment file | `LOGIN=user\nPASSWORD=pass` |
| `properties` | Java properties | `login=user\npassword=pass` |
| `yaml` | YAML format | `login: user\npassword: pass` |
| `ini` | INI format | `[secret]\nlogin=user\npassword=pass` |
| `raw` | Raw value (single field only) | `mypassword123` |
| `template` | Custom via Go template | User-defined format |

For `template` format, use the `template:` field to specify a Go template string. See [Template Guide](templates.md).

### Secrets by UID

If you prefer using record UIDs instead of titles:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/secret: "OqPt3Vd37My7G8rTb-8Q"  # 22-char UID
```

The injector auto-detects UIDs vs titles based on format.

### Keeper Notation

Use Keeper notation for precise field extraction with custom output paths:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  # Format: keeper://UID/field/FIELDNAME:OUTPUT_PATH
  keeper.security/secret-password: "keeper://QabbPIdM8Unw4hwVM-F8VQ/field/password:/app/secrets/db-pass"
  keeper.security/secret-login: "keeper://QabbPIdM8Unw4hwVM-F8VQ/field/login:/app/secrets/db-user"
```

Result:
- `/app/secrets/db-pass` contains the password value (raw text)
- `/app/secrets/db-user` contains the login value (raw text)

#### Notation Syntax

The Keeper notation format is: `keeper://RECORD_UID/TYPE/SELECTOR`

| Type | Selector | Description |
|------|----------|-------------|
| `field` | field name | Extract a specific field value |
| `file` | filename | Download a file attachment |
| `custom_field` | field label | Extract custom field by label |

#### Folder Path Notation

You can now use folder paths in Keeper notation to reference secrets by their location:

**Format:** `keeper://FOLDER_PATH/RECORD_NAME/TYPE/SELECTOR:OUTPUT_PATH`

**Examples:**

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"

  # Reference secret by folder path + title
  keeper.security/secret-db-pass: "keeper://Production/Databases/mysql-credentials/field/password:/app/secrets/db-pass"

  # Without keeper:// prefix
  keeper.security/secret-api-key: "Dev/APIs/stripe-api/field/api_key:/app/secrets/api-key"

  # Nested folders
  keeper.security/secret-cert: "Production/Region/US-East/Databases/postgres/field/certificate:/app/certs/cert.pem"
```

**Benefits:**
- More readable than UIDs
- Easier to maintain and understand
- Folder structure provides context
- Case-sensitive matching for precision

### File Attachments

Download file attachments from Keeper records:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  # Format: RECORD_TITLE:FILENAME:OUTPUT_PATH
  keeper.security/file-cert: "Database Credentials:cert.pem:/app/certs/server.pem"
  keeper.security/file-key: "Database Credentials:key.pem:/app/certs/server.key"
```

Result:
- `/app/certs/server.pem` contains the `cert.pem` file from "Database Credentials" record
- `/app/certs/server.key` contains the `key.pem` file from "Database Credentials" record

### Folder Support

Fetch all secrets from a Keeper folder:

#### By Folder UID

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/folder-uid: "FOLDER_UID_HERE"
  keeper.security/folder-path: "/app/folder-secrets"  # Output directory
```

Result: All secrets in the folder are written as JSON files to `/app/folder-secrets/`

#### By Folder Path

You can now reference folders by their path instead of UID:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-auth"
  keeper.security/folder: "Production/Databases"  # Folder path
  keeper.security/folder-path: "/app/db-secrets"  # Output directory
```

Result: All secrets in the `Production/Databases` folder are written to `/app/db-secrets/`

Folder paths are case-sensitive and must match the exact folder names in your Keeper vault.

### Complete Example

This example demonstrates multiple annotation types:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-auth"

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

---

## Helm Chart Values

### Installation

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm install keeper-injector keeper/keeper-injector -f values.yaml
```

### Custom Values

```yaml
# values.yaml

# Number of webhook replicas (HA)
replicaCount: 3

# Container images
image:
  repository: keeper/injector-webhook
  tag: "1.0.0"
  pullPolicy: IfNotPresent

sidecar:
  repository: keeper/injector-sidecar
  tag: "1.0.0"

# Resource limits for webhook
resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

# Resource limits for sidecar (per pod)
sidecarResources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 20m
    memory: 64Mi

# Exclude additional namespaces from injection
excludedNamespaces:
  - kube-system
  - kube-public
  - monitoring
  - istio-system

# Default settings for all pods (can be overridden per-pod)
defaults:
  refreshInterval: "10m"
  failOnError: true

# Prometheus metrics
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

# High availability
leaderElection:
  enabled: true

podDisruptionBudget:
  enabled: true
  minAvailable: 2

# TLS certificate management
tls:
  autoGenerate: true  # Auto-generate certs (default)
  certManager:
    enabled: false    # Use cert-manager (requires cert-manager installed)
    issuerKind: Issuer
    issuerName: ""    # Leave empty to auto-create

# Network policies
networkPolicy:
  enabled: false

# Logging configuration
logging:
  level: info  # debug, info, warn, error
  format: json  # json or text
```

### Production Configuration Example

```yaml
# production-values.yaml
replicaCount: 3

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 256Mi

podDisruptionBudget:
  enabled: true
  minAvailable: 2

metrics:
  enabled: true
  serviceMonitor:
    enabled: true

tls:
  certManager:
    enabled: true

logging:
  level: info
  format: json

excludedNamespaces:
  - kube-system
  - kube-public
  - cert-manager
  - monitoring
```

Install with production values:

```bash
helm install keeper-injector keeper/keeper-injector -f production-values.yaml
```

### Common Customizations

#### Enable cert-manager

```yaml
tls:
  certManager:
    enabled: true
```

Requires cert-manager to be installed first.

#### Exclude Namespaces

```yaml
excludedNamespaces:
  - my-excluded-namespace
  - another-namespace
```

Pods in these namespaces will not be injected.

#### Custom Sidecar Resources

```yaml
sidecarResources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 128Mi
```

#### Debug Logging

```yaml
logging:
  level: debug
  format: text  # Human-readable for debugging
```

---

**[← Back to Documentation Index](INDEX.md)**
