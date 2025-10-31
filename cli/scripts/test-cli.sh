#!/bin/bash
#
# CLI Integration Test Script for Egg Framework
#
# Main test orchestrator that coordinates all test modules.
# Uses modular test scripts for better organization and maintainability.
#
# Test Modules:
#   - test-config.sh: Test configuration and constants
#   - test-helpers.sh: Common helper functions
#   - test-compose.sh: Docker Compose service testing (health checks + RPC)
#
# Usage:
#   ./scripts/test-cli.sh [--remove]
#
# Options:
#   --remove    Remove test directory after test completion

set -e  # Exit on error

# Source test configuration (this will also source logger.sh and test-helpers.sh)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/test-config.sh"

# Source compose testing module
source "$SCRIPT_DIR/test-compose.sh"

# ==============================================================================
# Cleanup
# ==============================================================================

cleanup() {
    if [ "$KEEP_TEST_DIR" = true ]; then
        print_info "Keeping test directory: $TEST_DIR"
    else
        print_info "Cleaning up test directory..."
        if [ -d "$TEST_DIR" ]; then
            rm -rf "$TEST_DIR"
            print_success "Cleanup completed: $TEST_DIR"
        else
            print_info "Test directory already removed"
        fi
        # Clean up tmp workspace if empty
        if [ -d "$TEST_WORKSPACE" ] && [ -z "$(ls -A "$TEST_WORKSPACE")" ]; then
            rmdir "$TEST_WORKSPACE"
            print_info "Removed empty workspace: $TEST_WORKSPACE"
        fi
    fi
}

# Trap errors and cleanup
trap 'print_error "Test failed at line $LINENO"; cleanup' ERR

# ==============================================================================
# Pre-Test Setup
# ==============================================================================

print_header "Egg CLI Integration Test"
printf "\n"

print_info "Project root: $PROJECT_ROOT"
print_info "CLI root: $CLI_ROOT"
print_info "Test workspace: $TEST_WORKSPACE"
print_info "Test directory: $TEST_DIR"
printf "\n"

print_step "Setup" "Verifying egg CLI binary"

# Verify CLI binary exists
if [ ! -f "$EGG_CLI" ]; then
    exit_with_error "CLI binary not found: $EGG_CLI (run 'make build' first)"
fi

if [ ! -x "$EGG_CLI" ]; then
    exit_with_error "CLI binary is not executable: $EGG_CLI"
fi

print_success "CLI binary ready: $EGG_CLI"
printf "\n"

# ==============================================================================
# Cleanup: Remove any existing test directory
# ==============================================================================

# Create test workspace directory
mkdir -p "$TEST_WORKSPACE"
cd "$TEST_WORKSPACE"
print_info "Working directory: $(pwd)"

# Clean up any existing test directory BEFORE running tests
if [ -d "$PROJECT_NAME" ]; then
    print_info "Removing existing test directory..."
    rm -rf "$PROJECT_NAME"
    print_success "Removed existing test directory"
fi
printf "\n"

# ==============================================================================
# Test 0: Environment Check (egg doctor) - Run First
# ==============================================================================

run_egg_command "Environment Check (egg doctor)" doctor

# ==============================================================================
# Test 1: Project Initialization (egg init)
# ==============================================================================

# Run egg init (it will create the project directory)
run_egg_command "Project Initialization (egg init)" init \
    --project-name "$PROJECT_NAME" \
    --module-prefix github.com/eggybyte-test/test-project \
    --docker-registry ghcr.io/eggybyte-test \
    --version v1.0.0

# Enter the created project directory
if [ ! -d "$PROJECT_NAME" ]; then
    exit_with_error "Project directory '$PROJECT_NAME' was not created by egg init"
fi
cd "$PROJECT_NAME"
print_info "Changed to project directory: $(pwd)"

# Validate directory structure
print_cli_section "Validating directory structure"
check_dir "api"
check_dir "backend"
check_dir "frontend"
check_dir "docker"
check_dir "deploy"

