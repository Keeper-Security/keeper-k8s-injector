# Keeper Kubernetes Injector - Overview

## What is Keeper K8s Injector?

Keeper K8s Injector automatically injects secrets from Keeper Secrets Manager into your Kubernetes pods at runtime. Instead of creating Kubernetes Secret objects, it writes secrets directly to a memory-backed volume (tmpfs) inside your pods.

## How is it different from ESO (External Secrets Operator)?

| Feature | Keeper K8s Injector | ESO |
|---------|---------------------|-----|
| Creates K8s Secrets | No | Yes |
| Secret storage | Pod-scoped tmpfs | etcd |
| Secret lifetime | Pod lifetime | Persistent |
| Configuration | Pod annotations | CRDs |
| Rotation | Sidecar auto-refresh | Sync interval |
| Setup complexity | 2 annotations | Multiple CRDs |

**Bottom line**: ESO syncs secrets to Kubernetes Secret objects. Keeper Injector writes secrets directly into pods with no K8s Secrets created.

## When should you use it?

**Use Keeper K8s Injector when:**
- You want secrets scoped to pod lifetime (removed when pod terminates)
- You need runtime rotation without pod restarts
- You want minimal Kubernetes RBAC exposure
- You prefer annotation-based configuration over CRDs
- Security requirements prohibit storing secrets in etcd

**Use ESO when:**
- You need secrets available before pods start
- Multiple pods share the same secret
- You're already using ESO for other providers
- You need GitOps-friendly Secret objects

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                    │
│                                                         │
│  ┌─────────────────┐    ┌─────────────────────────────┐ │
│  │  Keeper Webhook │    │         Your Pod            │ │
│  │   Controller    │    │  ┌───────┐  ┌───────────┐  │ │
│  │                 │    │  │ Init  │  │  Sidecar  │  │ │
│  │  Watches pod    │───▶│  │ Fetch │  │  Refresh  │  │ │
│  │  creation,      │    │  └───┬───┘  └─────┬─────┘  │ │
│  │  injects        │    │      │            │        │ │
│  │  containers     │    │      ▼            ▼        │ │
│  └─────────────────┘    │  ┌───────────────────────┐ │ │
│                         │  │  tmpfs /keeper/secrets │ │ │
│                         │  │  (memory-backed)       │ │ │
│                         │  └───────────┬───────────┘ │ │
│                         │              │             │ │
│                         │  ┌───────────▼───────────┐ │ │
│                         │  │    Your App Container  │ │ │
│                         │  │    (reads files)       │ │ │
│                         │  └───────────────────────┘ │ │
│                         └─────────────────────────────┘ │
│                                      │                   │
└──────────────────────────────────────┼───────────────────┘
                                       │
                                       ▼
                          ┌─────────────────────────┐
                          │  Keeper Secrets Manager │
                          │        (Cloud)          │
                          └─────────────────────────┘
```

## Components

1. **Webhook Controller**: Mutating admission webhook that modifies pod specs
2. **Init Container**: Fetches secrets before your app starts
3. **Sidecar Container**: Continuously refreshes secrets (optional, for rotation)
4. **tmpfs Volume**: Memory-backed volume where secrets are written

## Next Steps

- [Installation Guide](installation.md) - Get your first secret injected in 5 minutes
- [Configuration Guide](configuration.md) - All available annotations and Helm values
- [Production Deployment](production.md) - HA, monitoring, and best practices

---

**[← Back to Documentation Index](INDEX.md)**
