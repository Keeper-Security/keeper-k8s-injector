# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-01-17

### Added

- Retry with exponential backoff (3 attempts, 200ms-5s delays)
- In-memory secret caching (24-hour TTL)
- Cache fallback when Keeper API unavailable
- Thread-safe cache implementation
- KSM config validation (JSON or base64 format)
- `keeper.security/fail-on-error` annotation controls fallback behavior


## [0.3.0] - 2026-01-17

### Added

- AWS Secrets Manager authentication via IRSA (IAM Roles for Service Accounts)
- GCP Secret Manager authentication via Workload Identity
- Azure Key Vault authentication via Workload Identity
- KSM config can be stored in cloud secrets stores
- CloudTrail and Cloud Logging audit trails for config access
- Backward compatible (K8s Secret auth remains default)

## [0.2.0] - 2026-01-16

### Added

- Custom CA certificate support for corporate proxies (Zscaler, Palo Alto, Cisco Umbrella)
  - `keeper.security/ca-cert-secret` annotation
  - `keeper.security/ca-cert-configmap` annotation
- Go template rendering for custom secret formatting
  - 100+ Sprig template functions
  - Connection string building
  - Conditional logic and default values
- Additional secret formats: properties, YAML, INI
- Kubernetes support extended to 1.21+
- Go 1.25.6 with latest libraries

### Security

- Fixed 13 of 15 vulnerabilities (golang.org/x/oauth2, golang.org/x/net, Go stdlib)

## [0.1.3] - 2026-01-15

### Added

- Complete example applications suite:
  - 01-hello-secrets: Simple web page demo (5 min)
  - 02-database-postgres: PostgreSQL credentials injection
  - 03-database-mysql: MySQL credentials injection
  - 04-api-keys: Multiple SaaS API keys pattern
  - 05-tls-nginx: NGINX with TLS certificates from Keeper
  - 06-rotation-dashboard: Live secret rotation visualization
- Helm chart README for ArtifactHub display
- Docker Hub READMEs for all three images
- Updated documentation with all installation methods

### Fixed

- Linter issues: unchecked Write() errors and simplified string operations
- Updated CLAUDE.md with Docker development rules

## [0.1.2] - 2026-01-15

### Changed

- Helm chart now published to Docker Hub OCI registry
- Install via: `helm install keeper-injector oci://registry-1.docker.io/keeper/keeper-injector`

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
