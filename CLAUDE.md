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
