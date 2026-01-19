# Corporate Proxy Configuration

Guide for using Keeper K8s Injector in environments with SSL inspection (Zscaler, Palo Alto, Cisco Umbrella, etc.).

## Problem

Corporate proxies with SSL inspection intercept HTTPS connections and re-sign certificates with their own CA. Without the corporate CA certificate, the sidecar cannot connect to Keeper's API and fails with:

```
error: Post "https://keepersecurity.com/api/rest/sm/v1/get_secret": x509: certificate signed by unknown authority
```

## Solution

Add your corporate CA certificate to the sidecar using annotations.

---

## Zscaler Configuration

### Step 1: Export Zscaler Root CA

**Option A: From Browser**
1. Visit any HTTPS site in your browser
2. Click the padlock → View certificate
3. Look for "Zscaler Root CA" in the certificate chain
4. Export as PEM (.pem or .crt)

**Option B: From Zscaler Admin Portal**
1. Log into Zscaler admin portal
2. Navigate to Administration → SSL Inspection
3. Download the root CA certificate

**Option C: From macOS Keychain**
```bash
# Find Zscaler cert
security find-certificate -c "Zscaler" -p > zscaler-root-ca.pem
```

**Option D: From Windows**
1. Open `certmgr.msc`
2. Navigate to Trusted Root Certification Authorities → Certificates
3. Find "Zscaler Root CA"
4. Right-click → All Tasks → Export
5. Choose Base-64 encoded X.509 (.CER)

### Step 2: Create Kubernetes ConfigMap

```bash
kubectl create configmap zscaler-ca \
  --from-file=ca.crt=zscaler-root-ca.pem \
  --namespace default
```

Or from Secret (more secure):

```bash
kubectl create secret generic zscaler-ca \
  --from-file=ca.crt=zscaler-root-ca.pem \
  --namespace default
```

### Step 3: Use in Pod Annotations

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-credentials"
    keeper.security/secret: "my-secret"
    keeper.security/ca-cert-configmap: "zscaler-ca"  # ← Add this
spec:
  containers:
    - name: app
      image: myapp:latest
```

### Step 4: Verify

```bash
# Check sidecar logs for confirmation
kubectl logs pod/my-app -c keeper-secrets-sidecar | grep "custom CA certificate"

# Expected output:
# {"level":"info","msg":"loading custom CA certificate","path":"/usr/local/share/ca-certificates/keeper-ca.crt"}
# {"level":"info","msg":"custom CA certificate loaded successfully"}
```

---

## Palo Alto Networks Configuration

### Step 1: Export Palo Alto CA

1. Connect to Palo Alto GlobalProtect VPN
2. Export the certificate from browser or admin portal
3. Save as `palo-alto-ca.pem`

### Step 2: Create ConfigMap

```bash
kubectl create configmap palo-alto-ca \
  --from-file=ca.crt=palo-alto-ca.pem
```

### Step 3: Configure Pod

```yaml
annotations:
  keeper.security/ca-cert-configmap: "palo-alto-ca"
```

---

## Cisco Umbrella Configuration

### Step 1: Download Cisco Umbrella Root CA

Download from: https://docs.umbrella.com/deployment-umbrella/docs/rebrand-cisco-certificate-to-cisco-umbrella-certificate

### Step 2: Create ConfigMap

```bash
kubectl create configmap cisco-umbrella-ca \
  --from-file=ca.crt=cisco-umbrella-root.pem
```

### Step 3: Configure Pod

```yaml
annotations:
  keeper.security/ca-cert-configmap: "cisco-umbrella-ca"
```

---

## Multiple CA Certificates

If you have multiple corporate CAs (e.g., Zscaler + internal CA):

### Option 1: Concatenate Certificates

```bash
cat zscaler-ca.pem internal-ca.pem > combined-ca.pem

kubectl create configmap corporate-ca \
  --from-file=ca.crt=combined-ca.pem
```

### Option 2: Multiple ConfigMaps (Not Supported Yet)

Currently, only one CA cert annotation is supported. Use concatenation method above.

---

## Custom CA Certificate Key

If your CA cert is stored with a different key name:

```yaml
annotations:
  keeper.security/ca-cert-configmap: "corporate-ca"
  keeper.security/ca-cert-key: "root-ca.pem"  # Custom key name
```

---

## Troubleshooting

### Certificate still not trusted

1. Verify certificate format (PEM):
   ```bash
   kubectl exec pod/my-app -c keeper-secrets-sidecar -- \
     cat /usr/local/share/ca-certificates/keeper-ca.crt
   ```

   Should start with:
   ```
   -----BEGIN CERTIFICATE-----
   ```

2. Check sidecar logs for errors:
   ```bash
   kubectl logs pod/my-app -c keeper-secrets-sidecar
   ```

3. Verify the certificate is valid:
   ```bash
   openssl x509 -in zscaler-ca.pem -text -noout
   ```

### Wrong certificate exported

Make sure you export the ROOT CA, not:
- Intermediate certificates
- Server certificates
- Client certificates

Look for "CA:TRUE" in certificate details.

### ConfigMap/Secret not found

Ensure the ConfigMap/Secret exists in the same namespace as your pod:

```bash
kubectl get configmap zscaler-ca -n your-namespace
kubectl describe configmap zscaler-ca -n your-namespace
```

---

## Security Considerations

### Use Secrets for Sensitive Environments

For production, use Secrets instead of ConfigMaps:

```yaml
annotations:
  keeper.security/ca-cert-secret: "corporate-ca"  # More secure
```

Secrets are:
- Base64 encoded
- RBAC protected
- Encrypted at rest (if configured)

### Limit RBAC Access

Ensure only authorized ServiceAccounts can read the CA cert:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: ca-cert-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  resourceNames: ["corporate-ca"]
  verbs: ["get"]
```

---

## Verification Script

```bash
#!/bin/bash
# verify-ca-cert.sh - Test if CA cert is working

POD_NAME=$1

if [ -z "$POD_NAME" ]; then
  echo "Usage: $0 <pod-name>"
  exit 1
fi

echo "Checking CA cert in pod: $POD_NAME"
echo ""

# Check if CA cert file exists
echo "1. Checking CA cert file..."
kubectl exec $POD_NAME -c keeper-secrets-sidecar -- \
  cat /usr/local/share/ca-certificates/keeper-ca.crt | head -2

# Check sidecar logs for CA cert loading
echo ""
echo "2. Checking sidecar logs..."
kubectl logs $POD_NAME -c keeper-secrets-sidecar | grep "CA certificate"

# Check if secrets were fetched successfully
echo ""
echo "3. Checking if secrets were fetched..."
kubectl exec $POD_NAME -c keeper-secrets-sidecar -- ls -la /keeper/secrets/

echo ""
echo "If secrets are present, CA cert is working correctly!"
```

---

## See Also

- [Configuration Guide](configuration.md) - All annotations
- [Troubleshooting](troubleshooting.md) - Common issues
- [Cloud Authentication](cloud-auth.md) - AWS, GCP, Azure setup

---

**[← Back to Documentation Index](INDEX.md)**