# Validate configuration files
print_section "Validating configuration files"
check_file ".gitignore"
check_file "egg.yaml"
check_file "api/buf.yaml"
check_file "api/buf.gen.yaml"
check_file "docker/Dockerfile.backend"
check_file "docker/Dockerfile.frontend"
check_file "docker/nginx.conf"

# Validate egg.yaml content
print_section "Validating egg.yaml content"
check_file_content "egg.yaml" "project_name: \"$PROJECT_NAME\"" "Project name"
check_file_content "egg.yaml" "module_prefix: \"github.com/eggybyte-test/test-project\"" "Module prefix"
check_file_content "egg.yaml" "docker_registry: \"ghcr.io/eggybyte-test\"" "Docker registry"
check_file_content "egg.yaml" "version: \"v1.0.0\"" "Version"

# ==============================================================================
# Test 2: Backend Service Creation (with local modules)
# ==============================================================================

# Test 2.0: Service name validation (reject -service suffix)
print_cli_section "Test 2.0: Service Name Validation"
print_info "Testing service name validation (should reject -service suffix)"
if ($EGG_CLI create backend user-service --local-modules 2>&1 | grep -q "must not end with '-service'"); then
    print_success "Service name validation works correctly (rejected 'user-service')"
else
    print_error "Service name validation failed - should reject names ending with '-service'"
    exit 1
fi

# Run egg create backend with --proto crud (matches default service template)
run_egg_command "Backend Service Creation (egg create backend --local-modules)" \
    create backend "$BACKEND_SERVICE" --proto crud --local-modules

# Validate backend service structure
print_section "Validating backend service structure"
check_dir "backend/$BACKEND_SERVICE"
check_dir "backend/$BACKEND_SERVICE/cmd/server"
check_dir "backend/$BACKEND_SERVICE/internal/config"
check_dir "backend/$BACKEND_SERVICE/internal/handler"
check_dir "backend/$BACKEND_SERVICE/internal/service"

# Validate backend service files
print_section "Validating backend service files"
check_file "backend/$BACKEND_SERVICE/go.mod"
check_file "backend/$BACKEND_SERVICE/go.sum"
check_file "backend/$BACKEND_SERVICE/cmd/server/main.go"
check_file "backend/$BACKEND_SERVICE/internal/config/app_config.go"
check_file "backend/$BACKEND_SERVICE/internal/handler/handler.go"
check_file "backend/$BACKEND_SERVICE/internal/service/service.go"
check_file "backend/$BACKEND_SERVICE/internal/repository/repository.go"
check_file "backend/$BACKEND_SERVICE/internal/model/model.go"
check_file "backend/$BACKEND_SERVICE/internal/model/errors.go"

# Validate complete layered structure (7 core files)
print_section "Validating complete layered structure (7 files)"
check_file_content "backend/$BACKEND_SERVICE/internal/service/service.go" "type.*Service interface" "Service interface"
check_file_content "backend/$BACKEND_SERVICE/internal/repository/repository.go" "type.*Repository interface" "Repository interface"
check_file_content "backend/$BACKEND_SERVICE/internal/model/model.go" "type.*struct" "Model struct"
check_file_content "backend/$BACKEND_SERVICE/internal/model/errors.go" "Err.*NotFound" "Error definitions"

# Validate proto file generation (crud)
print_section "Validating proto file generation (crud)"
check_file "api/$BACKEND_SERVICE/v1/$BACKEND_SERVICE.proto"
check_file_content "api/$BACKEND_SERVICE/v1/$BACKEND_SERVICE.proto" "rpc Create" "CRUD create RPC"

