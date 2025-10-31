#!/bin/bash
#
# Test Helper Functions
#
# Provides common helper functions used across all test scripts.
# This file is sourced by other test scripts.
# 
# Note: All logging should use functions from scripts/logger.sh for consistency.

# Print CLI section header (uses logger.sh functions)
print_cli_section() {
    print_section "$1"
}

# Print command execution context
print_command_context() {
    local context="$1"
    printf "\n"
    print_section "$context"
}

# Print command output header (simplified, no borders)
print_output_header() {
    print_info "Command output:"
}

# Print command output footer (removed, not needed)
print_output_footer() {
    # No footer needed for clean output
    return 0
}

# Print command execution summary
# Uses logger.sh functions for status messages to ensure consistent formatting
print_command_summary() {
    local exit_code="$1"
    local elapsed_time="${2:-}"
    
    if [ $exit_code -eq 0 ]; then
        if [ -n "$elapsed_time" ]; then
            print_success "Completed successfully (Duration: $elapsed_time)"
        else
            print_success "Completed successfully"
        fi
    else
        if [ -n "$elapsed_time" ]; then
            print_error "Failed with exit code $exit_code (Duration: $elapsed_time)"
        else
            print_error "Failed with exit code $exit_code"
        fi
    fi
}

# Print troubleshooting suggestions
print_troubleshooting() {
    local error_type="$1"
    local command="$2"
    
    printf "\n"
    print_warning "Troubleshooting Guide"
    
    case "$error_type" in
        "command_not_found")
            print_info "Issue: Command not found"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Check if the command is installed and in PATH\n"
            printf "  2. Verify the command spelling\n"
            printf "  3. Check if required dependencies are installed\n"
            ;;
        "permission_denied")
            print_info "Issue: Permission denied"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Check file permissions: ls -l %s\n" "$(echo "$command" | awk '{print $NF}')"
            printf "  2. Add execute permission: chmod +x <file>\n"
            printf "  3. Check directory permissions\n"
            ;;
        "network_error")
            print_info "Issue: Network connection error"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Check network connectivity: ping <host>\n"
            printf "  2. Verify firewall settings\n"
            printf "  3. Check if proxy settings are correct\n"
            printf "  4. Verify DNS resolution: nslookup <host>\n"
            ;;
        "timeout")
            print_info "Issue: Operation timed out"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Check if the service is running\n"
            printf "  2. Verify service health: curl <health-endpoint>\n"
            printf "  3. Check service logs: docker compose logs <service>\n"
            printf "  4. Increase timeout if needed\n"
            ;;
        "build_error")
            print_info "Issue: Build failed"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Check Docker is running: docker ps\n"
            printf "  2. Verify Docker images exist: docker images\n"
            printf "  3. Check build logs for specific errors\n"
            printf "  4. Ensure all dependencies are available\n"
            ;;
        "rpc_error")
            print_info "Issue: RPC call failed"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Verify service is healthy: curl <health-endpoint>\n"
            printf "  2. Check service logs: docker compose logs <service>\n"
            printf "  3. Verify RPC path is correct\n"
            printf "  4. Check request format matches proto definition\n"
            printf "  5. Ensure service is listening on correct port\n"
            ;;
        *)
            print_info "Issue: $error_type"
            print_info "Command: $command"
            print_info "Possible Solutions:"
            printf "  1. Review error message above for details\n"
            printf "  2. Check command syntax and parameters\n"
            printf "  3. Verify prerequisites are met\n"
            ;;
    esac
}

# Run egg command with detailed output and enhanced formatting
run_egg_command() {
    local description="$1"
    shift
    local cmd="$@"
    local start_time=$(date +%s)
    
    print_command_context "$description"
    print_command "$EGG_CLI $cmd"
    print_info "Working Directory: $(pwd)"
    print_output_header
    
    # Run command and capture output, preserving exit code
    local output_file=$(mktemp)
    set +e  # Temporarily disable exit on error
    
    # Run command and format output (no borders, just plain output)
    $EGG_CLI $cmd 2>&1 | tee "$output_file"
    
    local exit_code=${PIPESTATUS[0]}
    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))
    local elapsed_time=$(printf "%ds" $elapsed)
    
    set -e  # Re-enable exit on error
    
    # Print summary
    print_command_summary $exit_code "$elapsed_time"
    print_output_footer
    
    if [ $exit_code -eq 0 ]; then
        rm -f "$output_file"
        return 0
    else
        # Analyze error and provide troubleshooting
        local error_output=$(cat "$output_file" 2>/dev/null || echo "")
        local error_type="generic"
        
        # Detect error type
        if echo "$error_output" | grep -qi "command not found\|No such file\|not found"; then
            error_type="command_not_found"
        elif echo "$error_output" | grep -qi "permission denied\|Permission denied"; then
            error_type="permission_denied"
        elif echo "$error_output" | grep -qi "timeout\|timed out"; then
            error_type="timeout"
        elif echo "$error_output" | grep -qi "build\|docker"; then
            error_type="build_error"
        fi
        
        # Special handling for doctor command failure
        if [[ "$description" == *"doctor"* ]]; then
            printf "\n"
            print_error "Environment check failed!"
            print_warning "Please install missing components by running:"
            printf "  %s doctor --install\n" "$EGG_CLI"
            printf "\n"
            exit_with_error "Test suite terminated due to environment issues"
        fi
        
        # Print troubleshooting for other errors
        print_troubleshooting "$error_type" "$EGG_CLI $cmd"
        
        rm -f "$output_file"
        return $exit_code
    fi
}

