# Keeper Kubernetes Secrets Injector

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.25+-blue.svg)](https://kubernetes.io/)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/keeper-injector)](https://artifacthub.io/packages/helm/keeper-injector/keeper-injector)

Automatically inject secrets from [Keeper Secrets Manager](https://www.keepersecurity.com/secrets-manager.html) into your Kubernetes pods at runtime.

## Features

- **No Kubernetes Secrets created** - Secrets are written directly to pod tmpfs
- **Pod-scoped lifetime** - Secrets are removed when pod terminates
- **Automatic rotation** - Sidecar refreshes secrets without pod restarts
- **Simple configuration** - Just two annotations to get started
- **Title-based lookup** - Reference secrets by name, not UIDs
- **Keeper Notation** - Use `keeper://UID/field/password` for precise extraction
- **File Attachments** - Download files from Keeper records
- **Folder Support** - Fetch all secrets from a Keeper folder
- **Production-ready** - HA, metrics, leader election

## Installation

### Option 1: Helm (OCI Registry) - Recommended

```bash
# Works for both new installation and upgrades
helm upgrade --install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

### Option 2: Helm (Repository)

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm repo update
helm upgrade --install keeper-injector keeper/keeper-injector \
  --namespace keeper-security \
  --create-namespace
```

### Option 3: kubectl (Direct YAML)

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

## Quick Start

### 1. Create KSM Auth Secret

**Option 1: Base64 Config (Recommended)**

From Keeper: Vault → Secrets Manager → Select Application → Devices → Add Device → Base64

```bash
kubectl create secret generic keeper-auth \
  --from-literal=config='<paste-base64-config-here>' \
  --namespace default
```

**Option 2: Config File**

```bash
kubectl create secret generic keeper-auth \
  --from-file=config=ksm-config.json \
  --namespace default
```

### 2. Annotate Your Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  annotations:
    keeper.security/inject: "true"
    keeper.security/auth-secret: "keeper-auth"
    keeper.security/secret: "database-credentials"
spec:
  containers:
    - name: app
      image: my-app:latest
```

Secrets are now available at `/keeper/secrets/database-credentials.json`.

## Examples

Try these working examples to see the injector in action:

| Example | Description | Time |
|---------|-------------|------|
| [Hello Secrets](examples/01-hello-secrets/) | Web page displaying secret values | 5 min |
| [PostgreSQL](examples/02-database-postgres/) | Real database credential injection | 10 min |
| [Rotation Dashboard](examples/06-rotation-dashboard/) | Live secret rotation visualization | 5 min |

### Try It Now

```bash
# Clone the repo
git clone https://github.com/Keeper-Security/keeper-k8s-injector.git
cd keeper-k8s-injector

# Run the hello-secrets example
kubectl apply -f examples/01-hello-secrets/
kubectl port-forward svc/hello-secrets 8080:80

# Open http://localhost:8080
```

## Documentation

- [Quick Start](docs/quickstart.md) - 5-minute setup guide
- [Complete Feature Reference](docs/features.md) - All features with examples
- [Annotations Reference](docs/annotations.md) - All configuration options
- [Template Guide](docs/templates.md) - Go templates and functions
- [Feature Comparison](docs/comparison.md) - vs Vault, ESO, AWS CSI, 1Password
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

## Annotation Examples

### Multiple Secrets

```yaml
keeper.security/secrets: "database-creds, api-keys, tls-cert"
```

### Custom Paths

```yaml
keeper.security/secret-db: "/app/config/database.json"
keeper.security/secret-api: "/etc/myapp/api.json"
```

### With Rotation

```yaml
keeper.security/refresh-interval: "5m"
keeper.security/signal: "SIGHUP"
```

### Keeper Notation (Specific Fields)

```yaml
keeper.security/secret-password: "keeper://QabbPIdM8Unw4hwVM-F8VQ/field/password:/app/secrets/db-pass"
```

### File Attachments

```yaml
keeper.security/file-cert: "Database Credentials:cert.pem:/app/certs/server.pem"
```

## Comparison with External Secrets Operator (ESO)

| Feature | Keeper Injector | External Secrets Operator |
|---------|-----------------|---------------------------|
| Creates K8s Secrets | No | Yes |
| Secret storage | Pod tmpfs (memory) | etcd |
| Secrets in etcd backups | No | Yes |
| Configuration | Annotations | CRDs |
| Runtime rotation | Yes (sidecar) | Sync interval |
| Pod isolation | Yes | Shared secrets |

**Use Keeper Injector when:** Security is paramount, you need secrets out of etcd, or require per-pod isolation.

**Use ESO when:** You need secrets as K8s Secret objects, or apps require environment variables only.

## Docker Images

| Image | Description |
|-------|-------------|
| `keeper/injector-webhook` | Mutating admission webhook |
| `keeper/injector-sidecar` | Sidecar container for secret fetching |

Images are available on [Docker Hub](https://hub.docker.com/u/keeper) with multi-arch support (amd64, arm64).

## Requirements

- Kubernetes 1.21+ (tested with 1.21-1.34)
- [cert-manager](https://cert-manager.io/) (for TLS certificates)
- Keeper Secrets Manager application

## Links

- [ArtifactHub](https://artifacthub.io/packages/helm/keeper-injector/keeper-injector)
- [Docker Hub - Webhook](https://hub.docker.com/r/keeper/injector-webhook)
- [Docker Hub - Sidecar](https://hub.docker.com/r/keeper/injector-sidecar)
- [Keeper Secrets Manager](https://docs.keeper.io/secrets-manager/)

## Contributing

Contributions are welcome! Please open an issue or pull request.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- [GitHub Issues](https://github.com/Keeper-Security/keeper-k8s-injector/issues)
- [Keeper Support](https://www.keepersecurity.com/support.html)
