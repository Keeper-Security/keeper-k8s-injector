# Secret Rotation Dashboard

A live dashboard demonstrating automatic secret rotation without pod restarts.

**Time to complete: ~5 minutes**

## What This Demonstrates

- **Live secret rotation** without pod restarts
- Real-time visualization of secret changes
- History tracking of secret updates
- The power of sidecar-based injection

## Why This Matters

Most secrets management solutions require pod restarts to pick up new secrets. With Keeper K8s Injector:

- ✅ Secrets update automatically
- ✅ Zero downtime during rotation
- ✅ No deployment changes needed
- ✅ Configurable refresh intervals

## Prerequisites

- Keeper K8s Injector installed
- `keeper-credentials` secret created

**First time?** See [Example 01 - Hello Secrets](../01-hello-secrets/#complete-setup-from-zero) for complete installation instructions (Steps 1-2).

## Quick Start

### 1. Create a Secret in Keeper

In your Keeper vault:
1. Create a new record titled **"rotation-demo"**
2. Add a **password** field with any value (e.g., `initial-secret-123`)
3. Save the record

### 2. Deploy the Dashboard

```bash
kubectl apply -f rotation-dashboard.yaml
```

### 3. Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=rotation-dashboard --timeout=120s
```

### 4. Open the Dashboard

```bash
kubectl port-forward svc/rotation-dashboard 8080:80
```

Open http://localhost:8080 in your browser.

## The Demo

Once the dashboard is open:

1. **Observe** the current secret value displayed
2. **Go to Keeper** and change the password field
3. **Watch** the dashboard - within 15 seconds, you'll see:
   - The new value appear
   - The timestamp update
   - The old value added to history

The pod continues running without restart or redeployment.

## Screenshot

```
┌──────────────────────────────────────────────┐
│         🔐 Secret Rotation Dashboard          │
│                                              │
│  ┌────────────────────────────────────────┐  │
│  │     Current Secret Value               │  │
│  │                                        │  │
│  │     ██████ my-new-password ██████      │  │
│  │                                        │  │
│  │  Updated: 2024-01-15 14:32:15          │  │
│  │  Next Check: 12s                       │  │
│  └────────────────────────────────────────┘  │
│                                              │
│  📜 Change History                           │
│  ├─ 14:32:15  my-new-password               │
│  ├─ 14:31:45  old-password-456              │
│  └─ 14:31:00  initial-secret-123            │
│                                              │
└──────────────────────────────────────────────┘
```

## How It Works

```
                     Every 15 seconds
                           │
    ┌──────────────────────┼──────────────────────┐
    │                      ▼                      │
    │  ┌─────────────────────────────────────┐   │
    │  │          keeper-sidecar             │   │
    │  │                                     │   │
    │  │   1. Check Keeper for updates       │   │
    │  │   2. If changed, write new value    │   │
    │  │   3. Update file on tmpfs volume    │   │
    │  │                                     │   │
    │  └─────────────────────────────────────┘   │
    │                      │                      │
    │                      ▼                      │
    │  ┌─────────────────────────────────────┐   │
    │  │           tmpfs volume              │   │
    │  │    /keeper/secrets/rotation-demo    │   │
    │  └─────────────────────────────────────┘   │
    │                      │                      │
    │                      ▼                      │
    │  ┌─────────────────────────────────────┐   │
    │  │          dashboard app              │   │
    │  │                                     │   │
    │  │   Reads file, displays on web page  │   │
    │  │                                     │   │
    │  └─────────────────────────────────────┘   │
    │                                             │
    └─────────────────────────────────────────────┘
                         Pod
```

## Configuration

| Annotation | Value | Description |
|------------|-------|-------------|
| `keeper.security/refresh-interval` | `15s` | Check for updates every 15 seconds |
| `keeper.security/secret` | `rotation-demo` | The Keeper record title |

You can adjust the refresh interval:
- `15s` - Great for demos
- `60s` - Reasonable for production
- `5m` - For infrequently changed secrets

## Common Questions

### How fast can rotation be?

Configurable to any interval. Set `keeper.security/refresh-interval: "5s"` for near-instant updates. Consider API rate limits for very short intervals.

### Does this work with all secret types?

Yes! This demo uses a password field, but rotation works with:
- API keys
- Database credentials
- TLS certificates
- Any Keeper record type

### What if Keeper is unavailable?

The sidecar keeps the last known good value. Your application continues running with cached secrets until Keeper is reachable again.

## Cleanup

```bash
kubectl delete -f rotation-dashboard.yaml
```

## Next Steps

- [Hello Secrets](../01-hello-secrets/) - Simpler getting started example
- [PostgreSQL Example](../02-database-postgres/) - Database credentials with rotation
