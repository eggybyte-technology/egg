# Makefile for egg framework
.PHONY: help build build-cli test test-cli test-cli-keep lint clean tools generate \
	release-snapshot release-test release \
	tag-all tag-modules tag-cli delete-tags \
	fmt vet security quality setup example run-example

# Go modules that need independent tags
MODULES := core runtimex connectx configx obsx k8sx storex

# Default target
help:
	@echo "🥚 Egg Framework - Build & Release Management"
	@echo ""
	@echo "Build & Development:"
	@echo "  build            - Build all modules"
	@echo "  build-cli        - Build egg CLI tool"
	@echo "  test             - Run tests for all modules"
	@echo "  test-cli         - Run CLI integration tests"
	@echo "  lint             - Run linter on all modules"
	@echo "  clean            - Clean build artifacts"
	@echo "  tools            - Install required tools"
	@echo "  setup            - Setup development environment"
	@echo "  example          - Build example service"
	@echo "  run-example      - Run example service"
	@echo ""
	@echo "Release Management:"
	@echo "  release          - One-click release (Usage: make release VERSION=v0.0.1)"
	@echo "  release-snapshot - Build snapshot release locally (test)"
	@echo "  release-test     - Test release configuration"
	@echo "  tag-all          - Create tags for all modules (Usage: make tag-all VERSION=v0.0.1)"
	@echo "  tag-modules      - Create tags for Go modules only"
	@echo "  tag-cli          - Create tag for CLI only"
	@echo "  delete-tags      - Delete all tags for a version (Usage: make delete-tags VERSION=v0.0.1)"
	@echo ""
	@echo "Quality:"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  security         - Check for security vulnerabilities"
	@echo "  quality          - Run all quality checks (fmt, vet, test, lint)"

# Build all modules
build:
	@echo "📦 Building all modules..."
	@for module in $(MODULES); do \
		echo "  Building $$module..."; \
		cd $$module && go build ./... && cd .. || exit 1; \
	done
	@echo "✅ Build completed successfully"

# Build egg CLI tool
build-cli:
	@echo "📦 Building egg CLI tool..."
	@cd cli && rm -f egg && go build -o egg ./cmd/egg
	@chmod +x cli/egg
	@echo "✅ CLI tool built successfully at cli/egg"

# Run tests for all modules
test:
	@echo "🧪 Running tests for all modules..."
	@for module in $(MODULES); do \
		echo "  Testing $$module..."; \
		cd $$module && go test -race -cover ./... && cd .. || exit 1; \
	done
	@cd cli && go test -race -cover ./...
	@echo "✅ All tests passed"

# Run CLI integration tests
test-cli: build-cli
	@echo "🧪 Running CLI integration tests..."
	@./scripts/test-cli.sh
	@echo "✅ CLI integration tests passed"

# Run CLI integration tests and keep test directory
test-cli-keep: build-cli
	@echo "🧪 Running CLI integration tests (keeping test directory)..."
	@./scripts/test-cli.sh --keep
	@echo "✅ CLI integration tests passed"

# Run linter (requires golangci-lint)
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		for module in $(MODULES) cli; do \
			echo "  Linting $$module..."; \
			cd $$module && golangci-lint run ./... && cd .. || exit 1; \
		done; \
		echo "✅ Linting completed"; \
	else \
		echo "❌ golangci-lint not found. Install with: make tools"; \
		exit 1; \
	fi

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf dist/
	@find . -name "*.exe" -delete
	@find . -name "*.out" -delete
	@find . -name "*.test" -delete
	@find . -name "coverage.out" -delete
	@rm -f cli/egg
	@echo "✅ Clean completed"

# Install required tools
tools:
	@echo "🔧 Installing required tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@echo "✅ Tools installed successfully"

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@for module in $(MODULES) cli; do \
		echo "  Formatting $$module..."; \
		cd $$module && go fmt ./... && cd .. || exit 1; \
	done
	@echo "✅ Code formatted"

# Vet code
vet:
	@echo "🔍 Running go vet..."
	@for module in $(MODULES) cli; do \
		echo "  Vetting $$module..."; \
		cd $$module && go vet ./... && cd .. || exit 1; \
	done
	@echo "✅ Vet completed"

