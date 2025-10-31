#!/bin/bash

# Check for large files in git working directory
# Detects both tracked and untracked files larger than 1MB
#
# Usage:
#   scripts/check-large-files.sh [--check-only|--check-staged|--no-prompt]
#
# Options:
#   --check-only     Only check and display, no prompt (for make git-large-files)
#   --check-staged   Check only files that will be added by git add (for release scripts)
#   --no-prompt      Exit with error code if large files found (for CI/CD)
#
# Exit codes:
#   0 - No large files found or user confirmed
#   1 - Large files found and user cancelled or --no-prompt set

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the unified logging library
# shellcheck source=./logger.sh
source "$SCRIPT_DIR/logger.sh"

# Parse flags
CHECK_ONLY=false
CHECK_STAGED=false
NO_PROMPT=false

for arg in "$@"; do
    case "$arg" in
        --check-only)
            CHECK_ONLY=true
            ;;
        --check-staged)
            CHECK_STAGED=true
            ;;
        --no-prompt)
            NO_PROMPT=true
            ;;
    esac
done

# Function: check_staged_large_files
# Description: Checks for large files that will be added by git add
#              Only checks files that are modified or untracked (not ignored)
# Returns:
#   0 if no large files found or user confirms, 1 if user cancels
# ============================================================================
check_staged_large_files() {
    print_info "Checking for large files (>1MB) that will be added to git..."
    
    local large_files
    
    # Get files that will be staged by git add
    # git status --porcelain shows files that are NOT ignored by .gitignore
    # However, we need to be more careful: git status --porcelain shows:
    #   M  = modified tracked file (will be staged)
    #   A  = added (already staged)
    #   ?? = untracked (NOT ignored, will be staged by git add)
    # 
    # To be safe, we use git ls-files --others --exclude-standard to verify
    # which untracked files are NOT ignored, then combine with modified files
    local staged_files
    
    # Get modified tracked files
    local modified_files
    modified_files=$(git status --porcelain 2>/dev/null | awk '
        {
            status = substr($0, 1, 2)
            file = substr($0, 4)
            
            # Only process modified tracked files (M, A, R, C)
            if (status ~ /^[MARC]/) {
                # For rename/copy (R or C), file path is in $2
                if (status ~ /^[RC]/) {
                    file = $2
                }
                print file
            }
        }' || true)
    
    # Get untracked files that are NOT ignored (will be staged by git add)
    local untracked_files
    untracked_files=$(git ls-files --others --exclude-standard 2>/dev/null || true)
    
    # Combine modified and untracked files
    if [ -n "$modified_files" ] && [ -n "$untracked_files" ]; then
        staged_files=$(printf '%s\n%s' "$modified_files" "$untracked_files" | sort -u || true)
    elif [ -n "$modified_files" ]; then
        staged_files="$modified_files"
    elif [ -n "$untracked_files" ]; then
        staged_files="$untracked_files"
    else
        staged_files=""
    fi
    
    if [ -z "$staged_files" ]; then
        print_success "No files to be staged"
        return 0
    fi
    
    # Check file sizes
    local temp_file_list
    temp_file_list=$(mktemp)
    echo "$staged_files" > "$temp_file_list"
    
    large_files=$(while IFS= read -r file_path || [ -n "$file_path" ]; do
        if [ -n "$file_path" ] && [ -f "$file_path" ]; then
            local size_bytes
            if command -v stat >/dev/null 2>&1 && stat -f%z /dev/null >/dev/null 2>&1 2>/dev/null; then
                # macOS: use stat -f%z
                size_bytes=$(stat -f%z "$file_path" 2>/dev/null || echo 0)
            else
                # Linux: use du -b
                size_bytes=$(du -b "$file_path" 2>/dev/null | awk '{print $1}' || echo 0)
            fi
            
            if [ "$size_bytes" -gt 1048576 ]; then
                printf "%d %s\n" "$size_bytes" "$file_path"
            fi
        fi
    done < "$temp_file_list" | sort -rn || true)
    
    rm -f "$temp_file_list"
    
    if [ -z "$large_files" ]; then
        print_success "No large files found (>1MB) to be staged"
        return 0
    fi
    
    # Display large files
    display_large_files "$large_files"
    
    # Handle --no-prompt flag
    if [ "$NO_PROMPT" = "true" ]; then
        print_error "Large files detected in staged files. Please remove or use git-lfs."
        return 1
    fi
    
    # Prompt for confirmation
    read -rp "Do you want to continue despite large files? (y/N): " confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        print_info "Continuing despite large files..."
        return 0
    else
        print_error "Operation cancelled due to large files"
        return 1
    fi
}

# Function: check_large_files
# Description: Checks for files larger than 1MB (both tracked and untracked)
#              Prompts user for confirmation if large files are found (unless --check-only)
# Returns:
#   0 if no large files found or user confirmed, 1 if user cancels
# ============================================================================
check_large_files() {
    print_info "Checking for large files (>1MB) in working directory..."
    
    # Find files larger than 1MB (1048576 bytes)
    # Check both tracked files (git ls-files) and untracked files
    # Use find to bypass .gitignore and catch all large files
    local large_files
    
    # Create temporary file list
    local temp_file_list
    temp_file_list=$(mktemp)
    
    # Get tracked files
    local tracked_files
    tracked_files=$(git ls-files -z 2>/dev/null || true)
    if [ -n "$tracked_files" ]; then
        printf '%s' "$tracked_files" | tr '\0' '\n' >> "$temp_file_list"
    fi
    
    # Find all untracked files (excluding .git directory)
    # This bypasses .gitignore to catch all large files
    find . -type f -not -path "./.git/*" 2>/dev/null | while IFS= read -r file; do
        # Skip if file is tracked
        if git ls-files --error-unmatch "$file" >/dev/null 2>&1; then
            continue
        fi
        # Add to list
        echo "$file"
    done >> "$temp_file_list"
    
    # Check file sizes
    large_files=$(while IFS= read -r file_path || [ -n "$file_path" ]; do
        if [ -n "$file_path" ] && [ -f "$file_path" ]; then
            local size_bytes
            if command -v stat >/dev/null 2>&1 && stat -f%z /dev/null >/dev/null 2>&1 2>/dev/null; then
                # macOS: use stat -f%z
                size_bytes=$(stat -f%z "$file_path" 2>/dev/null || echo 0)
            else
                # Linux: use du -b
                size_bytes=$(du -b "$file_path" 2>/dev/null | awk '{print $1}' || echo 0)
            fi
            
            if [ "$size_bytes" -gt 1048576 ]; then
                printf "%d %s\n" "$size_bytes" "$file_path"
            fi
        fi
    done < "$temp_file_list" | sort -rn || true)
    
    rm -f "$temp_file_list"
    
    if [ -z "$large_files" ]; then
        print_success "No large files found (>1MB)"
        return 0
    fi
    
    # Display large files
    display_large_files "$large_files"
    
    # If --check-only, just display and exit with 0
    if [ "$CHECK_ONLY" = "true" ]; then
        return 0
    fi
    
    # Handle --no-prompt flag
    if [ "$NO_PROMPT" = "true" ]; then
        print_error "Large files detected. Use --no-prompt flag only if large files are expected."
        return 1
    fi
    
    # Prompt for confirmation
    read -rp "Do you want to continue despite large files? (y/N): " confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        print_info "Continuing despite large files..."
        return 0
    else
        print_error "Operation cancelled due to large files"
        return 1
    fi
}

# Function: display_large_files
# Description: Helper function to display large files list
# Parameters:
#   $1 - Large files list (newline-separated)
# ============================================================================
display_large_files() {
    local large_files="$1"
    
    # Count large files
    local file_count
    file_count=$(echo "$large_files" | wc -l | tr -d ' ')
    
    print_warning "Found $file_count file(s) larger than 1MB:"
    echo ""
    
    # Display large files with human-readable sizes
    # Use a temporary file to avoid subshell issues with while loop
    local temp_file
    temp_file=$(mktemp)
    echo "$large_files" > "$temp_file"
    
    while IFS= read -r line || [ -n "$line" ]; do
        if [ -n "$line" ]; then
            local size_bytes
            local file_path
            size_bytes=$(echo "$line" | awk '{print $1}')
            file_path=$(echo "$line" | awk '{for(i=2;i<=NF;i++) printf "%s ", $i; print ""}' | sed 's/[[:space:]]*$//')
            
            # Convert to human-readable format
            local size_human
            if [ "$size_bytes" -ge 1048576 ]; then
                size_human=$(awk "BEGIN {printf \"%.2f MB\", $size_bytes/1048576}")
            else
                size_human=$(awk "BEGIN {printf \"%.2f KB\", $size_bytes/1024}")
            fi
            
            # Check if file is tracked or untracked
            local file_status
            if git ls-files --error-unmatch "$file_path" >/dev/null 2>&1; then
                file_status="(tracked)"
            else
                file_status="(untracked)"
            fi
            
            printf "  %8s  %s %s\n" "$size_human" "$file_path" "$file_status"
        fi
    done < "$temp_file"
    
    rm -f "$temp_file"
    
    echo ""
    print_warning "Large files in git repository can slow down clones and increase repository size."
    print_info "Consider using git-lfs or removing unnecessary large files before releasing."
    echo ""
}

# Main execution
main() {
    cd "$PROJECT_ROOT" || exit_with_error "Failed to change to project root"
    
    if [ "$CHECK_STAGED" = "true" ]; then
        check_staged_large_files
    else
        check_large_files
    fi
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi