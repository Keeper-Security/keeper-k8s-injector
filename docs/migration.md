# Migration Guide

Step-by-step guides for migrating to Keeper K8s Injector from other secret management solutions.

## From External Secrets Operator (ESO)

### Overview

| Aspect | ESO | Keeper K8s Injector |
|--------|-----|---------------------|
| Architecture | Controller → K8s Secrets | Webhook → tmpfs files |
| Configuration | CRDs (ExternalSecret, SecretStore) | Pod annotations |
| Secret storage | etcd | tmpfs (RAM) |
| Rotation | Polling controller | Sidecar polling |

### Migration Steps

#### Step 1: Install Keeper Injector

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

#### Step 2: Create KSM Auth Secret

```bash
kubectl create secret generic keeper-auth \
  --from-literal=config='<your-ksm-base64-config>' \
  -n production
```

#### Step 3: Convert ExternalSecret to Annotations

**Before (ESO)**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: app-secrets
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: keeper-store
    kind: SecretStore
  target:
    name: app-secrets
  data:
    - secretKey: username
      remoteRef:
        key: database-credentials
        property: login
    - secretKey: password
      remoteRef:
        key: database-credentials
        property: password
---
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
    - name: app
      env:
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: username
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: password
```

**After (Keeper Injector)**:

Option 1 - File-based (recommended):
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-auth"
    keeper.security/secret: "database-credentials"
    keeper.security/refresh-interval: "1h"
spec:
  containers:
    - name: app
      # App reads /keeper/secrets/database-credentials.json
```

Option 2 - K8s Secret (similar to ESO):
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-auth"
    keeper.security/inject-as-k8s-secret: "true"
    keeper.security/k8s-secret-name: "app-secrets"
    keeper.security/secret: "database-credentials"
    keeper.security/k8s-secret-rotation: "true"
    keeper.security/refresh-interval: "1h"
spec:
  containers:
    - name: app
      env:
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: login
        - name: DB_PASS
          valueFrom:
            secretKeyRef:
              name: app-secrets
              key: password
```

#### Step 4: Remove ESO Resources

```bash
# Delete ExternalSecrets
kubectl delete externalsecret app-secrets

# Delete SecretStore
kubectl delete secretstore keeper-store

# Optionally uninstall ESO
helm uninstall external-secrets -n external-secrets-system
```

### Key Differences

| Feature | ESO | Keeper Injector |
|---------|-----|-----------------|
| Field extraction | `.data[].property` | Keeper notation or templates |
| Template support | `template.data` | Go templates in config |
| Multiple secrets | Separate ExternalSecret per app | Single annotation config |
| Debugging | Check ExternalSecret status | Check init/sidecar logs |

---

## From HashiCorp Vault Agent

### Overview

| Aspect | Vault Agent | Keeper K8s Injector |
|--------|-------------|---------------------|
| Architecture | Webhook → sidecar | Webhook → sidecar (same!) |
| Configuration | Annotations | Annotations (similar!) |
| Secret storage | tmpfs files | tmpfs files (same!) |
| Rotation | Sidecar polling | Sidecar polling (same!) |

**Migration is very similar!**

### Migration Steps

#### Step 1: Install Keeper Injector

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

#### Step 2: Create KSM Auth Secret

```bash
kubectl create secret generic keeper-auth \
  --from-literal=config='<your-ksm-base64-config>' \
  -n production
```

#### Step 3: Convert Vault Annotations to Keeper Annotations

**Before (Vault Agent)**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    vault.hashicorp.com/agent-inject: "true"
    vault.hashicorp.com/role: "myapp"
    vault.hashicorp.com/agent-inject-secret-db: "database/creds/myapp"
    vault.hashicorp.com/agent-inject-template-db: |
      {{- with secret "database/creds/myapp" -}}
      export DB_USER="{{ .Data.username }}"
      export DB_PASS="{{ .Data.password }}"
      {{- end -}}
spec:
  containers:
    - name: app
```

**After (Keeper Injector)**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-auth"
    keeper.security/config: |
      secrets:
        - record: database-credentials
          path: /keeper/secrets/db
          template: |
            export DB_USER="{{ .login }}"
            export DB_PASS="{{ .password }}"
spec:
  containers:
    - name: app
```

### Annotation Mapping

| Vault Annotation | Keeper Equivalent |
|------------------|-------------------|
| `vault.hashicorp.com/agent-inject: "true"` | `keeper.security/inject: "true"` |
| `vault.hashicorp.com/role: "myapp"` | `keeper.security/ksm-config: "keeper-auth"` |
| `vault.hashicorp.com/agent-inject-secret-db` | `keeper.security/secret` or `keeper.security/config` |
| `vault.hashicorp.com/agent-inject-template-db` | Use `template:` in config |
| `vault.hashicorp.com/agent-limits-cpu` | Set via Helm `sidecarResources.limits.cpu` |
| `vault.hashicorp.com/agent-requests-cpu` | Set via Helm `sidecarResources.requests.cpu` |

### Template Migration

Vault and Keeper both use Go templates, but field names differ:

**Vault**:
```
{{ .Data.username }}
{{ .Data.password }}
{{ .Data.data.nested_field }}
```

**Keeper**:
```
{{ .login }}
{{ .password }}
{{ .custom_field_name }}
```

Map Vault fields to Keeper fields in your records.

---

## From AWS Secrets CSI Driver

### Overview

| Aspect | AWS CSI Driver | Keeper K8s Injector |
|--------|----------------|---------------------|
| Architecture | CSI volume mount | Webhook → tmpfs mount |
| Configuration | SecretProviderClass | Pod annotations |
| Secret storage | tmpfs volume | tmpfs volume (same!) |
| Rotation | CSI driver polling | Sidecar polling |

### Migration Steps

#### Step 1: Install Keeper Injector

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

#### Step 2: Create KSM Auth Secret

Option 1 - K8s Secret:
```bash
kubectl create secret generic keeper-auth \
  --from-literal=config='<your-ksm-base64-config>' \
  -n production
