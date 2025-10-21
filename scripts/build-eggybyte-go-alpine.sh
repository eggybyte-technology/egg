#!/bin/bash

# ================================================================
# EggyByte Go Alpine Base Image Build Script
# ================================================================
# 
# Purpose: Build and publish eggybyte-go-alpine base image for 
#          multi-architecture (arm64, amd64) deployment
# 
# Usage: ./scripts/build-eggybyte-go-alpine.sh [VERSION] [--push]
# 
# Arguments:
#   VERSION  - Image version tag (default: latest)
#   --push   - Push image to registry after building
# 
# Environment Variables:
#   DOCKER_REGISTRY - Registry URL (default: ghcr.io/eggybyte)
#   IMAGE_NAME      - Image name (default: eggybyte-go-alpine)
#   DOCKERFILE      - Dockerfile path (default: build/Dockerfile.eggybyte-go-alpine)
# 
# Examples:
#   ./scripts/build-eggybyte-go-alpine.sh
#   ./scripts/build-eggybyte-go-alpine.sh v1.0.0
#   ./scripts/build-eggybyte-go-alpine.sh v1.0.0 --push
# ================================================================

set -euo pipefail

# Color definitions for enhanced output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
RESET='\033[0m'

# Output formatting functions
print_header() {
    echo ""
    echo -e "${BLUE}================================================================================${RESET}"
    echo -e "${BLUE}${BOLD}▶ $1${RESET}"
    echo -e "${BLUE}================================================================================${RESET}"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${RESET} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${RESET} $1"
}

print_info() {
    echo -e "${CYAN}[INFO]${RESET} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${RESET} $1"
}

# Default configuration
DOCKER_REGISTRY="${DOCKER_REGISTRY:-ghcr.io/eggybyte-technology}"
IMAGE_NAME="${IMAGE_NAME:-eggybyte-go-alpine}"
DOCKERFILE="${DOCKERFILE:-build/Dockerfile.eggybyte-go-alpine}"
VERSION="latest"
PUSH_IMAGE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --help|-h)
            echo "Usage: $0 [VERSION] [--push]"
            echo ""
            echo "Arguments:"
            echo "  VERSION  - Image version tag (default: latest)"
            echo "  --push   - Push image to registry after building"
            echo ""
            echo "Environment Variables:"
            echo "  DOCKER_REGISTRY - Registry URL (default: ghcr.io/eggybyte-technology)"
            echo "  IMAGE_NAME      - Image name (default: eggybyte-go-alpine)"
            echo "  DOCKERFILE      - Dockerfile path (default: build/Dockerfile.eggybyte-go-alpine)"
            echo ""
            echo "Examples:"
            echo "  $0"
            echo "  $0 v1.0.0"
            echo "  $0 v1.0.0 --push"
            exit 0
            ;;
        --push)
            PUSH_IMAGE=true
            shift
            ;;
        -*)
            echo "Unknown option $1"
            echo "Use --help for usage information"
            exit 1
            ;;
        *)
            VERSION="$1"
            shift
            ;;
    esac
done

# Construct full image name
FULL_IMAGE_NAME="${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"

# Validate prerequisites
validate_prerequisites() {
    print_header "Validating Prerequisites"
    
    # Check if Docker is installed
    if ! command -v docker >/dev/null 2>&1; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    # Check if Docker is running
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    
    # Check if Dockerfile exists
    if [[ ! -f "$DOCKERFILE" ]]; then
        print_error "Dockerfile not found: $DOCKERFILE"
        exit 1
    fi
    
    # Check if buildx is available
    if ! docker buildx version >/dev/null 2>&1; then
        print_error "Docker buildx is not available"
        print_info "Please enable buildx: docker buildx create --use"
        exit 1
    fi
    
    print_success "Prerequisites validated"
}

