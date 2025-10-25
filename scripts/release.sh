#!/usr/bin/env bash
#
# Release script for Egg Framework
#
# This script automates the release process for all Go modules in the framework.
# It updates module dependencies, creates Git tags, and pushes them to the remote repository.
#
# Usage:
#   ./scripts/release.sh v0.x.y
#
# Example:
#   ./scripts/release.sh v0.1.0
#
# The script will:
#   1. Validate the version format
#   2. Update internal module dependencies to the new version
#   3. Create tags for all modules
#   4. Push all tags to the remote repository
#
# Requirements:
#   - Git repository with proper permissions
#   - Go workspace configured
#   - Clean working directory (no uncommitted changes)

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the unified logging library
# shellcheck source=./logger.sh
source "$SCRIPT_DIR/logger.sh"

# Module list - must match the actual module directories
MODULES=(
  core
  logx
  connectx
  configx
  runtimex
  servicex
  storex
  obsx
  k8sx
  clientx
  httpx
  testingx
)

# GitHub repository base path
REPO_BASE="github.com/eggybyte-technology/egg"

# ============================================================================
# Function: validate_version
# Description: Validates that the version string follows semantic versioning
# Parameters:
#   $1 - Version string (e.g., v0.1.0)
# ============================================================================
validate_version() {
    local version="$1"
    
    if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        exit_with_error "Invalid version format: $version. Expected format: v0.x.y or v0.x.y-beta.1"
    fi
    
    print_success "Version format is valid: $version"
}

