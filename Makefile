# Makefile for ClippingKK CLI (Go version)

# Variables
BINARY_NAME=ck-cli
CMD_DIR=./cmd/ck-cli
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_FLAGS=-ldflags="-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

# Default target
.DEFAULT_GOAL := build

# Development commands
.PHONY: build
build: ## Build the CLI for current platform
	go build $(BUILD_FLAGS) -o $(BINARY_NAME) $(CMD_DIR)

.PHONY: build-release
build-release: ## Build optimized release binary
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -trimpath -o $(BINARY_NAME) $(CMD_DIR)

.PHONY: install
install: ## Install the CLI to GOPATH/bin
	go install $(BUILD_FLAGS) $(CMD_DIR)

.PHONY: clean
clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	go clean

# Testing
.PHONY: test
test: ## Run tests
	go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: bench
bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

# Code quality
.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint to be installed)
	golangci-lint run

.PHONY: mod-tidy
mod-tidy: ## Tidy go modules
	go mod tidy

# Development setup
.PHONY: deps
deps: ## Download dependencies
	go mod download

.PHONY: deps-update
deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

# Cross-compilation
.PHONY: build-linux
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64 $(CMD_DIR)

.PHONY: build-windows
build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

.PHONY: build-macos
build-macos: ## Build for macOS
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

.PHONY: build-all
build-all: build-linux build-windows build-macos ## Build for all platforms

# Release
.PHONY: release-dry
release-dry: ## Dry run release with goreleaser
	goreleaser release --snapshot --rm-dist

.PHONY: release
release: ## Release with goreleaser
	goreleaser release --rm-dist

# Docker
.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME):$(VERSION) .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run --rm -it $(BINARY_NAME):$(VERSION)

# Examples and testing
.PHONY: run-example
run-example: build ## Run example with test data
	@echo "Building and running example..."
	@if [ -f "./fixtures/clippings_en.txt" ]; then \
		./$(BINARY_NAME) parse --input ./fixtures/clippings_en.txt; \
	else \
		echo "No test fixtures found. Create a sample clippings file to test."; \
	fi

.PHONY: test-parse-stdin
test-parse-stdin: build ## Test parsing from stdin
	@echo "Testing stdin parsing..."
	@if [ -f "./fixtures/clippings_en.txt" ]; then \
		cat ./fixtures/clippings_en.txt | ./$(BINARY_NAME) parse; \
	else \
		echo "No test fixtures found."; \
	fi

# Development utilities
.PHONY: dev-setup
dev-setup: deps ## Set up development environment
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@if ! command -v goreleaser > /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi

.PHONY: check
check: fmt vet test ## Run all checks (format, vet, test)

.PHONY: all
all: clean deps check build ## Clean, download deps, run checks, and build

# Help
.PHONY: help
help: ## Show this help message
	@echo "ClippingKK CLI (Go) - Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make build                 # Build for current platform"
	@echo "  make test                  # Run tests"
	@echo "  make build-all             # Build for all platforms"
	@echo "  make run-example           # Build and test with sample data"