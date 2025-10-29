# Makefile for Egg Framework Base Images
.PHONY: help build-all build-builder build-runtime push-all clean

# Color definitions for enhanced output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
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

# Configuration
DOCKER_REGISTRY ?= ghcr.io/eggybyte-technology
GO_VERSION ?= 1.25.1
ALPINE_VERSION ?= 3.22
FOUNDATION_VERSION ?= go$(GO_VERSION)-alpine$(ALPINE_VERSION)
BUILDER_IMAGE := $(DOCKER_REGISTRY)/eggybyte-go-builder:$(FOUNDATION_VERSION)
RUNTIME_IMAGE := $(DOCKER_REGISTRY)/eggybyte-go-alpine:$(FOUNDATION_VERSION)
DOCKER_PLATFORM ?= linux/amd64,linux/arm64
PUSH ?= false

# Default target
help:
	@echo "$(BOLD)$(BLUE)Egg Framework - Base Images$(RESET)"
	@echo ""
	@echo "$(BOLD)Purpose:$(RESET)"
	@echo "  Build foundation Docker images for Egg services"
	@echo ""
	@echo "$(BOLD)Available Targets:$(RESET)"
	@echo "  $(CYAN)build-all$(RESET)       - Build both builder and runtime images"
	@echo "  $(CYAN)build-builder$(RESET)   - Build Go builder image (compilation environment)"
	@echo "  $(CYAN)build-runtime$(RESET)   - Build Alpine runtime image (minimal execution environment)"
	@echo "  $(CYAN)push-all$(RESET)        - Build and push all images to registry"
	@echo "  $(CYAN)clean$(RESET)           - Remove local images"
	@echo ""
	@echo "$(BOLD)Configuration:$(RESET)"
	@echo "  DOCKER_REGISTRY  = $(DOCKER_REGISTRY)"
	@echo "  GO_VERSION       = $(GO_VERSION)"
	@echo "  ALPINE_VERSION   = $(ALPINE_VERSION)"
	@echo "  DOCKER_PLATFORM  = $(DOCKER_PLATFORM)"
	@echo "  PUSH             = $(PUSH)"
	@echo ""
	@echo "$(BOLD)Usage Examples:$(RESET)"
	@echo "  $(CYAN)make build-all$(RESET)                           # Build locally for current arch"
	@echo "  $(CYAN)make build-all PUSH=true$(RESET)                 # Build multi-arch and push"
	@echo "  $(CYAN)make build-builder GO_VERSION=1.25.2$(RESET)     # Build with specific Go version"
	@echo ""
	@echo "$(BOLD)Images:$(RESET)"
	@echo "  Builder: $(BUILDER_IMAGE)"
	@echo "  Runtime: $(RUNTIME_IMAGE)"

# Build all foundation images (builder + runtime)
build-all: build-builder build-runtime
	$(call print_success,Foundation images built successfully)
	@echo ""
	$(call print_info,Built images:)
	@echo "  - $(BUILDER_IMAGE)"
	@echo "  - $(RUNTIME_IMAGE)"

# Build and push all images
push-all:
	@$(MAKE) build-all PUSH=true

# Build builder image (Go compilation environment)
build-builder:
	$(call print_header,Building eggybyte-go-builder image)
	@if [ ! -f Dockerfile.builder ]; then \
		$(call print_error,Dockerfile.builder not found); \
		exit 1; \
	fi
	@if [ "$(PUSH)" = "true" ]; then \
		$(call print_info,Building and pushing $(BUILDER_IMAGE) for $(DOCKER_PLATFORM)...); \
		docker buildx build \
			--platform $(DOCKER_PLATFORM) \
			--push \
			-f Dockerfile.builder \
			-t $(BUILDER_IMAGE) \
			-t $(DOCKER_REGISTRY)/eggybyte-go-builder:latest \
			.; \
	else \
		$(call print_info,Building $(BUILDER_IMAGE) for local use...); \
		if echo "$(DOCKER_PLATFORM)" | grep -q ","; then \
			$(call print_warning,Multi-platform builds require PUSH=true); \
			$(call print_info,Building for linux/amd64 only...); \
			docker buildx build \
				--platform linux/amd64 \
				--load \
				-f Dockerfile.builder \
				-t $(BUILDER_IMAGE) \
				-t $(DOCKER_REGISTRY)/eggybyte-go-builder:latest \
				.; \
		else \
			docker buildx build \
				--platform $(DOCKER_PLATFORM) \
				--load \
				-f Dockerfile.builder \
				-t $(BUILDER_IMAGE) \
				-t $(DOCKER_REGISTRY)/eggybyte-go-builder:latest \
				.; \
		fi \
	fi
	$(call print_success,Builder image built: $(BUILDER_IMAGE))

# Build runtime image (Alpine-based minimal runtime)
build-runtime:
	$(call print_header,Building eggybyte-go-alpine runtime image)
	@if [ ! -f Dockerfile.runtime ]; then \
		$(call print_error,Dockerfile.runtime not found); \
		exit 1; \
	fi
	@if [ "$(PUSH)" = "true" ]; then \
		$(call print_info,Building and pushing $(RUNTIME_IMAGE) for $(DOCKER_PLATFORM)...); \
		docker buildx build \
			--platform $(DOCKER_PLATFORM) \
			--push \
			-f Dockerfile.runtime \
			-t $(RUNTIME_IMAGE) \
			-t $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest \
			.; \
	else \
		$(call print_info,Building $(RUNTIME_IMAGE) for local use...); \
		if echo "$(DOCKER_PLATFORM)" | grep -q ","; then \
			$(call print_warning,Multi-platform builds require PUSH=true); \
			$(call print_info,Building for linux/amd64 only...); \
			docker buildx build \
				--platform linux/amd64 \
				--load \
				-f Dockerfile.runtime \
				-t $(RUNTIME_IMAGE) \
				-t $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest \
				.; \
		else \
			docker buildx build \
				--platform $(DOCKER_PLATFORM) \
				--load \
				-f Dockerfile.runtime \
				-t $(RUNTIME_IMAGE) \
				-t $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest \
				.; \
		fi \
	fi
	$(call print_success,Runtime image built: $(RUNTIME_IMAGE))

# Clean local images
clean:
	$(call print_header,Cleaning local base images)
	@echo ""
	@echo "$(YELLOW)This will remove the following images:$(RESET)"
	@echo "  - $(BUILDER_IMAGE)"
	@echo "  - $(DOCKER_REGISTRY)/eggybyte-go-builder:latest"
	@echo "  - $(RUNTIME_IMAGE)"
	@echo "  - $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest"
	@echo ""
	@read -p "Continue? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		$(call print_info,Removing builder images...); \
		docker rmi $(BUILDER_IMAGE) 2>/dev/null || true; \
		docker rmi $(DOCKER_REGISTRY)/eggybyte-go-builder:latest 2>/dev/null || true; \
		$(call print_info,Removing runtime images...); \
		docker rmi $(RUNTIME_IMAGE) 2>/dev/null || true; \
		docker rmi $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest 2>/dev/null || true; \
		$(call print_success,Images removed); \
	else \
		$(call print_info,Cancelled); \
	fi

