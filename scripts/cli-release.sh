#!/usr/bin/env bash
#
# Release script for Egg CLI Tool
#
# This script automates the release process for the CLI tool independently from
# the framework modules. The CLI can be released with its own version number
# and may iterate faster than the framework.
#
# Usage:
#   ./scripts/cli-release.sh v1.0.0 --framework-version v0.3.0
#   FRAMEWORK_VERSION is REQUIRED - must specify the framework version
#
# Example:
#   ./scripts/cli-release.sh v1.0.0 --framework-version v0.3.0
#   ./scripts/cli-release.sh v1.1.0 --framework-version v0.3.0
#
# Environment Variables:
#   AUTO_COMMIT_CHANGES - Set to "true" to auto-commit without prompting (for CI/CD)
#
# Release Process:
#   1. Remove all replace directives (local development paths)
#   2. Update egg framework module dependencies to specified version
#   3. Generate framework_version.go with framework version
#   4. Run go mod tidy (with GOPROXY=direct to bypass proxy cache)
#   5. Commit changes
#   6. Create and push tag (cli/vX.Y.Z)
#
# Requirements:
#   - Git repository with proper permissions
#   - Go workspace configured
#   - Clean working directory (or use auto-commit)
#   - Framework modules must already be released at the specified version

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the unified logging library
# shellcheck source=./logger.sh
source "$SCRIPT_DIR/logger.sh"

# CLI module path
CLI_MODULE="cli"
CLI_PATH="$PROJECT_ROOT/$CLI_MODULE"

# GitHub repository base path
REPO_BASE="go.eggybyte.com/egg"

# Egg framework modules that CLI depends on
FRAMEWORK_MODULES=(
    core logx configx obsx httpx
    runtimex connectx clientx storex k8sx testingx
    servicex
)

# ============================================================================
# Function: validate_version
# Description: Validates that the version string follows semantic versioning
# Parameters:
#   $1 - Version string (e.g., v1.0.0)
# ============================================================================
validate_version() {
    local version="$1"
    
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        exit_with_error "Invalid version format: $version. Expected format: v1.x.y or v1.x.y-beta.1"
    fi
    
    print_success "Version format is valid: $version"
}

# ============================================================================
# Function: check_working_directory
# Description: Ensures the working directory is clean before release
#              Offers auto-commit for uncommitted changes (CI/CD friendly)
# Parameters:
#   $1 - Version string (for commit message)
# Environment Variables:
#   AUTO_COMMIT_CHANGES - Set to "true" to auto-commit without prompting (for CI/CD)
# ============================================================================
check_working_directory() {
    local version="$1"
    
    print_info "Checking working directory status..."
    
    if ! git diff-index --quiet HEAD -- 2>/dev/null; then
        print_warning "Working directory has uncommitted changes"
        echo ""
        
        # Show what's changed
        print_info "Uncommitted changes:"
        git status --short | head -20
        echo ""
        
        # Check if auto-commit is enabled (for CI/CD)
        local auto_commit="${AUTO_COMMIT_CHANGES:-false}"
        
        if [[ "$auto_commit" == "true" ]]; then
            print_info "AUTO_COMMIT_CHANGES=true, auto-committing without prompt..."
            confirm="y"
        else
            # Prompt for auto-commit
            read -rp "Auto-commit these changes before release? (y/N): " confirm
        fi
        
        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            print_info "Auto-committing changes..."
            
            git add -A
            local commit_msg="chore(cli): auto-commit before release ${version}"
            
            if git commit -m "$commit_msg"; then
                print_success "Changes committed: $commit_msg"
            else
                exit_with_error "Failed to commit changes"
            fi
        else
            exit_with_error "Working directory has uncommitted changes. Please commit or stash them first."
        fi
    fi
    
    print_success "Working directory is clean"
}

