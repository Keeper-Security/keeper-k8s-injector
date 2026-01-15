# Keeper K8s Injector Makefile
# All operations run inside Docker - no local Go required

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Docker configuration
REGISTRY ?= keeper
WEBHOOK_IMAGE ?= $(REGISTRY)/injector-webhook
SIDECAR_IMAGE ?= $(REGISTRY)/injector-sidecar
DEV_IMAGE ?= keeper-k8s-injector-dev

# Go configuration (used inside Docker)
GOOS ?= linux
GOARCH ?= amd64
CGO_ENABLED ?= 0

# Build flags
LDFLAGS := -w -s \
	-X main.Version=$(VERSION) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.BuildDate=$(BUILD_DATE)

# Docker run command for dev container
DOCKER_RUN = docker run --rm -v $(PWD):/app -w /app $(DEV_IMAGE)

.PHONY: all build test clean docker-build docker-push helm-lint dev-image

all: build

## Dev image (required for all other targets)
dev-image:
	@echo "Building development Docker image..."
	docker build -f Dockerfile.dev -t $(DEV_IMAGE) .

## Build targets (all run in Docker)

build: dev-image build-webhook build-sidecar

build-webhook: dev-image
	@echo "Building webhook (in Docker)..."
	$(DOCKER_RUN) sh -c "CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags='$(LDFLAGS)' -o bin/keeper-webhook ./cmd/webhook"

build-sidecar: dev-image
	@echo "Building sidecar (in Docker)..."
	$(DOCKER_RUN) sh -c "CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		go build -ldflags='$(LDFLAGS)' -o bin/keeper-sidecar ./cmd/sidecar"

## Test targets (all run in Docker)

test: dev-image
	@echo "Running all tests (in Docker)..."
	$(DOCKER_RUN) go test -v -race -coverprofile=coverage.out ./pkg/...

test-unit: dev-image
	@echo "Running unit tests (in Docker)..."
	$(DOCKER_RUN) go test -v -short ./pkg/...

test-integration: dev-image
	@echo "Running integration tests (in Docker)..."
	$(DOCKER_RUN) go test -v -run Integration ./...

test-verbose: dev-image
	@echo "Running tests with verbose output (in Docker)..."
	$(DOCKER_RUN) go test -v ./pkg/... 2>&1

coverage: dev-image
	@echo "Generating coverage report (in Docker)..."
	$(DOCKER_RUN) sh -c "go test -coverprofile=coverage.out ./pkg/... && go tool cover -html=coverage.out -o coverage.html"

## Lint targets (all run in Docker)

lint: dev-image
	@echo "Running linters (in Docker)..."
	$(DOCKER_RUN) golangci-lint run ./...

fmt: dev-image
	@echo "Formatting code (in Docker)..."
	$(DOCKER_RUN) sh -c "go fmt ./... && gofumpt -w . || true"

vet: dev-image
	@echo "Running go vet (in Docker)..."
	$(DOCKER_RUN) go vet ./...

## Docker targets

docker-build: docker-build-webhook docker-build-sidecar

docker-build-webhook:
	@echo "Building webhook Docker image..."
	docker build -f Dockerfile.webhook -t $(WEBHOOK_IMAGE):$(VERSION) .
	docker tag $(WEBHOOK_IMAGE):$(VERSION) $(WEBHOOK_IMAGE):latest

docker-build-sidecar:
	@echo "Building sidecar Docker image..."
	docker build -f Dockerfile.sidecar -t $(SIDECAR_IMAGE):$(VERSION) .
	docker tag $(SIDECAR_IMAGE):$(VERSION) $(SIDECAR_IMAGE):latest

docker-push: docker-push-webhook docker-push-sidecar

docker-push-webhook:
	@echo "Pushing webhook Docker image..."
	docker push $(WEBHOOK_IMAGE):$(VERSION)
	docker push $(WEBHOOK_IMAGE):latest

docker-push-sidecar:
	@echo "Pushing sidecar Docker image..."
	docker push $(SIDECAR_IMAGE):$(VERSION)
	docker push $(SIDECAR_IMAGE):latest

## Helm targets

helm-lint:
	@echo "Linting Helm chart..."
	helm lint charts/keeper-injector

helm-template:
	@echo "Rendering Helm templates..."
	helm template keeper-injector charts/keeper-injector

helm-package:
	@echo "Packaging Helm chart..."
	helm package charts/keeper-injector -d dist/

## Local development

dev-cluster:
	@echo "Creating Kind cluster..."
	kind create cluster --name keeper-injector-dev

dev-deploy: docker-build
	@echo "Loading images into Kind..."
	kind load docker-image $(WEBHOOK_IMAGE):$(VERSION) --name keeper-injector-dev
	kind load docker-image $(SIDECAR_IMAGE):$(VERSION) --name keeper-injector-dev
	@echo "Installing chart..."
	helm upgrade --install keeper-injector charts/keeper-injector \
		--set image.tag=$(VERSION) \
		--set sidecar.tag=$(VERSION)

dev-cleanup:
	@echo "Deleting Kind cluster..."
	kind delete cluster --name keeper-injector-dev

## Generate targets

generate:
	@echo "Running code generation..."
	go generate ./...

## Clean targets

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ dist/ coverage.out coverage.html

## Dependencies (all run in Docker)

deps: dev-image
	@echo "Downloading dependencies (in Docker)..."
	$(DOCKER_RUN) go mod download

deps-update: dev-image
	@echo "Updating dependencies (in Docker)..."
	$(DOCKER_RUN) sh -c "go get -u ./... && go mod tidy"

mod-tidy: dev-image
	@echo "Tidying go.mod (in Docker)..."
	$(DOCKER_RUN) go mod tidy

## Quick commands for development

quick-test: dev-image
	@echo "Quick test - compiling and running basic tests..."
	$(DOCKER_RUN) sh -c "go build ./... && go test -v ./pkg/config/... ./pkg/sidecar/..."

shell: dev-image
	@echo "Opening shell in dev container..."
	docker run -it --rm -v $(PWD):/app -w /app $(DEV_IMAGE) /bin/bash

## Help

help:
	@echo "Keeper K8s Injector Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  build           Build all binaries"
	@echo "  test            Run all tests"
	@echo "  lint            Run linters"
	@echo "  docker-build    Build Docker images"
	@echo "  docker-push     Push Docker images"
	@echo "  helm-lint       Lint Helm chart"
	@echo "  helm-package    Package Helm chart"
	@echo "  dev-cluster     Create Kind development cluster"
	@echo "  dev-deploy      Deploy to Kind cluster"
	@echo "  clean           Clean build artifacts"
	@echo "  help            Show this help"
