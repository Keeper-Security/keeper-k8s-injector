# Quick Start

Get secrets injected into your pods in 5 minutes.

## Prerequisites

- Kubernetes cluster (1.21+, tested with 1.21-1.34)
- `kubectl` configured
- Keeper Secrets Manager application configured

> **Note**: TLS certificates are auto-generated. cert-manager is optional.

## Step 1: Install the Injector

### Option 1: Helm (OCI Registry) - Recommended

```bash
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

### Option 2: Helm (Repository)

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm repo update
helm upgrade --install keeper-injector keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

### Option 3: kubectl (Direct YAML)

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

Verify installation:

```bash
kubectl get pods -n keeper-security
```

You should see the webhook pods running:
```
NAME                                READY   STATUS    RESTARTS   AGE
keeper-injector-webhook-xxxxx-xxx   1/1     Running   0          30s
keeper-injector-webhook-xxxxx-yyy   1/1     Running   0          30s
```

## Step 2: Create Auth Secret

Create a Kubernetes secret with your KSM configuration:

**Option 1: Base64 Config (Recommended)**

From Keeper Secrets Manager:
1. Navigate to: Vault → Secrets Manager → Select your Application
2. Go to Devices tab → Add Device
3. Select "Configuration File" method and "Base64" type
4. Copy the base64 string

```bash
kubectl create secret generic keeper-credentials \
  --from-literal=config='<paste-base64-config-here>' \
  -n default
```

**Option 2: Config File**

If you downloaded a JSON config file:

```bash
kubectl create secret generic keeper-credentials \
  --from-file=config=ksm-config.json \
  -n default
```

## Step 3: Test Secret Injection

### Quick Test (No Files Needed)

Run this one-liner to verify secrets are injected:

```bash
kubectl run test-secrets --image=busybox:latest --restart=Never \
  --annotations="keeper.security/inject=true,keeper.security/auth-secret=keeper-credentials,keeper.security/secret=my-database-credentials" \
  -- sh -c "cat /keeper/secrets/my-database-credentials.json && sleep 3600"

# Check the logs to see your secret content
kubectl logs test-secrets

# Or exec into the pod
kubectl exec test-secrets -- cat /keeper/secrets/my-database-credentials.json

# Cleanup
kubectl delete pod test-secrets
```

### Full Example with Visual Confirmation

For a nicer demo with browser display, create this test pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-secrets
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-credentials"
    keeper.security/secret: "my-database-credentials"  # ← Use any secret title from your KSM
spec:
  containers:
    - name: nginx
      image: nginx:alpine
      command: ["/bin/sh", "-c"]
      args:
        - |
          echo '<h1>Secret Injected Successfully!</h1><pre>' > /usr/share/nginx/html/index.html
          cat /keeper/secrets/my-database-credentials.json >> /usr/share/nginx/html/index.html
          echo '</pre>' >> /usr/share/nginx/html/index.html
          nginx -g 'daemon off;'
```

## Step 4: Deploy and Verify

```bash
# Save the YAML above as test-pod.yaml and apply it
kubectl apply -f test-pod.yaml

# Wait for pod to be ready
kubectl wait --for=condition=Ready pod/test-secrets --timeout=60s

# Option 1: View in browser
kubectl port-forward pod/test-secrets 8080:80
# Open http://localhost:8080 to see your secret!

# Option 2: Check via command line
kubectl exec test-secrets -- cat /keeper/secrets/my-database-credentials.json

# Cleanup when done
kubectl delete pod test-secrets
```

## Summary

Your secrets are now:
- Fetched from Keeper Secrets Manager
- Written to `/keeper/secrets/` inside your pod
- Stored in memory (tmpfs), not on disk
- Automatically refreshed (configurable interval)

## Try the Examples

See working examples in the repo:

```bash
git clone https://github.com/Keeper-Security/keeper-k8s-injector.git
cd keeper-k8s-injector/examples

# Hello Secrets - 5 minute demo
kubectl apply -f 01-hello-secrets/
kubectl port-forward svc/hello-secrets 8080:80
# Open http://localhost:8080
```

## Next Steps

- Add multiple secrets: `keeper.security/secrets: "secret1, secret2, secret3"`
- Custom paths: `keeper.security/secret-myapp: "/app/config/secrets.json"`
- Adjust refresh interval: `keeper.security/refresh-interval: "5m"`

See [Annotations Reference](annotations.md) for all options.

## Example: Complete Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
      annotations:
        keeper.security/inject: "true"
        keeper.security/auth-secret: "keeper-credentials"
        keeper.security/secrets: "database-creds, api-keys"
        keeper.security/refresh-interval: "5m"
    spec:
      containers:
        - name: app
          image: my-app:latest
          env:
            - name: DB_CONFIG_PATH
              value: /keeper/secrets/database-creds.json
            - name: API_KEYS_PATH
              value: /keeper/secrets/api-keys.json
```

## Where to Find Charts & Images

- **Helm Chart (OCI)**: `oci://registry-1.docker.io/keeper/keeper-injector`
- **Helm Chart (HTTP)**: https://keeper-security.github.io/keeper-k8s-injector
- **ArtifactHub**: https://artifacthub.io/packages/helm/keeper-injector/keeper-injector
- **Docker Images**:
  - `keeper/injector-webhook` on [Docker Hub](https://hub.docker.com/r/keeper/injector-webhook)
  - `keeper/injector-sidecar` on [Docker Hub](https://hub.docker.com/r/keeper/injector-sidecar)

---

**[← Back to Documentation Index](INDEX.md)**
