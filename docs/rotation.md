# Secret Rotation

Automatic secret rotation syncs changes from Keeper Secrets Manager to your pods without requiring pod restarts.

**Important**: This feature detects when secrets are **updated in Keeper** (for example, when you manually rotate a database password in Keeper) and automatically updates the injected files in your pods. The injector does not actively rotate credentials in target systems (like databases)—it only reflects changes you make in Keeper.

## How It Works

```
┌─────────────────────────────────────────────────────┐
│                    Your Pod                          │
│                                                      │
│  ┌──────────────┐        ┌──────────────────────┐  │
│  │   Sidecar    │───────▶│  /keeper/secrets/    │  │
│  │  Container   │ Updates│  (tmpfs volume)      │  │
│  │              │ Every  │                      │  │
│  │  Polls KSM   │ 5min   │  db-creds.json       │  │
│  │  for changes │        │  api-keys.json       │  │
│  └──────────────┘        └──────────┬───────────┘  │
│                                     │              │
│  ┌──────────────────────────────────▼───────────┐  │
│  │         App Container                        │  │
│  │         Reads updated secrets                │  │
│  └──────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

---

## Automatic Rotation

### Enable Rotation (Default)

By default, a sidecar container runs alongside your app and periodically checks for secret updates:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-credentials"
  keeper.security/secret: "database-credentials"
  keeper.security/refresh-interval: "5m"  # Check every 5 minutes
```

**What happens:**
1. Sidecar checks Keeper every 5 minutes
2. If secrets changed, sidecar rewrites files in `/keeper/secrets/`
3. App can re-read files to get new secrets
4. **No pod restart required**

### Configurable Intervals

| Interval | Use Case |
|----------|----------|
| `1m` | Frequently rotating credentials (dev/test) |
| `5m` | Standard rotation (recommended for production) |
| `15m` | Low-churn secrets |
| `1h` | Rarely changing configuration |

**Example:**
```yaml
keeper.security/refresh-interval: "10m"
```

**Note**: Shorter intervals increase API calls to Keeper. Balance freshness with load.

---

## Signal on Update

Notify your application when secrets change by sending a signal:

```yaml
annotations:
  keeper.security/signal: "SIGHUP"
```

**What happens:**
1. Sidecar detects secret changed
2. Rewrites files in `/keeper/secrets/`
3. Sends `SIGHUP` to app container
4. App handles signal to reload configuration

### Supported Signals

| Signal | Common Use |
|--------|------------|
| `SIGHUP` | Reload configuration (most common) |
| `SIGUSR1` | Custom reload logic |
| `SIGUSR2` | Custom reload logic |

### Application Requirements

Your app must handle the signal. Example in Go:

```go
package main

import (
    "os"
    "os/signal"
    "syscall"
)

func main() {
    // Setup signal handling
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGHUP)

    go func() {
        for {
            <-sigs
            // Reload secrets
            loadSecrets()
        }
    }()

    // Rest of application...
}
```

**Without signal handling**: App must periodically re-read secret files to detect changes.

---

## Init-Only Mode (No Rotation)

Disable the sidecar and fetch secrets only once at pod startup:

```yaml
annotations:
  keeper.security/init-only: "true"
```

**What happens:**
1. Init container fetches secrets at startup
2. Writes to `/keeper/secrets/`
3. **No sidecar created**
4. Secrets never updated (static for pod lifetime)

### When to Use Init-Only

✅ **Use for:**
- Static configuration that doesn't change
- Reducing resource usage (no sidecar overhead)
- Compliance requirements to freeze secrets at startup
- Development environments

❌ **Avoid for:**
- Production secrets that rotate
- Secrets with expiration dates
- Security policies requiring regular rotation

### Resource Savings

```yaml
# With sidecar (default)
Containers: init + sidecar + app = 3 containers
Memory: ~64 Mi (sidecar overhead)

# Init-only mode
Containers: init + app = 2 containers
Memory: 0 Mi (no sidecar)
```

---

## Resilience Features

### Retry with Exponential Backoff

Automatic retry on transient failures (network issues, temporary API downtime):

**Behavior:**
- 3 attempts
- Delays: 200ms, 400ms, 800ms (exponential)
- Respects context cancellation

**Example log output:**
```
level=warn msg="failed to fetch secrets, retrying" attempt=1 error="connection timeout"
level=warn msg="failed to fetch secrets, retrying" attempt=2 error="connection timeout"
level=info msg="successfully fetched secrets" attempt=3
```

No configuration needed - automatic.

### In-Memory Secret Caching

Secrets cached after successful fetch:

**Behavior:**
- 24-hour maximum age
- Thread-safe concurrent access
- Cleared on pod restart
- Memory-only (no disk persistence)

**Why caching matters:**
- Faster startup (no API call if cache valid)
- Resilience during Keeper API outages
- Reduced API load

### Cache Fallback

When Keeper API is unavailable after retry:

```yaml
keeper.security/fail-on-error: "true"   # Fail if no cache (default)
# OR
keeper.security/fail-on-error: "false"  # Use cache, or start without secrets
```

