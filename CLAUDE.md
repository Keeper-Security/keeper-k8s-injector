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
