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
#   examples    Test example services
#   all         Run all tests
#   clean       Clean test artifacts
#
# Examples:
#   ./scripts/test.sh cli
#   ./scripts/test.sh production
#   ./scripts/test.sh examples
#   ./scripts/test.sh all
#   ./scripts/test.sh clean

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Print colored output
print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}▶ $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_success() {
    echo -e "${GREEN}[✓] SUCCESS:${NC} $1"
}

print_error() {
    echo -e "${RED}[✗] ERROR:${NC} $1"
}

print_info() {
    echo -e "${CYAN}[i] INFO:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!] WARNING:${NC} $1"
}

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

# Test example services
test_examples() {
    print_header "Testing example services"
    
    local test_failed=0
    
    # Build all services first
    print_info "Building all services..."
    if ! "$PROJECT_ROOT/scripts/build.sh" all; then
        print_error "Failed to build services"
        return 1
    fi
    print_success "Services built successfully"
    
    # Start services with docker-compose
    print_info "Starting services with docker-compose..."
    cd "$PROJECT_ROOT/deploy"
    
    # Ensure config file exists
    if [ ! -f "otel-collector-config.yaml" ]; then
        print_error "OpenTelemetry Collector config file not found"
        return 1
    fi
    
    if ! docker-compose up -d; then
        print_error "Failed to start services with docker-compose"
        return 1
    fi
    print_success "Services started"
    
    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 15
    
    # Test minimal service health
    print_info "Testing minimal service health..."
    print_info "Testing endpoint: http://localhost:8081/health"
    if curl -f http://localhost:8081/health > /dev/null 2>&1; then
        print_success "Minimal service health check passed"
    else
        print_warning "Minimal service health check failed"
        test_failed=1
    fi
    
    # Test user service health
    print_info "Testing user service health..."
    print_info "Testing endpoint: http://localhost:8083/health"
    if curl -f http://localhost:8083/health > /dev/null 2>&1; then
        print_success "User service health check passed"
    else
        print_warning "User service health check failed"
        test_failed=1
    fi
    
    # Test Connect endpoints
    print_info "Testing Connect service endpoints..."
    if ! test_connect_endpoints; then
        test_failed=1
    fi
    
    # Stop services
    print_info "Stopping services..."
    docker-compose down --remove-orphans 2>/dev/null || true
    
    # Clean up any remaining containers
    print_info "Cleaning up remaining containers..."
    docker-compose rm -f 2>/dev/null || true
    
    if [ $test_failed -eq 0 ]; then
        print_success "Example services tests completed successfully"
    else
        print_warning "Example services tests completed with failures"
    fi
    
    return $test_failed
}

# Test Connect service endpoints
test_connect_endpoints() {
    print_header "Testing Connect service endpoints"
    
    local connect_test_failed=0
    
    # Build Connect tester if needed
    print_info "Building Connect tester..."
    cd "$PROJECT_ROOT/scripts/connect-tester"
    if ! go build -o connect-tester main.go 2>/dev/null; then
        print_error "Failed to build Connect tester"
        return 1
    fi
    print_success "Connect tester built"
    
    # Test minimal service Connect endpoints
    print_info "Testing minimal service Connect endpoints..."
    print_info "Testing endpoint: http://localhost:8080"
    local minimal_output
    if minimal_output=$(./connect-tester http://localhost:8080 minimal-service 2>&1); then
        print_success "Minimal service Connect endpoints test passed"
        echo "$minimal_output" | grep -E "(✓ PASS|✗ FAIL)" | while read line; do
            print_info "  $line"
        done
    else
        print_warning "Minimal service Connect endpoints test failed"
        echo "$minimal_output" | grep -E "(✓ PASS|✗ FAIL)" | while read line; do
            print_info "  $line"
        done
        connect_test_failed=1
    fi
    
    # Test user service Connect endpoints
    print_info "Testing user service Connect endpoints..."
    print_info "Testing endpoint: http://localhost:8082"
    local user_output
    if user_output=$(./connect-tester http://localhost:8082 user-service 2>&1); then
        print_success "User service Connect endpoints test passed"
        echo "$user_output" | grep -E "(✓ PASS|✗ FAIL)" | while read line; do
            print_info "  $line"
        done
    else
        print_warning "User service Connect endpoints test failed"
        echo "$user_output" | grep -E "(✓ PASS|✗ FAIL)" | while read line; do
            print_info "  $line"
        done
        connect_test_failed=1
    fi
    
    # Clean up
    rm -f connect-tester
    
    if [ $connect_test_failed -eq 0 ]; then
        print_success "Connect endpoints testing completed successfully"
    else
        print_warning "Connect endpoints testing completed with failures"
    fi
    
    return $connect_test_failed
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
    echo "  examples    Test example services"
    echo "  all         Run all tests"
    echo "  clean       Clean test artifacts"
    echo ""
    echo "Examples:"
    echo "  $0 cli"
    echo "  $0 production"
    echo "  $0 examples"
    echo "  $0 all"
    echo "  $0 clean"
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
