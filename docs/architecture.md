# Architecture Deep Dive

Technical overview of how Keeper K8s Injector works internally.

## System Architecture

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

---

## Components

### 1. Webhook Controller

**What it is**: A Kubernetes mutating admission webhook that intercepts pod creation requests.

**Responsibilities**:
- Watch for pods with `keeper.security/inject: "true"` annotation
- Modify pod spec before it's persisted to etcd
- Add init container, sidecar container, and tmpfs volume
- Validate annotations and fail fast on errors

**Deployment**:
- Runs as a Deployment with 2-3 replicas (HA)
- Exposed via Service on port 443 (HTTPS)
- Registered with MutatingWebhookConfiguration

**Performance**:
- Typical latency: 50-100ms per pod admission
- Processes pod specs in-memory (no external calls)
- Batches multiple secret annotations into one config

### 2. Init Container

**What it is**: A container that runs before your app starts.

**Image**: `keeper/injector-sidecar` (same as sidecar)

**Responsibilities**:
- Fetch KSM configuration (from K8s Secret or cloud provider)
- Call Keeper API to fetch all requested secrets
- Write secrets to tmpfs volume at `/keeper/secrets/`
- Exit after successful write (runs once)

**Lifecycle**:
1. Pod starts
2. Init container runs (blocks app start)
3. Fetches secrets from Keeper
4. Writes to `/keeper/secrets/`
5. Exits with code 0 (success) or 1 (failure)
6. App container starts (only if init succeeded)

**Failure handling**:
- If `fail-on-error: true` (default): Pod fails to start
- If `fail-on-error: false`: Pod starts without secrets

### 3. Sidecar Container

**What it is**: A container that runs alongside your app for secret rotation.

**Image**: `keeper/injector-sidecar`

**Responsibilities**:
- Continuously refresh secrets at configured interval
- Rewrite files in `/keeper/secrets/` when secrets change
- Send signals to app container on update (optional)
- Update Kubernetes Secrets (if K8s Secret injection enabled)

**Lifecycle**:
1. Starts after init container completes
2. Enters refresh loop:
   - Sleep for `refresh-interval` (default 5m)
   - Fetch secrets from Keeper
   - Compare with cached values
   - Update files if changed
   - Send signal to app (if configured)
3. Repeat until pod terminates

**Resource usage**:
- CPU: ~20m (idle), ~50m (during fetch)
- Memory: ~64 Mi
- Network: HTTPS to Keeper API

**Disabled when**:
- `init-only: "true"` annotation set
- No sidecar created, saves resources

### 4. tmpfs Volume

**What it is**: A memory-backed temporary filesystem mounted at `/keeper/secrets/`.

**Characteristics**:
- Stored in RAM (not disk)
- Shared between init, sidecar, and app containers
- Deleted when pod terminates
- Not included in backups
- Not visible to `kubectl cp`

**Security benefits**:
- Never written to disk
- Not persisted in etcd
- Isolated per-pod (not shared)
- Cleared on pod deletion

**Size**:
- Default: No size limit (uses pod memory)
- Configurable via standard tmpfs mount options
- Counts against pod memory limits

---

## Webhook Flow

Detailed flow when a pod with `keeper.security/inject: "true"` is created:

### Step 1: API Server Receives Pod Create

```
Developer: kubectl apply -f pod.yaml
    │
    ▼
Kubernetes API Server: Receives pod create request
    │
    ▼
Admission Chain: Runs mutating webhooks
```

### Step 2: Webhook Intercepts

```
API Server ──HTTP POST──▶ Webhook Controller
    │
    ├─ Request Body: PodSpec (JSON)
    ├─ Headers: Content-Type, Authorization
    └─ Timeout: 10 seconds (default)
```

### Step 3: Webhook Processes

```
Webhook Controller:
  1. Check annotations
      ├─ keeper.security/inject == "true"? Continue : Skip
      └─ Validate required annotations (auth-secret, etc.)

  2. Build injection config
      ├─ Parse annotations
      ├─ Extract secret names, paths, formats
      └─ Create YAML config for sidecar

  3. Modify PodSpec
      ├─ Add init container (keeper-secrets-init)
      ├─ Add sidecar container (keeper-secrets-sidecar)
      ├─ Add tmpfs volume (keeper-secrets)
      ├─ Mount volume to all containers
      └─ Set annotation: keeper.security/injected: "true"
```

