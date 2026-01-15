# Keeper Secrets Injector - Webhook

The mutating admission webhook component of the Keeper Kubernetes Secrets Injector.

## What This Does

This webhook intercepts Kubernetes pod creation requests and automatically injects:
- A sidecar container (`keeper/injector-sidecar`) to fetch secrets
- A tmpfs volume for secure, memory-only secret storage

## Usage

This image is deployed automatically when you install the Keeper Injector via Helm. You don't typically pull this image directly.

### Install via Helm

```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm install keeper-injector keeper/keeper-injector --namespace keeper-system --create-namespace
```

### Or via kubectl

```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/latest/download/install.yaml
```

## Architecture

```
Pod Creation Request
        │
        ▼
┌───────────────────┐
│  injector-webhook │  ◄── This image
│  (admission ctrl) │
└───────────────────┘
        │
        ▼
   Mutated Pod with
   injector-sidecar
```

## Configuration

The webhook is configured via the Helm chart. See [values.yaml](https://github.com/Keeper-Security/keeper-k8s-injector/blob/main/charts/keeper-injector/values.yaml) for options.

## Links

- [GitHub Repository](https://github.com/Keeper-Security/keeper-k8s-injector)
- [Documentation](https://github.com/Keeper-Security/keeper-k8s-injector/tree/main/docs)
- [Helm Chart](https://artifacthub.io/packages/helm/keeper-injector/keeper-injector)
- [Keeper Secrets Manager](https://www.keepersecurity.com/secrets-manager.html)

## Tags

- `latest` - Latest stable release
- `X.Y.Z` - Specific version (e.g., `0.1.2`)

## License

MIT License - [Keeper Security, Inc.](https://www.keepersecurity.com)
