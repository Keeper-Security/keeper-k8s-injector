# Feature Comparison: Keeper vs Competitors

This document compares Keeper Kubernetes Secrets Injector with industry-standard secret management solutions.

## Products Compared

| Product | Type | Vendor |
|---------|------|--------|
| **Keeper K8s Injector** | Sidecar injection | Keeper Security |
| **Vault Agent Injector** | Sidecar injection | HashiCorp |
| **External Secrets Operator (ESO)** | Controller-based sync | Community |
| **AWS Secrets CSI Driver** | CSI Driver | AWS |
| **1Password Injector** | Sidecar injection | 1Password |

---

## Architecture Comparison

| Feature | Keeper | Vault Agent | ESO | AWS CSI | 1Password |
|---------|--------|-------------|-----|---------|-----------|
| **Creates K8s Secrets** | No | No | Yes | No | No |
| **Secrets in etcd** | No | No | Yes | No | No |
| **Secrets in backups** | No | No | Yes | No | No |
| **Storage location** | tmpfs | tmpfs | etcd | tmpfs | tmpfs |
| **Configuration method** | Annotations | Annotations | CRDs | CRDs | Annotations |
| **Injection method** | Webhook | Webhook | Controller | CSI Driver | Webhook |

**Winner for security:** Keeper, Vault, 1Password, AWS CSI (tie) - None persist secrets in cluster

---

## Secret Formats

| Format | Keeper | Vault Agent | ESO | AWS CSI | 1Password |
|--------|--------|-------------|-----|---------|-----------|
| **JSON** | ✅ Default | ✅ Via template | ✅ Default | ✅ | ✅ |
| **.env format** | ✅ Built-in | ✅ Via template | ✅ Via template | ❌ | ❌ |
| **Properties** | ❌ Missing | ✅ Via template | ✅ Via template | ❌ | ❌ |
| **YAML** | ❌ Missing | ✅ Via template | ✅ Via template | ❌ | ❌ |
| **Custom templates** | ❌ Missing | ✅ Go templates | ✅ Go templates | ❌ | ❌ |
| **Shell scripts** | ⚠️ Via .env | ✅ Via template | ✅ Via template | ❌ | ❌ |

**Winner:** Vault Agent, ESO (tie) - Full template flexibility

---

## Field Extraction

| Feature | Keeper | Vault Agent | ESO | AWS CSI | 1Password |
|---------|--------|-------------|-----|---------|-----------|
| **Specific field extraction** | ✅ Keeper notation | ✅ Template syntax | ✅ dataFrom | ❌ | ⚠️ Limited |
| **Field remapping** | ❌ | ✅ Templates | ✅ Templates | ❌ | ❌ |
| **Computed values** | ❌ | ✅ Template functions | ✅ Template functions | ❌ | ❌ |
| **String concatenation** | ❌ | ✅ Templates | ✅ Templates | ❌ | ❌ |

**Example - Build connection string:**

**Vault Agent:**
```yaml
template: |
  postgresql://{{ .Data.username }}:{{ .Data.password }}@postgres:5432/mydb
```

**ESO:**
```yaml
template:
  data:
    connectionString: "postgresql://{{ .username }}:{{ .password }}@postgres:5432/mydb"
```

**Keeper (current):**
```yaml
# Can't do this - would need templates
# Workaround: Fetch fields separately and build string in app code
```

**Winner:** Vault Agent, ESO (tie) - Template-based concatenation

---

## Secret Rotation

| Feature | Keeper | Vault Agent | ESO | AWS CSI | 1Password |
|---------|--------|-------------|-----|---------|-----------|
| **Automatic refresh** | ✅ Sidecar polls | ✅ Sidecar polls | ✅ Controller syncs | ✅ Driver polls | ✅ Sidecar |
| **Configurable interval** | ✅ Per-pod | ✅ Per-pod | ✅ Per-ExternalSecret | ✅ Global | ⚠️ Limited |
| **Signal on update** | ✅ SIGHUP, etc. | ✅ process-supervisor | ❌ Needs Reloader | ❌ | ❌ |
| **File-based rotation** | ✅ In-place update | ✅ In-place update | ❌ Recreates Secret | ✅ | ✅ |
| **Zero-downtime** | ✅ | ✅ | ⚠️ Depends on app | ✅ | ✅ |

