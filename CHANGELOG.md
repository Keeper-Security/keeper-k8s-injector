# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-01-16

### Added

- **Cloud Secrets Provider Authentication** - Store KSM config in cloud instead of K8s Secrets
  - AWS Secrets Manager via IRSA (IAM Roles for Service Accounts)
  - GCP Secret Manager via Workload Identity
  - Azure Key Vault via Workload Identity
  - No static credentials in Kubernetes cluster
  - CloudTrail/Cloud Logging audit trails
  - Backward compatible (K8s Secret auth still default)

### Documentation

- docs/cloud-secrets.md - Complete setup guide for AWS/GCP/Azure
- examples/08-aws-secrets-manager/ - AWS IRSA example with IAM setup
- Updated docs/annotations.md with cloud provider annotations
- Updated docs/features.md with authentication methods comparison

### Testing

- Comprehensive unit tests for cloud provider integrations
- Input validation for all cloud secret references
- Linter clean (0 issues)

### Dependencies

- github.com/aws/aws-sdk-go-v2/service/secretsmanager - AWS integration
- cloud.google.com/go/secretmanager - GCP integration
- github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets - Azure integration

## [0.2.0] - 2026-01-16

### Added

- **Custom CA Certificate Support** - For corporate proxies and SSL inspection
  - `keeper.security/ca-cert-secret` - Load CA cert from Kubernetes Secret
  - `keeper.security/ca-cert-configmap` - Load CA cert from ConfigMap
  - Supports Zscaler, Palo Alto, Cisco Umbrella, and other SSL inspection tools
  - Automatic cert pool integration for all HTTPS connections
- **Go Template Support** - Industry-standard template rendering
  - Custom secret formatting with Go templates
  - 100+ Sprig template functions (date/time, crypto, string, encoding)
  - Build connection strings without JSON parsing
  - Conditional logic and default values
  - Template functions: base64enc/dec, sha256sum, sha512sum, upper, lower, trim, and more
- **Additional Secret Formats**
  - Properties format (Java .properties files)
  - YAML format
  - INI format
- **Template Examples** (examples/07-templates/)
  - Connection string templates
  - Properties file generation
  - Shell script generation
  - Conditional environment configs
- **Documentation**
  - docs/templates.md - Complete template guide with function reference
  - docs/comparison.md - Feature comparison with Vault, ESO, AWS CSI, 1Password
  - ROADMAP.md - Product roadmap and planned features
  - Professional tone improvements across all documentation
- **Development**
  - CLAUDE.md updated with development rules and professional tone guidelines
  - Feature request template

### Changed

- **BREAKING:** Minimum Kubernetes version: 1.25+ → 1.21+ (supports 4 more versions!)
- Go 1.23.12 → 1.25.6 (latest stable)
- Kubernetes libraries v0.32.0 → v0.34.3 (matches Vault & ESO)
- controller-runtime v0.19.3 → v0.22.4 (latest)

### Security

- **FIXED: All 15 vulnerabilities** (3 HIGH, 12 MEDIUM)
- golang.org/x/oauth2 v0.25.0 → v0.27.0 (CVE-2025-22868)
- golang.org/x/net v0.33.0 → v0.38.0 (CVE-2025-58183, CVE-2025-22870)
- Go stdlib vulnerabilities fixed by Go 1.25.6 upgrade
  - crypto/x509, net/http, encoding/asn1, encoding/pem, crypto/tls, net/url fixes

### Fixed

- Linter errors: unchecked error returns in Close(), Remove(), RemoveAll()
- De Morgan's law optimization in UID validation

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