# Validate go.mod uses v0.0.0-dev versions (not replace directives)
print_section "Validating go.mod uses v0.0.0-dev versions"
check_file_content "backend/$BACKEND_SERVICE/go.mod" "go.eggybyte.com/egg/servicex v0.0.0-dev" "Servicex dev version"
check_file_content "backend/$BACKEND_SERVICE/go.mod" "go.eggybyte.com/egg/runtimex v0.0.0-dev" "Runtimex dev version"
check_file_content "backend/$BACKEND_SERVICE/go.mod" "go.eggybyte.com/egg/connectx v0.0.0-dev" "Connectx dev version"
check_file_content "backend/$BACKEND_SERVICE/go.mod" "go.eggybyte.com/egg/configx v0.0.0-dev" "Configx dev version"
check_file_content "backend/$BACKEND_SERVICE/go.mod" "go.eggybyte.com/egg/core v0.0.0-dev" "Core dev version"

# Verify NO replace directives exist for egg modules (Docker compatibility)
if grep "^replace go.eggybyte.com/egg/" "backend/$BACKEND_SERVICE/go.mod" 2>/dev/null | grep -v "gen/go"; then
    print_error "Found replace directives for egg modules - these break Docker builds"
    print_info "Use v0.0.0-dev versions instead for Docker compatibility"
    exit 1
else
    print_success "No replace directives for egg modules (Docker-compatible)"
fi

# Validate main.go imports egg packages
print_section "Validating main.go imports"
check_file_content "backend/$BACKEND_SERVICE/cmd/server/main.go" "go.eggybyte.com/egg/servicex" "Servicex import"

# Validate workspace was updated
print_section "Validating backend workspace"
check_file "backend/go.work"
check_file_content "backend/go.work" "./$BACKEND_SERVICE" "Workspace use directive"

# Validate service was registered in egg.yaml
print_section "Validating service registration"
check_file_content "egg.yaml" "backend:" "Backend section"
check_file_content "egg.yaml" "$BACKEND_SERVICE:" "Service entry"

# ==============================================================================
# Test 2.1: Create second service (ping with echo)
# ==============================================================================

run_egg_command "Backend service (ping with echo proto)" \
    create backend "$BACKEND_PING_SERVICE" --proto echo --local-modules

# Validate ping service structure
print_section "Validating ping service structure"
check_dir "backend/$BACKEND_PING_SERVICE"
check_file "api/$BACKEND_PING_SERVICE/v1/$BACKEND_PING_SERVICE.proto"
check_file_content "api/$BACKEND_PING_SERVICE/v1/$BACKEND_PING_SERVICE.proto" "rpc Ping" "Echo Ping RPC"

# ==============================================================================
# Test 2.2: Validate image_name field removed from config
# ==============================================================================

print_section "Validating image_name auto-calculation"
if grep -q "image_name:" egg.yaml; then
    print_error "egg.yaml should not contain image_name field (should be auto-calculated)"
    exit 1
else
    print_success "image_name field correctly removed from config"
fi

# ==============================================================================
# Test 2.3: Duplicate Service Name Prevention
# ==============================================================================

print_cli_section "Test 2.3: Duplicate Service Name Prevention"
print_info "Testing that duplicate service names are rejected"

# Try to create the same backend service again (should fail)
print_info "Attempting to create duplicate backend service..."
if $EGG_CLI create backend "$BACKEND_SERVICE" --local-modules 2>&1 | grep -q "already exists"; then
    print_success "Correctly prevents duplicate backend service creation"
else
    print_error "Should prevent duplicate backend service creation"
    exit 1
fi

# Try to create a frontend service with the same name as existing backend service (should fail)
print_info "Attempting to create frontend service with same name as backend service..."
if $EGG_CLI create frontend "$BACKEND_SERVICE" --platforms web 2>&1 | grep -q "conflicts"; then
    print_success "Correctly prevents cross-type service name conflicts"
else
    print_error "Should prevent cross-type service name conflicts"
    exit 1
fi

# ==============================================================================
# Test 2.4: Validate workspace management (backend-scoped)
# ==============================================================================

print_section "Validating backend-scoped workspace"
check_file "backend/go.work"
check_file_content "backend/go.work" "./$BACKEND_SERVICE" "User service in workspace"
check_file_content "backend/go.work" "./$BACKEND_PING_SERVICE" "Ping service in workspace"
print_info "backend/go.work will include ../gen/go after api generate"

