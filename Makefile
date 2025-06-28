.PHONY: help test test-verbose test-coverage lint fmt vet build clean install run

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

test: ## Run tests
	go test ./...

test-verbose: ## Run tests with verbose output
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-race: ## Run tests with race detection
	go test -race ./...

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code
	gofmt -s -w .
	go mod tidy

vet: ## Run go vet
	go vet ./...

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

build: ## Build the binary
	go build -o gh-search-docs .

build-all: ## Build binaries for all platforms
	GOOS=linux GOARCH=amd64 go build -o dist/gh-search-docs-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o dist/gh-search-docs-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o dist/gh-search-docs-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o dist/gh-search-docs-windows-amd64.exe .

install: ## Install the binary
	go install .

clean: ## Clean build artifacts
	rm -f gh-search-docs
	rm -f coverage.out coverage.html
	rm -rf dist/

deps: ## Download dependencies
	go mod download
	go mod verify

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

run: ## Run the application (pass args with ARGS="...")
	go run . $(ARGS)

# CI targets
ci-test: ## Run tests for CI
	go test -race -coverprofile=coverage.out -covermode=atomic ./...

ci-lint: ## Run linting for CI
	golangci-lint run --timeout=5m

ci-check: fmt vet ci-lint ci-test ## Run all CI checks

# Development helpers
dev-setup: ## Set up development environment
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go mod download

watch-test: ## Watch files and run tests on changes (requires entr)
	find . -name "*.go" | entr -c make test

# Release helpers
tag: ## Create and push a new tag (use with TAG=v1.0.0)
	@if [ -z "$(TAG)" ]; then echo "Usage: make tag TAG=v1.0.0"; exit 1; fi
	git tag $(TAG)
	git push origin $(TAG)
