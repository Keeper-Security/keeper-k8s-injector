# Resilience Demo

Demonstrates retry logic, caching, and graceful degradation when Keeper API is unavailable.

**Time to complete: ~15 minutes**

## What This Demonstrates

- Retry with exponential backoff (3 attempts)
- In-memory secret caching (24-hour TTL)
- Cache fallback when Keeper API unreachable
- Graceful degradation mode

## Prerequisites

- Keeper K8s Injector installed
- `keeper-credentials` secret created
- kubectl configured

**First time?** See [Example 01 - Hello Secrets](../01-hello-secrets/#complete-setup-from-zero) for complete installation instructions (Steps 1-2).

## Quick Start

### Step 1: Deploy the Example

```bash
kubectl apply -f https://raw.githubusercontent.com/Keeper-Security/keeper-k8s-injector/main/examples/11-resilience-demo/resilience-demo.yaml
```

### Step 2: Verify Normal Operation

```bash
# Wait for pod
kubectl wait --for=condition=ready pod -l app=resilience-demo --timeout=120s

# Check logs - should fetch successfully
kubectl logs -l app=resilience-demo -c keeper-secrets-sidecar | grep "fetching secret"

# Verify secret exists
kubectl exec deployment/resilience-demo -- cat /keeper/secrets/demo-secret.json
```

### Step 3: Simulate Keeper API Outage

The resilience-demo.yaml includes a NetworkPolicy that blocks Keeper API. To activate it, apply just the NetworkPolicy:

```bash
kubectl apply -f https://raw.githubusercontent.com/Keeper-Security/keeper-k8s-injector/main/examples/11-resilience-demo/resilience-demo.yaml
```

Then enable the policy by labeling the pod (the policy targets the app=resilience-demo label).

This blocks outbound traffic to keepersecurity.com

### Step 4: Observe Cache Fallback

```bash
# Trigger manual refresh (delete pod to force init)
kubectl delete pod -l app=resilience-demo

# Watch new pod start
kubectl get pods -l app=resilience-demo -w

# Check logs - should use cached value
kubectl logs -l app=resilience-demo -c keeper-secrets-sidecar | grep "cache"

# Expected output:
# {"level":"warn","msg":"using cached secret (Keeper API unavailable after retry)","secret":"demo-secret","cache_age":"2m30s"}
```

### Step 5: Verify Pod Still Works

```bash
# Pod should be running with cached secrets
kubectl exec deployment/resilience-demo -- cat /keeper/secrets/demo-secret.json

# Secret file exists (from cache)
```

### Step 6: Restore Keeper Access

```bash
# Remove network policy
kubectl delete -f network-policy-block-keeper.yaml

# Wait ~30s for next refresh
sleep 35

# Check logs - should fetch fresh secret
kubectl logs -l app=resilience-demo -c keeper-secrets-sidecar --tail=20
```

## How It Works

### Retry Logic

When secret fetch fails:
1. Attempt 1: Immediate
2. Attempt 2: After 200ms
3. Attempt 3: After 400ms

If all 3 fail → check cache

### Cache Behavior

```
First fetch → Success → Cache secret
Next refresh → Keeper down → Use cache (warn in logs)
Pod restart → Keeper down + cache exists → Use cache
Pod restart → Keeper down + no cache → Fail (or start without secrets if fail-on-error=false)
```

### Graceful Degradation

With `keeper.security/fail-on-error: "false"`:
- Pod starts even if Keeper unavailable
- Uses cached secrets if available
- Logs errors but doesn't crash

## Annotations Explained

```yaml
keeper.security/fail-on-error: "false"  # Enable graceful degradation
keeper.security/refresh-interval: "30s" # Frequent refresh for demo
```

**fail-on-error behavior:**

| Value | Keeper down, cache exists | Keeper down, no cache |
|-------|---------------------------|----------------------|
| `true` (default) | Use cache, warn | Pod fails |
| `false` | Use cache, warn | Pod starts, no secrets |

## Monitoring

Watch for these log patterns:

**Normal operation:**
```
{"level":"info","msg":"fetching secret","secret":"demo-secret"}
{"level":"debug","msg":"secret written","path":"/keeper/secrets/demo-secret.json"}
```

**Retry in action:**
```
{"level":"debug","msg":"fetch attempt failed, retrying","attempt":1}
{"level":"debug","msg":"fetch attempt failed, retrying","attempt":2}
```

**Cache fallback:**
```
{"level":"warn","msg":"using cached secret (Keeper API unavailable after retry)","cache_age":"5m"}
```

**Degraded mode:**
```
{"level":"error","msg":"secret unavailable, no cache, continuing with degraded state"}
```

## Cleanup

```bash
kubectl delete -f deployment.yaml
kubectl delete -f network-policy-block-keeper.yaml
kubectl delete secret keeper-credentials
```

## See Also

- [Features Guide](../../docs/features.md) - Complete feature reference
- [Cloud Secrets](../../docs/cloud-secrets.md) - AWS/GCP/Azure authentication
