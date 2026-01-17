# Advanced Configuration

## Helm Installation

For production deployments, use the Helm chart:

```bash
helm repo add keeper https://charts.keeper.security
helm install keeper-injector keeper/keeper-injector
```

### Custom Values

```yaml
# values.yaml
replicaCount: 3

image:
  repository: keeper/injector-webhook
  tag: "1.0.0"

sidecar:
  repository: keeper/injector-sidecar
  tag: "1.0.0"

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

# Exclude additional namespaces
excludedNamespaces:
  - kube-system
  - kube-public
  - monitoring
  - istio-system

# Default settings for all pods
defaults:
  refreshInterval: "10m"
  failOnError: true

# Enable Prometheus ServiceMonitor
metrics:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s

# High availability
leaderElection:
  enabled: true

podDisruptionBudget:
  enabled: true
  minAvailable: 2
```

Install with custom values:

```bash
helm install keeper-injector keeper/keeper-injector -f values.yaml
```

## OIDC Authentication

For enhanced security, OIDC authentication allows pods to authenticate using their Kubernetes ServiceAccount tokens instead of static credentials.

### Status: Coming Soon

OIDC authentication requires backend support from Keeper Secrets Manager for token exchange. The client-side infrastructure is implemented, but full functionality requires:

1. Keeper Secrets Manager to support OIDC token exchange
2. Kubernetes cluster OIDC issuer to be registered with Keeper

Contact Keeper support for availability information.

### Planned Configuration

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-method: "oidc"
  keeper.security/service-account: "my-app-sa"
  keeper.security/secret: "database-credentials"
```

### Current Workaround

Until OIDC is available, use the K8s Secret-based authentication:

```yaml
annotations:
  keeper.security/inject: "true"
  keeper.security/auth-secret: "keeper-auth"
  keeper.security/secret: "database-credentials"
```

The auth secret should contain your KSM configuration:

```bash
kubectl create secret generic keeper-auth \
  --from-file=config=ksm-config.json \
  -n your-namespace
```

## Multi-Cluster Setup

Each cluster needs its own injector installation:

```bash
# Cluster 1
kubectl config use-context cluster-1
helm install keeper-injector keeper/keeper-injector

# Cluster 2
kubectl config use-context cluster-2
helm install keeper-injector keeper/keeper-injector
```

Use different KSM applications per cluster for isolation.

## Network Policies

If you use NetworkPolicies, allow the sidecar to reach KSM:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-keeper-egress
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes:
    - Egress
  egress:
    # Allow DNS
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
    # Allow KSM API
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 443
```

## Resource Tuning

### Sidecar Resources

Configure sidecar resources in Helm values:

```yaml
sidecarResources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 20m
    memory: 64Mi
```

### Memory Considerations

The tmpfs volume counts against pod memory limits. If injecting large secrets:

```yaml
spec:
  containers:
    - name: app
      resources:
        limits:
          memory: 512Mi  # Include headroom for secrets
```

## GitOps Compatibility

The injector is designed to work with ArgoCD and Flux:

1. **Predictable mutations**: Only adds containers and volumes, doesn't modify existing fields
2. **Idempotent**: Safe to re-apply manifests
3. **Annotation marker**: Sets `keeper.security/injected: "true"` after injection

To prevent drift detection issues in ArgoCD:

```yaml
# ArgoCD Application
spec:
  ignoreDifferences:
    - group: ""
      kind: Pod
      jsonPointers:
        - /spec/initContainers
        - /spec/containers
        - /spec/volumes
```

## Logging and Debugging

### Webhook Logs

```bash
kubectl logs -n keeper-security -l app.kubernetes.io/name=keeper-injector
```

### Sidecar Logs

```bash
kubectl logs my-pod -c keeper-secrets-sidecar
```

### Debug Mode

Enable debug logging:

```yaml
# Helm values
logging:
  level: debug
  format: text  # Human-readable
```

## TLS Configuration

### Auto-Generated (Default)

Uses cert-manager to generate certificates automatically.

### Bring Your Own Certificates

```yaml
# Helm values
tls:
  autoGenerate: false
  certificate:
    crt: <base64-encoded-cert>
    key: <base64-encoded-key>
    ca: <base64-encoded-ca>
```

## Prometheus Metrics

Available metrics:

| Metric | Type | Description |
|--------|------|-------------|
| `keeper_injector_requests_total` | Counter | Total injection requests |
| `keeper_injector_errors_total` | Counter | Total injection errors |
| `keeper_injector_latency_seconds` | Histogram | Injection latency |
| `keeper_sidecar_refresh_total` | Counter | Total secret refreshes |
| `keeper_sidecar_refresh_errors_total` | Counter | Refresh errors |

### Grafana Dashboard

Import the included dashboard:

```bash
kubectl apply -f https://keeper.security/k8s/grafana-dashboard.yaml
```
