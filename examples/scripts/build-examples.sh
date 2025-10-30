#!/bin/bash

# Build Examples Script for Egg Framework
# Provides comprehensive build functionality for example services
#
# Usage:
#   ./scripts/build-examples.sh [command]
#
# Commands:
#   service     Build a specific service binary and Docker image
#   all         Build all example services
#   clean       Clean build artifacts
#   help        Show this help message
#
# Examples:
#   ./scripts/build-examples.sh all
#   ./scripts/build-examples.sh service minimal-connect-service
#   ./scripts/build-examples.sh clean

set -e

# Source the unified logger
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Get the examples root directory (parent of scripts/)
EXAMPLES_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROJECT_ROOT="$(cd "$EXAMPLES_ROOT/.." && pwd)"
source "$PROJECT_ROOT/scripts/logger.sh"

# Check if Docker is running
check_docker() {
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
}

# Build a specific service
# Usage: build_service <service_name> <service_dir> <build_path> <http_port> <health_port> <metrics_port>
build_service() {
    local service_name="$1"
    local service_dir="$2"
    local build_path="$3"
    local http_port="$4"
    local health_port="$5"
    local metrics_port="$6"
    
    # Validate arguments
    if [ -z "$service_name" ] || [ -z "$service_dir" ] || [ -z "$build_path" ]; then
        print_error "Usage: build_service <service_name> <service_dir> <build_path> [http_port] [health_port] [metrics_port]"
        return 1
    fi
    
    # Validate service directory
    if [ ! -d "$EXAMPLES_ROOT/$service_dir" ]; then
        print_error "Service directory $EXAMPLES_ROOT/$service_dir does not exist"
        return 1
    fi
    
    # Validate build path
    if [ ! -d "$EXAMPLES_ROOT/$service_dir/$build_path" ]; then
        print_error "Build path $EXAMPLES_ROOT/$service_dir/$build_path does not exist"
        return 1
    fi
    
    print_header "Building service: $service_name"
    
    print_info "Service: $service_dir"
    print_info "Binary: $service_name"
    print_info "Build path: $build_path"
    print_info "Ports: HTTP=$http_port, Health=$health_port, Metrics=$metrics_port"
    
    # Create bin directory if it doesn't exist
    mkdir -p "$EXAMPLES_ROOT/bin"
    
    # Detect native architecture for matching binary and Docker platform
    # This ensures the binary architecture matches the Docker image platform others build
    local ARCH="amd64"
    local DOCKER_PLATFORM="linux/amd64"
    if [ "$(uname -m)" = "arm64" ] || [ "$(uname -m)" = "aarch64" ]; then
        ARCH="arm64"
        DOCKER_PLATFORM="linux/arm64"
    fi
    
    # Build the binary for the detected architecture
    # Note: eggybyte-go-alpine base image supports multi-platform (amd64/arm64),
    # but Docker needs explicit --platform to select the matching variant
    print_info "Compiling Go binary for $ARCH architecture..."
    cd "$EXAMPLES_ROOT/$service_dir"
    
    if ! CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -a -installsuffix cgo \
        -o "$EXAMPLES_ROOT/bin/$service_name" "./$build_path"; then
        print_error "Failed to compile $service_name"
        return 1
    fi
    
    print_success "Binary built: $EXAMPLES_ROOT/bin/$service_name (linux/$ARCH)"
    
    # Build Docker image with explicit platform to match binary architecture
    # Pass TARGETPLATFORM as build arg so Dockerfile FROM can use it to pull correct base image
    # Use --no-cache for base image to ensure we pull the matching platform variant
    print_info "Building Docker image for $DOCKER_PLATFORM (matching binary architecture)..."
    cd "$PROJECT_ROOT"
    
    # Use buildx if available (better multi-platform support), fallback to regular build
    if docker buildx version >/dev/null 2>&1; then
        print_info "Using docker buildx for better platform handling..."
        if ! docker buildx build -f examples/docker/Dockerfile \
            --platform "$DOCKER_PLATFORM" \
            --load \
            --build-arg SERVICE_NAME="$service_name" \
            --build-arg BINARY_PATH="examples/bin/$service_name" \
            --build-arg HTTP_PORT="$http_port" \
            --build-arg HEALTH_PORT="$health_port" \
            --build-arg METRICS_PORT="$metrics_port" \
            -t "$service_name:latest" \
            .; then
            print_error "Failed to build Docker image for $service_name"
            return 1
        fi
    else
        # Fallback to regular docker build with explicit platform
        # Note: --pull ensures we fetch the correct platform variant (not use cached wrong arch)
        if ! docker build -f examples/docker/Dockerfile \
            --platform "$DOCKER_PLATFORM" \
            --pull \
            --build-arg SERVICE_NAME="$service_name" \
            --build-arg BINARY_PATH="examples/bin/$service_name" \
            --build-arg HTTP_PORT="$http_port" \
            --build-arg HEALTH_PORT="$health_port" \
            --build-arg METRICS_PORT="$metrics_port" \
            -t "$service_name:latest" \
            .; then
            print_error "Failed to build Docker image for $service_name"
            return 1
        fi
    fi
    
    print_success "Docker image built: $service_name:latest"
}

# Build all example services
build_all() {
    check_docker
    
    print_header "Building all example services"
    
    local build_failed=0
    
    # Build minimal-connect-service
    print_info "Step 1/2: Building minimal-connect-service..."
    if ! build_service "minimal-connect-service" "minimal-connect-service" "cmd/server" "8080" "8081" "9091"; then
        print_error "Failed to build minimal-connect-service"
        build_failed=1
    fi
    
    # Build user-service
    print_info "Step 2/2: Building user-service..."
    if ! build_service "user-service" "user-service" "cmd/server" "8082" "8083" "9092"; then
        print_error "Failed to build user-service"
        build_failed=1
    fi
    
    if [ $build_failed -eq 0 ]; then
        print_success "All example services built successfully!"
        echo ""
        print_info "Available binaries:"
        print_info "  - $EXAMPLES_ROOT/bin/minimal-connect-service"
        print_info "  - $EXAMPLES_ROOT/bin/user-service"
        echo ""
        print_info "Available images:"
        print_info "  - minimal-connect-service:latest"
        print_info "  - user-service:latest"
        echo ""
        print_info "You can now run docker-compose in the deploy directory."
        return 0
    else
        print_error "Some services failed to build"
        return 1
    fi
}

# Clean build artifacts
clean_build() {
    print_header "Cleaning build artifacts"
    
    print_info "Removing binary files..."
    if [ -d "$EXAMPLES_ROOT/bin" ]; then
        # Keep .gitkeep but remove binaries
        find "$EXAMPLES_ROOT/bin" -type f ! -name '.gitkeep' -delete
        print_success "Binary files removed"
    else
        print_info "No bin directory found (nothing to clean)"
    fi
    
    print_info "Removing Docker images..."
    docker rmi -f minimal-connect-service:latest 2>/dev/null || true
    docker rmi -f user-service:latest 2>/dev/null || true
    
    print_success "Build artifacts cleaned"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  all         Build all example services"
    echo "  service     Build a specific service (requires additional args)"
    echo "  clean       Clean build artifacts"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 all"
    echo "  $0 clean"
}

# Main script logic
case "${1:-}" in
    "all")
        init_logging "Example Services Build"
        build_all
        finalize_logging $? "Example Services Build"
        ;;
    "service")
        shift
        init_logging "Service Build"
        build_service "$@"
        finalize_logging $? "Service Build"
        ;;
    "clean")
        init_logging "Cleanup"
        clean_build
        finalize_logging $? "Cleanup"
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