### Step 4: Webhook Responds

```
Webhook Controller ──HTTP 200 OK──▶ API Server
    │
    ├─ Response Body: Patch (JSONPatch format)
    ├─ allowed: true
    └─ patch: base64-encoded mutations
```

### Step 5: API Server Applies Patch

```
API Server:
  1. Apply JSONPatch to PodSpec
  2. Continue admission chain
  3. Persist final PodSpec to etcd
  4. Schedule pod to node
```

### Step 6: Pod Starts on Node

```
Node (kubelet):
  1. Pull images (keeper/injector-sidecar + app)
  2. Create tmpfs volume
  3. Start init container
      ├─ Fetch secrets from Keeper
      └─ Write to /keeper/secrets/
  4. Start sidecar container (refresh loop)
  5. Start app container (reads /keeper/secrets/)
```

---

## Security Model

### Principle: Zero Trust

**No secrets stored in Kubernetes cluster**:
- Not in etcd (no K8s Secret objects created)*
- Not in backups
- Not in ConfigMaps
- Not in node filesystem

*Unless using K8s Secret injection mode

### Principle: Least Privilege

**RBAC Permissions**:
- Webhook: Read pods, mutate pods
- Sidecar: Read secrets (for auth config)
- No cluster-admin required

**Pod Security**:
- Non-root user (UID 65534)
- Read-only root filesystem
- All capabilities dropped
- No privilege escalation

**Network Isolation**:
- Webhook: TLS-only (port 443)
- Sidecar: HTTPS to Keeper API (egress)
- No ingress traffic to sidecar

### Principle: Defense in Depth

**Multiple layers**:
1. **Admission control**: Webhook validates annotations
2. **Runtime isolation**: tmpfs per-pod (not shared)
3. **Network security**: TLS for all communications
4. **Audit logging**: All mutations logged
5. **Secret lifecycle**: Deleted on pod termination

---

## TLS Certificate Management

The injector webhook requires TLS certificates for secure communication with the Kubernetes API server. Three modes are available:

### Mode 1: Auto-TLS (Default)

Certificates are auto-generated using `kube-webhook-certgen` during installation.

**Benefits:**
- No external dependencies
- Simple installation
- Certificates valid for 100 years
- Automatic CA bundle injection

**How it works:**
1. **Pre-install Job**: Generates TLS certificate and CA
   ```
   Job: keeper-injector-cert-create
   └─ Creates Secret: keeper-injector-tls
       ├─ tls.crt (server certificate)
       ├─ tls.key (private key)
       └─ ca.crt (CA certificate)
   ```

2. **Post-install Job**: Patches webhook configuration
   ```
   Job: keeper-injector-cert-patch
   └─ Updates MutatingWebhookConfiguration
       └─ Injects caBundle (base64-encoded CA cert)
   ```

3. **Webhook pod**: Mounts secret and uses certificate
   ```
   Pod: keeper-injector-webhook
   └─ Mounts Secret keeper-injector-tls
       ├─ /certs/tls.crt
       └─ /certs/tls.key
   ```

**Configuration:**
```yaml
# Helm values (default)
tls:
  autoGenerate: true
  certManager:
    enabled: false
```

### Mode 2: cert-manager

Use cert-manager for managed certificate lifecycle.

**Benefits:**
- Certificate rotation handled by cert-manager
- Integration with existing cert-manager infrastructure
- Custom issuer support

**Requirements:**
- cert-manager 1.11+ installed in cluster

**How it works:**
1. Helm creates `Certificate` CRD:
   ```yaml
   apiVersion: cert-manager.io/v1
   kind: Certificate
   metadata:
     name: keeper-injector-tls
   spec:
     secretName: keeper-injector-tls
     dnsNames:
       - keeper-injector-webhook
       - keeper-injector-webhook.keeper-security
       - keeper-injector-webhook.keeper-security.svc
   ```

2. cert-manager provisions certificate:
   - Creates Secret `keeper-injector-tls`
   - Renews before expiration
   - Updates Secret automatically

3. cert-manager CA Injector updates webhook:
   - Watches MutatingWebhookConfiguration
   - Injects `cert-manager.io/inject-ca-from` annotation
   - Keeps caBundle up-to-date

**Configuration:**
```yaml
# Helm values
tls:
  certManager:
    enabled: true
    issuerKind: Issuer  # or ClusterIssuer
    issuerName: ""  # Leave empty to auto-create
```