# Check for security vulnerabilities
security:
	@echo "🔒 Checking for security vulnerabilities..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		for module in $(MODULES) cli; do \
			echo "  Checking $$module..."; \
			cd $$module && govulncheck ./... && cd .. || exit 1; \
		done; \
		echo "✅ Security check completed"; \
	else \
		echo "❌ govulncheck not found. Install with: make tools"; \
		exit 1; \
	fi

# Run all quality checks
quality: fmt vet test lint
	@echo "✅ All quality checks completed"

# Setup development environment
setup: tools
	@echo "🔧 Setting up development environment..."
	@go work sync
	@echo "✅ Development environment setup completed"

# Build example service
example:
	@echo "📦 Building example service..."
	@cd examples/minimal-connect-service && go build -o minimal-connect-service .
	@echo "✅ Example service built successfully"

# Run example service
run-example: example
	@echo "🚀 Running example service..."
	@cd examples/minimal-connect-service && ./minimal-connect-service

# ==============================================================================
# Release Management
# ==============================================================================

# Build snapshot release (local test without pushing)
release-snapshot:
	@echo "📦 Building snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
		echo "✅ Snapshot built in dist/"; \
	else \
		echo "❌ goreleaser not found. Install with: make tools"; \
		exit 1; \
	fi

# Test release configuration
release-test:
	@echo "🧪 Testing release configuration..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check && \
		goreleaser build --snapshot --clean && \
		echo "✅ Release configuration is valid"; \
	else \
		echo "❌ goreleaser not found. Install with: make tools"; \
		exit 1; \
	fi

# ==============================================================================
# Tag Management
# ==============================================================================

# Create tags for Go modules only
# Usage: make tag-modules VERSION=v0.0.1
tag-modules:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make tag-modules VERSION=v0.0.1"; \
		exit 1; \
	fi
	@echo "🏷️  Creating module tags for version $(VERSION)..."
	@for module in $(MODULES); do \
		tag="$$module/$(VERSION)"; \
		echo "  Creating tag $$tag..."; \
		git tag -a "$$tag" -m "Release $$module $(VERSION)" || exit 1; \
	done
	@echo "✅ Module tags created successfully"
	@echo ""
	@echo "📌 Push tags with:"
	@echo "   git push origin --tags"

# Create tag for CLI only
# Usage: make tag-cli VERSION=v0.0.1
tag-cli:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make tag-cli VERSION=v0.0.1"; \
		exit 1; \
	fi
	@echo "🏷️  Creating CLI tag $(VERSION)..."
	@git tag -a "$(VERSION)" -m "Release CLI $(VERSION)" || exit 1
	@echo "✅ CLI tag created successfully"
	@echo ""
	@echo "📌 Push tag with:"
	@echo "   git push origin $(VERSION)"

# Create tags for all modules and CLI
# Usage: make tag-all VERSION=v0.0.1
tag-all:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make tag-all VERSION=v0.0.1"; \
		exit 1; \
	fi
	@echo "🏷️  Creating all tags for version $(VERSION)..."
	@echo ""
	@echo "Step 1: Creating module tags..."
	@for module in $(MODULES); do \
		tag="$$module/$(VERSION)"; \
		echo "  Creating tag $$tag..."; \
		git tag -a "$$tag" -m "Release $$module $(VERSION)" || exit 1; \
	done
	@echo ""
	@echo "Step 2: Creating CLI tag..."
	@git tag -a "$(VERSION)" -m "Release $(VERSION)" || exit 1
	@echo ""
	@echo "✅ All tags created successfully!"
	@echo ""
	@echo "📌 Tags created:"
	@for module in $(MODULES); do \
		echo "   - $$module/$(VERSION)"; \
	done
	@echo "   - $(VERSION) (CLI)"
	@echo ""
	@echo "📌 Push all tags with:"
	@echo "   git push origin --tags"

