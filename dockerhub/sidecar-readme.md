# Keeper Secrets Injector - Sidecar

The sidecar container component of the Keeper Kubernetes Secrets Injector.

## What This Does

This container runs alongside your application and:
- Fetches secrets from Keeper Secrets Manager at pod startup
- Writes secrets to a shared tmpfs volume (memory-only, never touches disk)
- Continuously refreshes secrets at configurable intervals
- Optionally sends signals (e.g., SIGHUP) to your app when secrets update

## Usage

This image is automatically injected into your pods by the webhook. You don't typically pull this image directly.

### Enable Injection

Add annotations to your pod:

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

### Secrets Location

Secrets are written to `/keeper/secrets/` by default:
- `/keeper/secrets/database-credentials.json`

## Architecture

```
┌─────────────────────────────────────────┐
│                  Pod                     │
│  ┌─────────────┐    ┌─────────────────┐ │
│  │  Your App   │    │ injector-sidecar│ │ ◄── This image
│  │             │    │                 │ │
│  │  Reads from │◄───│ Writes secrets  │ │
│  │  /keeper/   │    │ from KSM        │ │
│  └─────────────┘    └─────────────────┘ │
│         ▲                   │           │
│         └───────────────────┘           │
│              tmpfs volume               │
└─────────────────────────────────────────┘
```

## Features

- **Memory-only storage** - Secrets never written to disk
- **Auto-rotation** - Refreshes secrets without pod restart
- **Signal support** - Notify your app when secrets change
- **Keeper Notation** - Extract specific fields with `keeper://UID/field/password`
- **File attachments** - Download files from Keeper records

## Configuration Annotations

| Annotation | Description | Default |
|------------|-------------|---------|
| `keeper.security/inject` | Enable injection | Required |
| `keeper.security/auth-secret` | K8s secret with KSM config | Required |
| `keeper.security/secret` | Secret title to fetch | - |
| `keeper.security/secrets` | Multiple secrets (comma-separated) | - |
| `keeper.security/refresh-interval` | Rotation interval | `0` (disabled) |
| `keeper.security/signal` | Signal on refresh (e.g., SIGHUP) | - |

## Links

- [GitHub Repository](https://github.com/Keeper-Security/keeper-k8s-injector)
- [Annotations Reference](https://github.com/Keeper-Security/keeper-k8s-injector/blob/main/docs/annotations.md)
- [Keeper Secrets Manager](https://www.keepersecurity.com/secrets-manager.html)

## Tags

- `latest` - Latest stable release
- `X.Y.Z` - Specific version (e.g., `0.1.2`)

## License

MIT License - [Keeper Security, Inc.](https://www.keepersecurity.com)
