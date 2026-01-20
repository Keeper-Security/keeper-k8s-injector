# Injection Modes

Keeper K8s Injector supports three methods for injecting secrets into pods. Choose based on your security requirements and application compatibility.

## Quick Comparison

| Feature | Files | Env Vars | K8s Secrets |
|---------|-------|----------|-------------|
| **Storage** | tmpfs (RAM) | Pod spec | etcd (disk) |
| **Sync from Keeper** | ✅ Yes | ❌ No | ✅ Yes |
| **Visibility** | Hidden | `kubectl describe` | `kubectl get secret` |
| **Security** | Highest | Medium | Medium |
| **Use For** | Production | Legacy apps | GitOps/K8s-native |

**Sync from Keeper**: When you update a secret in Keeper, the sidecar detects the change and updates the injected files/K8s Secrets without pod restart.

---

## Method 1: Files (Recommended)

Secrets are written to tmpfs-backed files in `/keeper/secrets/`. This is the default and recommended approach.

### Advantages

- ✅ Not visible in pod metadata or process listings
- ✅ Syncs changes from Keeper without pod restart (via sidecar)
- ✅ More secure for sensitive data
- ✅ Supports all output formats (JSON, env, YAML, etc.)

### Basic Usage

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/secret: "database-credentials"
```

**Result:** `/keeper/secrets/database-credentials.json`

### Multiple Secrets

```yaml
annotations:
  keeper.security/secret: "db-creds, api-keys, tls-cert"
```

**Result:**
- `/keeper/secrets/db-creds.json`
- `/keeper/secrets/api-keys.json`
- `/keeper/secrets/tls-cert.json`

### Custom Paths

```yaml
annotations:
  keeper.security/secret-db: "/app/config/database.json"
  keeper.security/secret-api: "/etc/app/api-keys.json"
```

### With Format Conversion

```yaml
annotations:
  keeper.security/config: |
    secrets:
      - record: database-credentials
        path: /app/config/db.env
        format: env
      - record: api-config
        path: /app/config/api.properties
        format: properties
```

**Result:**
- `/app/config/db.env` in environment file format
- `/app/config/api.properties` in Java properties format

### When to Use Files

✅ **Recommended for:**
- Production environments
- Secrets that change frequently in Keeper
- Sensitive credentials (database passwords, API keys)
- Compliance requirements (SOC2, PCI-DSS)
- Applications that can read from files

❌ **Not suitable for:**
- Legacy apps that only support environment variables
- Applications that cannot read from `/keeper/secrets/`

---

## Method 2: Environment Variables

Secrets can be injected directly as environment variables in all containers. Use this for legacy applications that only support env vars.

### Security Notice

**⚠️ Warning**: Environment variables are visible in `kubectl describe pod` and process listings. Cannot sync changes from Keeper without pod restart.

### Basic Usage

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/ksm-config: "keeper-credentials"
  keeper.security/inject-env-vars: "true"
  keeper.security/secret: "database-credentials"
```

**Result**: Environment variables injected into all containers:
```bash
LOGIN=demouser
PASSWORD=secret123
HOSTNAME=db.example.com
```

### With Prefix

Add a prefix to namespace your environment variables:

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

### Mixed Mode (Files + Env Vars)

Inject some secrets as files, others as environment variables:

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
```

**Result**:
- Environment variables: `DB_LOGIN`, `DB_PASSWORD`, `DB_HOSTNAME`
- File: `/keeper/secrets/tls.json`

### When to Use Environment Variables

✅ **Use for:**
- Legacy applications that only support env vars
- Simple read-once patterns (secrets don't change often in Keeper)
- Development/testing environments
- Apps that expect specific env var names

❌ **Avoid for:**
- Production secrets (use files instead)
- Secrets that change frequently in Keeper
- Compliance environments (SOC2, PCI-DSS)
- Sensitive credentials

### Limitations

- ❌ Cannot sync changes from Keeper without pod restart
- ❌ Visible in `kubectl describe pod` output
- ❌ Visible in process listings (`ps aux`)
- ❌ May be captured in logs or debugging output
- ✅ Secrets never stored in etcd (not K8s Secrets)

---

## Method 3: Kubernetes Secrets

Secrets can be injected as Kubernetes Secret objects for GitOps workflows and K8s-native applications.

### Security Notice

**⚠️ Warning**: K8s Secrets are stored in etcd. Use file-based injection for higher security.

### When to Use

✅ **Use for:**
- Apps that mount K8s Secrets as volumes
- GitOps workflows requiring K8s Secret manifests
- CSI driver compatibility
- Sharing secrets across multiple pods

❌ **Avoid for:**
- Maximum security (use files instead)
- Applications that can read from tmpfs
- Compliance-sensitive workloads

### Basic Usage

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
      env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: password
```

