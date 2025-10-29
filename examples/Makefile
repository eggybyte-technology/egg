# Makefile for egg examples
.PHONY: help build test docker-build docker-clean \
	deploy-up deploy-down deploy-restart deploy-logs deploy-status \
	infra-up infra-down infra-restart infra-status infra-clean \
	services-up services-down services-restart

# Color definitions
GREEN := \033[0;32m
BLUE := \033[0;34m
CYAN := \033[0;36m
RESET := \033[0m

# Output formatting
define print_header
	@echo ""
	@echo "$(BLUE)================================================================================$(RESET)"
	@echo "$(BLUE)▶ $(1)$(RESET)"
	@echo "$(BLUE)================================================================================$(RESET)"
endef

define print_success
	@echo "$(GREEN)[SUCCESS]$(RESET) $(1)"
endef

help:
	@echo "$(BLUE)Egg Examples - Build, Test & Deploy$(RESET)"
	@echo ""
	@echo "Build & Test:"
	@echo "  $(CYAN)build$(RESET)              - Build all example services"
	@echo "  $(CYAN)test$(RESET)               - Test example services (full integration)"
	@echo "  $(CYAN)docker-build$(RESET)       - Build Docker images for examples"
	@echo "  $(CYAN)docker-clean$(RESET)       - Clean Docker images"
	@echo ""
	@echo "Deployment (All Services):"
	@echo "  $(CYAN)deploy-up$(RESET)          - Start all services (infra + app)"
	@echo "  $(CYAN)deploy-down$(RESET)        - Stop all services"
	@echo "  $(CYAN)deploy-restart$(RESET)     - Restart all services"
	@echo "  $(CYAN)deploy-logs$(RESET)        - Show service logs"
	@echo "  $(CYAN)deploy-status$(RESET)      - Show service status"
	@echo ""
	@echo "Infrastructure Only:"
	@echo "  $(CYAN)infra-up$(RESET)           - Start infrastructure (MySQL, Jaeger, OTEL)"
	@echo "  $(CYAN)infra-down$(RESET)         - Stop infrastructure"
	@echo "  $(CYAN)infra-restart$(RESET)      - Restart infrastructure"
	@echo "  $(CYAN)infra-status$(RESET)       - Show infrastructure status"
	@echo "  $(CYAN)infra-clean$(RESET)        - Clean infrastructure (including volumes)"
	@echo ""
	@echo "Application Services Only:"
	@echo "  $(CYAN)services-up$(RESET)        - Start application services"
	@echo "  $(CYAN)services-down$(RESET)      - Stop application services"
	@echo "  $(CYAN)services-restart$(RESET)   - Restart application services"

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