**Behavior with `fail-on-error: true` (default):**
- Keeper up → Fetch and cache
- Keeper down, cache exists → Use cached value (warn in logs)
- Keeper down, no cache → **Pod fails to start**

**Behavior with `fail-on-error: false`:**
- Keeper up → Fetch and cache
- Keeper down, cache exists → Use cached value (warn in logs)
- Keeper down, no cache → **Pod starts without secrets** (error in logs)

### When to Use fail-on-error: false

✅ **Use for:**
- Non-critical applications
- Development environments
- Apps with graceful degradation

❌ **Avoid for:**
- Production databases requiring credentials
- Security-critical applications
- Compliance environments

**Example:**
```yaml
annotations:
  keeper.security/fail-on-error: "false"
```

**Warning log when using cache fallback:**
```
level=warn msg="using cached secrets" age="6h" reason="keeper API unavailable"
```

---

## Rotation with K8s Secrets

If using [Kubernetes Secret injection](injection-modes.md#method-3-kubernetes-secrets), enable rotation:

```yaml
annotations:
  keeper.security/inject-as-k8s-secret: "true"
  keeper.security/k8s-secret-name: "app-secrets"
  keeper.security/k8s-secret-rotation: "true"
  keeper.security/refresh-interval: "5m"
```

**What happens:**
1. Sidecar checks Keeper every 5 minutes
2. If secrets changed, sidecar updates K8s Secret
3. Apps using the Secret must watch for changes or use [Reloader](https://github.com/stakater/Reloader)

**Limitation**: Most apps don't auto-reload when K8s Secrets change. You need:
- App-level Secret watching (e.g., Kubernetes client)
- External tool like Reloader to restart pods
- File-based injection (recommended)

---

## Rotation Best Practices

### 1. Match Rotation to Secret Lifecycle

```yaml
# Database password (rotates daily)
keeper.security/refresh-interval: "5m"

# TLS certificate (rotates monthly)
keeper.security/refresh-interval: "1h"

# Static config (never rotates)
keeper.security/init-only: "true"
```

### 2. Implement Signal Handling

Don't rely on app to poll for changes - use signals:

```yaml
keeper.security/signal: "SIGHUP"
```

**Benefits:**
- Instant reload (no polling delay)
- Lower resource usage
- Deterministic behavior

### 3. Test Rotation in Dev

Verify your app handles rotation correctly:

```bash
# Force refresh by restarting sidecar
kubectl exec my-pod -c keeper-secrets-sidecar -- kill 1

# Check if app reloaded
kubectl logs my-pod -c app
```

### 4. Monitor Sidecar Logs

Watch for rotation events:

```bash
kubectl logs my-pod -c keeper-secrets-sidecar -f | grep "secrets updated"
```

Expected output:
```
level=info msg="secrets updated" count=3 duration="142ms"
level=info msg="sent signal to app" signal="SIGHUP"
```

### 5. Set Appropriate Resource Limits

Sidecar has low overhead, but allocate resources:

```yaml
# Helm values
sidecarResources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 20m
    memory: 64Mi
```

---

## Troubleshooting Rotation

### Secrets not updating

**Check sidecar is running:**
```bash
kubectl get pod my-pod -o jsonpath='{.spec.containers[*].name}'
# Expected: keeper-secrets-sidecar
```

**Check sidecar logs:**
```bash
kubectl logs my-pod -c keeper-secrets-sidecar
```

**Common issues:**
- `init-only: "true"` set (no sidecar)
- Sidecar crashed (check logs)
- Refresh interval too long

### App not reloading

**Verify signal configured:**
```bash
kubectl get pod my-pod -o yaml | grep keeper.security/signal
```

**Check app handles signal:**
- Add signal handler to app code
- Test with `kill -HUP <pid>`

**Alternative**: Poll for file changes in app:
```go
// Check file modification time
stat, _ := os.Stat("/keeper/secrets/db.json")
if stat.ModTime().After(lastLoaded) {
    loadSecrets()
}
```

### High API load

**Symptom**: Too many API calls to Keeper

**Solution**: Increase refresh interval:
```yaml
keeper.security/refresh-interval: "15m"  # Reduce frequency
```

**Formula**:
```
API calls/hour = (60 minutes / refresh interval) * num pods
```

**Example**:
- 100 pods × refresh every 5min = 1,200 calls/hour
- 100 pods × refresh every 15min = 400 calls/hour

---

## Comparison: Rotation Methods

| Method | Pod Restart Required | App Code Changes | Latency |
|--------|---------------------|------------------|---------|
| **File rotation + signal** | ❌ No | Signal handler | Instant |
| **File rotation + polling** | ❌ No | File watch logic | Seconds |
| **K8s Secret rotation** | ✅ Yes* | None | Minutes |
| **No rotation (init-only)** | ✅ Yes | None | N/A |

*Unless using Reloader or similar tool

**Recommendation**: Use file-based rotation with signals for zero-downtime updates.

---

**[← Back to Documentation Index](INDEX.md)**
