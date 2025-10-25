# Egg Framework Scripts

This directory contains all the scripts for the Egg Framework, organized with a unified logging system for consistent output formatting.

## 📁 Directory Structure

```
scripts/
├── logger.sh              # Unified logging system
├── test.sh                # Main test runner (CLI-focused)
├── test-examples.sh       # Example services testing
├── test-cli.sh           # CLI integration testing
├── test-cli-production.sh # Production CLI testing
├── build.sh              # Build system
├── deploy.sh             # Deployment management
├── cleanup-ports.sh      # Port cleanup utility
└── connect-tester/       # Connect service testing tool
    ├── main.go
    └── go.mod
```

## 🎨 Unified Logging System

All scripts now use a unified logging system via `logger.sh` for consistent formatting and colors.

### Available Log Functions

- `print_header "Title"` - Print a formatted header with borders
- `print_success "Message"` - Print a success message (green ✓)
- `print_error "Message"` - Print an error message (red ✗)
- `print_info "Message"` - Print an info message (cyan i)
- `print_warning "Message"` - Print a warning message (yellow !)
- `print_debug "Message"` - Print a debug message (only if DEBUG=true)
- `print_command "Command"` - Print a command being executed
- `print_section "Section"` - Print a formatted section header

### Color Scheme

- 🔵 **Blue**: Headers and section dividers
- 🟢 **Green**: Success messages and confirmations
- 🔴 **Red**: Error messages and failures
- 🟡 **Yellow**: Warnings and important notices
- 🔷 **Cyan**: Information and progress updates
- 🟣 **Magenta**: Commands and technical details

## 🧪 Testing Scripts

### Main Test Runner (`test.sh`)

The main entry point for all testing operations.

```bash
# Show available commands
./scripts/test.sh help

# Test CLI functionality (local modules)
./scripts/test.sh cli

# Test CLI functionality (remote modules)
./scripts/test.sh production

# Test example services (rebuild & restart)
./scripts/test.sh examples

# Test example services (update only, no rebuild)
./scripts/test.sh examples-update

# Run all tests
./scripts/test.sh all

# Clean test artifacts
./scripts/test.sh clean
```

### Example Services Testing (`test-examples.sh`)

Comprehensive testing for example services with enhanced scenarios.

```bash
# Full test suite (rebuild and restart services)
./scripts/test-examples.sh examples

# Quick test (assumes services are already running)
./scripts/test-examples.sh examples-update

# Show help
./scripts/test-examples.sh help
```

**Enhanced Testing Features:**
- ✅ Multiple user CRUD operations
- ✅ Concurrent performance testing
- ✅ Error scenario validation
- ✅ Data validation and business logic testing
- ✅ Pagination and large dataset testing
- ✅ Connect endpoint comprehensive testing

### CLI Testing (`test-cli.sh`)

Tests the Egg CLI tool with real project generation and validation.

```bash
# Basic CLI testing
./scripts/test-cli.sh

# Keep test directory after completion
./scripts/test-cli.sh --keep
```

### Production CLI Testing (`test-cli-production.sh`)

Tests CLI with remote dependencies and multi-platform Docker builds.

```bash
# Production testing
./scripts/test-cli-production.sh

# Keep test directory after completion
./scripts/test-cli-production.sh --keep
```

## 🏗️ Build Scripts

### Build System (`build.sh`)

Manages building of services and Docker images.

```bash
# Build base image
./scripts/build.sh base

# Build a specific service
./scripts/build.sh service examples/minimal-connect-service minimal-connect-service

# Build all services
./scripts/build.sh all

# Clean build artifacts
./scripts/build.sh clean
```

## 🚀 Deployment Scripts

### Deployment Management (`deploy.sh`)

Manages Docker Compose services.

```bash
# Start all services
./scripts/deploy.sh up

# Stop all services
./scripts/deploy.sh down

# Restart all services
./scripts/deploy.sh restart

# Show logs
./scripts/deploy.sh logs

# Show status
./scripts/deploy.sh status

# Health check
./scripts/deploy.sh health

# Clean deployment artifacts
./scripts/deploy.sh clean
```

## 🧹 Utility Scripts

### Port Cleanup (`cleanup-ports.sh`)

Ensures all required ports are free before starting services.

```bash
./scripts/cleanup-ports.sh
```

### Connect Tester (`connect-tester/`)

Comprehensive testing tool for Connect services.

```bash
# Build the tester
cd scripts/connect-tester && go build -o connect-tester main.go

# Test full service suite
./connect-tester http://localhost:8080 minimal-service
./connect-tester http://localhost:8082 user-service

# Test single operations (for scripting)
./connect-tester http://localhost:8082 user-service create user@example.com "Test User"
./connect-tester http://localhost:8082 user-service get <user_id>
./connect-tester http://localhost:8082 user-service list 1 10
```

## 🔧 Development Guidelines

### Adding New Scripts

1. **Source the logger**: Add `source "$SCRIPT_DIR/logger.sh"` at the top
2. **Use project root**: Use `PROJECT_ROOT="$(get_project_root)"` for consistent paths
3. **Use log functions**: Replace echo/print statements with appropriate log functions
4. **Follow naming**: Use descriptive names and consistent structure

### Log Function Usage

```bash
#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

init_logging "My Script"              # Initialize logging
print_header "Starting Operation"     # Section headers
print_info "Processing..."            # Progress updates
print_success "Completed"             # Success messages
print_warning "Check this"            # Warnings
print_error "Failed"                  # Error messages
finalize_logging $? "My Script"       # Exit with proper code
```

### Error Handling

```bash
# Check commands exist
check_command "docker" "Docker is required for this script"

# Check files exist
check_file "config.yaml" "Configuration file not found"

# Check directories exist
check_directory "examples" "Examples directory not found"

# Wait for conditions
wait_for_condition "curl -f http://localhost:8080/health" 30 "Service health"

# Exit with appropriate messages
exit_with_error "Something went wrong"
exit_with_success "All operations completed"
```

## 🎯 Script Features

- **Consistent Formatting**: All scripts use the same visual style and colors
- **Error Handling**: Comprehensive error checking and reporting
- **Progress Tracking**: Clear progress indicators and status updates
- **Help Systems**: Built-in help for all major scripts
- **Cross-Platform**: Compatible with Linux and macOS
- **Extensible**: Easy to add new scripts following the same patterns

## 📋 Testing Workflow

1. **Development Testing**: Use `make test-examples UPDATE_ONLY=true` for quick tests
2. **Full Testing**: Use `make test-examples` for complete rebuild and test
3. **CLI Testing**: Use `make test-cli` for CLI functionality validation
4. **Production Testing**: Use `make test-cli-production` for production readiness

All scripts are designed to work together seamlessly and provide comprehensive testing coverage for the entire Egg Framework ecosystem.




