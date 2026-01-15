# Keeper Kubernetes Secrets Injector

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.25+-blue.svg)](https://kubernetes.io/)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://go.dev/)

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

## Quick Start

### Install

```bash
kubectl apply -f https://keeper.security/k8s/injector.yaml
```

### Configure Auth

```bash
kubectl create secret generic keeper-auth --from-file=config=ksm-config.json
```

### Annotate Your Pod

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

That's it! Your secrets are now available at `/keeper/secrets/database-credentials.json`.

## Documentation

- [Overview](docs/overview.md) - Architecture and concepts
- [Quick Start](docs/quickstart.md) - 5-minute setup guide
- [Annotations Reference](docs/annotations.md) - All configuration options
- [Advanced Configuration](docs/advanced.md) - Helm, OIDC, multi-cluster
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

## Comparison with ESO

| Feature | Keeper Injector | External Secrets Operator |
|---------|-----------------|---------------------------|
| Creates K8s Secrets | No | Yes |
| Secret storage | Pod tmpfs | etcd |
| Configuration | Annotations | CRDs |
| Runtime rotation | Yes (sidecar) | Sync interval |

## Requirements

- Kubernetes 1.25+
- cert-manager (for TLS)
- Keeper Secrets Manager application

## Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md).

## License

MIT License - see [LICENSE](LICENSE) for details.

## Support

- [Documentation](https://docs.keeper.io/k8s)
- [GitHub Issues](https://github.com/keeper-security/keeper-k8s-injector/issues)
- [Keeper Support](https://www.keepersecurity.com/support.html)
