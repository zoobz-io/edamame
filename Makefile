.PHONY: test test-unit test-integration test-bench test-all bench lint lint-fix coverage clean all help check ci install-tools install-hooks

.DEFAULT_GOAL := help

# Default target
all: test lint

# Display help - self-documenting via grep
help: ## Display available commands
	@echo "edamame Development Commands"
	@echo "============================"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# Run tests with race detector
test: ## Run all tests with race detector
	@echo "Running tests..."
	@go test -v -race ./...

# Run unit tests only (short mode)
test-unit: ## Run unit tests only (short mode)
	@echo "Running unit tests..."
	@go test -v -short -race $(shell go list ./... | grep -v '/testing/')

# Run benchmarks
bench: ## Run benchmarks (legacy alias)
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -benchtime=1s ./...

# Run linters
lint: ## Run linters
	@echo "Running linters..."
	@golangci-lint run --config=.golangci.yml --timeout=5m

# Run linters with auto-fix
lint-fix: ## Run linters with auto-fix
	@echo "Running linters with auto-fix..."
	@golangci-lint run --config=.golangci.yml --fix

# Generate coverage report
coverage: ## Generate coverage report (HTML)
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "Coverage report generated: coverage.html"

# Clean generated files
clean: ## Remove generated files
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@find . -name "*.test" -delete
	@find . -name "*.prof" -delete
	@find . -name "*.out" -delete

# Install development tools
install-tools: ## Install required development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2

# Install git pre-commit hook
install-hooks: ## Install git pre-commit hook
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make check' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

# Quick check - run tests and lint
check: test lint ## Quick validation (test + lint)
	@echo "All checks passed!"

# Run integration tests
test-integration: ## Run integration tests (Docker required)
	@echo "Running integration tests..."
	@go test -v -tags=integration ./testing/integration/...

# Run benchmarks
test-bench: ## Run performance benchmarks
	@echo "Running benchmarks..."
	@go test -v -bench=. -benchmem ./testing/benchmarks/...

# Run all tests (unit + integration)
test-all: test test-integration ## Run all tests (unit + integration)
	@echo "All tests passed!"

# CI simulation - what CI runs locally
ci: clean lint test test-integration coverage ## Full CI simulation
	@echo "Full CI simulation complete!"
