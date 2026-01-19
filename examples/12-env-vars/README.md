# Environment Variable Injection

Demonstrates injecting secrets directly as environment variables instead of files. Useful for legacy applications that only support env vars.

**Time to complete: ~5 minutes**

## ⚠️ Security Notice

**Environment variables are less secure than file-based injection:**
- Visible in `kubectl describe pod` output
- Visible in process listings inside containers
- May be captured in logs or debugging output
- Cannot be rotated without pod restart

**For production, prefer file-based injection (see example 01-hello-secrets).**

## What This Demonstrates

- Injecting secrets as environment variables
- Optional env var prefixes (e.g., `DB_LOGIN`, `DB_PASSWORD`)
- Security trade-offs and when to use each method
- Mixed mode (some secrets as files, some as env vars)

## Prerequisites

- Kubernetes cluster (minikube, kind, EKS, GKE, AKS, or any K8s 1.21+)
- kubectl configured
- Keeper Secrets Manager account
- Keeper K8s Injector installed (see Setup below)

## Complete Setup (From Zero)

### Step 1: Install Keeper K8s Injector

**Why:** This installs the webhook that injects secrets into your pods.

**Option A: Helm (recommended)**

```bash
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

**Option B: kubectl**

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

Verify installation:
```bash
kubectl get pods -n keeper-security

# Expected output:
# NAME                                      READY   STATUS    RESTARTS   AGE
# keeper-injector-webhook-xxxxx-xxx         1/1     Running   0          30s
# keeper-injector-webhook-xxxxx-yyy         1/1     Running   0          30s
```

### Step 2: Create KSM Auth Secret

Get your KSM config from Keeper:
1. Log into Keeper Vault
2. Navigate to: Vault → Secrets Manager → Select your Application
3. Go to Devices tab → Add Device
4. Select "Configuration File" method and "Base64" type
5. Copy the base64 string

```bash
kubectl create secret generic keeper-credentials \
  --from-literal=config='<paste-base64-config-here>'
```

**Important:** Don't wrap the config in quotes - use it exactly as copied from Keeper.

### Step 3: Create a Test Secret in Keeper

Create a secret in Keeper with some fields:
- **Title**: `demo-secret` (or update the YAML with your secret's title)
- **Fields**: Add some custom fields like:
  - `login`: `myuser`
  - `password`: `mypassword`
  - `hostname`: `db.example.com`

### Step 4: Deploy the Example

```bash
# If your secret has a different title, edit the YAML first:
vi env-vars.yaml
# Change keeper.security/secret to match your secret's title

kubectl apply -f env-vars.yaml
```

### Step 5: Access the Demo

```bash
# Port forward to the service
kubectl port-forward svc/env-vars-demo 8080:80

# Open in browser
open http://localhost:8080
```

You should see a web page displaying your secret's fields as environment variables with the `APP_` prefix.

## Verify Environment Variables

Check that environment variables are set in the pod:

```bash
# List all APP_* environment variables
kubectl exec deployment/env-vars-demo -- env | grep APP_

# Expected output:
# APP_LOGIN=myuser
# APP_PASSWORD=mypassword
# APP_HOSTNAME=db.example.com
```

## Configuration Options

### Simple Usage (All Secrets as Env Vars)

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/inject-env-vars: "true"
  keeper.security/secret: "database-credentials"
```

**Result**: Environment variables like `LOGIN`, `PASSWORD`, `HOSTNAME`

### With Prefix

```yaml
annotations:
  keeper.security/inject-env-vars: "true"
  keeper.security/env-prefix: "DB_"
  keeper.security/secret: "database-credentials"
```

**Result**: Environment variables like `DB_LOGIN`, `DB_PASSWORD`, `DB_HOSTNAME`

### Mixed Mode (Advanced)

Some secrets as files, some as env vars:

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

**Result**:
- Env vars: `DB_LOGIN`, `DB_PASSWORD`, `DB_HOSTNAME`
- File: `/keeper/secrets/tls.json`

## When to Use Environment Variables

**Good use cases:**
- Legacy applications that only support env vars
- Simple read-once patterns (not frequently rotated secrets)
- Development/testing environments
- 12-factor apps designed for env var configuration

**Bad use cases:**
- Production environments with sensitive data
- Secrets that rotate frequently
- Compliance requirements (SOC2, PCI-DSS)
- Long-lived credentials

## When to Use Files (Recommended)

**Advantages:**
- ✅ Not visible in pod metadata or process listings
- ✅ Can be rotated without pod restart (via sidecar)
- ✅ More secure for sensitive data
- ✅ tmpfs storage prevents disk persistence

**Use files for:**
- Production environments
- Database passwords, API keys
- Frequently rotated secrets
- Compliance requirements

## Important Limitations

1. **No Rotation**: Environment variables are set at pod creation time and cannot be changed without restarting the pod. That's why this example uses `init-only: "true"`.

2. **Visibility**: Environment variables are visible in:
   - `kubectl describe pod` output
   - Container process listings (`ps auxe`)
   - Application logs (if not careful)
   - Core dumps and crash reports

3. **Not in etcd**: Unlike using K8s Secrets with `envFrom`, the Keeper Injector fetches secrets at pod creation time and never stores them in etcd.

## Troubleshooting

### Environment variables not set

Check the init container logs:
```bash
kubectl logs deployment/env-vars-demo -c keeper-secrets-init

# Look for errors like:
# - Failed to fetch secret
# - Invalid annotation configuration
# - Authentication errors
```

### Can't access the web page

```bash
# Check if pod is running
kubectl get pods -l app=env-vars-demo

# Check pod events
kubectl describe pod -l app=env-vars-demo

# If init container failed, it means the secret couldn't be fetched
```

### Want to update the secret

Environment variables cannot be rotated without pod restart:

```bash
# Update the secret in Keeper first, then restart:
kubectl rollout restart deployment/env-vars-demo
```

## Cleanup

```bash
kubectl delete -f env-vars.yaml
```

To completely remove Keeper Injector:
```bash
helm uninstall keeper-injector -n keeper-security
kubectl delete namespace keeper-security
```

## Next Steps

- **Example 01 (Hello Secrets)**: File-based injection with automatic rotation
- **Example 02 (Database Postgres)**: Real database connection with secrets
- **Example 06 (Rotation Dashboard)**: Visualize secret rotation in real-time
- **Documentation**: Read [features.md](../../docs/features.md) for all injection methods

## Security Best Practices

1. **Prefer files over env vars** for production
2. **Use env var prefixes** to avoid naming conflicts (`DB_`, `API_`, etc.)
3. **Use init-only mode** with env vars (rotation doesn't work anyway)
4. **Never log env vars** in your application code
5. **Consider mixed mode** - use env vars only for non-sensitive config

## Learn More

- [Annotations Reference](../../docs/annotations.md)
- [Feature Guide](../../docs/features.md)
- [Security Model](../../docs/security.md)
- [Production Guide](../../docs/production.md)