# ============================================================================
# Function: check_working_directory
# Description: Ensures the working directory is clean before release
# ============================================================================
check_working_directory() {
    print_info "Checking working directory status..."
    
    if ! git diff-index --quiet HEAD -- 2>/dev/null; then
        exit_with_error "Working directory has uncommitted changes. Please commit or stash them first."
    fi
    
    print_success "Working directory is clean"
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
# Function: check_existing_tags
# Description: Checks if tags already exist for the given version
# Parameters:
#   $1 - Version string
# Returns:
#   0 if no tags exist, 1 if tags exist
# ============================================================================
check_existing_tags() {
    local version="$1"
    local found_tags=0
    
    print_info "Checking for existing tags..."
    
    for mod in "${MODULES[@]}"; do
        local tag="${mod}/${version}"
        if git rev-parse "$tag" >/dev/null 2>&1; then
            print_warning "Found existing tag: $tag"
            found_tags=1
        fi
    done
    
    if [ $found_tags -eq 1 ]; then
        print_warning "Some tags already exist for version $version"
        return 1
    else
        print_success "No existing tags found"
        return 0
    fi
}

# ============================================================================
# Function: update_module_dependencies
# Description: Updates go.mod files to reference the new version
# Parameters:
#   $1 - Version string
# ============================================================================
update_module_dependencies() {
    local version="$1"
    
    print_section "Updating Module Dependencies"
    
    for mod in "${MODULES[@]}"; do
        local mod_path="$PROJECT_ROOT/$mod"
        
        if [ ! -d "$mod_path" ]; then
            print_warning "Module directory not found: $mod_path (skipping)"
            continue
        fi
        
        if [ ! -f "$mod_path/go.mod" ]; then
            print_warning "No go.mod found in $mod (skipping)"
            continue
        fi
        
        print_info "Updating dependencies in $mod..."
        
        (
            cd "$mod_path"
            
            # Update dependencies to other egg modules
            for dep in "${MODULES[@]}"; do
                if [[ "$dep" != "$mod" ]]; then
                    # Check if this module actually depends on the other module
                    if grep -q "$REPO_BASE/$dep" go.mod 2>/dev/null; then
                        print_info "  → Setting $dep@$version"
                        go mod edit -require="$REPO_BASE/$dep@$version" || true
                    fi
                fi
            done
            
            # Tidy up the go.mod file
            print_info "  → Running go mod tidy..."
            go mod tidy
        )
        
        print_success "Updated $mod"
    done
    
    print_success "All module dependencies updated"
}

# ============================================================================
# Function: create_module_tags
# Description: Creates Git tags for all modules
# Parameters:
#   $1 - Version string
# ============================================================================
create_module_tags() {
    local version="$1"
    
    print_section "Creating Module Tags"
    
    for mod in "${MODULES[@]}"; do
        local tag="${mod}/${version}"
        
        print_info "Creating tag: $tag"
        
        if git tag -a "$tag" -m "Release $mod $version" 2>/dev/null; then
            print_success "  ✓ $tag"
        else
            exit_with_error "Failed to create tag: $tag"
        fi
    done
    
    print_success "All module tags created"
}

# ============================================================================
# Function: commit_dependency_updates
# Description: Commits the go.mod changes after updating dependencies
# Parameters:
#   $1 - Version string
# ============================================================================
commit_dependency_updates() {
    local version="$1"
    
    print_section "Committing Dependency Updates"
    
    # Check if there are any changes to go.mod files
    if git diff --quiet -- '*/go.mod' '*/go.sum'; then
        print_info "No changes to commit (go.mod files unchanged)"
        return 0
    fi
    
    print_info "Detected changes in go.mod/go.sum files"
    
    # Stage all go.mod and go.sum changes
    print_info "Staging go.mod and go.sum changes..."
    git add '*/go.mod' '*/go.sum' 2>/dev/null || true
    
    # Commit the changes
    local commit_msg="chore: update module dependencies to ${version}"
    print_info "Creating commit: ${commit_msg}"
    
    if git commit -m "$commit_msg"; then
        print_success "Dependency updates committed"
    else
        exit_with_error "Failed to commit dependency updates"
    fi
    
    # Update tags to point to the new commit
    print_info "Updating tags to point to new commit..."
    for mod in "${MODULES[@]}"; do
        local tag="${mod}/${version}"
        # Delete the old tag
        git tag -d "$tag" >/dev/null 2>&1 || true
        # Create new tag at current HEAD
        git tag -a "$tag" -m "Release $mod ${version}"
    done
    
    print_success "Tags updated to new commit"
}

# ============================================================================
# Function: push_tags
# Description: Pushes all tags and commits to the remote repository
# ============================================================================
push_tags() {
    print_section "Pushing Changes to Remote"
    
    print_info "Pushing commits to origin..."
    if git push origin HEAD; then
        print_success "Commits pushed successfully"
    else
        exit_with_error "Failed to push commits to remote"
    fi
    
    print_info "Pushing all tags to origin..."
    if git push origin --tags --force; then
        print_success "All tags pushed successfully"
    else
        exit_with_error "Failed to push tags to remote"
    fi
}

# ============================================================================
# Function: display_release_summary
# Description: Displays a summary of the release
# Parameters:
#   $1 - Version string
# ============================================================================
display_release_summary() {
    local version="$1"
    
    print_header "Release Summary"
    
    echo ""
    print_success "Release $version completed successfully!"
    echo ""
    
    print_info "Tags created and pushed:"
    for mod in "${MODULES[@]}"; do
        echo "  ✓ ${mod}/${version}"
    done
    
    echo ""
    print_info "Users can now install modules with:"
    echo ""
    for mod in "${MODULES[@]}"; do
        echo "  go get ${REPO_BASE}/${mod}@${version}"
    done
    echo ""
    
    print_info "To verify tags:"
    echo "  git tag -l '*${version}'"
    echo ""
}

# ============================================================================
# Main Execution
# ============================================================================
main() {
    local version="${1:-}"
    
    print_header "Egg Framework Release Script"
    
    # Validate input
    if [[ -z "$version" ]]; then
        print_error "Version argument is required"
        echo ""
        echo "Usage: $0 v0.x.y"
        echo ""
        echo "Example:"
        echo "  $0 v0.1.0"
        echo "  $0 v0.2.0-beta.1"
        exit 1
    fi
    
    # Pre-flight checks
    print_section "Pre-flight Checks"
    validate_version "$version"
    check_command "git" "Git is required but not installed"
    check_command "go" "Go is required but not installed"
    check_working_directory
    check_git_remote
    
    # Check for existing tags and prompt user
    if ! check_existing_tags "$version"; then
        echo ""
        read -rp "Tags already exist. Do you want to continue and recreate them? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            print_info "Release cancelled by user"
            exit 0
        fi
        
        print_warning "Deleting existing tags..."
        for mod in "${MODULES[@]}"; do
            local tag="${mod}/${version}"
            git tag -d "$tag" 2>/dev/null || true
            git push --delete origin "$tag" 2>/dev/null || true
        done
        print_success "Existing tags deleted"
    fi
    
    echo ""
    print_info "Preparing release for ${version}"
    echo ""
    
    # Step 1: Create tags (MUST be done before updating dependencies)
    print_step "Step 1/4" "Creating Git tags"
    create_module_tags "$version"
    echo ""
    
    # Step 2: Update module dependencies (now tags exist locally for go mod tidy)
    print_step "Step 2/4" "Updating module dependencies"
    update_module_dependencies "$version"
    echo ""
    
    # Step 3: Commit dependency updates
    print_step "Step 3/4" "Committing dependency updates"
    commit_dependency_updates "$version"
    echo ""
    
    # Step 4: Push tags and commits
    print_step "Step 4/4" "Pushing changes to remote"
    push_tags
    echo ""
    
    # Display summary
    display_release_summary "$version"
}

# Entry point
main "$@"

