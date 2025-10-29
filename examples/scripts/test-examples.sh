#!/bin/bash

# Test Examples Script for Egg Framework
# Provides comprehensive testing for example services
#
# Usage:
#   ./scripts/test-examples.sh [command]
#
# Commands:
#   examples        Test example services (rebuild & restart)
#   examples-update Test example services (update only, no rebuild)
#   help           Show this help message
#
# Examples:
#   ./scripts/test-examples.sh examples
#   ./scripts/test-examples.sh examples-update

set -e

# Source the unified logger
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

# Get the examples root directory (parent of scripts/)
EXAMPLES_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROJECT_ROOT="$(cd "$EXAMPLES_ROOT/.." && pwd)"

# Test example services (smart infrastructure + rebuild services)
test_examples() {
    print_header "Testing example services"

    # Check if Docker is running
    if ! docker ps >/dev/null 2>&1; then
        print_error "Docker daemon is not running or not accessible"
        print_info "Please start Docker Desktop and ensure it's fully initialized"
        print_info "You can check Docker status with: docker ps"
        print_info "If Docker Desktop is running but daemon is not accessible, try:"
        print_info "  1. Restart Docker Desktop"
        print_info "  2. Wait a few seconds for Docker to fully start"
        print_info "  3. Run: docker ps to verify connection"
        exit_with_error "Docker daemon not accessible"
    fi

    local test_failed=0
    cd "$EXAMPLES_ROOT/deploy"

    # Step 1: Build latest example service images
    print_header "Step 1: Building latest example service images"
    print_info "Building application services..."
    if ! "$EXAMPLES_ROOT/scripts/build-examples.sh" all; then
        exit_with_error "Failed to build services"
    fi
    print_success "Services built successfully"
    
    # Verify Docker images exist
    print_info "Verifying Docker images..."
    if ! docker images minimal-connect-service:latest -q | grep -q .; then
        exit_with_error "minimal-connect-service image not found"
    fi
    if ! docker images user-service:latest -q | grep -q .; then
        exit_with_error "user-service image not found"
    fi
    print_success "All Docker images verified"

    # Step 2: Check infrastructure services (mysql only - tracing disabled)
    # Note: Infrastructure services are never stopped during tests - they persist across test runs
    print_header "Step 2: Checking infrastructure services"
    print_info "Checking infrastructure status..."
    
    # Check if MySQL is running
    local mysql_running=false
    
    if docker ps --format '{{.Names}}' | grep -q "egg-mysql"; then
        mysql_running=true
        print_info "MySQL is already running"
    fi
    
    # If MySQL is not running, start infrastructure
    if [ "$mysql_running" = false ]; then
        print_info "MySQL is not running, starting infrastructure..."
        
        # Clean up ports if needed
        if ! "$PROJECT_ROOT/scripts/cleanup-ports.sh"; then
            print_warning "Some ports could not be freed, but continuing anyway..."
        fi
        
        # Start infrastructure
        docker-compose -f docker-compose.infra.yaml up -d
        
        # Wait for infrastructure to be ready
        print_info "Waiting for infrastructure to be ready..."
        if ! wait_for_condition "docker ps --format '{{.Names}}' | grep -q egg-mysql && docker inspect egg-mysql --format='{{.State.Health.Status}}' 2>/dev/null | grep -q healthy" 60 "MySQL healthy"; then
            print_error "Infrastructure failed to start properly"
            docker-compose -f docker-compose.infra.yaml logs --tail=50
            exit_with_error "Infrastructure startup failed"
        fi
        print_success "Infrastructure started successfully"
    else
        print_success "All infrastructure services are running"
    fi

    # Step 3: Stop all application services, then start with latest images
    print_header "Step 3: Restarting application services with latest images"
    print_info "Stopping existing application services..."
    # Use 'down' without --remove-orphans to preserve infrastructure services
    docker-compose -f docker-compose.services.yaml down 2>/dev/null || true
    print_success "Application services stopped"

    # Start application services with latest images (force recreate to use new images)
    print_info "Starting application services with latest images..."
    print_info "Note: Any 'orphan containers' or 'version obsolete' warnings are expected and can be ignored"

    if ! docker-compose -f docker-compose.services.yaml up -d --force-recreate; then
        exit_with_error "Failed to start application services"
    fi
    print_success "Application services started"

    # Wait for services to be ready with health checks
    print_info "Waiting for services to be ready..."

    # Test minimal service health with retry
    if ! wait_for_condition "curl -f -s http://localhost:8081/health > /dev/null 2>&1" 30 "Minimal service health"; then
        print_warning "Minimal service health check failed"
        test_failed=1
    fi

    # Test user service health with retry
    if ! wait_for_condition "curl -f -s http://localhost:8083/health > /dev/null 2>&1" 30 "User service health"; then
        print_warning "User service health check failed"
        test_failed=1
    fi

    # If health checks failed, show logs and exit early
    if [ $test_failed -eq 1 ]; then
        print_error "Health checks failed, showing container logs..."
        docker-compose -f docker-compose.services.yaml logs --tail=50
        print_info "Stopping application services (infrastructure services remain running)..."
        # Use 'down' without --remove-orphans to preserve infrastructure services
        docker-compose -f docker-compose.services.yaml down 2>/dev/null || true
        exit_with_error "Health checks failed"
    fi

    # Test database connectivity for user-service
    print_info "Testing user-service database connectivity..."
    print_info "Checking if user-service is using real database..."
    if docker logs egg-user-service 2>&1 | grep -q "Database initialized and migrated successfully"; then
        print_success "User service is using real database connection"
    elif docker logs egg-user-service 2>&1 | grep -q "Using mock repository"; then
        print_warning "User service is using mock repository (no database)"
    else
        print_warning "Cannot determine user service database status"
    fi

    # Step 4: Run connect-tester tests
    print_header "Step 4: Running connect-tester tests"
    if ! run_connect_tester_tests; then
        test_failed=1
    fi

    # Keep all services running after tests
    # Note: Infrastructure services persist across test runs for better performance
    print_info "Tests completed - all services remain running"
    print_info "Infrastructure services (persistent): mysql"
    print_info "Application services: minimal-service, user-service"
    print_info ""
    print_info "To stop services manually:"
    print_info "  - Stop application services only: make services-down"
    print_info "  - Stop infrastructure (after tests): make infra-down"
    print_info "  - Stop everything: make deploy-down"
    print_info ""
    print_info "Note: Infrastructure services are NOT stopped after tests to improve performance"

    if [ $test_failed -eq 0 ]; then
        print_success "Example services tests completed successfully"
    else
        print_warning "Example services tests completed with failures"
    fi

    return $test_failed
}

