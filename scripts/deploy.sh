#!/bin/bash

# Unified deployment script for Egg Framework
# This script provides comprehensive deployment functionality
#
# Usage:
#   ./scripts/deploy.sh [command] [options]
#
# Commands:
#   up          Start all services
#   down        Stop all services
#   restart     Restart all services
#   logs        Show service logs
#   status      Show service status
#   health      Check service health
#   clean       Clean deployment artifacts
#
# Examples:
#   ./scripts/deploy.sh up
#   ./scripts/deploy.sh down
#   ./scripts/deploy.sh restart
#   ./scripts/deploy.sh logs
#   ./scripts/deploy.sh status
#   ./scripts/deploy.sh health
#   ./scripts/deploy.sh clean

set -e

# Source the unified logger
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

# Get the project root directory
PROJECT_ROOT="$(get_project_root)"



# Start all services (infrastructure + application services)
deploy_up() {
    print_header "Starting all services"
    
    # Clean up any existing containers first
    print_info "Cleaning up any existing containers..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose -f docker-compose.services.yaml down --remove-orphans 2>/dev/null || true
    docker-compose -f docker-compose.infra.yaml down --remove-orphans 2>/dev/null || true
    
    # Run port cleanup
    print_info "Running port cleanup..."
    if ! "$PROJECT_ROOT/scripts/cleanup-ports.sh"; then
        print_warning "Some ports could not be freed, but continuing anyway..."
    fi
    
    # Build services if needed
    print_info "Building services..."
    "$PROJECT_ROOT/scripts/build.sh" all
    
    # Start infrastructure first
    print_info "Starting infrastructure services..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose -f docker-compose.infra.yaml up -d
    
    # Wait for infrastructure to be ready
    print_info "Waiting for infrastructure to be ready..."
    sleep 15
    
    # Start application services
    print_info "Starting application services..."
    docker-compose -f docker-compose.services.yaml up -d
    
    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 10
    
    # Check service health
    print_info "Checking service health..."
    deploy_health
    
    print_success "All services started successfully!"
    print_info "Service URLs:"
    print_info "  - Minimal Service: http://localhost:8080"
    print_info "  - User Service: http://localhost:8082"
    print_info "  - Jaeger UI: http://localhost:16686"
    print_info "  - Prometheus Metrics: http://localhost:8889/metrics"
}

# Stop all services (application + infrastructure)
deploy_down() {
    print_header "Stopping all services"
    
    print_info "Stopping application services..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose -f docker-compose.services.yaml down --remove-orphans 2>/dev/null || true
    
    print_info "Stopping infrastructure services..."
    docker-compose -f docker-compose.infra.yaml down --remove-orphans 2>/dev/null || true
    
    print_success "All services stopped"
}

# Restart all services (application + infrastructure)
deploy_restart() {
    print_header "Restarting all services"
    
    deploy_down
    sleep 2
    deploy_up
    
    print_success "All services restarted"
}

# Show service logs
deploy_logs() {
    local service="${1:-}"
    
    print_header "Showing service logs"
    
    cd "$PROJECT_ROOT/deploy"
    
    if [ -n "$service" ]; then
        print_info "Showing logs for service: $service"
        docker-compose -f docker-compose.services.yaml logs -f "$service" 2>/dev/null || \
        docker-compose -f docker-compose.infra.yaml logs -f "$service"
    else
        print_info "Showing logs for all services"
        docker-compose -f docker-compose.services.yaml logs -f
    fi
}

# Show service status
deploy_status() {
    print_header "Service status"
    
    cd "$PROJECT_ROOT/deploy"
    print_info "Infrastructure services:"
    docker-compose -f docker-compose.infra.yaml ps
    echo
    print_info "Application services:"
    docker-compose -f docker-compose.services.yaml ps
    
    print_info "Service URLs:"
    print_info "  - Minimal Service: http://localhost:8080"
    print_info "  - User Service: http://localhost:8082"
    print_info "  - Jaeger UI: http://localhost:16686"
    print_info "  - Prometheus Metrics: http://localhost:8889/metrics"
}

# Check service health
deploy_health() {
    print_header "Checking service health"
    
    # Check minimal service
    print_info "Checking minimal service..."
    if curl -f http://localhost:8081/health > /dev/null 2>&1; then
        print_success "Minimal service is healthy"
    else
        print_warning "Minimal service health check failed"
    fi
    
    # Check user service
    print_info "Checking user service..."
    if curl -f http://localhost:8083/health > /dev/null 2>&1; then
        print_success "User service is healthy"
    else
        print_warning "User service health check failed"
    fi
    
    # Check Jaeger
    print_info "Checking Jaeger..."
    if curl -f http://localhost:16686 > /dev/null 2>&1; then
        print_success "Jaeger is healthy"
    else
        print_warning "Jaeger health check failed"
    fi
    
    # Check Prometheus metrics
    print_info "Checking Prometheus metrics..."
    if curl -f http://localhost:8889/metrics > /dev/null 2>&1; then
        print_success "Prometheus metrics are available"
    else
        print_warning "Prometheus metrics check failed"
    fi
}

# Clean deployment artifacts
deploy_clean() {
    print_header "Cleaning deployment artifacts"
    
    # Stop services
    print_info "Stopping services..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose -f docker-compose.services.yaml down 2>/dev/null || true
    docker-compose -f docker-compose.infra.yaml down 2>/dev/null || true
    
    # Remove containers
    print_info "Removing containers..."
    docker-compose -f docker-compose.services.yaml rm -f 2>/dev/null || true
    docker-compose -f docker-compose.infra.yaml rm -f 2>/dev/null || true
    
    # Remove volumes
    print_info "Removing volumes..."
    docker-compose -f docker-compose.infra.yaml down -v 2>/dev/null || true
    
    # Remove images
    print_info "Removing images..."
    docker rmi -f egg-minimal-service 2>/dev/null || true
    docker rmi -f egg-user-service 2>/dev/null || true
    docker rmi -f otel/opentelemetry-collector-contrib:latest 2>/dev/null || true
    docker rmi -f jaegertracing/all-in-one:latest 2>/dev/null || true
    docker rmi -f mysql:9.4 2>/dev/null || true
    
    print_success "Deployment artifacts cleaned"
}

# Show usage information
show_usage() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  up          Start all services"
    echo "  down        Stop all services"
    echo "  restart     Restart all services"
    echo "  logs        Show service logs"
    echo "  status      Show service status"
    echo "  health      Check service health"
    echo "  clean       Clean deployment artifacts"
    echo ""
    echo "Examples:"
    echo "  $0 up"
    echo "  $0 down"
    echo "  $0 restart"
    echo "  $0 logs"
    echo "  $0 logs minimal-service"
    echo "  $0 status"
    echo "  $0 health"
    echo "  $0 clean"
}

# Main script logic
case "${1:-}" in
    "up")
        deploy_up
        ;;
    "down")
        deploy_down
        ;;
    "restart")
        deploy_restart
        ;;
    "logs")
        shift
        deploy_logs "$@"
        ;;
    "status")
        deploy_status
        ;;
    "health")
        deploy_health
        ;;
    "clean")
        deploy_clean
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
