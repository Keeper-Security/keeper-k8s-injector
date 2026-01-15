# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-01-15

### Security

- Updated `golang.org/x/net` to v0.33.0 (CVE fixes)
- Updated `golang.org/x/oauth2` to v0.25.0 (CVE fixes)

## [0.1.0] - 2026-01-15

### Added

- Initial release
- Mutating admission webhook for automatic sidecar injection
- Init container for secret fetching at pod startup
- Sidecar container for continuous secret rotation
- tmpfs volume for memory-only secret storage
- Annotation-based configuration:
  - `keeper.security/inject` - Enable injection
  - `keeper.security/auth-secret` - KSM authentication
  - `keeper.security/secret` - Single secret by title
  - `keeper.security/secrets` - Multiple secrets
  - `keeper.security/secret-{name}` - Custom paths
  - `keeper.security/refresh-interval` - Rotation interval
  - `keeper.security/init-only` - Disable sidecar
  - `keeper.security/fail-on-error` - Error handling
  - `keeper.security/signal` - Refresh signal
  - `keeper.security/strict-lookup` - Strict title matching
- Keeper Notation support (`keeper://UID/field/password`)
- File attachment downloads
- Folder support (by UID)
- Helm chart for production deployment
- Prometheus metrics
- Health check endpoints (`/healthz`, `/readyz`)
- Multi-architecture images (amd64, arm64)

### Security

- Secrets stored in memory-only tmpfs
- Read-only root filesystem for containers
- Non-root container execution
- Capability dropping (all capabilities dropped)
- Secrets scoped to pod lifetime
