#!/bin/bash
#
# Docker Compose Service Testing
#
# Tests Docker Compose services including health checks and RPC endpoints.
# This file is sourced by test-cli.sh or can be run standalone.
#
# Requires: test-config.sh (which sources test-helpers.sh)

# Source test configuration (this will also source logger.sh)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test-config.sh"

# Helper function to execute wget inside Docker network
# Uses docker compose exec to run wget in a container connected to the same network
docker_compose_wget() {
    local url="$1"
    shift
    local wget_args="$@"
    
    # Use docker compose exec to run wget in one of the running containers
    # We'll use the first backend service container as our wget executor
    local compose_dir="$TEST_DIR/deploy/compose"
    cd "$compose_dir" || return 1
    
    # Get the first backend service name
    local service_name=$(docker compose config --services 2>/dev/null | grep -E "^($BACKEND_SERVICE|$BACKEND_PING_SERVICE)" | head -1)
    
    if [ -z "$service_name" ]; then
        print_error "No backend service found for wget execution"
        return 1
    fi
    
    # Execute wget inside the container
    docker compose exec -T "$service_name" wget $wget_args "$url"
    local exit_code=$?
    
    cd - > /dev/null
    return $exit_code
}

# Helper function to wait for endpoint in Docker network
wait_for_endpoint_docker() {
    local service_name="$1"
    local port="$2"
    local path="${3:-/health}"
    local max_attempts="${4:-30}"
    local description="${5:-Service}"
    local pattern="${6:-}"
    
    local url="http://${service_name}:${port}${path}"
    local attempt=1
    
    print_info "Waiting for $description..."
    print_info "URL: $url"
    print_info "Max attempts: $max_attempts (${max_attempts}s)"
    
    local compose_dir="$TEST_DIR/deploy/compose"
    cd "$compose_dir" || return 1
    
    # Get the first backend service name to use as wget executor
    local executor_service=$(docker compose config --services 2>/dev/null | grep -E "^($BACKEND_SERVICE|$BACKEND_PING_SERVICE)" | head -1)
    
    if [ -z "$executor_service" ]; then
        print_error "No backend service found for health check"
        cd - > /dev/null
        return 1
    fi
    
    while [ $attempt -le $max_attempts ]; do
        # Use wget instead of curl (wget is available in containers, curl may not be)
        # First, get HTTP status code using --spider (non-destructive check)
        local http_code
        http_code=$(docker compose exec -T "$executor_service" wget --spider -S -q -O /dev/null "$url" 2>&1 | grep -i "HTTP/" | tail -1 | awk '{print $2}' || echo "000")
        
        # Then get response body only if status code is 200
        local body=""
        local wget_exit=1
        if [ "$http_code" = "200" ]; then
            body=$(docker compose exec -T "$executor_service" wget -qO- "$url" 2>&1)
            wget_exit=$?
        fi
        
        # Check if pattern is specified
        if [ -n "$pattern" ]; then
            if [ $wget_exit -eq 0 ] && [ "$http_code" = "200" ] && echo "$body" | grep -q "$pattern"; then
                printf "\n"
                print_success "$description ready (attempt $attempt/$max_attempts)"
                print_info "HTTP Status: $http_code"
                print_info "Pattern found: $pattern"
                cd - > /dev/null
                return 0
            fi
        else
            if [ "$http_code" = "200" ]; then
                printf "\n"
                print_success "$description ready (attempt $attempt/$max_attempts)"
                print_info "HTTP Status: $http_code"
                cd - > /dev/null
                return 0
            fi
        fi
        
        if [ $((attempt % 5)) -eq 0 ]; then
            printf " [%d/%d]" $attempt $max_attempts
        else
            printf "."
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    printf "\n"
    print_warning "$description not ready after $max_attempts attempts"
    cd - > /dev/null
    return 1
}

