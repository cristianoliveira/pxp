BINARY_NAME := pxp
BUILD_DIR := bin
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X main.version=$(VERSION)"

.PHONY: help build clean run test test-short test-race test-cover fmt vet deps tidy install-hooks run-hooks version
help: ## Show available targets
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
build: ## Build pxp
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/pxp
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
run: ## Run pxp
	go run ./cmd/pxp
test: ## Run all tests
	go test ./...
test-short: ## Run short tests
	go test ./... -short
test-race: ## Run tests with the race detector
	go test ./... -race
test-cover: ## Run tests with coverage
	go test ./... -cover
fmt: ## Format Go files
	go fmt ./...
vet: ## Run go vet
	go vet ./...
deps: ## Download dependencies
	go mod download
tidy: ## Tidy dependencies
	go mod tidy
install-hooks: ## Install pre-commit hooks
	lefthook install
run-hooks: ## Run pre-commit hooks
	lefthook run pre-commit
version: ## Show version
	@echo "Version: $(VERSION)"
.DEFAULT_GOAL := help
