# Keeper K8s Injector Roadmap

## Current Version: 0.1.3

---

## v0.2.0 - Template Support (Q1 2026)

### Goal
Match industry standard (Vault Agent, ESO) with Go template support for secret formatting.

### Features

#### 1. Go Template Support

Allow users to transform secrets using Go templates:

```yaml
annotations:
  keeper.security/config: |
    secrets:
      - record: postgres-credentials
        path: /app/config/db.properties
        template: |
          db.url=jdbc:postgresql://postgres:5432/mydb
          db.username={{ .login }}
          db.password={{ .password }}
```

**Use cases:**
- Build connection strings
- Generate config files (.properties, .ini, etc.)
- Custom format conversion
- Field remapping

#### 2. Basic Template Functions

| Function | Purpose | Example |
|----------|---------|---------|
| `base64enc` | Base64 encode | `{{ .password | base64enc }}` |
| `base64dec` | Base64 decode | `{{ .token | base64dec }}` |
| `upper` | Uppercase | `{{ .username | upper }}` |
| `lower` | Lowercase | `{{ .dbname | lower }}` |
| `trim` | Trim whitespace | `{{ .value | trim }}` |

#### 3. Additional Output Formats

- ✅ JSON (already implemented)
- ✅ .env (already implemented)
- ✅ raw (already implemented)
- 🆕 .properties (Java properties format)
- 🆕 YAML
- 🆕 INI

### Implementation

- Add `text/template` package
- Parse template in sidecar before writing
- Provide Keeper record data as template context
- Support both simple `format:` and `template:` fields

---

## v0.3.0 - Enhanced Templates (Q2 2026)

### Features

#### 1. Advanced Template Functions

- String manipulation: `replace`, `split`, `join`
- Crypto: `sha256sum`, `sha512sum`, `md5sum`
- Encoding: `urlEncode`, `htmlEscape`, `jsonEscape`
- Logic: `if`, `else`, `range`, `with`

#### 2. Multi-Secret Templates

Build config files from multiple Keeper records:

```yaml
template: |
  [database]
  host={{ index .secrets "db-config" "hostname" }}
  user={{ index .secrets "db-creds" "login" }}
  pass={{ index .secrets "db-creds" "password" }}

  [redis]
  url={{ index .secrets "redis-creds" "url" }}
```

#### 3. Environment-Based Templates

```yaml
template: |
  {{- if eq .env "prod" -}}
  DB_HOST=prod-db.example.com
  {{- else -}}
  DB_HOST=staging-db.example.com
  {{- end -}}
```

---

## v0.4.0 - Enterprise Features (Q3 2026)

### Features

#### 1. Template from ConfigMap

Reference external template files:

```yaml
keeper.security/config: |
  secrets:
    - record: db-creds
      path: /app/config/database.properties
      templateConfigMap: app-templates
      templateKey: database.properties.tmpl
```

#### 2. Sprig Functions

Add Sprig template function library (used by Helm):
- Date/time functions
- Crypto functions
- Advanced string manipulation
- Network functions

#### 3. Secret Validation

Validate secrets before writing:

```yaml
keeper.security/config: |
  secrets:
    - record: tls-cert
      path: /app/certs/server.crt
      validate: x509-cert
    - record: tls-key
      path: /app/certs/server.key
      validate: rsa-private-key
```

---

## v0.5.0 - OIDC Authentication (Q4 2026)

### Features

#### 1. Kubernetes ServiceAccount OIDC

Authenticate to Keeper using K8s OIDC tokens:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-method: "oidc"
  keeper.security/service-account: "my-app-sa"
```

**Requires:** KSM backend support for OIDC token exchange

#### 2. Workload Identity Integration

- GKE Workload Identity
- EKS IRSA
- AKS Managed Identity

---

## v1.0.0 - Production Hardening (2027)

### Features

1. **Comprehensive metrics** - Detailed Prometheus metrics
2. **Audit logging** - Track all secret access
3. **Performance optimization** - Reduced memory footprint
4. **Webhook optimization** - Faster pod mutation
5. **E2E test suite** - Complete integration tests
6. **Security hardening** - Third-party security audit

---

## Feature Requests

Track feature requests at: https://github.com/Keeper-Security/keeper-k8s-injector/issues

## Comparison with Competitors

See [docs/comparison.md](docs/comparison.md) for detailed feature comparison with Vault Agent, ESO, and others.
