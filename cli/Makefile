# Makefile for egg CLI
.PHONY: help build test test-integration test-production clean

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
	printf "$${BOLD}$${BLUE}Egg CLI - Build & Test$${RESET}\n"; \
	echo ""; \
	echo "Targets:"; \
	printf "  $${CYAN}build$${RESET}              - Build egg CLI tool\n"; \
	printf "  $${CYAN}test$${RESET}               - Run unit tests\n"; \
	printf "  $${CYAN}test-integration$${RESET}   - Run CLI integration tests (local modules)\n"; \
	printf "  $${CYAN}test-production$${RESET}    - Run CLI integration tests (remote modules)\n"; \
	printf "  $${CYAN}clean$${RESET}              - Clean build artifacts\n"

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
	@./scripts/test-cli.sh
	$(call print_success,CLI integration tests passed)

# Run CLI integration tests with remote modules (production)
test-production: build
	$(call print_header,Running CLI integration tests with remote modules)
	@echo "Production test script not yet implemented"
	@echo "TODO: Create scripts/test-cli-production.sh for testing with remote modules"
	@exit 1

# Clean build artifacts
clean:
	$(call print_header,Cleaning CLI build artifacts)
	@rm -rf bin/
	@rm -f egg
	$(call print_success,CLI clean completed)

