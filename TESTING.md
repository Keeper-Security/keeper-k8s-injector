# Testing Guide

All commands must be run in Docker containers (see CLAUDE.md for development rules).

## Unit Tests

Run all unit tests:

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -w /app \
  golang:1.25.6 sh -c "update-ca-certificates 2>/dev/null && go test ./..."
```

## Integration Tests

Integration tests use real Keeper vault credentials and make actual API calls.

### Prerequisites

1. Real KSM config file at `config.base64` in project root
2. Test records in vault:
   - `salesdb`
   - `demo-secret`
   - `postgres-credentials`
   - `Stripe POS API`

### Running Integration Tests

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -w /app \
  golang:1.25.6 sh -c "update-ca-certificates 2>/dev/null && go test -v -tags integration ./pkg/webhook/... -run Integration"
```

### Run Specific Integration Test

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -w /app \
  golang:1.25.6 sh -c "update-ca-certificates 2>/dev/null && go test -v -tags integration ./pkg/webhook/... -run TestIntegration_CreateK8sSecret_SingleRecord"
```

### Custom Config Path

Override config file location:

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -e KEEPER_CONFIG_FILE=/custom/path/config.base64 \
  -w /app \
  golang:1.25.6 sh -c "update-ca-certificates 2>/dev/null && go test -v -tags integration ./pkg/webhook/..."
```

## Linting

Run golangci-lint:

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -w /app \
  golangci/golangci-lint:v1.65.2 sh -c "update-ca-certificates 2>/dev/null && golangci-lint run --timeout=5m"
```

## Build Test

Verify code compiles:

```bash
docker run --rm \
  -v "$(pwd)":/app \
  -v /path/to/zscaler-root-cert.pem:/usr/local/share/ca-certificates/zscaler.crt:ro \
  -w /app \
  golang:1.25.6 sh -c "update-ca-certificates 2>/dev/null && go build -buildvcs=false ./cmd/webhook ./cmd/sidecar"
```

## Pre-Commit Checklist

Before every commit, run all three checks:

```bash
# 1. Build
docker run --rm -v $(pwd):/app -w /app golang:1.25.6 go build -buildvcs=false ./cmd/webhook ./cmd/sidecar

# 2. Tests
docker run --rm -v $(pwd):/app -w /app golang:1.25.6 go test ./...

# 3. Lint
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.65.2 golangci-lint run --timeout=5m
```

## Test Coverage

Current test suite:

- **Unit Tests**: 23 tests (config, webhook logic)
- **Integration Tests**: 11 tests (real Keeper API)
- **Total**: 34 tests

### Unit Tests by Package

- `pkg/config`: Annotation parsing, YAML config
- `pkg/webhook`: K8s Secret building, conflict resolution, size validation
- `pkg/ksm`: KSM client, folder trees, notation parsing
- `pkg/sidecar`: Agent lifecycle, caching, retry logic

### Integration Tests

- K8s Secret creation from real records
- Custom key mapping
- All conflict modes (overwrite, merge, skip, fail)
- Owner references (enabled/disabled)
- Batch fetching efficiency
- Error handling (missing secrets, size limits)
