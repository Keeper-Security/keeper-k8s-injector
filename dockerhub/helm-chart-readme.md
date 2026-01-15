# Keeper Secrets Injector - Helm Chart

The official Helm chart for the Keeper Kubernetes Secrets Injector, distributed as an OCI artifact.

## What This Is

This is a **Helm chart** (not a Docker image) stored in Docker Hub's OCI registry. It deploys the complete Keeper Secrets Injector to your Kubernetes cluster.

## Installation

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-system \
  --create-namespace
```

### With Custom Values

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --namespace keeper-system \
  --create-namespace \
  --set replicaCount=3 \
  --set metrics.enabled=true
```

### Specific Version

```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --version 0.1.2 \
  --namespace keeper-system \
  --create-namespace
```

## Alternative Installation Methods

### Via Helm Repository

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm install keeper-injector keeper/keeper-injector
```

### Via kubectl

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

## What Gets Deployed

- **Deployment** - The webhook server (`keeper/injector-webhook`)
- **Service** - ClusterIP service for the webhook
- **MutatingWebhookConfiguration** - Registers with Kubernetes API
- **RBAC** - ServiceAccount, ClusterRole, ClusterRoleBinding
- **Certificate** - TLS certificate for webhook (via cert-manager or self-signed)

## Configuration

See the full list of configuration options:
- [values.yaml](https://github.com/Keeper-Security/keeper-k8s-injector/blob/main/charts/keeper-injector/values.yaml)
- [Helm Chart on ArtifactHub](https://artifacthub.io/packages/helm/keeper-injector/keeper-injector)

## Links

- [GitHub Repository](https://github.com/Keeper-Security/keeper-k8s-injector)
- [Documentation](https://github.com/Keeper-Security/keeper-k8s-injector/tree/main/docs)
- [ArtifactHub](https://artifacthub.io/packages/helm/keeper-injector/keeper-injector)
- [Keeper Secrets Manager](https://www.keepersecurity.com/secrets-manager.html)

## Tags

- `latest` - Latest stable release
- `X.Y.Z` - Specific version (e.g., `0.1.2`)

## License

MIT License - [Keeper Security, Inc.](https://www.keepersecurity.com)
