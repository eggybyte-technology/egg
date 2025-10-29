# Makefile for egg examples
.PHONY: help build test docker-build docker-clean \
	deploy-up deploy-down deploy-restart deploy-logs deploy-status \
	infra-up infra-down infra-restart infra-status infra-clean \
	services-up services-down services-restart

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

define print_info
	$(call log,print_info,$(1))
endef

help:
	@source $(LOGGER); \
	echo ""; \
	printf "$${BOLD}$${BLUE}Egg Examples - Build, Test & Deploy$${RESET}\n"; \
	echo ""; \
	echo "Build & Test:"; \
	printf "  $${CYAN}build$${RESET}              - Build all example services\n"; \
	printf "  $${CYAN}test$${RESET}               - Test example services (full integration)\n"; \
	printf "  $${CYAN}docker-build$${RESET}       - Build Docker images for examples\n"; \
	printf "  $${CYAN}docker-clean$${RESET}       - Clean Docker images\n"; \
	echo ""; \
	echo "Deployment (All Services):"; \
	printf "  $${CYAN}deploy-up$${RESET}          - Start all services (infra + app)\n"; \
	printf "  $${CYAN}deploy-down$${RESET}        - Stop all services\n"; \
	printf "  $${CYAN}deploy-restart$${RESET}     - Restart all services\n"; \
	printf "  $${CYAN}deploy-logs$${RESET}        - Show service logs\n"; \
	printf "  $${CYAN}deploy-status$${RESET}      - Show service status\n"; \
	echo ""; \
	echo "Infrastructure Only:"; \
	printf "  $${CYAN}infra-up$${RESET}           - Start infrastructure (MySQL, Jaeger, OTEL)\n"; \
	printf "  $${CYAN}infra-down$${RESET}         - Stop infrastructure\n"; \
	printf "  $${CYAN}infra-restart$${RESET}      - Restart infrastructure\n"; \
	printf "  $${CYAN}infra-status$${RESET}       - Show infrastructure status\n"; \
	printf "  $${CYAN}infra-clean$${RESET}        - Clean infrastructure (including volumes)\n"; \
	echo ""; \
	echo "Application Services Only:"; \
	printf "  $${CYAN}services-up$${RESET}        - Start application services\n"; \
	printf "  $${CYAN}services-down$${RESET}      - Stop application services\n"; \
	printf "  $${CYAN}services-restart$${RESET}   - Restart application services\n"

# Build all example services
build:
	$(call print_header,Building example services)
	@for dir in */; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "Building $$dir..."; \
			cd "$$dir" && go build ./... && cd ..; \
		fi \
	done
	$(call print_success,Example services built successfully)

# Test example services (full integration test)
test:
	$(call print_header,Testing example services)
	@./scripts/test-examples.sh examples
	$(call print_success,Example services tests completed)

# Build Docker images for all example services
docker-build:
	$(call print_header,Building Docker images for examples)
	@./scripts/build-examples.sh all
	$(call print_success,All example service images built successfully)

# Clean Docker images
docker-clean:
	$(call print_header,Cleaning Docker images)
	@./scripts/build-examples.sh clean
	$(call print_success,Docker cleanup completed)

# ==============================================================================
# Deployment Targets
# ==============================================================================

# Start all services (infrastructure + application)
deploy-up: infra-up services-up
	$(call print_success,All services started)

# Stop all services
deploy-down: services-down infra-down
	$(call print_success,All services stopped)

# Restart all services
deploy-restart: services-restart
	$(call print_success,All services restarted)

# Show service logs
deploy-logs:
	$(call print_header,Showing service logs)
	@cd deploy && docker-compose -f docker-compose.infra.yaml logs -f &
	@cd deploy && docker-compose -f docker-compose.services.yaml logs -f

# Show service status
deploy-status:
	$(call print_header,Infrastructure status)
	@cd deploy && docker-compose -f docker-compose.infra.yaml ps
	@echo ""
	$(call print_header,Application services status)
	@cd deploy && docker-compose -f docker-compose.services.yaml ps

# ==============================================================================
# Infrastructure Management
# ==============================================================================

# Start infrastructure services
infra-up:
	$(call print_header,Starting infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml up -d
	$(call print_success,Infrastructure services started)

# Stop infrastructure services
infra-down:
	$(call print_header,Stopping infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml down
	$(call print_success,Infrastructure services stopped)

# Restart infrastructure services
infra-restart:
	$(call print_header,Restarting infrastructure services)
	@cd deploy && docker-compose -f docker-compose.infra.yaml restart
	$(call print_success,Infrastructure services restarted)

# Show infrastructure status
infra-status:
	$(call print_header,Infrastructure services status)
	@cd deploy && docker-compose -f docker-compose.infra.yaml ps

# Clean infrastructure (including volumes)
infra-clean:
	$(call print_header,Cleaning infrastructure)
	@cd deploy && docker-compose -f docker-compose.infra.yaml down -v
	$(call print_success,Infrastructure cleaned (including volumes))

# ==============================================================================
# Application Services Management
# ==============================================================================

# Start application services
services-up:
	$(call print_header,Starting application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml up -d
	$(call print_success,Application services started)

# Stop application services
services-down:
	$(call print_header,Stopping application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml down
	$(call print_success,Application services stopped)

# Restart application services
services-restart:
	$(call print_header,Restarting application services)
	@cd deploy && docker-compose -f docker-compose.services.yaml restart
	$(call print_success,Application services restarted)

