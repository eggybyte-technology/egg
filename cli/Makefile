# Makefile for egg CLI
.PHONY: help build test test-integration test-production clean

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
	@echo "$(BLUE)Egg CLI - Build & Test$(RESET)"
	@echo ""
	@echo "Targets:"
	@echo "  $(CYAN)build$(RESET)              - Build egg CLI tool"
	@echo "  $(CYAN)test$(RESET)               - Run unit tests"
	@echo "  $(CYAN)test-integration$(RESET)   - Run CLI integration tests (local modules)"
	@echo "  $(CYAN)test-production$(RESET)    - Run CLI integration tests (remote modules)"
	@echo "  $(CYAN)clean$(RESET)              - Clean build artifacts"

# Build egg CLI tool
build:
	$(call print_header,Building egg CLI tool)
	@mkdir -p bin
	@rm -f bin/egg
	@go build -o bin/egg ./cmd/egg
	@chmod +x bin/egg
	$(call print_success,CLI tool built successfully at cli/bin/egg)

# Run unit tests
test:
	$(call print_header,Running CLI unit tests)
	@go test -race -cover ./...
	$(call print_success,CLI unit tests passed)

# Run CLI integration tests with local modules
test-integration: build
	$(call print_header,Running CLI integration tests with local modules)
	@cd .. && ./scripts/test-cli.sh
	$(call print_success,CLI integration tests passed)

# Run CLI integration tests with remote modules (production)
test-production: build
	$(call print_header,Running CLI integration tests with remote modules)
	@cd .. && ./scripts/test-cli-production.sh
	$(call print_success,CLI production integration tests passed)

# Clean build artifacts
clean:
	$(call print_header,Cleaning CLI build artifacts)
	@rm -rf bin/
	@rm -f egg
	$(call print_success,CLI clean completed)

