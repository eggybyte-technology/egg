#!/bin/bash

# Unified logging script for Egg Framework
# Provides consistent logging format and colors across all scripts
#
# Usage:
#   source scripts/logger.sh
#   print_header "My Header"
#   print_success "Operation completed"
#   print_error "Something went wrong"
#   print_info "Information message"
#   print_warning "Warning message"

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

# Print a formatted header with borders
# Usage: print_header "My Header Title"
print_header() {
    local title="$1"
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
    echo -e "${BLUE}▶ ${BOLD}$title${RESET}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
}

# Print a success message
# Usage: print_success "Operation completed successfully"
print_success() {
    local message="$1"
    echo -e "${GREEN}[✓] SUCCESS:${RESET} $message"
}

# Print an error message
# Usage: print_error "Something went wrong"
print_error() {
    local message="$1"
    echo -e "${RED}[✗] ERROR:${RESET} $message" >&2
}

# Print an info message
# Usage: print_info "Information here"
print_info() {
    local message="$1"
    echo -e "${CYAN}[i] INFO:${RESET} $message"
}

# Print a warning message
# Usage: print_warning "This might be a problem"
print_warning() {
    local message="$1"
    echo -e "${YELLOW}[!] WARNING:${RESET} $message"
}

# Print a debug message (only if DEBUG is set)
# Usage: print_debug "Debug information"
print_debug() {
    local message="$1"
    if [ "${DEBUG:-false}" = "true" ]; then
        echo -e "${MAGENTA}[DEBUG]:${RESET} $message"
    fi
}

# Print a step indicator
# Usage: print_step "Step 1" "Doing something"
print_step() {
    local step="$1"
    local description="$2"
    echo -e "${BOLD}${step}${RESET}: ${description}"
}

# Print a section divider
# Usage: print_section "Section Name"
print_section() {
    local section="$1"
    echo ""
    echo -e "${BLUE}┌─ ${BOLD}$section${RESET} ${BLUE}────────────────────────────────────────────────────────────${RESET}"
}

# Print a command being executed (for transparency)
# Usage: print_command "make build"
print_command() {
    local command="$1"
    echo -e "${CYAN}[CMD]${RESET} $command"
}

# Exit with error message
# Usage: exit_with_error "Something went wrong"
exit_with_error() {
    local message="$1"
    print_error "$message"
    exit 1
}

# Exit with success message
# Usage: exit_with_success "All done"
exit_with_success() {
    local message="$1"
    print_success "$message"
    exit 0
}

# Check if command exists
# Usage: check_command "docker" "Docker is required"
check_command() {
    local command="$1"
    local error_msg="${2:-Command '$command' not found}"

    if ! command -v "$command" >/dev/null 2>&1; then
        exit_with_error "$error_msg"
    fi
}

# Check if file exists
# Usage: check_file "/path/to/file" "Config file not found"
check_file() {
    local file="$1"
    local error_msg="${2:-File '$file' not found}"

    if [ ! -f "$file" ]; then
        exit_with_error "$error_msg"
    fi
}

# Check if directory exists
# Usage: check_directory "/path/to/dir" "Directory not found"
check_directory() {
    local directory="$1"
    local error_msg="${2:-Directory '$directory' not found}"

    if [ ! -d "$directory" ]; then
        exit_with_error "$error_msg"
    fi
}

# Wait for a condition with progress indicator
# Usage: wait_for_condition "curl -f http://localhost:8080/health" 30 "Service health check"
wait_for_condition() {
    local command="$1"
    local max_attempts="${2:-30}"
    local description="${3:-Condition}"
    local attempt=1

    print_info "Waiting for $description..."
    while [ $attempt -le $max_attempts ]; do
        if eval "$command" >/dev/null 2>&1; then
            print_success "$description ready (attempt $attempt/$max_attempts)"
            return 0
        fi
        echo -n "."
        sleep 1
        attempt=$((attempt + 1))
    done
    echo ""
    exit_with_error "$description failed after $max_attempts attempts"
}

# Get the project root directory (useful for other scripts)
get_project_root() {
    echo "$(cd "$(dirname "${BASH_SOURCE[1]}")/.." && pwd)"
}

# Initialize logging for a script
# Usage: init_logging "script-name"
init_logging() {
    local script_name="$1"
    print_info "Starting $script_name..."
}

# Finalize logging for a script
# Usage: finalize_logging $exit_code "script-name"
finalize_logging() {
    local exit_code="$1"
    local script_name="$2"

    if [ "$exit_code" -eq 0 ]; then
        print_success "$script_name completed successfully"
    else
        print_error "$script_name completed with errors (exit code: $exit_code)"
    fi
    exit "$exit_code"
}