# Test ping service Ping RPC
test_ping_service_rpc() {
    # Use Docker service name instead of localhost
    local service_url="http://ping:${PING_HTTP_PORT}"
    local rpc_path="$PING_SERVICE_PATH"
    
    print_section "Testing Ping Service RPC"
    
    # Test Ping RPC with a message
    local test_message="Hello from test"
    local request_data="{\"message\":\"$test_message\"}"
    
    local compose_dir="$TEST_DIR/deploy/compose"
    cd "$compose_dir" || return 1
    
    # Get executor service (use ping service itself or another backend service)
    local executor_service=$(docker compose config --services 2>/dev/null | grep -E "^($BACKEND_SERVICE|$BACKEND_PING_SERVICE)" | head -1)
    
    if [ -z "$executor_service" ]; then
        print_error "No backend service found for RPC test"
        cd - > /dev/null
        return 1
    fi
    
    # Execute wget inside Docker network (POST request)
    local url="${service_url}${rpc_path}"
    print_info "Calling RPC: $url"
    
    # Use wget with --post-data for POST requests
    # First get HTTP status code
    local http_code
    http_code=$(docker compose exec -T "$executor_service" wget --spider -S -q -O /dev/null \
        --post-data="$request_data" \
        --header="Content-Type: application/json" \
        "$url" 2>&1 | grep -i "HTTP/" | tail -1 | awk '{print $2}' || echo "000")
    
    # Then get response body
    local response_body=""
    local wget_exit=1
    if [ "$http_code" = "200" ]; then
        response_body=$(docker compose exec -T "$executor_service" wget -qO- \
            --post-data="$request_data" \
            --header="Content-Type: application/json" \
            "$url" 2>&1)
        wget_exit=$?
    fi
    
    cd - > /dev/null
    
    if [ $wget_exit -eq 0 ] && [ "$http_code" = "200" ] && echo "$response_body" | grep -q "message"; then
        print_success "Ping RPC test passed"
        print_info "Response: $response_body"
        return 0
    else
        print_error "Ping RPC test failed"
        print_info "HTTP Status: $http_code"
        print_info "Response: $response_body"
        return 1
    fi
}

