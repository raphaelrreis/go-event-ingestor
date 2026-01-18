# Makefile for Go Event Ingestor
# Follows standard conventions.

# Variables
BINARY_NAME=ingestor
DOCKER_IMAGE=go-event-ingestor
DOCKER_TAG=latest
GO=go

.PHONY: help tidy build run test test-int bench lint docker-build docker-run clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

tidy: ## Run go mod tidy
	$(GO) mod tidy

build: ## Build the binary
	$(GO) build -o $(BINARY_NAME) cmd/ingestor/main.go

run: build ## Run the application locally
	./$(BINARY_NAME)

test: ## Run unit tests
	$(GO) test -v -race ./...

test-int: ## Run integration tests
	$(GO) test -v -tags=integration ./tests/integration/...

bench: ## Run benchmarks
	$(GO) test -bench=. ./tests/benchmark/...

lint: ## Run linters (go vet + golangci-lint if installed)
	$(GO) vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping"; \
	fi

docker-build: ## Build Docker image
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env $(DOCKER_IMAGE):$(DOCKER_TAG)

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	$(GO) clean
