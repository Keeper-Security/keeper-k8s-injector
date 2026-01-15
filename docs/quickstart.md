# Quick Start

Get secrets injected into your pods in 5 minutes.

## Prerequisites

- Kubernetes cluster (1.25+)
- `kubectl` configured
- cert-manager installed ([install guide](https://cert-manager.io/docs/installation/))
- Keeper Secrets Manager application configured

## Step 1: Install the Injector

```bash
kubectl apply -f https://keeper.security/k8s/injector.yaml
```

This creates:
- `keeper-system` namespace
- Webhook deployment with 2 replicas
- TLS certificates (via cert-manager)
- RBAC resources

Verify installation:
```bash
kubectl get pods -n keeper-system
```

## Step 2: Create Auth Secret

Export your KSM configuration and create a Kubernetes secret:

```bash
# From Keeper Secrets Manager, download your config.json
# Then create the secret:
kubectl create secret generic keeper-auth \
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
    keeper.security/auth-secret: "keeper-auth"
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
- Automatically refreshed every 5 minutes

## Next Steps

- Add multiple secrets: `keeper.security/secrets: "secret1, secret2, secret3"`
- Custom paths: `keeper.security/secret-myapp: "/app/config/secrets.json"`
- Adjust refresh interval: `keeper.security/refresh-interval: "10m"`

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
        keeper.security/auth-secret: "keeper-auth"
        keeper.security/secrets: "database-creds, api-keys"
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
