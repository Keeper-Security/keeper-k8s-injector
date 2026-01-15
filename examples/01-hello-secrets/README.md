# Hello Secrets

The simplest possible demo of Keeper K8s Injector. A web page that displays your secret value and updates automatically when you change it in Keeper.

**Time to complete: ~5 minutes**

## What This Demonstrates

- Basic secret injection using annotations
- Automatic secret rotation (no pod restart!)
- Visual confirmation that secrets are loaded

## Prerequisites

1. Keeper K8s Injector installed in your cluster
2. A Keeper Secrets Manager application with a config file

## Quick Start

### 1. Create Your KSM Auth Secret

```bash
# If you haven't already, create the auth secret from your KSM config
kubectl create secret generic keeper-credentials \
  --from-file=config=path/to/your/ksm-config.json
```

### 2. Create a Secret in Keeper

In your Keeper vault:
1. Create a new record titled **"demo-secret"** (or update `deployment.yaml`)
2. Add any fields you want (password, notes, custom fields)
3. Save the record

### 3. Deploy the Example

```bash
kubectl apply -f .
```

### 4. Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=hello-secrets --timeout=120s
```

### 5. View the Demo

```bash
kubectl port-forward svc/hello-secrets 8080:80
```

Open http://localhost:8080 in your browser.

You should see your secret displayed on the page!

## Try Secret Rotation

This is the magic part:

1. Go to Keeper and **modify your secret** (change the password, add a field, etc.)
2. Wait ~30 seconds (the configured refresh interval)
3. **Watch the page update automatically!**

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
│              (memory only - secure!)                │
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
