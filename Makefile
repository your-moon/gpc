.PHONY: build test lint fmt clean install coverage help

# Variables
BINARY_NAME=preloadcheck
CMD_PATH=./cmd/preloadcheck
COVERAGE_FILE=coverage.out

# Default target
.DEFAULT_GOAL := help

## help: Display this help message
help:
	@echo "GORM Preload Checker - Makefile Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(CMD_PATH)
	@echo "✓ Build complete: ./$(BINARY_NAME)"

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(CMD_PATH)
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## test: Run all tests
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "✓ Tests passed"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -race ./...
	@echo "✓ Tests passed"

## coverage: Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -html=$(COVERAGE_FILE)
	@echo "✓ Coverage report generated: $(COVERAGE_FILE)"

## coverage-text: Show coverage in terminal
coverage-text:
	@echo "Generating coverage report..."
	@go test -cover ./...

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Running go vet..."; \
		go vet ./...; \
	fi
	@echo "✓ Linting complete"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "✓ Code formatted"

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "✓ Vet complete"

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	@go mod tidy
	@echo "✓ Modules tidied"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f $(BINARY_NAME)
	@rm -f $(COVERAGE_FILE)
	@echo "✓ Clean complete"

## run-testdata: Run analyzer on testdata
run-testdata: build
	@echo "Running analyzer on testdata..."
	@./$(BINARY_NAME) ./testdata/ || true

## run-correct: Run analyzer on correct test file
run-correct: build
	@echo "Running analyzer on correct.go..."
	@./$(BINARY_NAME) ./testdata/correct.go

## run-error: Run analyzer on file with errors
run-error: build
	@echo "Running analyzer on testdata.go (should show errors)..."
	@./$(BINARY_NAME) ./testdata/testdata.go || true

## run-examples: Run analyzer on all examples
run-examples: build
	@echo "Running analyzer on examples..."
	@./$(BINARY_NAME) ./examples/ || true

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "✓ All checks passed"

## ci: Run CI checks
ci: fmt vet test
	@echo "✓ CI checks passed"

## dev: Development setup
dev:
	@echo "Setting up development environment..."
	@go mod download
	@echo "✓ Development environment ready"

## all: Build and test everything
all: clean fmt vet lint test build
	@echo "✓ Build and test complete"

