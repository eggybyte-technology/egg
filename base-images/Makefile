# Makefile for Egg Framework Base Images
.PHONY: help build-all build-builder build-runtime push-all clean

# Logger script for unified output
LOGGER := ../scripts/logger.sh

# Helper function to call logger.sh functions
# Usage: $(call log,function_name,message)
define log
	@bash -c 'source $(LOGGER) && $(1) "$(2)"'
endef

# Output formatting functions (using logger.sh)
define print_header
	$(call log,print_header,$(1))
endef

define print_success
	$(call log,print_success,$(1))
endef

define print_error
	$(call log,print_error,$(1))
endef

define print_info
	$(call log,print_info,$(1))
endef

define print_warning
	$(call log,print_warning,$(1))
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
	@source $(LOGGER); \
	echo ""; \
	printf "$${BOLD}$${BLUE}Egg Framework - Base Images$${RESET}\n"; \
	echo ""; \
	printf "$${BOLD}Purpose:$${RESET}\n"; \
	echo "  Build foundation Docker images for Egg services"; \
	echo ""; \
	printf "$${BOLD}Available Targets:$${RESET}\n"; \
	printf "  $${CYAN}build-all$${RESET}       - Build both builder and runtime images\n"; \
	printf "  $${CYAN}build-builder$${RESET}   - Build Go builder image (compilation environment)\n"; \
	printf "  $${CYAN}build-runtime$${RESET}   - Build Alpine runtime image (minimal execution environment)\n"; \
	printf "  $${CYAN}push-all$${RESET}        - Build and push all images to registry\n"; \
	printf "  $${CYAN}clean$${RESET}           - Remove local images\n"; \
	echo ""; \
	printf "$${BOLD}Configuration:$${RESET}\n"; \
	echo "  DOCKER_REGISTRY  = $(DOCKER_REGISTRY)"; \
	echo "  GO_VERSION       = $(GO_VERSION)"; \
	echo "  ALPINE_VERSION   = $(ALPINE_VERSION)"; \
	echo "  DOCKER_PLATFORM  = $(DOCKER_PLATFORM)"; \
	echo "  PUSH             = $(PUSH)"; \
	echo ""; \
	printf "$${BOLD}Usage Examples:$${RESET}\n"; \
	printf "  $${CYAN}make build-all$${RESET}                           # Build locally for current arch\n"; \
	printf "  $${CYAN}make build-all PUSH=true$${RESET}                 # Build multi-arch and push\n"; \
	printf "  $${CYAN}make build-builder GO_VERSION=1.25.2$${RESET}     # Build with specific Go version\n"; \
	echo ""; \
	printf "$${BOLD}Images:$${RESET}\n"; \
	echo "  Builder: $(BUILDER_IMAGE)"; \
	echo "  Runtime: $(RUNTIME_IMAGE)"

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
	@source $(LOGGER); \
	echo ""; \
	printf "$${YELLOW}This will remove the following images:$${RESET}\n"; \
	echo "  - $(BUILDER_IMAGE)"; \
	echo "  - $(DOCKER_REGISTRY)/eggybyte-go-builder:latest"; \
	echo "  - $(RUNTIME_IMAGE)"; \
	echo "  - $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest"; \
	echo ""; \
	read -p "Continue? (y/N): " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		print_info "Removing builder images..."; \
		docker rmi $(BUILDER_IMAGE) 2>/dev/null || true; \
		docker rmi $(DOCKER_REGISTRY)/eggybyte-go-builder:latest 2>/dev/null || true; \
		print_info "Removing runtime images..."; \
		docker rmi $(RUNTIME_IMAGE) 2>/dev/null || true; \
		docker rmi $(DOCKER_REGISTRY)/eggybyte-go-alpine:latest 2>/dev/null || true; \
		print_success "Images removed"; \
	else \
		print_info "Cancelled"; \
	fi