# ============================================================================
# Function: check_large_files
# Description: Checks for git-tracked files larger than 1MB
#              Prompts user for confirmation if large files are found
# Returns:
#   0 if no large files found or user confirms, 1 if user cancels
# ============================================================================
check_large_files() {
    print_info "Checking for large files (>1MB) in git-tracked files..."
    
    # Find files larger than 1MB (1048576 bytes)
    # Use du to get actual file sizes, then filter for files > 1MB
    local large_files
    large_files=$(git ls-files -z | xargs -0 du -b 2>/dev/null | awk '$1 > 1048576 {size=$1; $1=""; print size " " substr($0,2)}' | sort -rn || true)
    
    if [ -z "$large_files" ]; then
        print_success "No large files found (>1MB)"
        return 0
    fi
    
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
            
            printf "  %8s  %s\n" "$size_human" "$file_path"
        fi
    done < "$temp_file"
    
    rm -f "$temp_file"
    
    echo ""
    print_warning "Large files in git repository can slow down clones and increase repository size."
    print_info "Consider using git-lfs or removing unnecessary large files before releasing."
    echo ""
    
    read -rp "Do you want to continue with the release? (y/N): " confirm
    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        print_info "Release cancelled by user"
        return 1
    fi
    
    return 0
}

# ============================================================================
# Function: check_git_remote
# Description: Verifies that a Git remote is configured
# ============================================================================
check_git_remote() {
    print_info "Checking Git remote configuration..."
    
    if ! git remote get-url origin >/dev/null 2>&1; then
        exit_with_error "No Git remote 'origin' configured"
    fi
    
    local remote_url
    remote_url=$(git remote get-url origin)
    print_success "Remote configured: $remote_url"
}

# ============================================================================
# Function: check_existing_tag
# Description: Checks if tag already exists for the given version
# Parameters:
#   $1 - Version string
# Returns:
#   0 if no tag exists, 1 if tag exists
# ============================================================================
check_existing_tag() {
    local version="$1"
    local tag="cli/${version}"
    
    print_info "Checking for existing tag..."
    
    if git rev-parse "$tag" >/dev/null 2>&1; then
        print_warning "Found existing tag: $tag"
        return 1
    else
        print_success "No existing tag found"
        return 0
    fi
}

