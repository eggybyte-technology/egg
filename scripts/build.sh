#!/bin/bash

# Unified build script for Egg Framework
# This script provides comprehensive build functionality for all components
#
# Usage:
#   ./scripts/build.sh [command] [options]
#
# Commands:
#   base        Build the eggybyte-go-alpine base image
#   service     Build a Go service binary and Docker image
#   all         Build all services (base + examples)
#   clean       Clean build artifacts
#
# Examples:
#   ./scripts/build.sh base
#   ./scripts/build.sh service examples/minimal-connect-service minimal-connect-service
#   ./scripts/build.sh service examples/user-service user-service cmd/server
#   ./scripts/build.sh all
#   ./scripts/build.sh clean

set -e

# Source the unified logger
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

# Get the project root directory
PROJECT_ROOT="$(get_project_root)"

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

# Pull the base image (no longer building locally)
pull_base() {
    check_docker
    print_header "Pulling eggybyte-go-alpine base image"
    
    print_info "Pulling base image from remote registry..."
    docker pull ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest
    
    print_success "Base image pulled: ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest"
    
    # Tag for local registry (optional)
    print_info "Tagging for local registry..."
    docker tag ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest localhost:5000/eggybyte-go-alpine:latest
    
    print_success "Base image ready for use!"
}

# Build a Go service
build_service() {
    check_docker
    
    local service_dir="$1"
    local binary_name="$2"
    local build_path="${3:-.}"
    local http_port="${4:-8080}"
    local health_port="${5:-8081}"
    local metrics_port="${6:-9091}"
    local image_name="${7:-${binary_name}:latest}"
    
    # Validate arguments
    if [ -z "$service_dir" ] || [ -z "$binary_name" ]; then
        print_error "Usage: $0 service <service_dir> <binary_name> [build_path] [http_port] [health_port] [metrics_port] [image_name]"
        print_info "Examples:"
        print_info "  $0 service examples/minimal-connect-service minimal-connect-service"
        print_info "  $0 service examples/user-service user-service cmd/server"
        exit 1
    fi
    
    # Validate service directory
    if [ ! -d "$PROJECT_ROOT/$service_dir" ]; then
        print_error "Service directory $PROJECT_ROOT/$service_dir does not exist"
        exit 1
    fi
    
    # Validate build path
    if [ ! -d "$PROJECT_ROOT/$service_dir/$build_path" ]; then
        print_error "Build path $PROJECT_ROOT/$service_dir/$build_path does not exist"
        exit 1
    fi
    
    print_header "Building Go service: $binary_name"
    
    print_info "Service: $service_dir"
    print_info "Binary: $binary_name"
    print_info "Build path: $build_path"
    print_info "Ports: HTTP=$http_port, Health=$health_port, Metrics=$metrics_port"
    print_info "Image: $image_name"
    
    # Create bin directory if it doesn't exist
    mkdir -p "$PROJECT_ROOT/bin"
    
    # Build the binary
    print_info "Compiling Go binary..."
    cd "$PROJECT_ROOT/$service_dir"
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o "$PROJECT_ROOT/bin/$binary_name" "./$build_path"
    
    print_success "Binary built: $PROJECT_ROOT/bin/$binary_name"
    
    # Build Docker image
    print_info "Building Docker image..."
    cd "$PROJECT_ROOT"
    docker build -f build/Dockerfile.backend \
        --build-arg BINARY_NAME="$binary_name" \
        --build-arg HTTP_PORT="$http_port" \
        --build-arg HEALTH_PORT="$health_port" \
        --build-arg METRICS_PORT="$metrics_port" \
        -t "$image_name" \
        .
    
    print_success "Docker image built: $image_name"
}

# Build all services
build_all() {
    print_header "Building all services"
    
    # Pull base image first
    print_info "Step 1: Pulling base image..."
    pull_base
    
    # Build minimal-connect-service
    print_info "Step 2: Building minimal-connect-service..."
    build_service "examples/minimal-connect-service" "minimal-connect-service" "cmd/server"
    
    # Build user-service
    print_info "Step 3: Building user-service..."
    build_service "examples/user-service" "user-service" "cmd/server"
    
    print_success "All services built successfully!"
    echo ""
    print_info "Available images:"
    print_info "  - ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest (base image)"
    print_info "  - minimal-connect-service:latest"
    print_info "  - user-service:latest"
    echo ""
    print_info "You can now run docker-compose in the deploy directory."
}

# Clean build artifacts
clean_build() {
    print_header "Cleaning build artifacts"
    
    print_info "Removing binary files..."
    rm -rf "$PROJECT_ROOT/bin"
    
    print_info "Removing Docker images..."
    docker rmi -f minimal-connect-service:latest 2>/dev/null || true
    docker rmi -f user-service:latest 2>/dev/null || true
    docker rmi -f ghcr.io/eggybyte-technology/eggybyte-go-alpine:latest 2>/dev/null || true
    
    print_success "Build artifacts cleaned"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  base        Pull the eggybyte-go-alpine base image from remote registry"
    echo "  service     Build a Go service binary and Docker image"
    echo "  all         Build all services (pull base + examples)"
    echo "  clean       Clean build artifacts"
    echo ""
    echo "Examples:"
    echo "  $0 base"
    echo "  $0 service examples/minimal-connect-service minimal-connect-service"
    echo "  $0 service examples/user-service user-service cmd/server"
    echo "  $0 all"
    echo "  $0 clean"
}

# Main script logic
case "${1:-}" in
    "base")
        pull_base
        ;;
    "service")
        shift
        build_service "$@"
        ;;
    "all")
        build_all
        ;;
    "clean")
        clean_build
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