**Winner:** Keeper, Vault Agent (tie) - Both have signal support for app notification

---

## Configuration Complexity

### Minimal Configuration (Hello World)

**Keeper (2 annotations):**
```yaml
keeper.security/inject: "true"
keeper.security/secret: "my-secret"
```

**Vault Agent (3 annotations):**
```yaml
vault.hashicorp.com/agent-inject: "true"
vault.hashicorp.com/role: "myapp"
vault.hashicorp.com/agent-inject-secret-db: "database/creds"
```

**ESO (2 CRDs):**
```yaml
# SecretStore + ExternalSecret = more YAML
kind: SecretStore
---
kind: ExternalSecret
```

**Winner:** Keeper - Simplest for basic use cases

---

## Advanced Features

| Feature | Keeper | Vault Agent | ESO | AWS CSI | 1Password |
|---------|--------|-------------|-----|---------|-----------|
| **File attachments** | ✅ Native | ❌ | ❌ | ❌ | ✅ Documents |
| **Folder support** | ✅ By UID | ❌ | ⚠️ Via multiple secrets | ❌ | ⚠️ Vaults |
| **Dynamic secrets** | ❌ | ✅ DB, PKI, etc. | ⚠️ Via provider | ⚠️ Via provider | ❌ |
| **Secret versioning** | ⚠️ Via Keeper | ✅ Built-in | ⚠️ Via provider | ❌ | ⚠️ Via 1Password |
| **Multi-namespace** | ✅ | ✅ | ✅ | ✅ | ✅ |
| **HA support** | ✅ Leader election | ✅ | ✅ | ✅ | ✅ |

**Winner:** Vault Agent - Most comprehensive feature set

---

## Template Capabilities (Detailed)

### What Templates Enable

1. **Format conversion** - JSON → .env, .properties, YAML, shell scripts
2. **Field remapping** - `username` → `DB_USER`
3. **String building** - Construct connection strings, URLs
4. **Conditional logic** - Different values based on environment
5. **Data transformation** - Base64 encode, URL encode, etc.

### Vault Agent Template Example

```yaml
vault.hashicorp.com/agent-inject-template-config: |
  {{- with secret "database/creds" -}}
  # Database Configuration
  DB_HOST=postgres
  DB_PORT=5432
  DB_NAME=myapp
  DB_USER={{ .Data.username }}
  DB_PASS={{ .Data.password }}
  DB_URL=postgresql://{{ .Data.username }}:{{ .Data.password }}@postgres:5432/myapp
  {{- end -}}
```

Output: `/vault/secrets/config` contains:
```
# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_NAME=myapp
DB_USER=v-kubernetes-myapp-abc123
DB_PASS=xyz789
DB_URL=postgresql://v-kubernetes-myapp-abc123:xyz789@postgres:5432/myapp
```

### ESO Template Example

```yaml
spec:
  target:
    template:
      engineVersion: v2
      data:
        application.properties: |
          spring.datasource.url=jdbc:postgresql://{{ .dbhost }}:5432/{{ .dbname }}
          spring.datasource.username={{ .username }}
          spring.datasource.password={{ .password }}
```

### Keeper (Current - No Templates)

```yaml
keeper.security/config: |
  secrets:
    - record: postgres-credentials
      path: /app/config/db.env
      format: env
```

Output: `/app/config/db.env`:
```
LOGIN=demouser
PASSWORD=mypassword
```

**Limitation:** Can't build `DB_URL` - user must construct it in app code.

---

## What Keeper Needs to Add

### Priority 1: Go Template Support

```yaml
keeper.security/config: |
  secrets:
    - record: postgres-credentials
      path: /app/config/db-init.sh
      template: |
        export DB_USER="{{ .login }}"
        export DB_PASS="{{ .password }}"
        export DB_URL="postgresql://{{ .login }}:{{ .password }}@postgres:5432/mydb"
```

