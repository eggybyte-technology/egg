#!/bin/bash

# Port Cleanup Script for Egg Framework
# This script ensures all required ports are free before starting services
#
# Usage:
#   ./scripts/cleanup-ports.sh
#
# Note: Compatible with both Linux and macOS (bash 3.x+)

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}[✓] SUCCESS:${NC} $1"
}

print_info() {
    echo -e "${CYAN}[i] INFO:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!] WARNING:${NC} $1"
}

# Define ports used by Egg services (using arrays instead of associative arrays for bash 3.x compatibility)
PORTS=(
    "3306:MySQL"
    "4317:OTLP gRPC"
    "4318:OTLP HTTP"
    "8080:Minimal Service HTTP"
    "8081:Minimal Service Health"
    "8082:User Service HTTP"
    "8083:User Service Health"
    "8889:Prometheus Metrics"
    "9091:Minimal Service Metrics"
    "9092:User Service Metrics"
    "14268:Jaeger Collector"
    "16686:Jaeger UI"
)

print_info "Checking for port conflicts..."

ports_freed=0
for port_info in "${PORTS[@]}"; do
    port="${port_info%%:*}"
    service="${port_info#*:}"
    
    # Check if port is in use
    if lsof -ti:$port >/dev/null 2>&1; then
        print_warning "Port $port ($service) is in use"
        
        # Get process info
        pid=$(lsof -ti:$port)
        process=$(ps -p $pid -o comm= 2>/dev/null || echo "unknown")
        
        print_info "  Process: $process (PID: $pid)"
        print_info "  Attempting to free port $port..."
        
        # Try to kill the process
        if kill -9 $pid 2>/dev/null; then
            print_success "  Port $port freed"
            ports_freed=$((ports_freed + 1))
        else
            print_warning "  Failed to free port $port (may require sudo)"
        fi
    fi
done

# Wait a moment for ports to be fully released
if [ $ports_freed -gt 0 ]; then
    print_info "Waiting for ports to be fully released..."
    sleep 2
fi

# Verify all ports are free
print_info "Verifying all ports are free..."
conflicts=0
for port_info in "${PORTS[@]}"; do
    port="${port_info%%:*}"
    service="${port_info#*:}"
    if lsof -ti:$port >/dev/null 2>&1; then
        print_warning "Port $port ($service) is still in use"
        conflicts=$((conflicts + 1))
    fi
done

if [ $conflicts -eq 0 ]; then
    print_success "All ports are free and ready for use"
    exit 0
else
    print_warning "$conflicts port(s) still in use"
    print_info "You may need to manually stop conflicting services or use sudo"
    exit 1
fi

