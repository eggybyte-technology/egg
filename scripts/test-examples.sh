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

# Get the project root directory
PROJECT_ROOT="$(get_project_root)"

# Test example services (smart infrastructure + rebuild services)
test_examples() {
    print_header "Testing example services"

    local test_failed=0
    cd "$PROJECT_ROOT/deploy"

    # Check if infrastructure is already running
    print_info "Checking infrastructure status..."
    local infra_running=false
    
    if docker ps --format '{{.Names}}' | grep -q "egg-mysql"; then
        print_info "MySQL is already running, will reuse existing infrastructure"
        infra_running=true
    else
        print_info "Infrastructure not detected, will start fresh"
        infra_running=false
    fi

    # Stop any existing application services (but keep infrastructure if running)
    print_info "Stopping existing application services..."
    docker-compose -f docker-compose.services.yaml down --remove-orphans 2>/dev/null || true

    # If infrastructure is not running, start it
    if [ "$infra_running" = false ]; then
        print_info "Starting infrastructure services..."
        
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
        print_success "Using existing infrastructure"
    fi

    # Always rebuild application services
    print_info "Building application services..."
    if ! "$PROJECT_ROOT/scripts/build.sh" all; then
        exit_with_error "Failed to build services"
    fi
    print_success "Services built successfully"

    # Start application services
    print_info "Starting application services..."
    
    # Ensure config file exists
    check_file "otel-collector-config.yaml" "OpenTelemetry Collector config file not found"

    if ! docker-compose -f docker-compose.services.yaml up -d; then
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
        print_info "Stopping application services..."
        docker-compose -f docker-compose.services.yaml down --remove-orphans 2>/dev/null || true
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

    # Run comprehensive tests
    if ! run_comprehensive_tests; then
        test_failed=1
    fi

    # Keep all services running after tests
    print_info "Tests completed - all services remain running"
    print_info "Infrastructure services: mysql, jaeger, otel-collector"
    print_info "Application services: minimal-service, user-service"
    print_info ""
    print_info "To stop services manually:"
    print_info "  - Stop application services: make services-down"
    print_info "  - Stop infrastructure: make infra-down"
    print_info "  - Stop everything: make deploy-down"

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

# Run comprehensive tests for both minimal and user services
run_comprehensive_tests() {
    local test_failed=0

    # Enhanced user service testing with complex scenarios
    print_info "Running enhanced user service tests..."
    if ! test_user_service_comprehensive; then
        test_failed=1
    fi

    # Test Connect endpoints
    print_info "Testing Connect service endpoints..."
    if ! test_connect_endpoints; then
        test_failed=1
    fi

    # Test performance and load scenarios
    print_info "Testing performance and load scenarios..."
    if ! test_performance_scenarios; then
        test_failed=1
    fi

    # Test error scenarios and edge cases
    print_info "Testing error scenarios and edge cases..."
    if ! test_error_scenarios; then
        test_failed=1
    fi

    # Test data validation and business logic
    print_info "Testing data validation and business logic..."
    if ! test_data_validation; then
        test_failed=1
    fi

    return $test_failed
}

# Enhanced comprehensive user service testing
test_user_service_comprehensive() {
    print_header "Running comprehensive user service tests"

    local test_failed=0

    # Build Connect tester if needed
    print_info "Building Connect tester..."
    cd "$PROJECT_ROOT/scripts/connect-tester"
    if ! go build -o connect-tester main.go 2>/dev/null; then
        exit_with_error "Failed to build Connect tester"
    fi
    print_success "Connect tester built"

    # Test multiple user creation and operations
    print_info "Testing multiple user operations..."

    # Create multiple users for testing
    local user_ids=()
    local test_emails=()

    for i in {1..5}; do
        timestamp=$(date +%s%N)
        email="comprehensive-test-${timestamp}-${i}@example.com"
        test_emails+=("$email")

        print_info "Creating user $i: $email"
        if user_id=$(./connect-tester http://localhost:8082 user-service create "$email" "Test User $i" 2>/dev/null | grep -o 'with ID: [^[:space:]]*' | cut -d: -f2 | tr -d ' '); then
            user_ids+=("$user_id")
            print_success "User $i created with ID: $user_id"
        else
            print_warning "Failed to create user $i"
            test_failed=1
        fi
    done

    # Test user retrieval for all created users
    print_info "Testing user retrieval..."
    for i in "${!user_ids[@]}"; do
        if [ "${user_ids[$i]}" != "" ]; then
            print_info "Retrieving user ${user_ids[$i]}"
            if ./connect-tester http://localhost:8082 user-service get "${user_ids[$i]}" 2>/dev/null | grep -q "✓ PASS GetUser"; then
                print_success "User ${user_ids[$i]} retrieved successfully"
            else
                print_warning "Failed to retrieve user ${user_ids[$i]}"
                test_failed=1
            fi
        fi
    done

    # Test user updates
    print_info "Testing user updates..."
    for i in "${!user_ids[@]}"; do
        if [ "${user_ids[$i]}" != "" ]; then
            new_name="Updated Test User $i"
            print_info "Updating user ${user_ids[$i]} to name: $new_name"
            if ./connect-tester http://localhost:8082 user-service update "${user_ids[$i]}" "${test_emails[$i]}" "$new_name" 2>/dev/null | grep -q "✓ PASS UpdateUser"; then
                print_success "User ${user_ids[$i]} updated successfully"
            else
                print_warning "Failed to update user ${user_ids[$i]}"
                test_failed=1
            fi
        fi
    done

    # Test pagination
    print_info "Testing user pagination..."
    if ./connect-tester http://localhost:8082 user-service list 1 2 2>/dev/null | grep -q "✓ PASS ListUsers"; then
        print_success "Pagination test passed"
    else
        print_warning "Pagination test failed"
        test_failed=1
    fi

    # Test user deletion
    print_info "Testing user deletion..."
    for i in "${!user_ids[@]}"; do
        if [ "${user_ids[$i]}" != "" ]; then
            print_info "Deleting user ${user_ids[$i]}"
            if ./connect-tester http://localhost:8082 user-service delete "${user_ids[$i]}" 2>/dev/null | grep -q "✓ PASS DeleteUser"; then
                print_success "User ${user_ids[$i]} deleted successfully"
            else
                print_warning "Failed to delete user ${user_ids[$i]}"
                test_failed=1
            fi
        fi
    done

    # Clean up
    rm -f connect-tester

    if [ $test_failed -eq 0 ]; then
        print_success "Comprehensive user service tests completed successfully"
    else
        print_warning "Comprehensive user service tests completed with failures"
    fi

    return $test_failed
}

# Test performance and load scenarios
test_performance_scenarios() {
    print_header "Testing performance and load scenarios"

    local test_failed=0

    # Build Connect tester if needed
    print_info "Building Connect tester..."
    cd "$PROJECT_ROOT/scripts/connect-tester"
    if ! go build -o connect-tester main.go 2>/dev/null; then
        exit_with_error "Failed to build Connect tester"
    fi
    print_success "Connect tester built"

    # Test concurrent operations
    print_info "Testing concurrent user creation..."
    local concurrent_failed=0

    # Create users concurrently
    for i in {1..3}; do
        (
            timestamp=$(date +%s%N)
            email="perf-test-${timestamp}-${i}@example.com"
            if ./connect-tester http://localhost:8082 user-service create "$email" "Perf Test User $i" 2>/dev/null | grep -q "✓ PASS CreateUser"; then
                echo "Concurrent user $i created successfully"
            else
                echo "Failed to create concurrent user $i"
                concurrent_failed=1
            fi
        ) &
    done

    # Wait for all background jobs to complete
    wait

    if [ $concurrent_failed -eq 0 ]; then
        print_success "Concurrent operations test passed"
    else
        print_warning "Concurrent operations test failed"
        test_failed=1
    fi

    # Test large list operations
    print_info "Testing large list operations..."
    if ./connect-tester http://localhost:8082 user-service list 1 50 2>/dev/null | grep -q "✓ PASS ListUsers"; then
        print_success "Large list operations test passed"
    else
        print_warning "Large list operations test failed"
        test_failed=1
    fi

    # Clean up
    rm -f connect-tester

    if [ $test_failed -eq 0 ]; then
        print_success "Performance scenarios tests completed successfully"
    else
        print_warning "Performance scenarios tests completed with failures"
    fi

    return $test_failed
}

# Test error scenarios and edge cases
test_error_scenarios() {
    print_header "Testing error scenarios and edge cases"

    local test_failed=0

    # Build Connect tester if needed
    print_info "Building Connect tester..."
    cd "$PROJECT_ROOT/scripts/connect-tester"
    if ! go build -o connect-tester main.go 2>/dev/null; then
        exit_with_error "Failed to build Connect tester"
    fi
    print_success "Connect tester built"

    # Test invalid user ID
    print_info "Testing invalid user ID..."
    if ./connect-tester http://localhost:8082 user-service get "invalid-user-id" 2>/dev/null | grep -q "✗ FAIL GetUser"; then
        print_success "Invalid user ID test passed (proper error handling)"
    else
        print_warning "Invalid user ID test failed (no proper error handling)"
        test_failed=1
    fi

    # Test empty email
    print_info "Testing empty email validation..."
    if ./connect-tester http://localhost:8082 user-service create "" "Test User" 2>/dev/null | grep -q "✗ FAIL CreateUser"; then
        print_success "Empty email validation test passed"
    else
        print_warning "Empty email validation test failed"
        test_failed=1
    fi

    # Test empty name
    print_info "Testing empty name validation..."
    if ./connect-tester http://localhost:8082 user-service create "test@example.com" "" 2>/dev/null | grep -q "✗ FAIL CreateUser"; then
        print_success "Empty name validation test passed"
    else
        print_warning "Empty name validation test failed"
        test_failed=1
    fi

    # Test duplicate email (if supported by the service)
    print_info "Testing duplicate email handling..."
    timestamp=$(date +%s%N)
    email="duplicate-test-${timestamp}@example.com"

    # First creation should succeed
    if ! ./connect-tester http://localhost:8082 user-service create "$email" "Test User" 2>/dev/null | grep -q "✓ PASS CreateUser"; then
        print_warning "First creation of duplicate test user failed"
        test_failed=1
    else
        # Second creation should fail or handle gracefully
        if ./connect-tester http://localhost:8082 user-service create "$email" "Test User 2" 2>/dev/null | grep -q "✗ FAIL CreateUser"; then
            print_success "Duplicate email handling test passed"
        else
            print_info "Duplicate email allowed (may be by design)"
        fi
    fi

    # Clean up
    rm -f connect-tester

    if [ $test_failed -eq 0 ]; then
        print_success "Error scenarios tests completed successfully"
    else
        print_warning "Error scenarios tests completed with failures"
    fi

    return $test_failed
}

# Test data validation and business logic
test_data_validation() {
    print_header "Testing data validation and business logic"

    local test_failed=0

    # Build Connect tester if needed
    print_info "Building Connect tester..."
    cd "$PROJECT_ROOT/scripts/connect-tester"
    if ! go build -o connect-tester main.go 2>/dev/null; then
        exit_with_error "Failed to build Connect tester"
    fi
    print_success "Connect tester built"

    # Test email format validation
    print_info "Testing email format validation..."

    # Test invalid email formats
    local invalid_emails=("invalid-email" "@example.com" "user@" "user@.com" "user..user@example.com")

    for invalid_email in "${invalid_emails[@]}"; do
        print_info "Testing invalid email format: $invalid_email"
        if ./connect-tester http://localhost:8082 user-service create "$invalid_email" "Test User" 2>/dev/null | grep -q "✗ FAIL CreateUser"; then
            print_success "Email format validation passed for: $invalid_email"
        else
            print_info "Email format validation may not be enforced for: $invalid_email"
        fi
    done

    # Test valid email formats
    print_info "Testing valid email formats..."
    local valid_emails=("user@example.com" "test.user+tag@example.org" "user123@test-domain.co.uk")

    for valid_email in "${valid_emails[@]}"; do
        timestamp=$(date +%s%N)
        test_email="validation-${timestamp}-${valid_email}"
        print_info "Testing valid email format: $test_email"
        if ./connect-tester http://localhost:8082 user-service create "$test_email" "Validation Test User" 2>/dev/null | grep -q "✓ PASS CreateUser"; then
            print_success "Valid email format test passed for: $test_email"
        else
            print_warning "Valid email format test failed for: $test_email"
            test_failed=1
        fi
    done

    # Test name length validation (if any)
    print_info "Testing name length handling..."
    local long_name="This is a very long name that might exceed normal validation limits for testing purposes and should be handled gracefully by the system"
    timestamp=$(date +%s%N)
    email="length-test-${timestamp}@example.com"

    if ./connect-tester http://localhost:8082 user-service create "$email" "$long_name" 2>/dev/null | grep -q "✓ PASS CreateUser"; then
        print_success "Long name handling test passed"
    else
        print_warning "Long name handling test failed"
        test_failed=1
    fi

    # Test special characters in names
    print_info "Testing special characters in names..."
    local special_names=("José" "Müller" "李明" "O'Connor" "Smith-Jones")

    for special_name in "${special_names[@]}"; do
        timestamp=$(date +%s%N)
        email="special-${timestamp}@example.com"
        print_info "Testing special name: $special_name"
        if ./connect-tester http://localhost:8082 user-service create "$email" "$special_name" 2>/dev/null | grep -q "✓ PASS CreateUser"; then
            print_success "Special characters test passed for: $special_name"
        else
            print_warning "Special characters test failed for: $special_name"
            test_failed=1
        fi
    done

    # Clean up
    rm -f connect-tester

    if [ $test_failed -eq 0 ]; then
        print_success "Data validation tests completed successfully"
    else
        print_warning "Data validation tests completed with failures"
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
        exit_with_error "Failed to build Connect tester"
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
