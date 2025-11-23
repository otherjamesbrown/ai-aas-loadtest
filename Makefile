.PHONY: help build test docker-build docker-push deploy clean

# Variables
BINARY_NAME=load-test-worker
DOCKER_IMAGE?=otherjamesbrown/ai-aas-loadtest
VERSION?=latest
PLATFORMS=linux/amd64,linux/arm64

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) ./cmd/load-test-worker

run: ## Run locally (requires CONFIG_PATH env var)
	@echo "Running $(BINARY_NAME)..."
	go run ./cmd/load-test-worker

test: ## Run unit tests
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.txt ./...

test-integration: ## Run integration tests (requires platform running)
	@echo "Running integration tests..."
	go test -v -tags=integration ./test/integration/...

lint: ## Run linters
	@echo "Running golangci-lint..."
	golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/ dist/ coverage.txt
	go clean

##@ Docker

docker-build: ## Build Docker image
	@echo "Building Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

docker-build-multiarch: ## Build multi-architecture Docker image
	@echo "Building multi-arch Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker buildx build --platform $(PLATFORMS) -t $(DOCKER_IMAGE):$(VERSION) --push .

docker-push: ## Push Docker image
	@echo "Pushing Docker image $(DOCKER_IMAGE):$(VERSION)..."
	docker push $(DOCKER_IMAGE):$(VERSION)

##@ Deployment

deploy: ## Deploy to Kubernetes (requires kubectl context set)
	@echo "Deploying to Kubernetes..."
	./scripts/deploy.sh

deploy-example: ## Deploy example test (smoke test)
	@echo "Deploying smoke test..."
	./scripts/deploy.sh --config smoke-test

##@ Utilities

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	go mod verify

update-deps: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
