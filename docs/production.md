# Production Deployment

Best practices for deploying Keeper K8s Injector in production environments.

## High Availability

### Webhook Replicas

Run multiple webhook replicas for fault tolerance:

```yaml
# Helm values
replicaCount: 3

podDisruptionBudget:
  enabled: true
  minAvailable: 2
```

**Benefits**:
- Survives node failures
- Rolling updates without downtime
- Load distribution across replicas

**Sizing guidance**:
- Small cluster (<50 nodes): 2 replicas
- Medium cluster (50-200 nodes): 3 replicas
- Large cluster (200+ nodes): 3-5 replicas

### Leader Election

Enable leader election for high-traffic clusters:

```yaml
# Helm values
leaderElection:
  enabled: true
```

**When to use**:
- High pod creation rate (>100 pods/min)
- Prevents thundering herd on webhook

**How it works**:
- One replica is elected leader
- Leader handles all webhook requests
- Followers standby for failover

### Anti-Affinity

Spread replicas across nodes:

```yaml
# Helm values
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app.kubernetes.io/name: keeper-injector
          topologyKey: kubernetes.io/hostname
```

**Benefits**:
- Survives node failures
- No single point of failure
- Better distribution

---

## Resource Limits

### Webhook Resources

```yaml
# Helm values - Production
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 200m
    memory: 256Mi
```

**Tuning guidance**:
- CPU: 100m baseline + 100m per 100 pods/hour
- Memory: 128Mi baseline + 128Mi per 1000 concurrent pods

**Example calculations**:
- 10 pods/hour: 100m CPU, 128Mi RAM
- 100 pods/hour: 200m CPU, 256Mi RAM
- 1000 pods/hour: 1000m CPU, 512Mi RAM

### Sidecar Resources

Per-pod sidecar overhead:

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

**Cluster-wide impact**:
- 100 pods × 64Mi = 6.4 GB cluster RAM
- 1000 pods × 64Mi = 64 GB cluster RAM

**Optimization**:
- Use `init-only: "true"` for static secrets (no sidecar)
- Increase refresh interval to reduce CPU usage

### tmpfs Volume Sizing

tmpfs counts against pod memory limits:

```yaml
spec:
  containers:
    - name: app
      resources:
        limits:
          memory: 512Mi  # Include headroom for secrets
```

**Sizing formula**:
```
Total memory = App memory + Sidecar memory + Secret size
```

**Example**:
- App: 256 Mi
- Sidecar: 64 Mi
- Secrets: 10 Mi
- **Total**: 330 Mi → set limit to 384 Mi (20% headroom)

---

## Monitoring and Metrics

### Prometheus Metrics

Enable Prometheus scraping:

```yaml
# Helm values
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
```

**Available metrics**:

| Metric | Type | Description |
|--------|------|-------------|
| `keeper_injector_requests_total` | Counter | Total injection requests |
| `keeper_injector_errors_total` | Counter | Total injection errors |
| `keeper_injector_latency_seconds` | Histogram | Injection latency |
| `keeper_sidecar_refresh_total` | Counter | Total secret refreshes |
| `keeper_sidecar_refresh_errors_total` | Counter | Refresh errors |
| `keeper_sidecar_secrets_fetched_total` | Counter | Total secrets fetched |
| `keeper_sidecar_fetch_duration_seconds` | Histogram | Secret fetch duration |

### Grafana Dashboard

**Key panels to include**:
1. Injection rate (requests/min)
2. Error rate (errors/total)
3. P50/P95/P99 latency
4. Active pods with injection
5. Sidecar refresh rate
6. API call rate to Keeper

**Example PromQL queries**:

```promql
# Injection rate
rate(keeper_injector_requests_total[5m])

# Error rate
rate(keeper_injector_errors_total[5m]) / rate(keeper_injector_requests_total[5m])

# P95 latency
histogram_quantile(0.95, rate(keeper_injector_latency_seconds_bucket[5m]))

# Refresh errors
sum(rate(keeper_sidecar_refresh_errors_total[5m])) by (pod)
```

### Alerts

**Critical alerts**:

```yaml
# High error rate
alert: KeeperInjectorHighErrorRate
expr: |
  rate(keeper_injector_errors_total[5m]) / rate(keeper_injector_requests_total[5m]) > 0.05
for: 5m
labels:
  severity: critical
annotations:
  summary: "Keeper Injector error rate > 5%"

# Webhook down
alert: KeeperInjectorDown
expr: |
  up{job="keeper-injector-webhook"} == 0
for: 2m
labels:
  severity: critical
annotations:
  summary: "Keeper Injector webhook is down"

# Refresh failures
alert: KeeperSidecarRefreshFailures
expr: |
  rate(keeper_sidecar_refresh_errors_total[10m]) > 0.1
for: 10m
labels:
  severity: warning
annotations:
  summary: "Sidecar refresh failures detected"
```