# Check if command succeeded
check_success() {
    if [ $? -eq 0 ]; then
        print_success "$1"
    else
        exit_with_error "$1 failed"
    fi
}

# Check if file exists (wrapper for consistency with test script)
# Enhanced with better error messages
check_file() {
    local file="$1"
    local hint="${2:-}"
    
    if [ -f "$file" ]; then
        print_success "File exists: $file"
        return 0
    else
        print_error "File missing: $file"
        if [ -n "$hint" ]; then
            print_info "Hint: $hint"
        fi
        exit_with_error "File validation failed"
    fi
}

# Check if directory exists (wrapper for consistency with test script)
# Enhanced with better error messages
check_dir() {
    local dir="$1"
    local hint="${2:-}"
    
    if [ -d "$dir" ]; then
        print_success "Directory exists: $dir"
        return 0
    else
        print_error "Directory missing: $dir"
        if [ -n "$hint" ]; then
            print_info "Hint: $hint"
        fi
        exit_with_error "Directory validation failed"
    fi
}

# Check if file contains expected content
# Enhanced with better error messages and context
check_file_content() {
    local file="$1"
    local expected="$2"
    local description="$3"
    local hint="${4:-}"
    
    if [ ! -f "$file" ]; then
        print_error "File not found: $file"
        if [ -n "$hint" ]; then
            print_info "Hint: $hint"
        fi
        exit_with_error "File validation failed"
    fi
    
    if grep -q "$expected" "$file"; then
        print_success "$description: found in $file"
        return 0
    else
        print_error "$description: not found in $file"
        print_info "Expected: $expected"
        
        # Show context around the expected content
        print_info "File content preview:"
        head -20 "$file" | sed "s/^/  /"
        
        if [ -n "$hint" ]; then
            print_info "Hint: $hint"
        fi
        
        exit_with_error "Content validation failed for $file"
    fi
}

# Wait for endpoint with retry (non-fatal version for tests)
# Enhanced with detailed progress and diagnostics
wait_for_endpoint() {
    local url="$1"
    local max_attempts="${2:-30}"
    local description="${3:-Endpoint}"
    local attempt=1
    local last_error=""
    
    print_info "Waiting for $description..."
    print_info "URL: $url"
    print_info "Max attempts: $max_attempts (${max_attempts}s)"
    
    while [ $attempt -le $max_attempts ]; do
        # Capture both stdout and stderr
        local response
        response=$(curl -sf -w "\n%{http_code}" "$url" 2>&1)
        local curl_exit=$?
        local http_code=$(echo "$response" | tail -1)
        local body=$(echo "$response" | sed '$d')
        
        if [ $curl_exit -eq 0 ] && [ "$http_code" = "200" ]; then
            printf "\n"
            print_success "$description ready (attempt $attempt/$max_attempts)"
            print_info "HTTP Status: $http_code"
            return 0
        else
            # Store error for diagnostics
            if [ $curl_exit -ne 0 ]; then
                last_error="curl_error_$curl_exit"
            elif [ "$http_code" != "200" ]; then
                last_error="http_$http_code"
            fi
            
            # Show progress indicator
            if [ $((attempt % 5)) -eq 0 ]; then
                printf " [%d/%d]" $attempt $max_attempts
            else
                printf "."
            fi
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    printf "\n"
    print_warning "$description not ready after $max_attempts attempts"
    
    # Provide diagnostics
    print_info "Last error: $last_error"
    print_info "Diagnostics:"
    print_info "Checking connectivity..."
    
    # Check if URL is reachable
    if curl -sf --connect-timeout 2 "$url" >/dev/null 2>&1; then
        print_info "URL is reachable"
    else
        print_warning "URL is not reachable"
        print_troubleshooting "network_error" "curl $url"
    fi
    
    return 1
}

