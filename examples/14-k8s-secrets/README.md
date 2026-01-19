# Example 14: Kubernetes Secret Injection

Create Kubernetes Secret objects directly from Keeper secrets.

## Prerequisites

1. Keeper Secrets Manager configured
2. Keeper Injector installed in cluster
3. Auth secret created (see main README)

## Security Notice

K8s Secrets are stored in etcd (disk) and visible via `kubectl get secret`. For higher security, use file-based injection (tmpfs).

Use K8s Secrets for:
- GitOps workflows
- Apps expecting K8s Secret mounts
- CSI driver integration

## Examples

### 1. Basic Secret Injection

Creates K8s Secret from Keeper record:

```bash
kubectl apply -f basic-secret.yaml
```

Result: Secret `app-secrets` created with all fields.

### 2. Custom Key Mapping

Maps Keeper fields to specific Secret keys:

```bash
kubectl apply -f custom-keys.yaml
```

Result: Secret with keys `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_HOST`.

### 3. TLS Certificate

Creates TLS Secret for Ingress:

```bash
kubectl apply -f tls-secret.yaml
```

Result: Secret of type `kubernetes.io/tls`.

### 4. Rotation with Sidecar

Automatically updates Secrets when Keeper records change:

```bash
kubectl apply -f rotation.yaml
```

Sidecar updates Secret every 5 minutes.

### 5. Conflict Resolution

Demonstrates different conflict modes:

```bash
kubectl apply -f conflict-modes.yaml
```

## Verification

```bash
# View created Secret
kubectl get secret app-secrets -o yaml

# Check Secret keys
kubectl get secret app-secrets -o jsonpath='{.data}' | jq

# Verify pod is using Secret
kubectl describe pod my-app

# Watch sidecar logs for rotation
kubectl logs my-app -c keeper-secrets-sidecar -f
```

## Cleanup

```bash
kubectl delete -f .
```

Secrets with owner references are deleted automatically when pods terminate.

## Integration Tests

Integration tests use real Keeper vault. Run with:

```bash
# Run all integration tests
go test -v -tags integration ./pkg/webhook/... -run Integration

# Run specific test
go test -v -tags integration ./pkg/webhook/... -run TestIntegration_CreateK8sSecret_SingleRecord
```

**Note**: Requires `config.base64` file in project root.
