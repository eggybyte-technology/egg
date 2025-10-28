# Makefile for egg framework
.PHONY: help build build-cli test test-cli test-cli-production test-examples test-all lint clean tools generate \
	release publish-modules delete-all-tags \
	fmt vet security quality setup \
	docker-build docker-build-alpine docker-backend docker-go-service docker-all docker-clean \
	deploy-up deploy-down deploy-restart deploy-logs deploy-status deploy-health deploy-clean deploy-ports \
	infra-up infra-down infra-restart infra-status infra-clean services-up services-down services-restart services-rebuild

# Color definitions for enhanced output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
MAGENTA := \033[0;35m
BOLD := \033[1m
RESET := \033[0m

# Output formatting functions
define print_header
	@echo ""
	@echo "$(BLUE)================================================================================$(RESET)"
	@echo "$(BLUE)$(BOLD)▶ $(1)$(RESET)"
	@echo "$(BLUE)================================================================================$(RESET)"
endef

define print_success
	@echo "$(GREEN)[SUCCESS]$(RESET) $(1)"
endef

define print_error
	@echo "$(RED)[ERROR]$(RESET) $(1)"
endef

define print_info
	@echo "$(CYAN)[INFO]$(RESET) $(1)"
endef

define print_warning
	@echo "$(YELLOW)[WARNING]$(RESET) $(1)"
endef

# Go modules that need independent tags (excluding cli and examples)
MODULES := clientx configx connectx core httpx k8sx logx obsx runtimex servicex storex testingx

# Default target
help:
	@echo "$(BOLD)$(BLUE)Egg Framework - Build & Release Management$(RESET)"
	@echo ""
	@echo "$(BOLD)Build & Development:$(RESET)"
	@echo "  $(CYAN)build$(RESET)            - Build all modules"
	@echo "  $(CYAN)build-cli$(RESET)        - Build egg CLI tool"
	@echo "  $(CYAN)test$(RESET)             - Run tests for all modules"
	@echo "  $(CYAN)test-cli$(RESET)         - Run CLI integration tests (local modules)"
	@echo "  $(CYAN)test-cli-production$(RESET) - Run CLI integration tests (remote modules)"
	@echo "  $(CYAN)test-examples$(RESET)    - Test example services (full test)"
	@echo "  $(CYAN)test-all$(RESET)         - Run all tests (CLI + examples)"
	@echo "  $(CYAN)lint$(RESET)             - Run linter on all modules"
	@echo "  $(CYAN)clean$(RESET)            - Clean build artifacts"
	@echo "  $(CYAN)tools$(RESET)            - Install required tools"
	@echo "  $(CYAN)setup$(RESET)           - Setup development environment"
	@echo ""
	@echo "$(BOLD)Release Management:$(RESET)"
	@echo "  $(CYAN)release$(RESET)          - Release all modules with specified version (Usage: make release VERSION=v0.1.0)"
	@echo "  $(CYAN)publish-modules$(RESET)  - Publish all modules with specified version (Usage: make publish-modules VERSION=v0.1.0)"
	@echo "  $(RED)delete-all-tags$(RESET)  - $(RED)$(BOLD)[DANGEROUS]$(RESET) Delete ALL version tags (requires 3 confirmations)"
	@echo ""
	@echo "$(BOLD)Quality:$(RESET)"
	@echo "  $(CYAN)fmt$(RESET)              - Format code"
	@echo "  $(CYAN)vet$(RESET)              - Run go vet"
	@echo "  $(CYAN)security$(RESET)         - Check for security vulnerabilities"
	@echo "  $(CYAN)quality$(RESET)          - Run all quality checks (fmt, vet, test, lint)"
	@echo ""
	@echo "$(BOLD)Docker & Containerization:$(RESET)"
	@echo "  $(CYAN)docker-build$(RESET)     - Pull eggybyte-go-alpine base image from remote registry"
	@echo "  $(CYAN)docker-build-alpine$(RESET) - Build and publish eggybyte-go-alpine base image (Usage: make docker-build-alpine [VERSION=latest] [PUSH=true])"
	@echo "  $(CYAN)docker-backend$(RESET)   - Build backend image (Usage: make docker-backend BINARY=service-name)"
	@echo "  $(CYAN)docker-go-service$(RESET) - Build Go service (Usage: make docker-go-service SERVICE=path BINARY=name [BUILD_PATH=path])"
	@echo "  $(CYAN)docker-all$(RESET)       - Build all example services"
	@echo "  $(CYAN)docker-clean$(RESET)    - Clean Docker images and containers"
	@echo ""
	@echo "$(BOLD)Deployment Scripts:$(RESET)"
	@echo "  $(CYAN)deploy-up$(RESET)        - Start all services (infra + services)"
	@echo "  $(CYAN)deploy-down$(RESET)      - Stop all services"
	@echo "  $(CYAN)deploy-restart$(RESET)   - Restart all services"
	@echo "  $(CYAN)deploy-logs$(RESET)      - Show service logs"
	@echo "  $(CYAN)deploy-status$(RESET)    - Show service status"
	@echo "  $(CYAN)deploy-health$(RESET)    - Check service health"
	@echo "  $(CYAN)deploy-ports$(RESET)     - Check and free required ports"
	@echo "  $(CYAN)deploy-clean$(RESET)     - Clean deployment artifacts"
	@echo ""
	@echo "$(BOLD)Infrastructure & Services Management:$(RESET)"
	@echo "  $(CYAN)infra-up$(RESET)         - Start infrastructure (MySQL, Jaeger, OTEL)"
	@echo "  $(CYAN)infra-down$(RESET)       - Stop infrastructure"
	@echo "  $(CYAN)infra-restart$(RESET)    - Restart infrastructure"
	@echo "  $(CYAN)infra-status$(RESET)     - Show infrastructure status"
	@echo "  $(CYAN)infra-clean$(RESET)      - Clean infrastructure (including volumes)"
	@echo "  $(CYAN)services-up$(RESET)      - Start application services only"
	@echo "  $(CYAN)services-down$(RESET)    - Stop application services only"
	@echo "  $(CYAN)services-restart$(RESET) - Restart application services only"
	@echo "  $(CYAN)services-rebuild$(RESET) - Rebuild and restart application services"

# Build all modules
build:
	$(call print_header,Building all modules)
	@for module in $(MODULES); do \
		$(call print_info,Building $$module...); \
		cd $$module && go build ./... && cd .. || exit 1; \
	done
	$(call print_success,Build completed successfully)

# Build egg CLI tool
build-cli:
	$(call print_header,Building egg CLI tool)
	@cd cli && rm -f egg && go build -o egg ./cmd/egg
	@chmod +x cli/egg
	$(call print_success,CLI tool built successfully at cli/egg)

# Run tests for all modules
test:
	$(call print_header,Running tests for all modules)
	@for module in $(MODULES); do \
		$(call print_info,Testing $$module...); \
		cd $$module && go test -race -cover ./... && cd .. || exit 1; \
	done
	@cd cli && go test -race -cover ./...
	$(call print_success,All tests passed)

# Run tests without race detection (for release)
test-no-race:
	$(call print_header,Running tests for all modules without race detection)
	@for module in $(MODULES); do \
		$(call print_info,Testing $$module...); \
		cd $$module && go test -cover ./... && cd .. || exit 1; \
	done
	@cd cli && go test -cover ./...
	$(call print_success,All tests passed)

# Run CLI integration tests
test-cli: build-cli
	$(call print_header,Running CLI integration tests with local modules)
	@./scripts/test-cli.sh
	$(call print_success,CLI integration tests passed)

# Run CLI integration tests with remote modules (production)
test-cli-production: build-cli
	$(call print_header,Running CLI integration tests with remote modules)
	@./scripts/test-cli-production.sh
	$(call print_success,CLI production integration tests passed)

# Test example services (default: rebuild and restart)
# Test example services (full test with rebuild)
test-examples:
	$(call print_header,Testing example services)
	@./scripts/test-examples.sh examples
	$(call print_success,Example services tests completed)

# Run all tests (CLI + examples)
test-all: build-cli
	$(call print_header,Running all tests)
	@bash scripts/test.sh all
	$(call print_success,All tests passed)

# Run linter (requires golangci-lint)
lint:
	$(call print_header,Running linter)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		for module in $(MODULES) cli; do \
			$(call print_info,Linting $$module...); \
			cd $$module && golangci-lint run ./... && cd .. || exit 1; \
		done; \
		$(call print_success,Linting completed); \
	else \
		$(call print_error,golangci-lint not found. Install with: make tools); \
		exit 1; \
	fi

# Clean build artifacts
clean:
	$(call print_header,Cleaning build artifacts)
	@rm -rf dist/
	@find . -name "*.exe" -delete
	@find . -name "*.out" -delete
	@find . -name "*.test" -delete
	@find . -name "coverage.out" -delete
	@rm -f cli/egg
	$(call print_success,Clean completed)

# Install required tools
tools:
	$(call print_header,Installing required tools)
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/goreleaser/goreleaser/v2@latest
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	$(call print_success,Tools installed successfully)

# Format code
fmt:
	$(call print_header,Formatting code)
	@for module in $(MODULES) cli; do \
		$(call print_info,Formatting $$module...); \
		cd $$module && go fmt ./... && cd .. || exit 1; \
	done
	$(call print_success,Code formatted)

# Vet code
vet:
	$(call print_header,Running go vet)
	@for module in $(MODULES) cli; do \
		$(call print_info,Vetting $$module...); \
		cd $$module && go vet ./... && cd .. || exit 1; \
	done
	$(call print_success,Vet completed)

# Check for security vulnerabilities
security:
	$(call print_header,Checking for security vulnerabilities)
	@if command -v govulncheck >/dev/null 2>&1; then \
		for module in $(MODULES) cli; do \
			$(call print_info,Checking $$module...); \
			cd $$module && govulncheck ./... && cd .. || exit 1; \
		done; \
		$(call print_success,Security check completed); \
	else \
		$(call print_error,govulncheck not found. Install with: make tools); \
		exit 1; \
	fi

# Run all quality checks
quality: fmt vet test lint
	$(call print_success,All quality checks completed)

# Setup development environment
setup: tools
	$(call print_header,Setting up development environment)
	@go work sync
	$(call print_success,Development environment setup completed)

# ==============================================================================
# Release Management
# ==============================================================================

# Release all modules with specified version using the unified release script
# Usage: make release VERSION=v0.1.0
release:
	@if [ -z "$(VERSION)" ]; then \
		$(call print_error,VERSION is required); \
		echo "Usage: make release VERSION=v0.1.0"; \
		exit 1; \
	fi
	$(call print_header,Releasing Egg Framework $(VERSION))
	@./scripts/release.sh $(VERSION)

# Publish all modules with specified version (create tags and push)
# Usage: make publish-modules VERSION=v0.1.0
publish-modules:
	@if [ -z "$(VERSION)" ]; then \
		$(call print_error,VERSION is required); \
		echo "Usage: make publish-modules VERSION=v0.1.0"; \
		exit 1; \
	fi
	$(call print_header,Publishing all modules with version $(VERSION))
	@echo ""
	$(call print_info,Step 1: Running tests for all modules...)
	@$(MAKE) test-no-race
	@echo ""
	$(call print_info,Step 2: Checking for existing tags...)
	@existing_tags=0; \
	for module in $(MODULES); do \
		if git rev-parse "$$module/$(VERSION)" >/dev/null 2>&1; then \
			echo "$(YELLOW)[WARNING]$(RESET) Found existing tag $$module/$(VERSION)"; \
			existing_tags=1; \
		fi; \
	done; \
	if [ $$existing_tags -eq 1 ]; then \
		echo ""; \
		echo "$(YELLOW)[WARNING]$(RESET) WARNING: Some tags already exist!"; \
		echo "This will delete and recreate tags for version $(VERSION)"; \
		read -p "Continue? (y/N) " confirm; \
		if [ "$$confirm" != "y" ] && [ "$$confirm" != "Y" ]; then \
			echo "$(CYAN)[INFO]$(RESET) Cancelled"; \
			exit 0; \
		fi; \
		echo ""; \
		echo "$(CYAN)[INFO]$(RESET) Deleting existing tags..."; \
		for module in $(MODULES); do \
			git tag -d "$$module/$(VERSION)" 2>/dev/null || true; \
			git push --delete origin "$$module/$(VERSION)" 2>/dev/null || true; \
		done; \
		echo "$(GREEN)[SUCCESS]$(RESET) Old tags deleted"; \
	else \
		echo "$(CYAN)[INFO]$(RESET) No existing tags found"; \
	fi
	@echo ""
	$(call print_info,Step 3: Creating module tags...)
	@for module in $(MODULES); do \
		tag="$$module/$(VERSION)"; \
		echo "$(CYAN)[INFO]$(RESET) Creating $$tag..."; \
		git tag -a "$$tag" -m "Release $$module $(VERSION)" || exit 1; \
	done
	@echo ""
	$(call print_info,Step 4: Pushing all tags to remote...)
	@git push origin --tags
	@echo ""
	$(call print_header,Successfully published all modules with version $(VERSION)!)
	@echo ""
	$(call print_info,Tags created and pushed:)
	@for module in $(MODULES); do \
		echo "   ✓ $$module/$(VERSION)"; \
	done
	@echo ""
	$(call print_info,Users can now install modules with:)
	@for module in $(MODULES); do \
		echo "   go get go.eggybyte.com/egg/$$module@$(VERSION)"; \
	done

# Delete ALL version tags (DANGEROUS OPERATION!)
# This will delete all tags for all modules, both locally and remotely
# Requires 3 confirmations to proceed
delete-all-tags:
	$(call print_header,⚠️  DANGER: Delete ALL Version Tags)
	@echo ""
	@echo "$(RED)$(BOLD)⚠️  WARNING: This will delete ALL version tags for ALL modules!$(RESET)"
	@echo "$(RED)$(BOLD)⚠️  This operation is IRREVERSIBLE!$(RESET)"
	@echo ""
	@./scripts/release.sh --delete-all-tags

# ==============================================================================
# Docker & Containerization
# ==============================================================================

# Pull eggybyte-go-alpine base image
docker-build:
	$(call print_header,Pulling eggybyte-go-alpine base image)
	@./scripts/build.sh base
	$(call print_success,Base image pulled successfully)

# Build and publish eggybyte-go-alpine base image (multi-arch)
# Usage: make docker-build-alpine [VERSION=latest] [PUSH=true]
docker-build-alpine:
	$(call print_header,Building and publishing eggybyte-go-alpine base image)
	@if [ "$(PUSH)" = "true" ]; then \
		$(call print_info,Building and pushing image with version $(VERSION)...); \
		./scripts/build-eggybyte-go-alpine.sh $(VERSION) --push; \
	else \
		$(call print_info,Building image locally with version $(VERSION)...); \
		./scripts/build-eggybyte-go-alpine.sh $(VERSION); \
	fi
	$(call print_success,EggyByte Go Alpine base image build completed)

# Build backend image from pre-built binary
# Usage: make docker-backend BINARY=service-name [HTTP_PORT=8080] [HEALTH_PORT=8081] [METRICS_PORT=9091] [IMAGE=service-name:latest]
docker-backend:
	@if [ -z "$(BINARY)" ]; then \
		$(call print_error,BINARY is required); \
		echo "Usage: make docker-backend BINARY=service-name [HTTP_PORT=8080] [HEALTH_PORT=8081] [METRICS_PORT=9091] [IMAGE=service-name:latest]"; \
		exit 1; \
	fi
	$(call print_header,Building backend image for $(BINARY))
	@./scripts/build.sh service examples/$(BINARY) $(BINARY) . "$(HTTP_PORT)" "$(HEALTH_PORT)" "$(METRICS_PORT)" "$(IMAGE)"
	$(call print_success,Backend image built successfully)

# Build Go service (compile + build image)
# Usage: make docker-go-service SERVICE=examples/service BINARY=service-name [BUILD_PATH=.] [HTTP_PORT=8080] [HEALTH_PORT=8081] [METRICS_PORT=9091] [IMAGE=service-name:latest]
docker-go-service:
	@if [ -z "$(SERVICE)" ] || [ -z "$(BINARY)" ]; then \
		$(call print_error,SERVICE and BINARY are required); \
		echo "Usage: make docker-go-service SERVICE=examples/service BINARY=service-name [BUILD_PATH=.] [HTTP_PORT=8080] [HEALTH_PORT=8081] [METRICS_PORT=9091] [IMAGE=service-name:latest]"; \
		echo ""; \
		echo "Examples:"; \
		echo "  make docker-go-service SERVICE=examples/minimal-connect-service BINARY=minimal-connect-service"; \
		echo "  make docker-go-service SERVICE=examples/user-service BINARY=user-service BUILD_PATH=cmd/server"; \
		exit 1; \
	fi
	$(call print_header,Building Go service $(BINARY) from $(SERVICE))
	@./scripts/build.sh service "$(SERVICE)" "$(BINARY)" "$(BUILD_PATH)" "$(HTTP_PORT)" "$(HEALTH_PORT)" "$(METRICS_PORT)" "$(IMAGE)"
	$(call print_success,Go service built successfully)

# Build all example services
docker-all: docker-build
	$(call print_header,Building all example services)
	@./scripts/build.sh all
	$(call print_success,All example services built successfully)
	@echo ""
	$(call print_info,Available images:)
	@echo "  - localhost:5000/eggybyte-go-alpine:latest (base image)"
	@echo "  - minimal-connect-service:latest"
	@echo "  - user-service:latest"

# Clean Docker images and containers
docker-clean:
	$(call print_header,Cleaning Docker images and containers)
	@./scripts/build.sh clean
	$(call print_success,Docker cleanup completed)

# ==============================================================================
# Deployment Scripts
# ==============================================================================

# Start all services
deploy-up:
	$(call print_header,Starting all services)
	@./scripts/deploy.sh up
	$(call print_success,All services started)

# Stop all services
deploy-down:
	$(call print_header,Stopping all services)
	@./scripts/deploy.sh down
	$(call print_success,All services stopped)

# Restart all services
deploy-restart:
	$(call print_header,Restarting all services)
	@./scripts/deploy.sh restart
	$(call print_success,All services restarted)

# Show service logs
# Usage: make deploy-logs [SERVICE=service-name]
deploy-logs:
	$(call print_header,Showing service logs)
	@./scripts/deploy.sh logs $(SERVICE)
	$(call print_success,Logs displayed)

# Show service status
deploy-status:
	$(call print_header,Showing service status)
	@./scripts/deploy.sh status
	$(call print_success,Status displayed)

# Check service health
deploy-health:
	$(call print_header,Checking service health)
	@./scripts/deploy.sh health
	$(call print_success,Health check completed)

# Check and free required ports
deploy-ports:
	$(call print_header,Checking and freeing required ports)
	@./scripts/cleanup-ports.sh
	$(call print_success,Port check completed)

# Clean deployment artifacts
deploy-clean:
	$(call print_header,Cleaning deployment artifacts)
	@./scripts/deploy.sh clean
	$(call print_success,Deployment artifacts cleaned)

# Infrastructure management (separate from services)
infra-up:
	$(call print_header,Starting infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml up -d
	$(call print_success,Infrastructure services started)

infra-down:
	$(call print_header,Stopping infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml down
	$(call print_success,Infrastructure services stopped)

infra-restart:
	$(call print_header,Restarting infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml restart
	$(call print_success,Infrastructure services restarted)

infra-status:
	$(call print_header,Infrastructure services status)
	@cd deploy && docker-compose -f docker-compose.infra.yaml ps

infra-clean:
	$(call print_header,Cleaning infrastructure)
	@cd deploy && docker-compose -f docker-compose.infra.yaml down -v
	$(call print_success,Infrastructure cleaned (including volumes))

# Application services management (requires infrastructure)
services-up:
	$(call print_header,Starting application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml up -d
	$(call print_success,Application services started)

services-down:
	$(call print_header,Stopping application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml down
	$(call print_success,Application services stopped)

services-restart:
	$(call print_header,Restarting application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml restart
	$(call print_success,Application services restarted)

services-rebuild:
	$(call print_header,Rebuilding and restarting application services)
	@./scripts/build.sh all
	@cd deploy && docker-compose -f docker-compose.services.yaml down
	@cd deploy && docker-compose -f docker-compose.services.yaml up -d
	$(call print_success,Application services rebuilt and restarted)