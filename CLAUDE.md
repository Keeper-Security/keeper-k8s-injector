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
