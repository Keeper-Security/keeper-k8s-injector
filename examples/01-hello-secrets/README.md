# Hello Secrets

The simplest possible demo of Keeper K8s Injector. A web page that displays your secret value and updates automatically when you change it in Keeper.

**Time to complete: ~5 minutes**

## What This Demonstrates

- Basic secret injection using annotations
- Automatic secret rotation without pod restarts
- Visual confirmation that secrets are loaded

## Prerequisites

- Kubernetes cluster (minikube, kind, EKS, GKE, AKS, or any K8s 1.21+)
- kubectl configured
- Keeper Secrets Manager account

## Complete Setup (From Zero)

### Step 1: Install cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.1/cert-manager.yaml

# Wait for ready
kubectl wait --for=condition=ready pod -l app.kubernetes.io/instance=cert-manager -n cert-manager --timeout=120s
```

### Step 2: Install Keeper K8s Injector

**Option A: Helm (recommended)**

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-system \
  --create-namespace
```

**Option B: kubectl**

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

Verify installation:
```bash
kubectl get pods -n keeper-system
```

### Step 3: Create KSM Auth Secret

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

### Step 4: Create a Secret in Keeper

In your Keeper vault:
1. Create a new record
2. Set the title to exactly: **demo-secret**
3. Add a password field with any value (e.g., "hello-world-123")
4. Save the record

**Important:** The title must be exactly "demo-secret" for this tutorial.

### Step 5: Deploy the Example

```bash
kubectl apply -f https://raw.githubusercontent.com/Keeper-Security/keeper-k8s-injector/main/examples/01-hello-secrets/deployment.yaml
kubectl apply -f https://raw.githubusercontent.com/Keeper-Security/keeper-k8s-injector/main/examples/01-hello-secrets/service.yaml
```

### Step 6: Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=hello-secrets --timeout=120s
```

### Step 7: View the Demo

```bash
kubectl port-forward svc/hello-secrets 8080:80
```

Open http://localhost:8080 in your browser.

You should see your secret displayed on the page.

## Try Secret Rotation

1. Go to Keeper and modify your secret (change the password, add a field, etc.)
2. Wait ~30 seconds (the configured refresh interval)
3. Refresh the page to see the updated value

The pod never restarts. The sidecar container fetches the new value and writes it to the shared volume.

## How It Works

```
┌─────────────────────────────────────────────────────┐
│                      Pod                             │
│  ┌──────────────┐         ┌──────────────────────┐  │
│  │    nginx     │         │   keeper-sidecar     │  │
│  │  (your app)  │         │   (injected)         │  │
│  │              │         │                      │  │
│  │  Reads from  │◄────────│  Writes secrets      │  │
│  │  /keeper/    │         │  from Keeper         │  │
│  │  secrets/    │         │  every 30s           │  │
│  └──────────────┘         └──────────────────────┘  │
│         ▲                          │                │
│         └──────────────────────────┘                │
│                  tmpfs volume                       │
│              (memory-backed tmpfs)                  │
└─────────────────────────────────────────────────────┘
```

## Configuration

Edit `deployment.yaml` to customize:

| Annotation | Description | Default |
|------------|-------------|---------|
| `keeper.security/secret` | Title of your Keeper record | `demo-secret` |
| `keeper.security/auth-secret` | K8s secret with KSM config | `keeper-credentials` |
| `keeper.security/refresh-interval` | How often to check for updates | `30s` |

## Cleanup

```bash
kubectl delete -f .
```

## Troubleshooting

### Page shows "Secret not loaded yet"

1. Check the sidecar logs:
   ```bash
   kubectl logs deployment/hello-secrets -c keeper-sidecar
   ```

2. Verify your auth secret exists:
   ```bash
   kubectl get secret keeper-credentials
   ```

3. Check the record title matches the annotation

### Pod won't start

1. Check pod events:
   ```bash
   kubectl describe pod -l app=hello-secrets
   ```

2. Ensure the injector webhook is running:
   ```bash
   kubectl get pods -n keeper-system
   ```

## Next Steps

- [Database Connection Example](../02-database-postgres/) - Real PostgreSQL credentials
- [Rotation Dashboard](../06-rotation-dashboard/) - Live rotation visualization