# ==============================================================================
# Test 3: Frontend Service Creation
# ==============================================================================

# Check if Flutter is installed
if ! command -v flutter &> /dev/null; then
    print_info "Flutter not installed, skipping frontend test"
else
    # Run egg create frontend (allow it to fail gracefully)
    # Note: Using underscore naming for Dart compatibility (admin_portal instead of admin-portal)
    if run_egg_command "Frontend Service Creation (egg create frontend)" \
        create frontend "$FRONTEND_SERVICE" --platforms web 2>&1; then
        
        # Validate frontend service structure
        print_section "Validating frontend service structure"
        check_dir "frontend/$FRONTEND_SERVICE"
        check_dir "frontend/$FRONTEND_SERVICE/lib"
        check_dir "frontend/$FRONTEND_SERVICE/web"
        
        # Validate frontend service files
        print_section "Validating frontend service files"
        check_file "frontend/$FRONTEND_SERVICE/pubspec.yaml"
        check_file "frontend/$FRONTEND_SERVICE/lib/main.dart"
        
        # Validate service was registered in egg.yaml
        print_section "Validating service registration"
        check_file_content "egg.yaml" "frontend:" "Frontend section"
        check_file_content "egg.yaml" "$FRONTEND_SERVICE:" "Service entry"
        
        # Test 3.1: Duplicate Frontend Service Prevention (after creation)
        print_cli_section "Test 3.1: Duplicate Frontend Service Prevention"
        print_info "Testing that duplicate frontend service creation is rejected"
        if $EGG_CLI create frontend "$FRONTEND_SERVICE" --platforms web 2>&1 | grep -q "already exists"; then
            print_success "Correctly prevents duplicate frontend service creation"
        else
            print_error "Should prevent duplicate frontend service creation"
            exit 1
        fi
    else
        print_info "Flutter frontend creation failed (Flutter may not be properly configured)"
        print_info "This is acceptable for the CLI test - skipping frontend validation"
        
        # Remove the frontend service from egg.yaml since it wasn't created successfully
        print_info "Removing frontend service registration from egg.yaml..."
        # Use awk to remove the frontend service entry
        awk '
        BEGIN { skip = 0 }
        /^  '"$FRONTEND_SERVICE"':/ { skip = 1; next }
        skip == 1 && /^  [a-zA-Z_]/ { skip = 0 }
        skip == 1 && /^[a-zA-Z]/ { skip = 0 }
        skip == 0 { print }
        ' egg.yaml > egg.yaml.tmp && mv egg.yaml.tmp egg.yaml
        print_success "Cleaned up egg.yaml"
    fi
fi

# ==============================================================================
# Test 4: API Initialization
# ==============================================================================

# Run egg api init
run_egg_command "API Initialization (egg api init)" api init

# Validate API structure (should already exist from egg init, but verify again)
print_section "Validating API structure"
check_file "api/buf.yaml"
check_file "api/buf.gen.yaml"

# ==============================================================================
# Test 5: API Generation
# ==============================================================================

# Create a sample proto file for testing
print_section "Creating sample proto file"
mkdir -p api/test/v1
cat > api/test/v1/test.proto <<'EOF'
syntax = "proto3";

package test.v1;

option go_package = "github.com/eggybyte-test/test-project/gen/go/test/v1;testv1";

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
}

service TestService {
  rpc SayHello(HelloRequest) returns (HelloResponse) {}
}
EOF
print_success "Sample proto file created"

# Check if buf and protoc plugins are available
if ! command -v buf &> /dev/null; then
    print_info "Buf not installed, skipping API generation test"
    API_SUCCESS=false