# ============================================================================
# Function: check_framework_version_exists
# Description: Verifies that framework modules exist at the specified version
# Parameters:
#   $1 - Framework version string
# ============================================================================
check_framework_version_exists() {
    local framework_version="$1"
    
    print_info "Verifying framework modules exist at version $framework_version..."
    
    local missing_modules=()
    for mod in "${FRAMEWORK_MODULES[@]}"; do
        local tag="${mod}/${framework_version}"
        if ! git rev-parse "$tag" >/dev/null 2>&1; then
            missing_modules+=("$tag")
        fi
    done
    
    if [ ${#missing_modules[@]} -gt 0 ]; then
        print_error "Missing framework module tags:"
        for tag in "${missing_modules[@]}"; do
            echo "  - $tag"
        done
        echo ""
        exit_with_error "Framework modules must be released before CLI release. Use './scripts/release.sh $framework_version' first."
    fi
    
    print_success "All framework modules verified at version $framework_version"
}

# ============================================================================
# Function: release_cli
# Description: Releases the CLI module
# Parameters:
#   $1 - CLI version string
#   $2 - Framework version string
# ============================================================================
release_cli() {
    local cli_version="$1"
    local framework_version="$2"
    
    print_info "Releasing CLI module: $cli_version (using framework $framework_version)"
    
    # Validate CLI module exists
    if [ ! -d "$CLI_PATH" ]; then
        exit_with_error "CLI directory not found: $CLI_PATH"
    fi
    
    if [ ! -f "$CLI_PATH/go.mod" ]; then
        exit_with_error "No go.mod found in $CLI_MODULE"
    fi
    
    # Step 1: Clean and update dependencies
    print_info "  → Updating dependencies for $CLI_MODULE..."
    (
        cd "$CLI_PATH"
        
        # Step 1a: Remove ALL replace directives (local development paths)
        print_info "    ↳ Removing all replace directives..."
        local removed_count=0
        for mod in "${FRAMEWORK_MODULES[@]}"; do
            if go mod edit -dropreplace="$REPO_BASE/$mod" 2>/dev/null; then
                removed_count=$((removed_count + 1))
            fi
        done
        
        if [ $removed_count -gt 0 ]; then
            print_success "    ↳ Removed $removed_count replace directives"
        else
            print_info "    ↳ No replace directives to remove"
        fi
        
        # Step 1b: Update framework module dependencies to specified version
        print_info "    ↳ Updating framework dependencies to $framework_version..."
        local updated_count=0
        for mod in "${FRAMEWORK_MODULES[@]}"; do
            # Check if CLI imports this module
            if grep -r "\"$REPO_BASE/$mod" . --include="*.go" --exclude-dir=vendor 2>/dev/null | head -1 > /dev/null; then
                print_info "    ↳ Setting $mod@$framework_version"
                if go mod edit -require="$REPO_BASE/$mod@$framework_version" 2>/dev/null; then
                    updated_count=$((updated_count + 1))
                fi
            fi
        done
        
        if [ $updated_count -gt 0 ]; then
            print_success "    ↳ Updated $updated_count framework dependencies"
        else
            print_info "    ↳ No framework dependencies to update"
        fi
        
        # Step 1c: Run go mod tidy with retry mechanism
        print_info "    ↳ Running go mod tidy with GOPROXY=direct..."
        
        local tidy_success=false
        local max_retries=3
        local retry_delay=5
        
        for attempt in $(seq 1 $max_retries); do
            if [ $attempt -gt 1 ]; then
                print_info "    ↳ Retry attempt $attempt/$max_retries (waiting ${retry_delay}s for GitHub to propagate tags)..."
                sleep $retry_delay
            fi
            
            if GOPROXY=direct go mod tidy 2>&1; then
                print_success "    ↳ go mod tidy completed successfully"
                tidy_success=true
                break
            else
                if [ $attempt -eq $max_retries ]; then
                    print_warning "    ↳ go mod tidy failed after $max_retries attempts"
                    print_warning "    ↳ This may be due to GitHub tag propagation delays"
                    print_info "    ↳ Users can run 'go get' later to fetch the correct dependencies"
                fi
            fi
        done
    ) || exit_with_error "Failed to update CLI dependencies"
    
    # Step 2: Update framework version in CLI code
    print_info "  → Updating framework version in CLI code..."
    local framework_version_file="$CLI_PATH/internal/generators/framework_version.go"
    
    # Create framework_version.go with framework version
    cat > "$framework_version_file" <<EOF
// Code generated by cli-release.sh. DO NOT EDIT.
package generators

// FrameworkVersion is the Egg framework version that this CLI release uses.
// This version is used when generating new projects.
const FrameworkVersion = "$framework_version"
EOF
    
    if [ -f "$framework_version_file" ]; then
        print_success "    ↳ Framework version updated to $framework_version"
    else
        print_warning "    ↳ Failed to create framework_version.go"
    fi
    
    # Step 3: Commit changes
    cd "$PROJECT_ROOT"
    if ! git diff --quiet -- "$CLI_MODULE/" 2>/dev/null; then
        print_info "  → Committing dependency updates for $CLI_MODULE..."
        git add "$CLI_MODULE/" 2>/dev/null || true
        
        local commit_msg="chore($CLI_MODULE): update dependencies to framework $framework_version for release $cli_version"
        
        # Also commit framework_version.go if it exists
        if [ -f "$CLI_PATH/internal/generators/framework_version.go" ]; then
            git add "$CLI_PATH/internal/generators/framework_version.go" 2>/dev/null || true
        fi
        
        if git commit -m "$commit_msg" 2>/dev/null; then
            print_info "    ↳ Changes committed"
        else
            print_warning "    ↳ No changes to commit or commit failed"
        fi
    else
        print_info "  → No dependency changes for $CLI_MODULE"
    fi
    
    # Step 3: Create and push tag
    local tag="cli/${cli_version}"
    print_info "  → Creating tag: $tag"
    
    if git tag -a "$tag" -m "Release CLI $cli_version (framework $framework_version)"; then
        print_info "  → Pushing tag: $tag"
        if git push origin "$tag"; then
            print_success "  ✓ Released CLI ($tag)"
            return 0
        else
            exit_with_error "Failed to push tag: $tag"
        fi
    else
        exit_with_error "Failed to create tag: $tag"
    fi
}

# ============================================================================
# Function: push_final_commits
# Description: Pushes any remaining commits to remote
# ============================================================================
push_final_commits() {
    print_section "Pushing Final Commits"
    
    # Check if there are any unpushed commits
    if git diff --quiet origin/$(git branch --show-current) HEAD 2>/dev/null; then
        print_info "No unpushed commits"
        return 0
    fi
    
    print_info "Pushing final commits to origin..."
    if git push origin HEAD; then
        print_success "Final commits pushed successfully"
    else
        print_warning "Failed to push some commits (may already be pushed)"
    fi
}

# ============================================================================
# Function: display_release_summary
# Description: Displays a summary of the release
# Parameters:
#   $1 - CLI version string
#   $2 - Framework version string
# ============================================================================
display_release_summary() {
    local cli_version="$1"
    local framework_version="$2"
    
    print_header "Release Summary"
    
    echo ""
    print_success "CLI release $cli_version completed successfully!"
    echo ""
    
    print_info "Release details:"
    echo "  CLI Version:      $cli_version"
    echo "  Framework Version: $framework_version"
    echo "  Tag:              cli/${cli_version}"
    echo ""
    
    print_info "Users can now install CLI with:"
    echo ""
    echo "  go install ${REPO_BASE}/${CLI_MODULE}/cmd/egg@${cli_version}"
    echo ""
    
    print_info "To verify tag:"
    echo "  git tag -l 'cli/${cli_version}'"
    echo ""
}

# ============================================================================
# Function: parse_arguments
# Description: Parses command line arguments
# Parameters:
#   $@ - Command line arguments
# Returns:
#   Sets global variables: CLI_VERSION, FRAMEWORK_VERSION
# ============================================================================
parse_arguments() {
    CLI_VERSION=""
    FRAMEWORK_VERSION=""
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --framework-version)
                FRAMEWORK_VERSION="$2"
                shift 2
                ;;
            *)
                if [[ -z "$CLI_VERSION" ]]; then
                    CLI_VERSION="$1"
                else
                    exit_with_error "Unexpected argument: $1"
                fi
                shift
                ;;
        esac
    done
    
    if [[ -z "$CLI_VERSION" ]]; then
        exit_with_error "CLI version argument is required"
    fi
    
    # Framework version is REQUIRED - no auto-detection
    if [[ -z "$FRAMEWORK_VERSION" ]]; then
        exit_with_error "Framework version is REQUIRED. Use --framework-version vX.Y.Z"
    fi
}