### Mode 3: Manual Certificates

Provide your own certificates for custom CA requirements.

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
openssl req -x509 -new -nodes -key ca.key -days 36500 -out ca.crt \
  -subj "/CN=keeper-injector-ca"

# Generate server certificate
openssl genrsa -out tls.key 2048
openssl req -new -key tls.key -out tls.csr \
  -subj "/CN=keeper-injector-webhook.keeper-security.svc"

# Create SAN config
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

### Troubleshooting TLS

**Check certificate:**
```bash
kubectl get secret -n keeper-security keeper-injector-tls \
  -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -noout -text
```

**Check webhook CA bundle:**
```bash
kubectl get mutatingwebhookconfiguration keeper-injector \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' | \
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
- **Certificate expired**: Regenerate using appropriate method
- **CA bundle mismatch**: Ensure webhook configuration has correct CA bundle

---

## Data Flow

### Secret Fetch Flow

```
1. Parse Annotations
   ├─ Extract secret names
   ├─ Extract output paths
   ├─ Extract formats/templates
   └─ Build fetch list

2. Fetch KSM Config
   ├─ From K8s Secret (default)
   ├─ From AWS Secrets Manager (IRSA)
   ├─ From GCP Secret Manager (Workload Identity)
   └─ From Azure Key Vault (Workload Identity)

3. Initialize KSM Client
   ├─ Parse KSM config
   ├─ Validate credentials
   └─ Establish HTTPS connection

4. Batch Fetch Secrets
   ├─ Single API call for all secrets
   ├─ Keeper returns JSON with all records
   └─ Cache secrets in memory (24h TTL)

5. Process Each Secret
   ├─ Extract fields (if notation used)
   ├─ Apply template (if configured)
   ├─ Format output (json, env, properties, etc.)
   └─ Write to file path

6. Set Permissions
   ├─ chmod 0600 (owner read/write only)
   └─ chown to app user (if specified)
```

### Refresh Flow (Sidecar)

```
1. Sleep for refresh-interval

2. Fetch Secrets (same as init)
   └─ Use cached KSM config

3. Compare with Previous
   ├─ Hash of each secret
   └─ Detect changes

4. If Changed:
   ├─ Rewrite affected files
   ├─ Log update event
   └─ Send signal to app (if configured)

5. Repeat from Step 1
```

---

## Performance Characteristics

### Webhook Latency

| Operation | Latency |
|-----------|---------|
| Annotation parsing | 1-5 ms |
| PodSpec mutation | 10-20 ms |
| Network roundtrip | 20-50 ms |
| **Total** | **50-100 ms** |

**Impact on pod startup**: Negligible (< 100ms added latency)

### Secret Fetch Performance

| Scenario | API Calls | Latency |
|----------|-----------|---------|
| 1 secret | 1 | 100-200 ms |
| 10 secrets | 1 | 100-200 ms |
| 100 secrets | 1 | 100-200 ms |

**Batching**: All secrets fetched in single API call, regardless of count.

### Memory Footprint

| Component | Memory Usage |
|-----------|--------------|
| Webhook controller | 128 Mi (per replica) |
| Init container | 64 Mi (ephemeral) |
| Sidecar container | 64 Mi (persistent) |
| tmpfs volume | Varies (secret size) |

**Example pod with 3 secrets (10 KB total)**:
- Init: 64 Mi (runs once)
- Sidecar: 64 Mi (continuous)
- tmpfs: < 1 Mi (10 KB secrets)
- **Total overhead**: ~64 Mi per pod

---

## High Availability

### Webhook HA

**Default deployment**:
- 2 replicas (can scale to 3+)
- PodDisruptionBudget: minAvailable=1
- Anti-affinity (spread across nodes)

**Failure handling**:
- API server retries on webhook timeout
- If all replicas down: Pod admission fails
- Health checks: /healthz endpoint

### Sidecar Resilience

**Retry logic**:
- Exponential backoff (200ms → 400ms → 800ms)
- 3 attempts per fetch
- Respects context cancellation

**Cache fallback**:
- 24-hour in-memory cache
- Used on API failure
- Logs warning on stale cache use

**Graceful shutdown**:
- SIGTERM handler
- Flushes pending writes
- Cleans up connections

---

**[← Back to Documentation Index](INDEX.md)**
