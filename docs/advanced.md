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

## TLS Certificate Management

The injector webhook requires TLS certificates for secure communication with the Kubernetes API server. Three modes are available:

### Mode 1: Auto-TLS (Default)

Certificates are auto-generated using `kube-webhook-certgen` during installation. This is the recommended mode for most users.

**Benefits:**
- No external dependencies
- Simple installation
- Certificates are valid for 100 years
- Automatic CA bundle injection

**Configuration:**
```yaml
# Helm values (default)
tls:
  autoGenerate: true
  certManager:
    enabled: false
```

**How it works:**
1. Pre-install Job generates TLS certificate and CA
2. Certificate stored in `keeper-injector-tls` Secret
3. Post-install Job patches webhook configuration with CA bundle
4. Webhook uses the generated certificate

**Installation:**
```bash
# Default installation uses auto-TLS
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

### Mode 2: cert-manager (Optional)

Use cert-manager for managed certificate lifecycle. Requires cert-manager to be pre-installed in the cluster.

**Benefits:**
- Certificate rotation handled by cert-manager
- Integration with existing cert-manager infrastructure
- Custom issuer support

**Requirements:**
- cert-manager 1.11+ installed in cluster

**Configuration:**
```yaml
# Helm values
tls:
  autoGenerate: true
  certManager:
    enabled: true
    issuerKind: Issuer  # or ClusterIssuer
    issuerName: ""  # Leave empty to auto-create
```

**Installation:**
```bash
# Install cert-manager first
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.1/cert-manager.yaml

# Install injector with cert-manager
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace \
  --set tls.certManager.enabled=true
```

### Mode 3: Manual Certificates (Advanced)

Provide your own certificates for custom CA requirements or corporate PKI integration.

**Use cases:**
- Corporate PKI integration
- Custom certificate authorities
- Compliance requirements

**Configuration:**
```yaml
# Helm values
tls:
  autoGenerate: false
  certificate:
    crt: LS0tLS1CRUd...  # base64-encoded certificate
    key: LS0tLS1CRUd...  # base64-encoded private key
    ca: LS0tLS1CRUd...   # base64-encoded CA certificate
```

**Generate certificates:**
```bash
# Generate CA
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -days 36500 -out ca.crt -subj "/CN=keeper-injector-ca"

# Generate server certificate
openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -out tls.csr -subj "/CN=keeper-injector-webhook.keeper-security.svc"

# Create config for SAN
cat > csr.conf <<EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name

[req_distinguished_name]

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = keeper-injector-webhook
DNS.2 = keeper-injector-webhook.keeper-security
DNS.3 = keeper-injector-webhook.keeper-security.svc
EOF

# Sign certificate
openssl x509 -req -in tls.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out tls.crt -days 36500 -extensions v3_req -extfile csr.conf

# Base64 encode for Helm values
cat tls.crt | base64 | tr -d '\n'
cat tls.key | base64 | tr -d '\n'
cat ca.crt | base64 | tr -d '\n'
```

**Installation:**
```bash
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace \
  --set tls.autoGenerate=false \
  --set tls.certificate.crt="LS0tLS..." \
  --set tls.certificate.key="LS0tLS..." \
  --set tls.certificate.ca="LS0tLS..."
```

### Migrating Between Modes

**From cert-manager to Auto-TLS:**
```bash
helm upgrade keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --reuse-values \
  --set tls.certManager.enabled=false
```

**From Auto-TLS to cert-manager:**
```bash
# Install cert-manager first if not already installed
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.1/cert-manager.yaml

# Upgrade to cert-manager mode
helm upgrade keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --reuse-values \
  --set tls.certManager.enabled=true
```

### Troubleshooting TLS

**Check certificate:**
```bash
kubectl get secret -n keeper-security keeper-injector-tls -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -noout -text
```

**Check webhook CA bundle:**
```bash
kubectl get mutatingwebhookconfiguration keeper-injector -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | \
  base64 -d | openssl x509 -noout -subject
```

**Verify Jobs (Auto-TLS mode):**
```bash
kubectl get jobs -n keeper-security | grep cert
kubectl logs -n keeper-security job/keeper-injector-cert-create
kubectl logs -n keeper-security job/keeper-injector-cert-patch
```

**Common issues:**
- **Job failures**: Check RBAC permissions for cert-manager ServiceAccount
- **Certificate expired**: Regenerate using the appropriate method for your mode
- **CA bundle mismatch**: Ensure webhook configuration has correct CA bundle

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