**Result**: K8s Secret `app-secrets` created with all fields from record.

### Custom Key Mapping

Map Keeper fields to specific Secret keys:

```yaml
annotations:
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

### Conflict Resolution

Control what happens when a Secret already exists:

#### Overwrite (Default)
```yaml
keeper.security/k8s-secret-mode: "overwrite"
```
Replaces all data in existing Secret. Use when this pod is the primary secret source.

#### Merge
```yaml
keeper.security/k8s-secret-mode: "merge"
```
Preserves existing keys, adds/updates new ones. Use when multiple pods update the same Secret.

#### Skip If Exists
```yaml
keeper.security/k8s-secret-mode: "skip-if-exists"
```
Does nothing if Secret already exists. Use for idempotent deployments.

#### Fail
```yaml
keeper.security/k8s-secret-mode: "fail"
```
Returns error if Secret already exists. Use for strict validation.

### Sync Changes from Keeper via Sidecar

Enable automatic K8s Secret updates when you change secrets in Keeper:

```yaml
annotations:
  keeper.security/inject-as-k8s-secret: "true"
  keeper.security/k8s-secret-name: "app-secrets"
  keeper.security/k8s-secret-rotation: "true"
  keeper.security/refresh-interval: "5m"
  keeper.security/secret: "database-credentials"
```

**Result**: Sidecar updates K8s Secret every 5 minutes with latest values from Keeper.

**Note**: Applications must watch for Secret updates or use a tool like [Reloader](https://github.com/stakater/Reloader) to restart pods on Secret changes.

### Owner Reference Control

By default, K8s Secrets are deleted when the pod terminates (via owner reference). To keep Secrets after pod deletion:

```yaml
annotations:
  keeper.security/k8s-secret-owner-ref: "false"
```

**Use cases:**
- Secrets shared across multiple pods
- Manual Secret lifecycle management
- StatefulSet deployments

### TLS Certificate Injection

Create a `kubernetes.io/tls` Secret for Ingress use:

```yaml
annotations:
  keeper.security/config: |
    secrets:
      - record: "TLS Certificate"
        fileName: "cert.pem"
        injectAsK8sSecret: true
        k8sSecretName: "tls-cert"
        k8sSecretType: "kubernetes.io/tls"
        k8sSecretKeys:
          tls.crt: "cert.pem"
          tls.key: "key.pem"
```

**Result**: K8s Secret of type `kubernetes.io/tls` ready for Ingress use.

### Supported Secret Types

- `Opaque` - Default, arbitrary key-value pairs
- `kubernetes.io/tls` - TLS certificates (requires `tls.crt` and `tls.key`)
- `kubernetes.io/dockerconfigjson` - Docker registry auth
- `kubernetes.io/basic-auth` - Basic authentication
- `kubernetes.io/ssh-auth` - SSH authentication

---

## Security Comparison

| Aspect | Files (tmpfs) | Env Vars | K8s Secrets |
|--------|--------------|----------|-------------|
| **Storage** | RAM (tmpfs) | Pod spec | etcd (disk) |
| **Persistence** | Pod lifetime | Pod lifetime | Survives pod deletion |
| **Backups** | Not included | Not included | ✅ Included in backups |
| **Encryption** | N/A (RAM) | N/A | Requires etcd encryption |
| **Audit** | Container logs | Pod metadata | K8s audit logs |
| **Visibility** | Hidden | `kubectl describe` | `kubectl get secret` |
| **Sync from Keeper** | ✅ Yes (sidecar) | ❌ No | ✅ Yes (sidecar) |
| **Best For** | Production | Legacy apps | K8s-native apps |

---

## Security Recommendations

1. **Enable etcd encryption** if using K8s Secret injection
2. **Use RBAC** to limit who can read Secrets
3. **Prefer file-based injection** for maximum security
4. **Use K8s Secrets only when**:
   - GitOps requires K8s Secret manifests
   - Apps expect K8s Secret mounts
   - CSI driver integration needed
5. **Monitor Secret access** via K8s audit logs

---

## Performance Considerations

### Efficient Batching

The injector makes **ONE** Keeper API call to fetch all records, then creates multiple outputs (files, env vars, or K8s Secrets). This minimizes API load and speeds up pod admission.

**Example**: 10 secrets = 1 API call (not 10)

### Memory Usage

- **Files (tmpfs)**: Counts against pod memory limits
- **Env Vars**: Minimal overhead (part of pod spec)
- **K8s Secrets**: Stored in etcd, not in pod memory

**Tip**: If injecting large secrets via files, ensure adequate memory limits:

```yaml
spec:
  containers:
    - name: app
      resources:
        limits:
          memory: 512Mi  # Include headroom for secrets in tmpfs
```

---

**[← Back to Documentation Index](INDEX.md)**
