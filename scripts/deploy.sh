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

# Start all services
deploy_up() {
    print_header "Starting all services"
    
    # Build services if needed
    print_info "Building services..."
    "$PROJECT_ROOT/scripts/build.sh" all
    
    # Start services
    print_info "Starting services with docker-compose..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose up -d
    
    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 15
    
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

# Stop all services
deploy_down() {
    print_header "Stopping all services"
    
    print_info "Stopping services..."
    cd "$PROJECT_ROOT/deploy"
    docker-compose down
    
    print_success "All services stopped"
}

# Restart all services
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
        docker-compose logs -f "$service"
    else
        print_info "Showing logs for all services"
        docker-compose logs -f
    fi
}

# Show service status
deploy_status() {
    print_header "Service status"
    
    cd "$PROJECT_ROOT/deploy"
    docker-compose ps
    
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
    docker-compose down 2>/dev/null || true
    
    # Remove containers
    print_info "Removing containers..."
    docker-compose rm -f 2>/dev/null || true
    
    # Remove volumes
    print_info "Removing volumes..."
    docker volume rm deploy_mysql_data 2>/dev/null || true
    
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
