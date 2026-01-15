# NGINX with TLS Certificates

Enterprise-grade TLS certificate management using Keeper file attachments. Demonstrates automatic certificate rotation without pod restarts.

**Time to complete: ~15 minutes**

## What This Demonstrates

- TLS certificate injection from Keeper file attachments
- Separate cert and key file management
- Automatic certificate rotation without downtime
- Enterprise certificate management pattern

## Use Case

Managing TLS certificates is a common operational challenge:
- Certificates expire and need rotation
- Private keys must be kept secure
- Manual updates cause downtime

Keeper solves this by:
- Storing certificates as file attachments
- Automatically injecting them into pods
- Rotating certificates without pod restart

## Prerequisites

1. Keeper K8s Injector installed in your cluster
2. A Keeper Secrets Manager application with a config file
3. TLS certificate and private key files

## Quick Start

### 1. Generate Test Certificates (Optional)

If you don't have certificates, generate self-signed ones for testing:

```bash
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout server.key -out server.crt \
  -days 365 -subj "/CN=localhost"
```

This creates:
- `server.crt` - Certificate file
- `server.key` - Private key file

### 2. Upload Certificates to Keeper

In your Keeper vault:
1. Create a new record titled **"tls-certificate"**
2. Click "Add File" and upload `server.crt`
3. Click "Add File" again and upload `server.key`
4. Save the record

### 3. Create Your KSM Auth Secret

```bash
# If you haven't already
kubectl create secret generic keeper-credentials \
  --from-file=config=path/to/your/ksm-config.json
```

### 4. Deploy NGINX

```bash
kubectl apply -f deployment.yaml
```

### 5. Wait for Ready

```bash
kubectl wait --for=condition=ready pod -l app=nginx-tls --timeout=120s
```

### 6. Test HTTPS

```bash
kubectl port-forward svc/nginx-tls 8443:443
```

Open https://localhost:8443 in your browser.

You'll need to accept the self-signed certificate warning (for production, use a proper CA-signed cert).

You should see:
- 🔒 TLS connection active
- Certificate details
- Confirmation that certs are from Keeper

## How It Works

### File Attachment Annotations

```yaml
annotations:
  keeper.security/file-cert: "tls-certificate:server.crt:/app/certs/server.crt"
  keeper.security/file-key: "tls-certificate:server.key:/app/certs/server.key"
```

Format: `keeper.security/file-{name}: "{record-title}:{attachment-filename}:{destination-path}"`

### Certificate Location

The sidecar downloads file attachments and writes them to the specified paths:
- `/app/certs/server.crt` - Certificate
- `/app/certs/server.key` - Private key

NGINX reads these files to serve HTTPS traffic.

## Try Certificate Rotation

Rotating certificates without downtime:

### Step 1: Generate New Certificates

```bash
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout server-new.key -out server-new.crt \
  -days 365 -subj "/CN=localhost"
```

### Step 2: Update Keeper

1. Go to your **"tls-certificate"** record in Keeper
2. Delete the old `server.crt` and `server.key` attachments
3. Upload the new `server-new.crt` as `server.crt`
4. Upload the new `server-new.key` as `server.key`
5. Save the record

### Step 3: Verify the rotation

- Wait ~12 hours (or force refresh by restarting the pod)
- The sidecar detects the change
- New certificate files are written
- NGINX picks up the new certificates
- No downtime or pod restart required

## Production Considerations

### Use proper CA-signed certificates

For production, obtain certificates from a Certificate Authority (Let's Encrypt, DigiCert, etc.) instead of self-signed.

### Set appropriate refresh intervals

```yaml
keeper.security/refresh-interval: "24h"  # Check daily
```

### Monitor certificate expiration

Set up monitoring to alert when certificates are close to expiration (30 days before).

### Use cert-manager for automation

For full automation, combine with [cert-manager](https://cert-manager.io/) to automatically request and renew certificates, storing them in Keeper.

## File Permissions

The sidecar creates certificate files with secure permissions:
- Certificate: `0644` (readable by all)
- Private key: `0600` (readable only by owner)

## Cleanup

```bash
kubectl delete -f deployment.yaml
kubectl delete configmap nginx-tls-config
```

## Troubleshooting

### NGINX won't start

1. Check if certificates were injected:
   ```bash
   kubectl exec deployment/nginx-tls -- ls -la /app/certs/
   ```

2. Verify certificate format:
   ```bash
   kubectl exec deployment/nginx-tls -- openssl x509 -in /app/certs/server.crt -text -noout
   ```

3. Check sidecar logs:
   ```bash
   kubectl logs deployment/nginx-tls -c keeper-sidecar
   ```

### "File not found" in Keeper

1. Verify the attachment filename matches exactly (case-sensitive)
2. Ensure the Keeper application has access to the record
3. Check the record title matches the annotation

### Certificate errors

1. Verify certificate and key match:
   ```bash
   kubectl exec deployment/nginx-tls -- sh -c "
     openssl x509 -noout -modulus -in /app/certs/server.crt | openssl md5 &&
     openssl rsa -noout -modulus -in /app/certs/server.key | openssl md5
   "
   ```

   Both MD5 hashes should match.

## Real-World Patterns

### Multiple domains

For multiple domains, use SNI (Server Name Indication):

```yaml
keeper.security/file-cert-example-com: "cert-example-com:server.crt:/app/certs/example.com.crt"
keeper.security/file-key-example-com: "cert-example-com:server.key:/app/certs/example.com.key"
keeper.security/file-cert-api: "cert-api:server.crt:/app/certs/api.example.com.crt"
keeper.security/file-key-api: "cert-api:server.key:/app/certs/api.example.com.key"
```

### Certificate chain

Include intermediate certificates:

```yaml
keeper.security/file-cert: "tls-certificate:fullchain.pem:/app/certs/server.crt"
keeper.security/file-key: "tls-certificate:privkey.pem:/app/certs/server.key"
```

## Next Steps

- [API Keys](../04-api-keys/) - Multiple secrets pattern
- [Rotation Dashboard](../06-rotation-dashboard/) - Visualize rotation
