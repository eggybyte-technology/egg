# Makefile for egg framework
.PHONY: help build build-cli test lint clean tools generate release-snapshot release-test release-publish tag

# Default target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build & Development:"
	@echo "  build            - Build all modules"
	@echo "  build-cli        - Build egg CLI tool"
	@echo "  test             - Run tests for all modules"
	@echo "  lint             - Run linter on all modules"
	@echo "  clean            - Clean build artifacts"
	@echo "  tools            - Install required tools (including goreleaser)"
	@echo "  generate         - Generate code (if needed)"
	@echo "  example          - Build example service"
	@echo ""
	@echo "Release Management:"
	@echo "  release-snapshot - Build snapshot release locally (test)"
	@echo "  release-test     - Test release configuration"
	@echo "  tag              - Create a new version tag (interactive)"
	@echo "  release-publish  - Publish release to GitHub (requires GITHUB_TOKEN)"
	@echo ""
	@echo "Quality:"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  security         - Check for security vulnerabilities"
	@echo "  quality          - Run all quality checks"

# Build all modules
build:
	@echo "Building all modules..."
	@cd core && go build ./...
	@cd runtimex && go build ./...
	@cd connectx && go build ./...
	@cd configx && go build ./...
	@cd obsx && go build ./...
	@cd k8sx && go build ./...
	@cd storex && go build ./...
	@echo "Build completed successfully"

# Build egg CLI tool
build-cli:
	@echo "Building egg CLI tool..."
	@cd cli && rm -f egg && go build -o egg ./cmd/egg
	@chmod +x cli/egg
	@echo "CLI tool built successfully at cli/egg"

# Run tests for all modules
test:
	@echo "Running tests for all modules..."
	@cd core && go test -v ./...
	@cd runtimex && go test -v ./...
	@cd connectx && go test -v ./...
	@cd configx && go test -v ./...
	@cd obsx && go test -v ./...
	@cd k8sx && go test -v ./...
	@cd storex && go test -v ./...
	@echo "All tests passed"

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config .golangci.yml; \
	else \
		echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@find . -name "*.exe" -delete
	@find . -name "*.out" -delete
	@find . -name "*.test" -delete
	@find . -name "coverage.out" -delete
	@echo "Clean completed"

# Install required tools
tools:
	@echo "Installing required tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/goreleaser/goreleaser/v2@latest
	@echo "Tools installed successfully"

# Generate code (placeholder for future use)
generate:
	@echo "Code generation not implemented yet"

# Build example service
example:
	@echo "Building example service..."
	@cd examples/minimal-connect-service && go build -o minimal-connect-service .
	@echo "Example service built successfully"

# Run example service
run-example: example
	@echo "Running example service..."
	@cd examples/minimal-connect-service && ./minimal-connect-service

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

# Vet code
vet:
	@echo "Running go vet..."
	@go vet ./...
	@echo "Vet completed"

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Install it with: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

# Run all quality checks
quality: fmt vet test lint
	@echo "All quality checks completed"

# Setup development environment
setup: tools
	@echo "Setting up development environment..."
	@go work sync
	@echo "Development environment setup completed"

# Create release build
release: clean build test
	@echo "Release build completed successfully"

# Build snapshot release (local test without pushing)
release-snapshot:
	@echo "Building snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "goreleaser not found. Install it with: go install github.com/goreleaser/goreleaser/v2@latest"; \
	fi

# Test release configuration
release-test:
	@echo "Testing release configuration..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
		goreleaser build --snapshot --clean; \
	else \
		echo "goreleaser not found. Install it with: go install github.com/goreleaser/goreleaser/v2@latest"; \
	fi

# Create and push a release tag
tag:
	@echo "Current tags:"
	@git tag -l
	@echo ""
	@read -p "Enter new version tag (e.g., v0.0.1): " VERSION; \
	if [ -z "$$VERSION" ]; then \
		echo "Error: Version tag cannot be empty"; \
		exit 1; \
	fi; \
	echo "Creating tag $$VERSION..."; \
	git tag -a $$VERSION -m "Release $$VERSION"; \
	echo "Tag created. Push with: git push origin $$VERSION"

# Publish release (requires GITHUB_TOKEN)
release-publish:
	@echo "Publishing release..."
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "Error: GITHUB_TOKEN environment variable is not set"; \
		echo "Create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not found. Install it with: go install github.com/goreleaser/goreleaser/v2@latest"; \
	fi