# Test user service CRUD operations
test_user_service_crud() {
    # Use Docker service name instead of localhost
    local service_url="http://user:${USER_HTTP_PORT}"
    local user_id=""
    
    print_section "Testing User Service CRUD Operations"
    
    local compose_dir="$TEST_DIR/deploy/compose"
    cd "$compose_dir" || return 1
    
    # Get executor service (use user service itself or another backend service)
    local executor_service=$(docker compose config --services 2>/dev/null | grep -E "^($BACKEND_SERVICE|$BACKEND_PING_SERVICE)" | head -1)
    
    if [ -z "$executor_service" ]; then
        print_error "No backend service found for CRUD test"
        cd - > /dev/null
        return 1
    fi
    
    # Helper function to execute wget in Docker network
    docker_wget() {
        local method="$1"
        local path="$2"
        local data="${3:-}"
        local url="${service_url}${path}"
        
        if [ "$method" = "POST" ] && [ -n "$data" ]; then
            # POST request with data
            docker compose exec -T "$executor_service" wget -qO- \
                --post-data="$data" \
                --header="Content-Type: application/json" \
                "$url" 2>&1
            return $?
        else
            # GET request or POST without data
            docker compose exec -T "$executor_service" wget -qO- \
                --header="Content-Type: application/json" \
                "$url" 2>&1
            return $?
        fi
    }
    
    # 1. Create User
    print_info "Creating user..."
    local create_data="{\"email\":\"test@example.com\",\"name\":\"Test User\"}"
    local create_response
    create_response=$(docker_wget "POST" "$USER_SERVICE_CREATE" "$create_data")
    local wget_exit=$?
    
    if [ $wget_exit -eq 0 ] && echo "$create_response" | grep -q "\"id\""; then
        print_success "User created successfully"
        # Extract user ID from response (basic extraction - assumes JSON format)
        user_id=$(echo "$create_response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4 || echo "")
        if [ -n "$user_id" ]; then
            print_info "Created user ID: $user_id"
        fi
        echo "$create_response" | head -10
    else
        print_error "Failed to create user"
        echo "$create_response"
        return 1
    fi
    
    # 2. Get User
    if [ -n "$user_id" ]; then
        print_info "Getting user..."
        local get_response
        get_response=$(docker_wget "POST" "$USER_SERVICE_GET" "{\"id\":\"$user_id\"}")
        local wget_exit=$?
        
        if [ $wget_exit -eq 0 ] && echo "$get_response" | grep -q "$user_id"; then
            print_success "User retrieved successfully"
            echo "$get_response" | head -10
        else
            print_error "Failed to get user"
            echo "$get_response"
            return 1
        fi
    fi
    
    # 3. Update User
    if [ -n "$user_id" ]; then
        print_info "Updating user..."
        local update_data="{\"id\":\"$user_id\",\"email\":\"updated@example.com\",\"name\":\"Updated User\"}"
        local update_response
        update_response=$(docker_wget "POST" "$USER_SERVICE_UPDATE" "$update_data")
        local wget_exit=$?
        
        if [ $wget_exit -eq 0 ] && echo "$update_response" | grep -q "updated@example.com"; then
            print_success "User updated successfully"
            echo "$update_response" | head -10
        else
            print_error "Failed to update user"
            echo "$update_response"
            return 1
        fi
    fi
    
    # 4. List Users
    print_info "Listing users..."
    local list_response
    list_response=$(docker_wget "POST" "$USER_SERVICE_LIST" '{"page":1,"page_size":10}')
    local wget_exit=$?
    
    if [ $wget_exit -eq 0 ] && echo "$list_response" | grep -q "users"; then
        print_success "Users listed successfully"
        echo "$list_response" | head -15
    else
        print_error "Failed to list users"
        echo "$list_response"
        return 1
    fi
    
    # 5. Delete User
    if [ -n "$user_id" ]; then
        print_info "Deleting user..."
        local delete_response
        delete_response=$(docker_wget "POST" "$USER_SERVICE_DELETE" "{\"id\":\"$user_id\"}")
        local wget_exit=$?
        
        if [ $wget_exit -eq 0 ]; then
            print_success "User deleted successfully"
            echo "$delete_response" | head -5
        else
            print_error "Failed to delete user"
            echo "$delete_response"
            return 1
        fi
    fi
    
    cd - > /dev/null
    print_success "All User Service CRUD operations completed successfully"
    return 0
}

# Test Docker Compose services with health checks and RPC endpoints
test_compose_services() {
    print_cli_section "Test 12: Docker Compose Service Validation"
    
    # Only test if API generation and builds were successful
    # API_SUCCESS is set by the main test script
    if [ "${API_SUCCESS:-false}" != true ]; then
        print_warning "Skipping Docker Compose validation (API generation failed)"
        return 0
    fi
    
    print_section "Starting services with Docker Compose"
    
    # Change to compose directory
    local original_dir=$(pwd)
    cd "$TEST_DIR/deploy/compose" || return 1
    
    # Start services in detached mode
    print_command "docker compose up -d"
    if ! docker compose up -d 2>&1; then
        print_error "Failed to start Docker Compose services"
        cd "$original_dir"
        return 1
    fi
    
    print_success "Services started successfully"
    
    # Check service status
    print_section "Checking service status"
    print_command "docker compose ps"
    docker compose ps
    
    # Wait for services to be healthy
    print_section "Waiting for services to become healthy"
    
    # Test ping service health endpoint with retry (using Docker service name)
    print_section "Testing ping service health endpoint"
    if ! wait_for_endpoint_docker "ping" "$PING_HEALTH_PORT" "/health" 30 "ping service health"; then
        print_error "Ping service health check failed"
        docker compose logs ping --tail=50
        cd "$original_dir"
        return 1
    fi
    
    # Test user service health endpoint with retry (using Docker service name)
    print_section "Testing user service health endpoint"
    if ! wait_for_endpoint_docker "user" "$USER_HEALTH_PORT" "/health" 30 "user service health"; then
        print_error "User service health check failed"
        docker compose logs user --tail=50
        cd "$original_dir"
        return 1
    fi
    
    # Test user service metrics endpoint with retry
    print_section "Testing user service metrics endpoint"
    if ! wait_for_endpoint_docker "user" "$USER_METRICS_PORT" "/metrics" 30 "user service metrics" "go_"; then
        print_warning "User service metrics check failed (non-critical)"
    fi
    
    # Test ping service metrics endpoint with retry
    print_section "Testing ping service metrics endpoint"
    if ! wait_for_endpoint_docker "ping" "$PING_METRICS_PORT" "/metrics" 30 "ping service metrics" "go_"; then
        print_warning "Ping service metrics check failed (non-critical)"
    fi
    
    # Test frontend service if it exists (using Docker service name)
    local frontend_service_name=$(docker compose config --services 2>/dev/null | grep -iE "admin|frontend" | head -1)
    if [ -n "$frontend_service_name" ]; then
        print_section "Testing frontend service endpoint"
        if ! wait_for_endpoint_docker "$frontend_service_name" "$FRONTEND_PORT" "" 30 "frontend service"; then
            print_warning "Frontend service check failed (non-critical)"
        fi
    fi
    
    # Now test RPC endpoints (after health checks pass)
    print_section "Testing RPC Endpoints"
    
    # Test Ping Service RPC
    if ! test_ping_service_rpc; then
        print_error "Ping service RPC test failed"
        docker compose logs ping --tail=50
        cd "$original_dir"
        return 1
    fi
    
    # Test User Service CRUD
    if ! test_user_service_crud; then
        print_error "User service CRUD test failed"
        docker compose logs user --tail=50
        cd "$original_dir"
        return 1
    fi
    
    # Get container logs for debugging (last 20 lines)
    print_section "Container logs (last 20 lines)"
    print_command "docker compose logs --tail=20"
    docker compose logs --tail=20 2>&1 | head -50
    
    # Keep services running after tests (don't stop them)
    print_section "Tests completed - services remain running"
    print_info "Services are still running for manual testing"
    print_info "To stop services manually, run: docker compose down"
    
    # Get network name for port mapping instructions
    # Docker Compose creates network names as: {project-name}_{network-name}
    # Try to get actual network name from running containers
    local network_name=""
    local first_container=$(docker compose ps -q 2>/dev/null | head -1)
    if [ -n "$first_container" ]; then
        # Get network name from container's network settings
        network_name=$(docker inspect "$first_container" --format='{{range $key, $value := .NetworkSettings.Networks}}{{$key}}{{end}}' 2>/dev/null | head -1)
    fi
    
    # Fallback to default network name format if not found
    if [ -z "$network_name" ]; then
        # Get project name from compose directory name
        local compose_dir=$(basename "$TEST_DIR" 2>/dev/null || echo "$PROJECT_NAME")
        network_name="${compose_dir}_${PROJECT_NAME}-network"
    fi
    
    # Provide port mapping instructions for local access
    print_section "Port Mapping for Local Access"
    print_info "Docker Compose services don't expose ports to localhost by default."
    print_info "To access services locally, use socat to create port proxies:"
    echo ""
    print_info "Frontend service (admin_portal):"
    echo "  docker run --rm -d \\"
    echo "    --name temp-frontend-proxy \\"
    echo "    -p 3000:3000 \\"
    echo "    --network ${network_name} \\"
    echo "    alpine/socat \\"
    echo "    tcp-listen:3000,fork,reuseaddr tcp-connect:${PROJECT_NAME}-admin-portal:3000"
    echo ""
    print_info "User service (HTTP):"
    echo "  docker run --rm -d \\"
    echo "    --name temp-user-proxy \\"
    echo "    -p 8080:8080 \\"
    echo "    --network ${network_name} \\"
    echo "    alpine/socat \\"
    echo "    tcp-listen:8080,fork,reuseaddr tcp-connect:${PROJECT_NAME}-user:8080"
    echo ""
    print_info "Ping service (HTTP):"
    echo "  docker run --rm -d \\"
    echo "    --name temp-ping-proxy \\"
    echo "    -p 8090:8090 \\"
    echo "    --network ${network_name} \\"
    echo "    alpine/socat \\"
    echo "    tcp-listen:8090,fork,reuseaddr tcp-connect:${PROJECT_NAME}-ping:8090"
    echo ""
    print_info "To stop port proxies:"
    echo "  docker stop temp-frontend-proxy temp-user-proxy temp-ping-proxy"
    echo ""
    print_info "After starting proxies, access services at:"
    echo "  - Frontend: http://localhost:3000"
    echo "  - User service: http://localhost:8080"
    echo "  - Ping service: http://localhost:8090"
    echo ""
    print_info "Note: Health check ports (8081, 8091) and metrics ports (9091, 9092) can be mapped similarly if needed."
    
    cd "$original_dir"
    print_success "Docker Compose services validated"
    return 0
}

# Allow script to be run standalone
# Check if script is being sourced or executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    # Script is being executed directly, not sourced
    # Parse command line arguments
    TEST_DIR_OVERRIDE=""
    API_SUCCESS_OVERRIDE=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --test-dir)
                TEST_DIR_OVERRIDE="$2"
                shift 2
                ;;
            --api-success)
                API_SUCCESS_OVERRIDE="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --test-dir DIR      Override test directory (default: cli/tmp/test-project)"
                echo "  --api-success BOOL  Override API_SUCCESS flag (default: true)"
                echo "  --help, -h         Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                                    # Run tests with defaults"
                echo "  $0 --test-dir /path/to/project        # Use custom test directory"
                echo "  $0 --api-success false                # Skip API success check"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
    
    # Override TEST_DIR if provided
    if [ -n "$TEST_DIR_OVERRIDE" ]; then
        TEST_DIR="$TEST_DIR_OVERRIDE"
        export TEST_DIR
    fi
    
    # Override API_SUCCESS if provided, otherwise default to true for standalone execution
    if [ -n "$API_SUCCESS_OVERRIDE" ]; then
        API_SUCCESS="$API_SUCCESS_OVERRIDE"
    else
        # Default to true when running standalone (assume services are already built)
        API_SUCCESS=true
    fi
    export API_SUCCESS
    
    # Check if test directory exists
    if [ ! -d "$TEST_DIR/deploy/compose" ]; then
        print_error "Test directory not found: $TEST_DIR/deploy/compose"
        print_info "Please ensure the project has been initialized and Docker Compose files generated"
        print_info "You can run: cd $(dirname "$TEST_DIR") && $EGG_CLI init $PROJECT_NAME"
        exit 1
    fi
    
    # Run the test
    print_cli_section "Standalone Docker Compose Service Test"
    print_info "Test directory: $TEST_DIR"
    print_info "API Success flag: $API_SUCCESS"
    
    if test_compose_services; then
        exit 0
    else
        exit 1
    fi
fi