# Wait for endpoint with pattern match (non-fatal version for tests)
# Enhanced with detailed progress and diagnostics
wait_for_endpoint_pattern() {
    local url="$1"
    local pattern="$2"
    local max_attempts="${3:-30}"
    local description="${4:-Endpoint}"
    local attempt=1
    local last_response=""
    
    print_info "Waiting for $description..."
    print_info "URL: $url"
    print_info "Expected pattern: $pattern"
    print_info "Max attempts: $max_attempts (${max_attempts}s)"
    
    while [ $attempt -le $max_attempts ]; do
        local response
        response=$(curl -sf -w "\n%{http_code}" "$url" 2>&1)
        local curl_exit=$?
        local http_code=$(echo "$response" | tail -1)
        local body=$(echo "$response" | sed '$d')
        last_response="$body"
        
        if [ $curl_exit -eq 0 ] && echo "$body" | grep -q "$pattern"; then
            printf "\n"
            print_success "$description ready (attempt $attempt/$max_attempts)"
            print_info "HTTP Status: $http_code"
            print_info "Pattern found in response"
            return 0
        else
            # Show progress indicator
            if [ $((attempt % 5)) -eq 0 ]; then
                printf " [%d/%d]" $attempt $max_attempts
            else
                printf "."
            fi
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    printf "\n"
    print_warning "$description not ready after $max_attempts attempts"
    
    # Provide diagnostics
    print_info "Last HTTP Status: $http_code"
    if [ -n "$last_response" ]; then
        print_info "Last response preview:"
        echo "$last_response" | head -5 | sed "s/^/  /"
    fi
    
    return 1
}

# Call Connect RPC endpoint using curl
# Parameters:
#   - service_url: Base URL of the service (e.g., http://localhost:8080)
#   - rpc_path: Connect RPC path (e.g., /eggybyte_test.test_project.ping.v1.PingService/Ping)
#   - method: HTTP method (GET or POST, default: POST)
#   - data: JSON data for POST requests (optional)
#   - expected_pattern: Pattern to match in response (optional)
call_connect_rpc() {
    local service_url="$1"
    local rpc_path="$2"
    local method="${3:-POST}"
    local data="${4:-}"
    local expected_pattern="${5:-}"
    
    local url="${service_url}${rpc_path}"
    local start_time=$(date +%s)
    
    print_command_context "Connect RPC Call"
    print_info "Method: $method"
    print_info "Endpoint: $url"
    if [ -n "$data" ]; then
        print_info "Request Data:"
        # Try to format as JSON, fallback to raw output
        echo "$data" | jq '.' 2>/dev/null | sed "s/^/  /" || echo "$data" | sed "s/^/  /"
    fi
    if [ -n "$expected_pattern" ]; then
        print_info "Expected Pattern: $expected_pattern"
    fi
    print_output_header
    
    # Build curl command
    local curl_opts="-sf -w '\n%{http_code}' -X $method"
    curl_opts="$curl_opts -H 'Content-Type: application/json'"
    
    if [ -n "$data" ]; then
        curl_opts="$curl_opts -d '$data'"
    fi
    
    # Execute curl command
    local response
    response=$(eval "curl $curl_opts '$url'" 2>&1)
    local curl_exit=$?
    local end_time=$(date +%s)
    local elapsed=$((end_time - start_time))
    
    # Parse response
    local http_code=$(echo "$response" | tail -1)
    local response_body=$(echo "$response" | sed '$d')
    
    print_info "Response:"
    print_info "  HTTP Status: $http_code"
    print_info "  Duration: ${elapsed}s"
    
    if [ $curl_exit -eq 0 ] && [ "$http_code" = "200" ]; then
        print_info "Response Body:"
        # Try to format JSON, fallback to raw output
        echo "$response_body" | jq '.' 2>/dev/null | sed "s/^/  /" || echo "$response_body" | head -20 | sed "s/^/  /"
        
        print_command_summary 0 "${elapsed}s"
        print_output_footer
        
        if [ -n "$expected_pattern" ]; then
            if echo "$response_body" | grep -q "$expected_pattern"; then
                print_success "Response matches expected pattern: $expected_pattern"
                return 0
            else
                print_error "Response does not match expected pattern: $expected_pattern"
                print_info "Expected pattern: $expected_pattern"
                print_info "Response preview:"
                echo "$response_body" | head -10 | sed "s/^/  /"
                print_troubleshooting "rpc_error" "curl -X $method '$url'"
                return 1
            fi
        else
            return 0
        fi
    else
        print_error "Error Details:"
        echo "$response_body" | head -20 | sed "s/^/  /"
        
        print_command_summary 1 "${elapsed}s"
        print_output_footer
        
        # Provide diagnostics
        print_info "Failed RPC call details:"
        printf "  Method: %s\n" "$method"
        printf "  URL: %s\n" "$url"
        printf "  HTTP Status: %s\n" "$http_code"
        if [ -n "$data" ]; then
            printf "  Request Data: %s\n" "$data"
        fi
        
        print_troubleshooting "rpc_error" "curl -X $method '$url'"
        return 1
    fi
}