### Logging

**Production logging configuration**:

```yaml
# Helm values
logging:
  level: info  # debug for troubleshooting
  format: json  # structured logging
```

**Log aggregation**:
- Ship logs to Elasticsearch, Loki, or CloudWatch
- Index on: `level`, `msg`, `pod`, `namespace`
- Retention: 30 days minimum for audit

**Key log messages to monitor**:
```json
{"level":"error","msg":"failed to fetch secrets","pod":"myapp-abc","error":"timeout"}
{"level":"warn","msg":"using cached secrets","age":"6h","reason":"keeper API unavailable"}
{"level":"info","msg":"secrets updated","count":3,"duration":"142ms"}
```

---

## GitOps Compatibility

### ArgoCD Integration

The injector is designed to work with ArgoCD:

**Features**:
1. **Predictable mutations**: Only adds containers and volumes
2. **Idempotent**: Safe to re-apply manifests
3. **Annotation marker**: Sets `keeper.security/injected: "true"`

**Prevent drift detection**:

```yaml
# ArgoCD Application
apiVersion: argoproj.io/v1alpha1
kind: Application
spec:
  ignoreDifferences:
    - group: ""
      kind: Pod
      jsonPointers:
        - /spec/initContainers
        - /spec/containers
        - /spec/volumes
        - /metadata/annotations/keeper.security~1injected
```

**Why this works**:
- Webhook mutations are ignored by ArgoCD
- App can sync without detecting drift
- Only user-defined fields are tracked

### Flux Integration

Works with Flux out-of-the-box:

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: myapp
spec:
  path: ./apps/myapp
  sourceRef:
    kind: GitRepository
    name: fleet
  interval: 5m
```

**No special configuration needed** - Flux doesn't compare runtime pod specs.

### GitOps Best Practices

1. **Store auth configs separately**:
   ```yaml
   # In Git: app-deployment.yaml
   annotations:
     keeper.security/inject: "true"
     keeper.security/ksm-config: "keeper-auth"  # Reference only

   # Not in Git: Created via sealed-secrets or external-secrets
   apiVersion: v1
   kind: Secret
   metadata:
     name: keeper-auth
   ```

2. **Use separate repos for sensitive config**:
   - Public repo: Application manifests
   - Private repo: Secret references and auth configs

3. **Version annotations**:
   ```yaml
   annotations:
     app.version: "v1.2.3"
     keeper.security/refresh-interval: "10m"  # Tracked in Git
   ```

---

## Network Policies

If using NetworkPolicies, allow sidecar egress to Keeper API:

### Allow Keeper API Access

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: keeper-injector-egress
  namespace: production
spec:
  podSelector:
    matchLabels:
      keeper.security/injected: "true"
  policyTypes:
    - Egress
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
    # Allow Keeper API (HTTPS)
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 10.0.0.0/8      # Block private IPs
              - 172.16.0.0/12
              - 192.168.0.0/16
      ports:
        - protocol: TCP
          port: 443
```

### Allow Webhook Admission

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: keeper-webhook-ingress
  namespace: keeper-security
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: keeper-injector
  policyTypes:
    - Ingress
  ingress:
    # Allow API server
    - from:
        - ipBlock:
            cidr: 0.0.0.0/0  # API server IP varies
      ports:
        - protocol: TCP
          port: 443
```

**Note**: API server source IP varies by provider (EKS, GKE, AKS). Use `0.0.0.0/0` or get specific IP range from provider docs.

---

## Multi-Cluster Setup

Each cluster needs its own injector installation:

### Architecture

```
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│    Cluster 1 (US)   │  │   Cluster 2 (EU)    │  │  Cluster 3 (APAC)   │
│                     │  │                     │  │                     │
│  Injector v1.0.0    │  │  Injector v1.0.0    │  │  Injector v1.0.0    │
│  ↓                  │  │  ↓                  │  │  ↓                  │
│  KSM App: prod-us   │  │  KSM App: prod-eu   │  │  KSM App: prod-apac │
└──────────┬──────────┘  └──────────┬──────────┘  └──────────┬──────────┘
           │                        │                         │
           └────────────────────────┴─────────────────────────┘
                                    │
                       ┌────────────▼────────────┐
                       │  Keeper Secrets Manager │
                       │    (Shared Backend)     │
                       └─────────────────────────┘
