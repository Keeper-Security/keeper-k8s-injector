# Releasing keeper-k8s-injector

## Overview

This repo publishes Docker images. The Helm chart is managed separately.

| What | Where | URL |
|------|-------|-----|
| Docker Images | Docker Hub | `keeper/injector-webhook`, `keeper/injector-sidecar` |
| GitHub Release | GitHub | Auto-generated release notes |
| Helm Chart | Separate repo | [Keeper-Security/helm-charts](https://github.com/Keeper-Security/helm-charts) |

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

The release workflow (`.github/workflows/build-and-release.yaml`) will:

1. Run tests
2. Build multi-arch Docker images (amd64, arm64)
3. Push images to Docker Hub (`keeper/injector-webhook`, `keeper/injector-sidecar`)
4. Create GitHub Release with auto-generated notes

### 4. Update the Helm Chart

After Docker images are pushed, update the Helm chart in the [helm-charts](https://github.com/Keeper-Security/helm-charts) repo:

1. Update `appVersion` in `charts/keeper-injector/Chart.yaml` to match the new tag
2. Bump `version` in `charts/keeper-injector/Chart.yaml`
3. Update `artifacthub.io/changes` annotation with the changelog
4. Push to main — the helm-charts CI will package and publish the chart

### 5. Verify the Release

- **GitHub Release**: https://github.com/Keeper-Security/keeper-k8s-injector/releases
- **Docker Hub**: https://hub.docker.com/r/keeper/injector-webhook
- **ArtifactHub**: https://artifacthub.io/packages/helm/keeper-security/keeper-injector

## Required Secrets (GitHub Actions)

These are configured in the `prod` environment:

| Secret | Description |
|--------|-------------|
| `DOCKERHUB_USERNAME` | Docker Hub username |
| `DOCKERHUB_TOKEN` | Docker Hub access token |

## Troubleshooting

### Release workflow failed?

1. Check the workflow logs: Actions > Build and Release > Click failed run
2. Common issues:
   - Docker Hub auth failed: Check `DOCKERHUB_TOKEN` secret
