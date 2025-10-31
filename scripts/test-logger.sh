#!/bin/bash

# Test script for logger.sh functions
# This script demonstrates all logging functions with sample outputs

# Source the logger script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

echo "=========================================="
echo "Logger Functions Test Output"
echo "=========================================="
echo ""

echo "--- print_header ---"
print_header "Test Header"

echo ""
echo "--- print_success ---"
print_success "Operation completed successfully"
print_success "File created: /path/to/file.txt"
print_success "Service started on port 8080"

echo ""
echo "--- print_error ---"
print_error "Operation failed"
print_error "File not found: /path/to/missing.txt"
print_error "Connection timeout after 30 seconds"

echo ""
echo "--- print_info ---"
print_info "Initializing application..."
print_info "Connecting to database..."
print_info "Processing 100 records"

echo ""
echo "--- print_warning ---"
print_warning "Configuration file not found, using defaults"
print_warning "Low disk space: 10% remaining"
print_warning "Deprecated feature will be removed in next version"

echo ""
echo "--- print_debug (DEBUG=true) ---"
export DEBUG=true
print_debug "Debug mode enabled"
print_debug "Variable value: x=42"
print_debug "Function call: process_data()"
unset DEBUG

echo ""
echo "--- print_debug (DEBUG=false, should not output) ---"
print_debug "This should not appear"

echo ""
echo "--- print_section ---"
print_section "Configuration"
print_section "Database Setup"
print_section "Service Deployment"

echo ""
echo "--- print_step ---"
print_step "Step 1" "Validating configuration"
print_step "Step 2" "Starting services"
print_step "Step 3" "Running health checks"

echo ""
echo "--- print_command ---"
print_command "make build"
print_command "docker compose up -d"
print_command "go test ./..."

echo ""
echo "--- Combined Example ---"
print_header "Deployment Process"
print_section "Pre-deployment Checks"
print_info "Checking prerequisites..."
print_success "Docker is installed"
print_success "Go compiler is available"
print_warning "Using default configuration"
print_section "Building Application"
print_step "Step 1" "Compiling Go code"
print_command "go build -o app ./cmd/server"
print_success "Build completed"
print_section "Deployment"
print_info "Deploying to staging..."
print_success "Deployment successful"

echo ""
echo "=========================================="
echo "Test completed"
echo "=========================================="

