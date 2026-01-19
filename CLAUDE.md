# Claude Instructions for keeper-k8s-injector

## Git Commit Rules

- **NEVER** add Claude as a co-author in commits
- **NEVER** include `Co-Authored-By: Claude` or similar lines
- Commits should only have the human author

## Project Context

This is the Keeper Kubernetes Secrets Injector - a mutating admission webhook that injects secrets from Keeper Secrets Manager into Kubernetes pods at runtime.

## Tech Stack

- Go 1.23+
- Kubernetes 1.25+
- controller-runtime for webhook framework
- Keeper Secrets Manager Go SDK

## Development Rules

- **ALWAYS** run code, tests, and linting in Docker containers
- **NEVER** run Go commands or linters directly on the host
- Use `docker run` or the dev Dockerfile for all development tasks

## Pre-Commit Checklist (MANDATORY)

**Before EVERY git push, run locally in Docker:**

1. **Build test:**
   ```bash
   docker run --rm -v $(pwd):/app -w /app golang:1.25.6 go build -buildvcs=false ./cmd/webhook ./cmd/sidecar
   ```

2. **Run all tests:**
   ```bash
   docker run --rm -v $(pwd):/app -w /app golang:1.25.6 go test ./...
   ```

3. **Run linter:**
   ```bash
   docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.65.2 golangci-lint run --timeout=5m
   ```

4. **Only after all pass:** git push

**If CI fails after push, you skipped local testing. Don't let this happen.**

## Documentation Style

- Use professional, matter-of-fact language
- Avoid marketing terms like "killer feature", "magic", "awesome"
- Avoid excessive exclamation marks in technical explanations
- Avoid phrases like "secure!", "powerful!", "amazing!"
- Use clear, technical descriptions instead of enthusiasm
- Emojis are acceptable in user-facing demo UIs, but minimize in READMEs

## Changelog Guidelines

- **Be factual, not promotional** - State what was added, not why it's good
- **No comparisons** - Don't mention competitors or "industry standard"
- **No marketing language** - Avoid "production-grade", "best practice", "enterprise"
- **No implementation details** - Remove Testing, Documentation, Dependencies sections
- **Concise** - List features, not explanations
- **Don't justify** - Don't explain "matches Vault" or "like ESO does"
- **Avoid "BREAKING"** - Just state the change
- **Keep Changed section minimal** - Or omit if changes are additions

Example of what NOT to write:
```
- Industry-standard template rendering (matches Vault)
- Production-grade resilience
- BREAKING: Minimum K8s version changed
- Testing: 15 comprehensive tests
```

Example of what TO write:
```
- Go template rendering with 100+ Sprig functions
- Retry with exponential backoff
- Kubernetes 1.21+ support
```

## Documentation Standards (MANDATORY)

**If it's not documented, it doesn't exist.**

### Rule: Implementation Must Match Documentation

Before any release:

1. **Verify all features are documented** in docs/features.md and docs/annotations.md
2. **Verify documentation matches implementation** exactly
3. **Add examples** for all new features in examples/
4. **Update both CHANGELOG files** (see Changelog Files section below)
5. **Verify no discrepancies** between docs and code

### Pre-Release Checklist

- [ ] All annotations documented in docs/annotations.md
- [ ] All formats/features documented in docs/features.md
- [ ] Working example exists in examples/
- [ ] CHANGELOG.md updated following guidelines
- [ ] Chart.yaml artifacthub.io/changes annotation updated
- [ ] No discrepancies between docs and code

### Changelog Files

**Two changelog files must be updated for every release:**

1. **CHANGELOG.md** (root directory)
   - Full detailed changelog following [Keep a Changelog](https://keepachangelog.com/) format
   - Includes all versions and changes
   - Detailed descriptions and examples

2. **charts/keeper-injector/Chart.yaml** (artifacthub.io/changes annotation)
   - Brief, structured changelog for ArtifactHub display
   - Only includes changes for the CURRENT version
   - Uses YAML format with kinds: `added`, `changed`, `deprecated`, `removed`, `fixed`, `security`
   - See `charts/keeper-injector/CHANGELOG-TEMPLATE.yaml` for format
   - Documented in `charts/keeper-injector/README.md` (Maintainers section)

### Enforcement

Claude must verify documentation accuracy before ANY release. Run comprehensive audit of:
- Documented annotations vs implemented annotations
- Documented formats vs implemented formats
- Documented auth methods vs implemented auth methods
- Example code vs actual behavior
- Both CHANGELOG.md and Chart.yaml updated