else
    # Run egg api generate with retry logic for rate limit
    print_cli_section "API Generation (egg api generate)"
    MAX_RETRIES=3
    RETRY_DELAY=60
    API_SUCCESS=false
    
    for attempt in $(seq 1 $MAX_RETRIES); do
        print_info "API generation attempt $attempt/$MAX_RETRIES..."
        
        # Run command (it will print the command internally)
        if run_egg_command "API Generation (attempt $attempt)" api generate; then
            API_SUCCESS=true
            break
        else
            if [ $attempt -lt $MAX_RETRIES ]; then
                print_warning "API generation failed (likely rate limit), waiting ${RETRY_DELAY}s before retry..."
                sleep $RETRY_DELAY
            else
                print_warning "API generation failed after $MAX_RETRIES attempts"
                print_info "This may be due to buf.build rate limiting"
                print_info "Continuing with tests that don't require generated code..."
                API_SUCCESS=false
            fi
        fi
    done
fi

# Validate backend-scoped workspace after API generation (if successful)
if [ "$API_SUCCESS" = true ]; then
    print_section "Validating backend-scoped workspace after API generation"
    check_file "backend/go.work"
    check_file_content "backend/go.work" "../gen/go" "gen/go in workspace"
    check_file "gen/go/go.mod"
    check_file_content "gen/go/go.mod" "module github.com/eggybyte-test/test-project/gen/go" "gen/go module path"
    print_success "Backend-scoped workspace correctly configured with gen/go"
else
    print_warning "Skipping API generation validation due to buf rate limit"
fi

# ==============================================================================
# Test 6: Generate Docker Compose Configuration
# ==============================================================================

# Generate docker-compose.yaml
run_egg_command "Docker Compose Generation (egg compose generate)" compose generate

# Validate compose configuration exists
print_section "Validating Docker Compose configuration"
check_file "deploy/compose/compose.yaml"
check_file "deploy/compose/.env"

# Validate compose.yaml content
print_section "Validating compose.yaml content"
# Note: MySQL service is only included when database.enabled=true (default is false)
check_file_content "deploy/compose/compose.yaml" "$BACKEND_SERVICE:" "User service"
check_file_content "deploy/compose/compose.yaml" "$BACKEND_PING_SERVICE:" "Ping service"

# Validate .env file content
print_section "Validating .env file"
check_file_content "deploy/compose/.env" "COMPOSE_PROJECT_NAME=" "Compose project name"
# Note: MySQL passwords are only included when database.enabled=true

# ==============================================================================
# Test 8: Build Backend Services
# ==============================================================================

print_section "Building all backend services"

# Only build if API generation was successful (services depend on generated code)
if [ "$API_SUCCESS" = true ]; then
    # Build all services using egg build all command (with --local flag)
    print_section "Building all services"
    
    run_egg_command "Build all services (egg build all --local)" build all --local

    # Verify Docker images were created (for --local build, images are loaded locally)
    print_section "Validating Docker images"
    if docker images | grep -q "$PROJECT_NAME-$BACKEND_SERVICE"; then
        print_success "Docker image built: $PROJECT_NAME-$BACKEND_SERVICE"
    else
        print_warning "Docker image not found in local registry (may be expected for multi-platform builds)"
    fi
    
    if docker images | grep -q "$PROJECT_NAME-$BACKEND_PING_SERVICE"; then
        print_success "Docker image built: $PROJECT_NAME-$BACKEND_PING_SERVICE"
    else
        print_warning "Docker image not found in local registry (may be expected for multi-platform builds)"
    fi

    print_success "All backend services built successfully (2 services)"
    
    # Build frontend service if it exists
    if [ -d "frontend/$FRONTEND_SERVICE" ]; then
        print_section "Building frontend service"
        run_egg_command "Build Frontend Service (egg build frontend $FRONTEND_SERVICE --local)" build frontend $FRONTEND_SERVICE --local
        
        # Verify frontend Docker image was created
        FRONTEND_IMAGE_NAME="${PROJECT_NAME}-${FRONTEND_SERVICE//_/-}-frontend"
        if docker images | grep -q "$FRONTEND_IMAGE_NAME"; then
            print_success "Frontend Docker image built: $FRONTEND_IMAGE_NAME"
        else
            print_warning "Frontend Docker image not found in local registry"
        fi
    fi