# ============================================================================
# Main Execution
# ============================================================================
main() {
    print_header "Egg CLI Release Script"
    
    # Parse arguments
    parse_arguments "$@"
    
    # Pre-flight checks
    print_section "Pre-flight Checks"
    validate_version "$CLI_VERSION"
    validate_version "$FRAMEWORK_VERSION"
    check_command "git" "Git is required but not installed"
    check_command "go" "Go is required but not installed"
    check_large_files || exit 0
    check_working_directory "$CLI_VERSION"
    check_git_remote
    
    # Check for existing tag
    if ! check_existing_tag "$CLI_VERSION"; then
        echo ""
        read -rp "Tag already exists. Do you want to continue and recreate it? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            print_info "Release cancelled by user"
            exit 0
        fi
        
        print_warning "Deleting existing tag..."
        local tag="cli/${CLI_VERSION}"
        git tag -d "$tag" 2>/dev/null || true
        git push --delete origin "$tag" 2>/dev/null || true
        print_success "Existing tag deleted"
    fi
    
    # Verify framework version exists
    check_framework_version_exists "$FRAMEWORK_VERSION"
    
    echo ""
    print_info "Preparing CLI release $CLI_VERSION"
    print_info "Using framework version: $FRAMEWORK_VERSION"
    echo ""
    
    # Release CLI
    release_cli "$CLI_VERSION" "$FRAMEWORK_VERSION"
    echo ""
    
    # Push any final commits
    push_final_commits
    echo ""
    
    # Display summary
    display_release_summary "$CLI_VERSION" "$FRAMEWORK_VERSION"
}

# Entry point
main "$@"