```

### Setup Steps

1. **Install injector in each cluster**:
   ```bash
   # Cluster 1
   kubectl config use-context cluster-1
   helm install keeper-injector keeper/keeper-injector

   # Cluster 2
   kubectl config use-context cluster-2
   helm install keeper-injector keeper/keeper-injector
   ```

2. **Create separate KSM applications** (recommended):
   - `prod-us-k8s` (for US cluster)
   - `prod-eu-k8s` (for EU cluster)
   - `prod-apac-k8s` (for APAC cluster)

   **Why separate apps**:
   - Isolation (rotate credentials per-cluster)
   - Audit trail (see which cluster accessed secrets)
   - Access control (different secrets per region)

3. **Alternative: Shared KSM application**:
   - Single KSM config shared across clusters
   - Simpler to manage
   - No isolation between clusters

### Version Management

**Keep injector versions consistent**:

```bash
# Check versions across clusters
kubectl config use-context cluster-1
kubectl get deployment -n keeper-security keeper-injector \
  -o jsonpath='{.spec.template.spec.containers[0].image}'

kubectl config use-context cluster-2
kubectl get deployment -n keeper-security keeper-injector \
  -o jsonpath='{.spec.template.spec.containers[0].image}'
```

**Upgrade strategy**:
1. Upgrade dev cluster first
2. Test pod injection
3. Upgrade staging
4. Upgrade production clusters one at a time

---

## Resource Quotas

Set resource quotas to prevent runaway sidecar usage:

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: keeper-sidecar-quota
  namespace: production
spec:
  hard:
    requests.cpu: "50"      # 50 CPUs for sidecars
    requests.memory: "64Gi" # 64 GB for sidecars (1000 pods × 64Mi)
    limits.cpu: "100"
    limits.memory: "128Gi"
```

**Calculation**:
- Expected pods: 1000
- Sidecar request: 20m CPU, 64Mi RAM
- Total request: 20 CPU, 64 GB RAM
- Quota: 2.5× headroom = 50 CPU, 160 GB RAM

---

## Backup and Disaster Recovery

### What to Backup

**Critical resources**:
1. Webhook deployment (backed up via GitOps)
2. MutatingWebhookConfiguration (backed up via GitOps)
3. TLS certificates (Secret: `keeper-injector-tls`)
4. KSM auth configs (Secrets in app namespaces)

**What NOT to backup**:
- Secrets in tmpfs (ephemeral by design)
- Sidecar state (stateless)

### Disaster Recovery Plan

**Scenario: Complete cluster loss**

1. **Restore cluster**:
   ```bash
   # Restore from backup tool (Velero, etc.)
   velero restore create --from-backup cluster-backup
   ```

2. **Reinstall injector**:
   ```bash
   helm install keeper-injector keeper/keeper-injector
   ```

3. **Recreate KSM auth secrets** (if not in backup):
   ```bash
   kubectl create secret generic keeper-auth \
     --from-literal=config='<base64-config>'
   ```

4. **Verify injection works**:
   ```bash
   kubectl run test --image=busybox \
     --annotations=keeper.security/inject=true \
     --annotations=keeper.security/ksm-config=keeper-auth
   ```

**Recovery time objective (RTO)**: < 15 minutes

---

## Compliance and Security

### Audit Logging

Enable Kubernetes audit logs for injection events:

```yaml
# Audit policy
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: RequestResponse
    verbs: ["create", "update", "patch"]
    resources:
      - group: ""
        resources: ["pods"]
    omitStages:
      - RequestReceived
```

**What gets logged**:
- Pod creation with injection annotations
- Webhook mutations (init container, sidecar added)
- Success/failure of injection

### Security Scanning

**Scan images regularly**:

```bash
# Scan webhook image
trivy image keeper/injector-webhook:1.0.0

# Scan sidecar image
trivy image keeper/injector-sidecar:1.0.0
```

**Enforce image signing**:

```yaml
# Cosign verification
apiVersion: v1
kind: Pod
metadata:
  annotations:
    cosign.sigstore.dev/verify: "true"
spec:
  containers:
    - image: keeper/injector-webhook:1.0.0
```

### RBAC Hardening

**Minimal permissions for webhook**:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: keeper-injector
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get", "list"]  # Read auth configs only
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]  # Read pod specs only
  # No write permissions to secrets or pods
```

**Minimal permissions for sidecar**:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: keeper-sidecar
  namespace: production
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["get"]  # Read auth config only
    resourceNames: ["keeper-auth"]  # Specific secret only
```

---

## Performance Tuning

### Webhook Timeout

Adjust timeout for slow networks:

```yaml
# MutatingWebhookConfiguration
webhooks:
  - name: keeper.security
    timeoutSeconds: 10  # Default, increase if needed
```

**When to increase**:
- High latency to Keeper API (>500ms)
- Large number of secrets (>50 per pod)
- Corporate proxy adds latency

### Caching Strategy

Leverage 24-hour secret cache:

```yaml
annotations:
  keeper.security/refresh-interval: "12h"  # Longer interval
```

**Benefits**:
- Fewer API calls
- Faster pod startup (cache hit)
- Resilience during outages

**Trade-off**:
- Longer time to detect secret changes
- Balance freshness vs performance

---

**[← Back to Documentation Index](INDEX.md)**