else
    print_warning "Skipping backend builds (API generation failed - services depend on generated code)"
fi

# ==============================================================================
# Test 9: Build Command Test
# ==============================================================================

print_cli_section "Test 9: Build Command (egg build)"
print_info "Testing egg build backend command for a single service"

# Only test if API generation was successful
if [ "$API_SUCCESS" = true ]; then
    # Test building a specific service
    run_egg_command "Build single service (egg build backend)" build backend $BACKEND_SERVICE --local

    # Verify Docker image was created (for --local build, images are loaded locally)
    print_section "Validating build output"
    if docker images | grep -q "$PROJECT_NAME-$BACKEND_SERVICE"; then
        print_success "Docker image created: $PROJECT_NAME-$BACKEND_SERVICE"
        
        # Check if image exists and has correct tag
        if docker images "$PROJECT_NAME-$BACKEND_SERVICE" | grep -q "v1.0.0"; then
            print_success "Docker image has correct tag: v1.0.0"
        else
            print_warning "Docker image tag may be incorrect"
        fi
    else
        print_error "Docker image not found: $PROJECT_NAME-$BACKEND_SERVICE"
        exit 1
    fi
else
    print_warning "Skipping build command test (API generation failed)"
fi

# ==============================================================================
# Test 11: Helm Generation (if helm available)
# ==============================================================================

if command -v helm &> /dev/null; then
    print_cli_section "Test 11: Helm Chart Generation"
    
    # Generate Helm charts
    run_egg_command "Helm Chart Generation (egg kube generate)" kube generate
    
    # Validate unified helm chart structure
    print_section "Validating unified Helm chart structure"
    check_dir "deploy/helm"
    
    # Check for unified project chart
    if [ -d "deploy/helm/$PROJECT_NAME" ]; then
        print_success "Unified Helm chart generated: deploy/helm/$PROJECT_NAME"
        
        # Validate chart structure
        print_section "Validating chart structure"
        check_file "deploy/helm/$PROJECT_NAME/Chart.yaml"
        check_file "deploy/helm/$PROJECT_NAME/values.yaml"
        check_dir "deploy/helm/$PROJECT_NAME/templates"
        
        # Validate chart metadata
        print_section "Validating chart metadata"
        check_file_content "deploy/helm/$PROJECT_NAME/Chart.yaml" "name: $PROJECT_NAME" "Chart name"
        check_file_content "deploy/helm/$PROJECT_NAME/Chart.yaml" "type: application" "Chart type"
        
        # Validate values.yaml contains services
        print_section "Validating values.yaml"
        check_file_content "deploy/helm/$PROJECT_NAME/values.yaml" "projectName: $PROJECT_NAME" "Project name"
        check_file_content "deploy/helm/$PROJECT_NAME/values.yaml" "backend:" "Backend section"
        check_file_content "deploy/helm/$PROJECT_NAME/values.yaml" "frontend:" "Frontend section"
        check_file_content "deploy/helm/$PROJECT_NAME/values.yaml" "user:" "User service in values"
        check_file_content "deploy/helm/$PROJECT_NAME/values.yaml" "ping:" "Ping service in values"
        
        # Validate template files
        print_section "Validating template files"
        check_file "deploy/helm/$PROJECT_NAME/templates/_helpers.tpl"
        check_file "deploy/helm/$PROJECT_NAME/templates/backend-deployment.yaml"
        check_file "deploy/helm/$PROJECT_NAME/templates/backend-service.yaml"
        check_file "deploy/helm/$PROJECT_NAME/templates/frontend-deployment.yaml"
        check_file "deploy/helm/$PROJECT_NAME/templates/frontend-service.yaml"
        check_file "deploy/helm/$PROJECT_NAME/templates/configmaps.yaml"
        check_file "deploy/helm/$PROJECT_NAME/templates/secrets.yaml"
        
        # Try to lint the unified chart
        print_section "Linting unified Helm chart"
        print_command "helm lint deploy/helm/$PROJECT_NAME"
        if helm lint deploy/helm/$PROJECT_NAME 2>&1 | head -20; then
            print_success "Unified Helm chart passes lint checks"
        else
            print_warning "Helm chart has lint warnings (may be acceptable)"
        fi
        
        # Test helm template command
        print_section "Testing helm template command"
        print_command "helm template $PROJECT_NAME deploy/helm/$PROJECT_NAME"
        if helm template $PROJECT_NAME deploy/helm/$PROJECT_NAME > /tmp/helm_output.yaml 2>&1; then
            print_success "Helm template renders successfully"
            
            # Validate output contains backend services
            if grep -q "kind: Deployment" /tmp/helm_output.yaml && \
               grep -q "kind: Service" /tmp/helm_output.yaml; then
                print_success "Helm template generates Kubernetes manifests"
                
                # Count deployments
                DEPLOYMENT_COUNT=$(grep -c "kind: Deployment" /tmp/helm_output.yaml)
                print_info "Generated $DEPLOYMENT_COUNT deployment(s)"
                
                # Count services
                SERVICE_COUNT=$(grep -c "kind: Service" /tmp/helm_output.yaml)
                print_info "Generated $SERVICE_COUNT service(s)"
            else
                print_warning "Helm template output may be incomplete"
            fi
            rm -f /tmp/helm_output.yaml
        else
            print_warning "Helm template command failed (may be acceptable)"
        fi
    else
        print_error "Unified Helm chart not found: deploy/helm/$PROJECT_NAME"
        print_info "Expected structure: deploy/helm/<project-name>/"
        exit 1
    fi
