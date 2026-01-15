# Quick Start

Get secrets injected into your pods in 5 minutes.

## Prerequisites

- Kubernetes cluster (1.25+)
- `kubectl` configured
- Keeper Secrets Manager application configured

## Step 1: Install the Injector

### Option 1: Helm (OCI Registry) - Recommended

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-system \
  --create-namespace
```

### Option 2: Helm (Repository)

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm repo update
helm install keeper-injector keeper/keeper-injector \
  --namespace keeper-system \
  --create-namespace
```

### Option 3: kubectl (Direct YAML)

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

Verify installation:

```bash
kubectl get pods -n keeper-system
```

You should see the webhook pods running:
```
NAME                                READY   STATUS    RESTARTS   AGE
keeper-injector-webhook-xxxxx-xxx   1/1     Running   0          30s
keeper-injector-webhook-xxxxx-yyy   1/1     Running   0          30s
```

## Step 2: Create Auth Secret

Export your KSM configuration and create a Kubernetes secret:

```bash
# From Keeper Secrets Manager, download your config.json
# Then create the secret:
kubectl create secret generic keeper-credentials \
  --from-file=config=ksm-config.json \
  -n default
```

## Step 3: Annotate Your Pod

Add two annotations to your pod:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-credentials"
    keeper.security/secret: "my-database-credentials"  # Your secret title in KSM
spec:
  containers:
    - name: app
      image: your-app:latest
```

## Step 4: Deploy and Verify

```bash
kubectl apply -f my-pod.yaml

# Check that secrets were injected
kubectl exec my-app -- cat /keeper/secrets/my-database-credentials.json
```

## That's It!

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
