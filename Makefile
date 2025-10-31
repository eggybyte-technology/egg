# Makefile for egg framework (monorepo root)
.PHONY: help test lint clean tools setup tidy coverage check quality \
	release delete-all-tags cli-release git-large-files git-large-objects reinit-workspace

# Logger script for unified output
LOGGER := ./scripts/logger.sh

# Helper function to call logger.sh functions
# Usage: $(call log,function_name,message)
define log
	@bash -c 'source $(LOGGER) && $(1) "$(2)"'
endef

# Color definitions for inline use (compatible with logger.sh)
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
CYAN := \033[0;36m
MAGENTA := \033[0;35m
BOLD := \033[1m
RESET := \033[0m

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

# Go modules that need independent tags (excluding cli and examples)
MODULES := clientx configx connectx core httpx k8sx logx obsx runtimex servicex storex testingx

# Default target
help:
	@echo "$(BOLD)$(BLUE)Egg Framework - Library Management$(RESET)"
	@echo ""
	@echo "$(BOLD)Core Development:$(RESET)"
	@echo "  $(CYAN)setup$(RESET)            - Setup development environment (install tools + init workspace)"
	@echo "  $(CYAN)reinit-workspace$(RESET) - Reinitialize entire Go workspace (delete & recreate all modules)"
	@echo "  $(CYAN)tidy$(RESET)             - Clean and update dependencies for all modules"
	@echo "  $(CYAN)test$(RESET)             - Run tests for all framework modules"
	@echo "  $(CYAN)lint$(RESET)             - Run linter on all modules (includes fmt + vet)"
	@echo "  $(CYAN)coverage$(RESET)         - Generate test coverage report (HTML + terminal)"
	@echo "  $(CYAN)check$(RESET)            - Quick validation (lint + test, no coverage)"
	@echo "  $(CYAN)quality$(RESET)          - Full quality check (tidy + lint + test + coverage)"
	@echo "  $(CYAN)clean$(RESET)            - Clean test artifacts and coverage files"
	@echo "  $(CYAN)tools$(RESET)            - Install required development tools"
	@echo ""
	@echo "$(BOLD)Sub-Projects:$(RESET)"
	@echo "  $(CYAN)CLI:$(RESET)             cd cli && make help"
	@echo "  $(CYAN)Examples:$(RESET)        cd examples && make help"
	@echo "  $(CYAN)Base Images:$(RESET)     cd base-images && make help"
	@echo ""
	@echo "$(BOLD)Release Management:$(RESET)"
	@echo "  $(CYAN)release$(RESET)          - Release all modules (Usage: make release VERSION=v0.3.0)"
	@echo "  $(CYAN)cli-release$(RESET)       - Release CLI tool (Usage: make cli-release CLI=v1.0.0 FW=v0.3.0)"
	@echo "  $(RED)delete-all-tags$(RESET)  - $(RED)$(BOLD)[DANGEROUS]$(RESET) Delete ALL version tags"
	@echo ""
	@echo "$(BOLD)Git Utilities:$(RESET)"
	@echo "  $(CYAN)git-large-files$(RESET)    - Check large files tracked by git (current working directory)"
	@echo "  $(CYAN)git-large-objects$(RESET)  - Check large objects in git history"
	@echo ""
	@echo "$(BOLD)Typical Workflow:$(RESET)"
	@echo "  1. $(CYAN)make setup$(RESET)       # First-time setup"
	@echo "  2. $(CYAN)make tidy$(RESET)        # Clean dependencies"
	@echo "  3. $(CYAN)make check$(RESET)       # Quick validation"
	@echo "  4. $(CYAN)make coverage$(RESET)    # Check test coverage"
	@echo "  5. $(CYAN)make quality$(RESET)     # Full check before release"
	@echo "  6. $(CYAN)make release VERSION=v0.x.y$(RESET)"

# Run tests for all framework modules
test:
	$(call print_header,Running tests for all framework modules)
	@source $(LOGGER); \
	failed_modules=""; \
	for module in $(MODULES); do \
		print_info "Testing $$module..."; \
		if ! (cd $$module && go test -race -cover ./...); then \
			failed_modules="$$failed_modules $$module"; \
		fi; \
	done; \
	if [ -n "$$failed_modules" ]; then \
		print_error "Tests failed in modules:$$failed_modules"; \
		exit 1; \
	fi
	$(call print_success,All framework tests passed)

# Run linter (requires golangci-lint)
lint:
	$(call print_header,Running linter on all modules)
	@source $(LOGGER); \
	if ! command -v golangci-lint >/dev/null 2>&1; then \
		print_error "golangci-lint not found. Install with: make tools"; \
		exit 1; \
	fi; \
	failed_modules=""; \
		for module in $(MODULES) cli; do \
		print_info "Linting $$module..."; \
		output=$$(cd $$module && golangci-lint run ./... 2>&1 | grep -v "level=warning" || true); \
		if [ -n "$$output" ]; then \
			echo "$$output"; \
			failed_modules="$$failed_modules $$module"; \
		fi; \
		done; \
	if [ -n "$$failed_modules" ]; then \
		print_error "Linting failed in modules:$$failed_modules"; \
		exit 1; \
	fi
	$(call print_success,Linting completed successfully)

# Clean test artifacts and coverage files
clean:
	$(call print_header,Cleaning test artifacts)
	@rm -rf dist/
	@find . -name "*.exe" -delete
	@find . -name "*.out" -delete
	@find . -name "*.test" -delete
	@find . -name "coverage.out" -delete
	@find . -name "coverage.html" -delete
	@rm -rf .coverage/
	$(call print_success,Clean completed)

# Install required tools
tools:
	$(call print_header,Installing required tools)
	$(call print_info,Installing golangci-lint...)
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(call print_info,Installing protobuf toolchain...)
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(call print_info,Installing release tools...)
	@go install github.com/goreleaser/goreleaser/v2@latest
	$(call print_info,Installing security tools...)
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	$(call print_success,All tools installed successfully)

# Tidy dependencies for all modules
tidy:
	$(call print_header,Tidying dependencies for all modules)
	@source $(LOGGER); \
		for module in $(MODULES) cli; do \
		print_info "Tidying $$module..."; \
		(cd $$module && go mod tidy) || exit 1; \
	done; \
	print_info "Syncing workspace..."; \
	go work sync
	$(call print_success,All modules tidied successfully)

# Generate test coverage report
coverage:
	$(call print_header,Generating test coverage report)
	@source $(LOGGER); \
	mkdir -p .coverage; \
	total_coverage=0; \
	module_count=0; \
	echo "" > .coverage/summary.txt; \
	for module in $(MODULES); do \
		print_info "Generating coverage for $$module..."; \
		(cd $$module && go test -coverprofile=../.coverage/$$module.out -covermode=atomic ./... >/dev/null 2>&1) || continue; \
		if [ -f .coverage/$$module.out ]; then \
			coverage=$$(go tool cover -func=.coverage/$$module.out | grep total | awk '{print $$3}' | sed 's/%//'); \
			printf "  %-15s %6.2f%%\n" "$$module:" "$$coverage" | tee -a .coverage/summary.txt; \
			total_coverage=$$(echo "$$total_coverage + $$coverage" | bc); \
			module_count=$$((module_count + 1)); \
		fi; \
		done; \
	if [ $$module_count -gt 0 ]; then \
		avg_coverage=$$(echo "scale=2; $$total_coverage / $$module_count" | bc); \
		echo "" | tee -a .coverage/summary.txt; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━" | tee -a .coverage/summary.txt; \
		printf "  Total Average:  %6.2f%%\n" "$$avg_coverage" | tee -a .coverage/summary.txt; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━" | tee -a .coverage/summary.txt; \
	fi; \
	print_info "Coverage reports saved to .coverage/"
	$(call print_success,Coverage analysis completed)

# Quick check (lint + test without coverage)
check: lint test
	$(call print_success,Quick check passed)

# Full quality check (tidy + lint + test + coverage)
quality: tidy lint test coverage
	$(call print_success,All quality checks completed)

# Setup development environment
setup: tools
	$(call print_header,Setting up development environment)
	@$(call print_info,Syncing workspace...)
	@go work sync
	@echo ""
	$(call print_success,Development environment ready!)
	@echo ""
	$(call print_info,Next steps:)
	@echo "  1. Run 'make tidy' to clean dependencies"
	@echo "  2. Run 'make check' for quick validation"
	@echo "  3. Run 'make coverage' to see test coverage"

# Reinitialize entire Go workspace
# This will delete all go.mod, go.sum, go.work files and recreate them from scratch
reinit-workspace:
	@./scripts/reinit-workspace.sh

# ==============================================================================
# Release Management
# ==============================================================================

# Release all modules with specified version using the unified release script
# Usage: make release VERSION=v0.3.0
release:
	@if [ -z "$(VERSION)" ]; then \
		$(call print_error,VERSION is required); \
		echo "Usage: make release VERSION=v0.3.0"; \
		exit 1; \
	fi
	$(call print_header,Releasing Egg Framework $(VERSION))
	@./scripts/release.sh $(VERSION)

# Release CLI tool independently with specified version
# Usage: make cli-release CLI=v1.0.0 FW=v0.3.0
# Also supports: make cli-release VERSION=v1.0.0 FRAMEWORK_VERSION=v0.3.0
# FRAMEWORK_VERSION is REQUIRED - must specify the framework version to use
cli-release:
	@CLI_VERSION="$(CLI)"; \
	FW_VERSION="$(FW)"; \
	if [ -z "$$CLI_VERSION" ] && [ -n "$(VERSION)" ]; then \
		CLI_VERSION="$(VERSION)"; \
	fi; \
	if [ -z "$$FW_VERSION" ] && [ -n "$(FRAMEWORK_VERSION)" ]; then \
		FW_VERSION="$(FRAMEWORK_VERSION)"; \
	fi; \
	if [ -z "$$CLI_VERSION" ]; then \
		echo "$(RED)Error: CLI version is required$(RESET)"; \
		echo "Usage: make cli-release CLI=v1.0.0 FW=v0.3.0"; \
		echo "   or: make cli-release VERSION=v1.0.0 FRAMEWORK_VERSION=v0.3.0"; \
		exit 1; \
	fi; \
	if [ -z "$$FW_VERSION" ]; then \
		echo "$(RED)Error: Framework version is required$(RESET)"; \
		echo "Usage: make cli-release CLI=v1.0.0 FW=v0.3.0"; \
		echo "   or: make cli-release VERSION=v1.0.0 FRAMEWORK_VERSION=v0.3.0"; \
		exit 1; \
	fi; \
	echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"; \
	echo "$(BOLD)$(BLUE)▶ Releasing Egg CLI $$CLI_VERSION with framework $$FW_VERSION$(RESET)"; \
	echo "$(BOLD)$(BLUE)━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━$(RESET)"; \
	./scripts/cli-release.sh $$CLI_VERSION --framework-version $$FW_VERSION

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
# Git Utilities
# ==============================================================================

# Check large files tracked by git in current working directory
# Shows top 20 largest files currently tracked by git
git-large-files:
	$(call print_header,Checking large files tracked by git)
	@echo ""
	@echo "$(CYAN)Top 20 largest files in git working directory:$(RESET)"
	@echo ""
	@git ls-files -z | xargs -0 du -h 2>/dev/null | sort -hr | head -n 20 | \
		awk '{printf "  %8s  %s\n", $$1, substr($$0, index($$0,$$2))}' || \
		(echo "$(YELLOW)No tracked files found or git not initialized$(RESET)" && exit 0)
	@echo ""
	$(call print_info,To check files larger than a specific size, use:)
	@echo "  git ls-files -z | xargs -0 du -h | awk '\$$1 > \"10M\"'"

# Check large objects in git history
# Shows top 20 largest objects ever committed to git repository
git-large-objects:
	$(call print_header,Checking large objects in git history)
	@echo ""
	@echo "$(CYAN)Top 20 largest objects in git history:$(RESET)"
	@echo ""
	@git rev-list --objects --all 2>/dev/null | \
		git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' 2>/dev/null | \
		awk '/^blob/ {print substr($$0,6)}' | \
		sort --numeric-sort --key=2 -r | \
		head -n 20 | \
		awk '{size=$$2; if(size>1073741824) printf "  %8.2f GB  %s\n", size/1073741824, substr($$0,index($$0,$$3)); \
		     else if(size>1048576) printf "  %8.2f MB  %s\n", size/1048576, substr($$0,index($$0,$$3)); \
		     else if(size>1024) printf "  %8.2f KB  %s\n", size/1024, substr($$0,index($$0,$$3)); \
		     else printf "  %8d B   %s\n", size, substr($$0,index($$0,$$3))}' || \
		(echo "$(YELLOW)Git repository not initialized or no history found$(RESET)" && exit 0)
	@echo ""
	$(call print_info,To remove large files from git history, use git-filter-repo or BFG Repo-Cleaner)