### Priority 2: Template Functions

Industry standard functions:

| Function | Purpose | Example |
|----------|---------|---------|
| `base64enc` | Base64 encode | `{{ .password | base64enc }}` |
| `base64dec` | Base64 decode | `{{ .token | base64dec }}` |
| `upper` | Uppercase | `{{ .username | upper }}` |
| `lower` | Lowercase | `{{ .dbname | lower }}` |
| `replace` | String replace | `{{ .url | replace "http:" "https:" }}` |
| `trim` | Trim whitespace | `{{ .value | trim }}` |
| `sha256sum` | Hash value | `{{ .password | sha256sum }}` |

### Priority 3: Conditional Logic

```yaml
template: |
  {{- if eq .environment "prod" -}}
  DB_HOST=prod-db.example.com
  {{- else -}}
  DB_HOST=staging-db.example.com
  {{- end -}}
```

---

## Implementation Gaps

| Gap | Impact | Workaround |
|-----|--------|------------|
| No Go templates | High | Users parse JSON/env in app code |
| No connection string builder | Medium | Users construct in app |
| No field remapping | Low | Use env format with source |
| No conditional logic | Low | Use separate records per environment |
| No template functions | Medium | Process in app code |

---

## Competitive Positioning

### Where Keeper Wins

1. **Simplicity** - Fewest annotations for basic use cases
2. **File attachments** - Native support (Vault doesn't have this)
3. **Folder support** - Batch fetch secrets
4. **No CRDs** - Pure annotations (simpler than ESO)
5. **Signal support** - App notification on rotation

### Where Keeper Loses

1. **Template flexibility** - Vault/ESO have full Go templates
2. **Dynamic secrets** - Vault generates short-lived DB credentials
3. **Format variety** - Limited to json/env/raw

### Where It Matters

**Templates matter for:**
- Complex applications with specific config file formats
- Legacy apps expecting specific property layouts
- Connection string construction
- Multi-environment deployments

**Templates don't matter for:**
- Modern apps that parse JSON
- Simple credential injection
- Apps using .env files

---

## Recommendations

### Short Term (v0.2.0)

Add basic Go template support:

```yaml
keeper.security/config: |
  secrets:
    - record: database-credentials
      path: /app/config/db.properties
      template: |
        db.user={{ .login }}
        db.password={{ .password }}
```

### Medium Term (v0.3.0)

Add template functions (base64enc, upper, lower, replace, trim).

### Long Term (v0.4.0)

Add conditional logic and advanced template features.

---

## Current Workarounds

Until templates are added, users can:

### Option 1: Use .env format + shell processing

```yaml
keeper.security/config: |
  secrets:
    - record: db-creds
      path: /secrets/db.env
      format: env
```

In app:
```bash
source /secrets/db.env
export DB_URL="postgresql://${LOGIN}:${PASSWORD}@postgres:5432/mydb"
```

### Option 2: Use JSON + jq

```yaml
keeper.security/secret: "db-creds"
```

In app:
```bash
DB_USER=$(cat /keeper/secrets/db-creds.json | jq -r '.login')
DB_PASS=$(cat /keeper/secrets/db-creds.json | jq -r '.password')
export DB_URL="postgresql://${DB_USER}:${DB_PASS}@postgres:5432/mydb"
```

### Option 3: Process in application code

Most modern apps (Python, Node.js, Go) have native JSON parsing - no workaround needed.

---

## Sources

- [Vault Agent Injector](https://developer.hashicorp.com/vault/docs/deploy/kubernetes/injector)
- [Vault Agent Templates](https://developer.hashicorp.com/vault/docs/deploy/kubernetes/injector/annotations)
- [External Secrets Operator Templating v2](https://external-secrets.io/latest/guides/templating/)
- [External Secrets Operator Templating v1](https://external-secrets.io/latest/guides/templating-v1/)
- [Secrets Store CSI Driver](https://secrets-store-csi-driver.sigs.k8s.io/)