```

Option 2 - AWS Secrets Manager (keep cloud-native):
```yaml
# ServiceAccount with IRSA
apiVersion: v1
kind: ServiceAccount
metadata:
  name: myapp
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789:role/keeper-access
---
# Pod annotation
annotations:
  keeper.security/auth-method: "aws-secrets-manager"
  keeper.security/aws-secret-id: "prod/keeper/ksm-config"
```

#### Step 3: Convert SecretProviderClass to Annotations

**Before (CSI Driver)**:
```yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: aws-secrets
spec:
  provider: aws
  parameters:
    objects: |
      - objectName: "database-credentials"
        objectType: "secretsmanager"
---
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  serviceAccountName: myapp
  volumes:
    - name: secrets
      csi:
        driver: secrets-store.csi.k8s.io
        readOnly: true
        volumeAttributes:
          secretProviderClass: "aws-secrets"
  containers:
    - name: app
      volumeMounts:
        - name: secrets
          mountPath: "/mnt/secrets"
          readOnly: true
```

**After (Keeper Injector)**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-method: "aws-secrets-manager"
    keeper.security/aws-secret-id: "prod/keeper/ksm-config"
    keeper.security/secret: "database-credentials"
spec:
  serviceAccountName: myapp  # Still needs IRSA
  containers:
    - name: app
      # Secrets at /keeper/secrets/ instead of /mnt/secrets/
```

#### Step 4: Update App Paths

Update file paths in application:
- Old: `/mnt/secrets/database-credentials`
- New: `/keeper/secrets/database-credentials.json`

---

## From 1Password

### Overview

| Aspect | 1Password | Keeper K8s Injector |
|--------|-----------|---------------------|
| Architecture | Webhook → sidecar | Webhook → sidecar (same!) |
| Configuration | Annotations | Annotations (similar!) |
| Secret storage | tmpfs files | tmpfs files (same!) |

Very similar migration to Vault Agent.

### Migration Steps

#### Step 1: Install Keeper Injector

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

#### Step 2: Create KSM Auth Secret

```bash
kubectl create secret generic keeper-auth \
  --from-literal=config='<your-ksm-base64-config>' \
  -n production
```

#### Step 3: Convert 1Password Annotations

**Before (1Password)**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    operator.1password.io/inject: "true"
    operator.1password.io/vault: "Production"
    operator.1password.io/item: "database-credentials"
spec:
  containers:
    - name: app
```

**After (Keeper Injector)**:
```yaml
apiVersion: v1
kind: Pod
metadata:
  annotations:
    keeper.security/inject: "true"
    keeper.security/ksm-config: "keeper-auth"
    keeper.security/secret: "database-credentials"
spec:
  containers:
    - name: app
```

---

## Migration Checklist

Use this checklist for any migration:

### Pre-Migration

- [ ] Install Keeper K8s Injector in test cluster
- [ ] Create KSM application in Keeper
- [ ] Test secret injection with sample pod
- [ ] Benchmark performance (latency, resource usage)
- [ ] Document field name mappings (if using templates)

### Migration

- [ ] Create KSM auth secrets in all namespaces
- [ ] Convert one deployment (canary)
- [ ] Verify secrets injected correctly
- [ ] Check sidecar logs for errors
- [ ] Test secret rotation
- [ ] Monitor for 24 hours

### Post-Migration

- [ ] Convert remaining deployments
- [ ] Remove old secret management system
- [ ] Update documentation
- [ ] Update CI/CD pipelines
- [ ] Train team on new annotations

---

## Common Migration Issues

### Issue: Field names don't match

**Problem**: Vault uses `.Data.username`, Keeper uses `.login`

**Solution**: Use templates to remap:
```yaml
template: |
  export USERNAME="{{ .login }}"
  export PASSWORD="{{ .password }}"
```

### Issue: Secrets in different format

**Problem**: ESO created K8s Secrets, app expects env vars

**Solution**: Use K8s Secret injection mode:
```yaml
annotations:
  keeper.security/inject-as-k8s-secret: "true"
  keeper.security/k8s-secret-name: "app-secrets"
```

### Issue: App expects /vault/secrets/ path

**Problem**: Vault Agent used `/vault/secrets/`, Keeper uses `/keeper/secrets/`

**Solution**: Use symlink in init container or update app paths:
```yaml
spec:
  initContainers:
    - name: symlink
      image: busybox
      command: ["sh", "-c", "ln -s /keeper/secrets /vault/secrets"]
      volumeMounts:
        - name: keeper-secrets
          mountPath: /keeper/secrets
```

### Issue: Multiple secret backends

**Problem**: Migrating from Vault (dynamic secrets) + ESO (static secrets)

**Solution**: Phased migration:
1. Migrate static secrets to Keeper first (ESO replacement)
2. Keep Vault for dynamic secrets (database credentials)
3. Gradually move to Keeper as backend supports rotation

---

**[← Back to Documentation Index](INDEX.md)**