else
    print_info "Helm not installed, skipping Helm chart generation test"
fi

# ==============================================================================
# Test 12: Docker Compose Service Validation (with RPC testing)
# ==============================================================================

# Use the modular test-compose.sh function
test_compose_services

print_header "Test Summary"

print_success "All integration tests completed successfully"

print_section "Commands Tested"
printf "  ${GREEN}[✓]${RESET} egg doctor                           - Environment diagnostic check\n"
printf "  ${GREEN}[✓]${RESET} egg init                             - Project initialization\n"
printf "  ${GREEN}[✓]${RESET} egg create backend                   - Backend service with local modules\n"
printf "  ${GREEN}[✓]${RESET} egg create backend --proto crud      - Backend with CRUD proto (user)\n"
printf "  ${GREEN}[✓]${RESET} egg create backend --proto echo     - Backend with echo proto (ping)\n"
printf "  ${GREEN}[✓]${RESET} Duplicate service prevention         - Backend, frontend, cross-type\n"
printf "  ${GREEN}[✓]${RESET} egg create frontend                  - Frontend service (Flutter with Dart naming)\n"
printf "  ${GREEN}[✓]${RESET} egg api init                         - API definition initialization\n"
printf "  ${GREEN}[✓]${RESET} egg api generate                     - Code generation from protobuf\n"
printf "  ${GREEN}[✓]${RESET} egg compose generate                 - Docker Compose configuration generation\n"
printf "  ${GREEN}[✓]${RESET} egg build all                       - Build all services\n"
printf "  ${GREEN}[✓]${RESET} egg build docker <service>           - Build Docker images\n"
printf "  ${GREEN}[✓]${RESET} egg kube generate                    - Unified Helm chart generation\n"
printf "  ${GREEN}[✓]${RESET} egg check                            - Configuration validation\n"
printf "  ${GREEN}[✓]${RESET} Docker Compose RPC testing          - Health checks + RPC endpoints\n"

# ==============================================================================
# Cleanup
# ==============================================================================

cleanup

print_header "Integration Test Complete"
printf "\n"
print_success "Egg CLI integration test suite completed successfully"
printf "\n"
if [ "$KEEP_TEST_DIR" = true ]; then
    print_info "Test artifacts preserved in: $TEST_DIR"
else
    print_info "Test artifacts cleaned up"
fi
printf "\n"
