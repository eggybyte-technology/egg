#!/bin/bash

# Unified test script for Egg Framework
# This script provides comprehensive testing functionality
#
# Usage:
#   ./scripts/test.sh [command] [options]
#
# Commands:
#   cli         Test CLI functionality (local modules)
#   production  Test CLI functionality (remote modules)
#   examples    Test example services (rebuild & restart)
#   examples-update Test example services (update only, no rebuild)
#   all         Run all tests
#   clean       Clean test artifacts
#
# Examples:
#   ./scripts/test.sh cli
#   ./scripts/test.sh production
#   ./scripts/test.sh examples
#   ./scripts/test.sh examples-update
#   ./scripts/test.sh all
#   ./scripts/test.sh clean

set -e

# Source the unified logger
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

# Get the project root directory
PROJECT_ROOT="$(get_project_root)"

# Test CLI functionality with local modules
test_cli() {
    print_header "Testing CLI functionality (local modules)"
    
    # Check if egg CLI is built
    if [ ! -f "$PROJECT_ROOT/cli/egg" ]; then
        print_info "Building egg CLI..."
        cd "$PROJECT_ROOT"
        make build-cli
        print_success "CLI built"
    fi
    
    # Run CLI tests
    print_info "Running CLI integration tests..."
    "$PROJECT_ROOT/scripts/test-cli.sh" "$@"
    
    print_success "CLI tests completed"
}

# Test CLI functionality with remote modules
test_production() {
    print_header "Testing CLI functionality (remote modules)"
    
    # Check if egg CLI is built
    if [ ! -f "$PROJECT_ROOT/cli/egg" ]; then
        print_info "Building egg CLI..."
        cd "$PROJECT_ROOT"
        make build-cli
        print_success "CLI built"
    fi
    
    # Run production tests
    print_info "Running production integration tests..."
    "$PROJECT_ROOT/scripts/test-cli-production.sh" "$@"
    
    print_success "Production tests completed"
}

# Test example services (delegated to separate script)
test_examples() {
    print_info "Delegating to test-examples.sh..."
    "$SCRIPT_DIR/test-examples.sh" examples
}

# Run all tests
test_all() {
    print_header "Running all tests"
    
    local overall_failed=0
    
    # Test CLI with local modules
    print_info "Starting CLI tests (local modules)..."
    if ! test_cli; then
        overall_failed=1
    fi
    
    # Test CLI with remote modules
    print_info "Starting CLI tests (remote modules)..."
    if ! test_production; then
        overall_failed=1
    fi
    
    # Test example services
    print_info "Starting example services tests..."
    if ! test_examples; then
        overall_failed=1
    fi
    
    if [ $overall_failed -eq 0 ]; then
        print_success "All tests completed successfully!"
    else
        print_warning "Some tests failed. Check the output above for details."
    fi
    
    return $overall_failed
}

# Clean test artifacts
clean_tests() {
    print_header "Cleaning test artifacts"
    
    print_info "Removing test directories..."
    rm -rf test-egg-project
    rm -rf test-egg-production
    
    print_info "Stopping any running containers..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose down --remove-orphans 2>/dev/null || true
    docker-compose rm -f 2>/dev/null || true
    
    print_info "Removing test images..."
    docker rmi -f test-project-user-service:latest 2>/dev/null || true
    docker rmi -f localhost:5000/api-service:v1.0.0 2>/dev/null || true
    
    print_success "Test artifacts cleaned"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  cli         Test CLI functionality (local modules)"
    echo "  production  Test CLI functionality (remote modules)"
    echo "  examples    Test example services (rebuild & restart)"
    echo "  examples-update Test example services (update only, no rebuild)"
    echo "  all         Run all tests"
    echo "  clean       Clean test artifacts"
    echo ""
    echo "Examples:"
    echo "  $0 cli"
    echo "  $0 production"
    echo "  $0 examples"
    echo "  $0 examples-update"
    echo "  $0 all"
    echo "  $0 clean"
}

# Test example services (update only - delegated to separate script)
test_examples_update() {
    print_info "Delegating to test-examples.sh..."
    "$SCRIPT_DIR/test-examples.sh" examples-update
}

# Main script logic
case "${1:-}" in
    "cli")
        shift
        test_cli "$@"
        ;;
    "production")
        shift
        test_production "$@"
        ;;
    "examples")
        test_examples
        ;;
    "examples-update")
        test_examples_update
        ;;
    "all")
        test_all
        ;;
    "clean")
        clean_tests
        ;;
    "help"|"-h"|"--help")
        show_usage
        ;;
    "")
        print_error "No command specified"
        show_usage
        exit 1
        ;;
    *)
        print_error "Unknown command: $1"
        show_usage
        exit 1
        ;;
esac