# Test example services (legacy compatibility - redirects to main test)
test_examples_update() {
    print_warning "test_examples_update is deprecated, redirecting to test_examples..."
    test_examples

}

# Run connect-tester tests for both minimal and user services
run_connect_tester_tests() {
    local test_failed=0

    # Change to connect-tester directory
    cd "$EXAMPLES_ROOT/connect-tester"

    # Test minimal service
    print_info "Testing minimal-service endpoints..."
    if ! go run main.go http://localhost:8080 minimal-service; then
        print_error "Minimal service tests failed"
        test_failed=1
    else
        print_success "Minimal service tests passed"
    fi

    # Test user service
    print_info "Testing user-service endpoints..."
    if ! go run main.go http://localhost:8082 user-service; then
        print_error "User service tests failed"
        test_failed=1
    else
        print_success "User service tests passed"
    fi

    return $test_failed
}


# Show usage information
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  examples        Test example services (rebuild & restart)"
    echo "  examples-update Test example services (update only, no rebuild)"
    echo "  help           Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 examples"
    echo "  $0 examples-update"
}

# Main script logic
case "${1:-}" in
    "examples")
        init_logging "Example Services Tests"
        test_examples
        finalize_logging $? "Example Services Tests"
        ;;
    "examples-update")
        init_logging "Example Services Tests (Update Only)"
        test_examples_update
        finalize_logging $? "Example Services Tests (Update Only)"
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
