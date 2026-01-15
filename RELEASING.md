# Releasing keeper-k8s-injector

## Overview

This project publishes to multiple locations:

| What | Where | URL |
|------|-------|-----|
| Docker Images | Docker Hub | `keeper/injector-webhook`, `keeper/injector-sidecar` |
| Helm Chart (OCI) | Docker Hub | `oci://registry-1.docker.io/keeper/keeper-injector` |
| Helm Chart (HTTP) | GitHub Pages | `https://keeper-security.github.io/keeper-k8s-injector` |
| GitHub Release | GitHub | Includes `.tgz` and `install.yaml` |
| ArtifactHub | artifacthub.io | Auto-syncs from GitHub Pages |

## How to Release a New Version

### 1. Update CHANGELOG.md

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added/Changed/Fixed/Security
- Description of changes
```

### 2. Commit and Tag

```bash
git add CHANGELOG.md
git commit -m "Release vX.Y.Z"
git push origin main

git tag vX.Y.Z
git push origin vX.Y.Z
```

### 3. What Happens Automatically

The release workflow (`.github/workflows/release.yaml`) will:

1. Run tests
2. Build multi-arch Docker images (amd64, arm64)
3. Push images to Docker Hub (`keeper/injector-webhook`, `keeper/injector-sidecar`)
4. Package Helm chart with the new version
5. Push Helm chart to Docker Hub OCI registry
6. Update GitHub Pages Helm repository (gh-pages branch)
7. Create GitHub Release with artifacts

### 4. Verify the Release

- **GitHub Release**: https://github.com/Keeper-Security/keeper-k8s-injector/releases
- **Docker Hub**: https://hub.docker.com/r/keeper/injector-webhook
- **ArtifactHub**: https://artifacthub.io/packages/helm/keeper-injector/keeper-injector

## Installation Methods for Users

### Method 1: Helm (OCI Registry)
```bash
helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector --version X.Y.Z
```

### Method 2: Helm (HTTP Repository)
```bash
helm repo add keeper https://keeper-security.github.io/keeper-k8s-injector
helm install keeper-injector keeper/keeper-injector --version X.Y.Z
```

### Method 3: kubectl (Direct YAML)
```bash
kubectl apply -f https://github.com/Keeper-Security/keeper-k8s-injector/releases/download/vX.Y.Z/install.yaml
```

## Required Secrets (GitHub Actions)

These are configured in the `prod` environment:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token |

## Branch Structure

| Branch | Purpose |
|--------|---------|
| `main` | Source code, development |
| `gh-pages` | Helm repository (index.yaml, .tgz files, artifacthub-repo.yml) |

## ArtifactHub Verified Publisher

The `gh-pages` branch contains `artifacthub-repo.yml` with:
```yaml
repositoryID: d64343f9-be2e-45ab-a82d-3180c9b03dff
```

This enables the "Verified Publisher" badge on ArtifactHub.

## Troubleshooting

### Release workflow failed?

1. Check the workflow logs: Actions → Release → Click failed run
2. Common issues:
   - Docker Hub auth failed: Check `DOCKERHUB_TOKEN` secret
   - Helm push failed: Ensure Helm registry login step exists

### ArtifactHub not updating?

- ArtifactHub scans every ~30 minutes
- Force rescan: ArtifactHub Control Panel → Repositories → Click refresh icon

### gh-pages not updating?

The release workflow updates gh-pages automatically. If manual update needed:
```bash
git checkout gh-pages
# Download chart from release
gh release download vX.Y.Z --pattern "*.tgz"
# Regenerate index
helm repo index . --url https://keeper-security.github.io/keeper-k8s-injector --merge index.yaml
git add -A && git commit -m "Add vX.Y.Z" && git push
git checkout main
```
