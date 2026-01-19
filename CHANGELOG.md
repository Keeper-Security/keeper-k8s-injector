# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2026-01-19

### Added

- Kubernetes Secret injection via `keeper.security/inject-as-k8s-secret` annotation
- Custom key mapping for K8s Secrets via `k8sSecretKeys` field
- Support for all K8s Secret types: Opaque, kubernetes.io/tls, dockerconfigjson, basic-auth, ssh-auth
- Conflict resolution modes: overwrite, merge, skip-if-exists, fail
- Sidecar rotation support for K8s Secrets via `keeper.security/k8s-secret-rotation`
- Owner references for automatic Secret cleanup when pod terminates
- Cross-namespace Secret creation via `keeper.security/k8s-secret-namespace`
- Folder-based batch creation with `k8sSecretNamePrefix`
- K8s Secret size validation (1MB limit)
- New annotations:
  - `keeper.security/inject-as-k8s-secret`
  - `keeper.security/k8s-secret-name`
  - `keeper.security/k8s-secret-namespace`
  - `keeper.security/k8s-secret-mode`
  - `keeper.security/k8s-secret-type`
  - `keeper.security/k8s-secret-rotation`
  - `keeper.security/k8s-secret-owner-ref`
- YAML config fields: `injectAsK8sSecret`, `k8sSecretName`, `k8sSecretKeys`, `k8sSecretType`, `k8sSecretNamePrefix`

### Changed

- RBAC permissions extended with Secret write verbs (create, update, patch, delete)
- Webhook handler calls K8s Secret injection
- Sidecar agent includes K8s client for Secret updates

### Security

K8s Secrets are stored in etcd and visible via kubectl. File-based injection (tmpfs) remains the default for higher security.

## [0.8.0] - 2026-01-18

### Added

- **Folder Path Support**: Reference secrets using folder paths instead of UIDs
  - New notation format: `keeper://Folder/Path/Record/field/password`
  - Works with all annotation types (simple, YAML, notation, templates)
  - Automatic folder tree building and path resolution
  - Support for nested folder hierarchies
- Folder path resolution for `keeper.security/folder` annotation
- Case-sensitive folder matching for precision
- Example 13: Folder-based secret lookup demo with comprehensive documentation

### Changed

- Folder annotation now supports both `folder-uid` and `folder` (path-based) lookups
- KSM client extended with `BuildFolderTree()` and `GetSecretByPath()` methods
- Notation parser now detects and resolves folder paths automatically
- Sidecar agent updated to resolve folder paths when fetching secrets
- Documentation updated with folder path notation in docs/annotations.md
- Features documentation enhanced with folder-based references in docs/features.md

### Implementation Details

- New `pkg/ksm/folder_tree.go`: Hierarchical folder tree implementation
- Updated `pkg/ksm/client.go`: Added `parseNotationPath()` for folder path parsing
- Updated `pkg/sidecar/agent.go`: Folder path resolution in `fetchSecretsFromFolder()`
- Comprehensive unit tests for folder tree operations and notation parsing
- Integration tests with real Keeper vault demonstrating folder path functionality

### Backward Compatibility

Fully backward compatible with existing configurations:
- UID-based lookups continue to work unchanged
- Title-based lookups continue to work unchanged
- Existing notation syntax remains supported
- Folder path support is additive only

## [0.7.0] - 2026-01-18

### Added

- Environment variable injection via `keeper.security/inject-env-vars` annotation
- Optional env var prefix via `keeper.security/env-prefix` annotation (e.g., `DB_LOGIN`, `DB_PASSWORD`)
- Per-secret env var control in YAML config: `injectAsEnvVars: true` and `envPrefix` fields
- Mixed mode support: some secrets as files, some as environment variables
- Example 12: Environment variable injection demo
- Security trade-offs documentation comparing env vars vs files

### Changed

- Webhook now supports two injection modes: files (default) and environment variables (opt-in)
- Documentation updated with environment variable injection guide in docs/features.md
- Documentation updated with environment variable annotations in docs/annotations.md

### Security Notes

Environment variables are visible in pod metadata and cannot be rotated without pod restart. File-based injection remains the recommended approach for production environments.

## [0.6.0] - 2026-01-17

### Added

- Auto-TLS certificate generation using `kube-webhook-certgen`
- Built-in certificate management (no cert-manager required)
- Pre-install Job for certificate generation
- Post-install Job for webhook configuration patching
- Dedicated ServiceAccount and RBAC for certificate management
- Three TLS modes: Auto-TLS (default), cert-manager (optional), Manual (advanced)
- Comprehensive TLS documentation in advanced.md

### Changed

- **Breaking**: Default TLS mode changed from cert-manager to auto-TLS
  - Existing installations with explicit `tls.certManager.enabled=true` continue working
  - New installations no longer require cert-manager
- cert-manager is now optional, not required
- NOTES.txt now displays TLS mode and secret location information
- Updated all documentation to reflect cert-manager as optional dependency

### Upgrading from 0.5.x

**To use auto-TLS (recommended):**
```bash
helm upgrade keeper-injector oci://registry-1.docker.io/keeper/keeper-injector --reuse-values
```

**To keep using cert-manager:**
```bash
helm upgrade keeper-injector oci://registry-1.docker.io/keeper/keeper-injector \
  --reuse-values \
  --set tls.certManager.enabled=true
```

**Note**: If you have an existing values.yaml file with explicit `tls.certManager.enabled=true`, no changes are needed. The upgrade will preserve your cert-manager configuration.

### Documentation

- Added comprehensive TLS Certificate Management section to advanced.md
- Updated quickstart.md to remove cert-manager prerequisite
- Updated README.md requirements section
- Added migration guide between TLS modes
- Added troubleshooting guide for TLS issues

## [0.5.0] - 2026-01-17

### Added

- Helm chart now automatically labels the installation namespace with `keeper.security/inject=disabled` to prevent webhook self-injection
- Added `namespaceManagement` configuration section in values.yaml for namespace creation and labeling
- Added NOTES.txt template to display post-installation information
- Added `artifacthub.io/changes` annotation to Chart.yaml to enable changelog display on ArtifactHub
- Added `artifacthub.io/links` annotation with documentation, examples, and support links
- Added CHANGELOG-TEMPLATE.yaml for future release changelog updates

### Changed

- Changed default namespace from `keeper-system` to `keeper-security` throughout documentation
- Updated Example 01 troubleshooting section with webhook self-injection fix

### Fixed

- Fixed chicken-and-egg problem where webhook would try to intercept its own pod creation, preventing installation
- Helm chart now creates namespace with proper labels to avoid "connection refused" webhook errors during initial deployment
- Fixed ArtifactHub changelog display by adding proper annotations to Chart.yaml

### Documentation

- Added maintainer documentation for updating ArtifactHub changelog in future releases
- Chart README now includes instructions for maintaining the `artifacthub.io/changes` annotation

## [0.4.1] - 2026-01-17

### Fixed

- All documentation updated to use `helm upgrade --install` (idempotent, works for both install and upgrade)
- Added troubleshooting for "invalid ownership metadata" error during reinstall
- Example 01 now includes complete from-zero setup with explanations
- Examples 02-11 reference Example 01 for installation instructions

### Documentation

- Example 01: Added "Why" explanations for cert-manager and injector installation
- Example 01: Direct GitHub URLs for kubectl apply (no git clone needed)
- Example 01: Explicit requirement that record title must be "demo-secret"
- All examples: Cleaner prerequisites sections

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