# Setup buildx builder
setup_buildx() {
    print_header "Setting up Docker Buildx"
    
    # Create and use buildx builder if it doesn't exist
    BUILDER_NAME="eggybyte-multiarch"
    
    if ! docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
        print_info "Creating buildx builder: $BUILDER_NAME"
        docker buildx create --name "$BUILDER_NAME" --use
    else
        print_info "Using existing buildx builder: $BUILDER_NAME"
        docker buildx use "$BUILDER_NAME"
    fi
    
    # Bootstrap the builder
    print_info "Bootstrapping buildx builder..."
    docker buildx inspect --bootstrap
    
    print_success "Buildx setup completed"
}

# Build multi-architecture image
build_image() {
    print_header "Building Multi-Architecture Image"
    
    print_info "Image: $FULL_IMAGE_NAME"
    print_info "Dockerfile: $DOCKERFILE"
    print_info "Platforms: linux/amd64, linux/arm64"
    
    # Build the image
    if [[ "$PUSH_IMAGE" == "true" ]]; then
        docker buildx build \
            --platform linux/amd64,linux/arm64 \
            --file "$DOCKERFILE" \
            --tag "$FULL_IMAGE_NAME" \
            --tag "${DOCKER_REGISTRY}/${IMAGE_NAME}:latest" \
            --metadata-file /tmp/build-metadata.json \
            --push \
            .
    else
        docker buildx build \
            --platform linux/amd64,linux/arm64 \
            --file "$DOCKERFILE" \
            --tag "$FULL_IMAGE_NAME" \
            --tag "${DOCKER_REGISTRY}/${IMAGE_NAME}:latest" \
            --metadata-file /tmp/build-metadata.json \
            --load \
            .
    fi
    
    if [[ $? -eq 0 ]]; then
        print_success "Image built successfully"
        
        # Show build metadata if available
        if [[ -f /tmp/build-metadata.json ]]; then
            print_info "Build metadata:"
            cat /tmp/build-metadata.json | jq '.' 2>/dev/null || cat /tmp/build-metadata.json
            rm -f /tmp/build-metadata.json
        fi
    else
        print_error "Image build failed"
        exit 1
    fi
}

# Verify image
verify_image() {
    print_header "Verifying Built Image"
    
    if [[ "$PUSH_IMAGE" == "true" ]]; then
        print_info "Image was pushed to registry - verification skipped"
        print_info "You can verify manually with:"
        print_info "  docker buildx imagetools inspect $FULL_IMAGE_NAME"
    else
        print_info "Verifying local image..."
        
        # Check if image exists locally
        if docker buildx imagetools inspect "$FULL_IMAGE_NAME" >/dev/null 2>&1; then
            print_success "Image verification passed"
            
            # Show image details
            print_info "Image details:"
            docker buildx imagetools inspect "$FULL_IMAGE_NAME"
        else
            print_warning "Image not found locally (this is normal for multi-arch builds)"
            print_info "Image was built successfully but not loaded locally"
            print_info "To load locally, use: docker buildx build --load"
        fi
    fi
}

# Show usage information
show_usage() {
    print_header "Usage Information"
    
    echo "Built image: $FULL_IMAGE_NAME"
    echo ""
    echo "To use this image in your Dockerfile:"
    echo "  FROM $FULL_IMAGE_NAME"
    echo ""
    echo "To push to registry:"
    echo "  ./scripts/build-eggybyte-go-alpine.sh $VERSION --push"
    echo ""
    echo "To inspect the image:"
    echo "  docker buildx imagetools inspect $FULL_IMAGE_NAME"
    echo ""
    echo "To test locally:"
    echo "  docker run --rm $FULL_IMAGE_NAME /bin/sh"
}

# Main execution
main() {
    print_header "EggyByte Go Alpine Base Image Builder"
    
    print_info "Configuration:"
    print_info "  Registry: $DOCKER_REGISTRY"
    print_info "  Image: $IMAGE_NAME"
    print_info "  Version: $VERSION"
    print_info "  Dockerfile: $DOCKERFILE"
    print_info "  Push: $PUSH_IMAGE"
    print_info "  Full name: $FULL_IMAGE_NAME"
    echo ""
    
    validate_prerequisites
    setup_buildx
    build_image
    verify_image
    show_usage
    
    print_header "Build Completed Successfully"
    print_success "EggyByte Go Alpine base image built and ready for use"
}

# Run main function
main "$@"