# Delete all tags for a version (locally and remotely)
# Usage: make delete-tags VERSION=v0.0.1
delete-tags:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make delete-tags VERSION=v0.0.1"; \
		exit 1; \
	fi
	@echo "🗑️  Deleting all tags for version $(VERSION)..."
	@echo ""
	@echo "⚠️  WARNING: This will delete tags both locally and remotely!"
	@read -p "Continue? (y/N) " confirm; \
	if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
		echo "Cancelled"; \
		exit 0; \
	fi
	@echo ""
	@echo "Deleting module tags..."
	@for module in $(MODULES); do \
		tag="$$module/$(VERSION)"; \
		echo "  Deleting $$tag..."; \
		git tag -d "$$tag" 2>/dev/null || echo "  Local tag not found"; \
		git push --delete origin "$$tag" 2>/dev/null || echo "  Remote tag not found"; \
	done
	@echo ""
	@echo "Deleting CLI tag..."
	@git tag -d "$(VERSION)" 2>/dev/null || echo "  Local tag not found"
	@git push --delete origin "$(VERSION)" 2>/dev/null || echo "  Remote tag not found"
	@echo ""
	@echo "✅ Tag deletion completed"

# ==============================================================================
# One-Click Release
# ==============================================================================

# One-click release with version (requires GITHUB_TOKEN)
# Usage: make release VERSION=v0.0.1
release:
	@if [ -z "$(VERSION)" ]; then \
		echo "❌ Error: VERSION is required"; \
		echo "Usage: make release VERSION=v0.0.1"; \
		exit 1; \
	fi
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "❌ Error: GITHUB_TOKEN environment variable is not set"; \
		echo "Create a token at: https://github.com/settings/tokens"; \
		exit 1; \
	fi
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "❌ goreleaser not found. Install with: make tools"; \
		exit 1; \
	fi
	@echo "=========================================================="
	@echo "  🚀 Releasing Egg Framework $(VERSION)"
	@echo "=========================================================="
	@echo ""
	@echo "Step 1: Checking for existing tags..."
	@existing_tags=0; \
	if git rev-parse $(VERSION) >/dev/null 2>&1; then \
		echo "  Found existing CLI tag $(VERSION)"; \
		existing_tags=1; \
	fi; \
	for module in $(MODULES); do \
		if git rev-parse "$$module/$(VERSION)" >/dev/null 2>&1; then \
			echo "  Found existing module tag $$module/$(VERSION)"; \
			existing_tags=1; \
		fi; \
	done; \
	if [ $$existing_tags -eq 1 ]; then \
		echo ""; \
		echo "⚠️  WARNING: Some tags already exist!"; \
		echo "This will delete and recreate all tags for version $(VERSION)"; \
		read -p "Continue? (y/N) " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "Cancelled"; \
			exit 0; \
		fi; \
		echo ""; \
		echo "Deleting existing tags..."; \
		git tag -d $(VERSION) 2>/dev/null || true; \
		git push --delete origin $(VERSION) 2>/dev/null || true; \
		for module in $(MODULES); do \
			git tag -d "$$module/$(VERSION)" 2>/dev/null || true; \
			git push --delete origin "$$module/$(VERSION)" 2>/dev/null || true; \
		done; \
		echo "✅ Old tags deleted"; \
	else \
		echo "  No existing tags found"; \
	fi
	@echo ""
	@echo "Step 2: Running quality checks..."
	@$(MAKE) test
	@echo ""
	@echo "Step 3: Creating module tags..."
	@for module in $(MODULES); do \
		tag="$$module/$(VERSION)"; \
		echo "  Creating $$tag..."; \
		git tag -a "$$tag" -m "Release $$module $(VERSION)" || exit 1; \
	done
	@echo ""
	@echo "Step 4: Creating CLI tag..."
	@git tag -a $(VERSION) -m "Release $(VERSION)" || exit 1
	@echo ""
	@echo "Step 5: Pushing all tags..."
	@git push origin --tags
	@echo ""
	@echo "Step 6: Running goreleaser..."
	@goreleaser release --clean
	@echo ""
	@echo "=========================================================="
	@echo "  ✅ Release $(VERSION) completed successfully!"
	@echo "=========================================================="
	@echo ""
	@echo "📦 Artifacts created:"
	@echo "   - CLI binaries (goreleaser)"
	@echo "   - Module tags for Go modules"
	@echo ""
	@echo "🔗 Check the release at:"
	@echo "   https://github.com/eggybyte-technology/egg/releases/tag/$(VERSION)"
	@echo ""
	@echo "📦 Users can now install modules with:"
	@for module in $(MODULES); do \
		echo "   go get github.com/eggybyte-technology/egg/$$module@$(VERSION)"; \
	done