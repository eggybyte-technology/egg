#!/usr/bin/env bash
#
# Release script for Egg Framework
#
# This script automates the release process for all Go modules in the framework.
# It follows a layer-by-layer release strategy to ensure dependency consistency.
#
# Usage:
#   ./scripts/release.sh v0.x.y
#
# Example:
#   ./scripts/release.sh v0.1.0
#
# Environment Variables:
#   AUTO_COMMIT_CHANGES - Set to "true" to auto-commit without prompting (for CI/CD)
#
# Release Strategy:
#   The script releases modules in dependency order (bottom-up):
#   Layer 0 (L0): core
#   Layer 1 (L1): logx
#   Layer 2 (L2): configx, obsx, httpx
#   Layer 3 (L3): runtimex, connectx, clientx, storex, k8sx, testingx
#   Layer 4 (L4): servicex
#
# For each module:
#   1. Update dependencies to already-released modules
#   2. Run go mod tidy (with GOPROXY=direct to bypass proxy cache)
#   3. Commit changes
#   4. Create and push tag
#
# Requirements:
#   - Git repository with proper permissions
#   - Go workspace configured
#   - Clean working directory (or use auto-commit)

set -euo pipefail

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Source the unified logging library
# shellcheck source=./logger.sh
source "$SCRIPT_DIR/logger.sh"

# Module release layers - ordered by dependency hierarchy (bottom-up)
# Each layer's modules can be released in parallel, but layers must be sequential
# Format: Each element is a space-separated list of modules at that layer
RELEASE_LAYERS=(
  "core"                                          # L0: Zero dependencies
  "logx"                                          # L1: Depends on core
  "configx obsx httpx"                            # L2: Depends on L0/L1
  "runtimex connectx clientx storex k8sx testingx" # L3+Aux: Depends on L0/L1/L2
  "servicex"                                      # L4: Depends on all above
)

# All modules (for validation and summary)
ALL_MODULES=(
  core logx configx obsx httpx
  runtimex connectx clientx storex k8sx testingx
  servicex
)

# GitHub repository base path
REPO_BASE="go.eggybyte.com/egg"

# Track released modules (used during release process)
RELEASED_MODULES=()

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
            local commit_msg="chore: auto-commit before release ${version}"
            
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
    
    for mod in "${ALL_MODULES[@]}"; do
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
# Function: release_single_module
# Description: Releases a single module (update deps, commit, tag, push)
# Parameters:
#   $1 - Module name
#   $2 - Version string
# ============================================================================
release_single_module() {
    local mod="$1"
    local version="$2"
    local mod_path="$PROJECT_ROOT/$mod"
    
    print_info "Releasing module: $mod"
    
    # Validate module exists
    if [ ! -d "$mod_path" ]; then
        print_warning "Module directory not found: $mod_path (skipping)"
        return 1
    fi
    
    if [ ! -f "$mod_path/go.mod" ]; then
        print_warning "No go.mod found in $mod (skipping)"
        return 1
    fi
    
    # Step 1: Update dependencies to already-released modules
    print_info "  → Updating dependencies for $mod..."
    (
        cd "$mod_path"
        
        # Update dependencies only to modules that have been released
        # Use array expansion that's safe for empty arrays (set -u compatible)
        if [ ${#RELEASED_MODULES[@]} -gt 0 ]; then
            for dep in "${RELEASED_MODULES[@]}"; do
                if [[ "$dep" != "$mod" ]]; then
                    # Check if this module actually depends on the released module
                    if grep -q "$REPO_BASE/$dep" go.mod 2>/dev/null; then
                        print_info "    ↳ Setting $dep@$version"
                        go mod edit -require="$REPO_BASE/$dep@$version" || true
                    fi
                fi
            done
        else
            print_info "    ↳ No dependencies to update (first module)"
        fi
        
        # Tidy up the go.mod file (will resolve from already-pushed tags)
        # Use GOPROXY=direct to bypass proxy cache and fetch directly from Git
        # This avoids waiting for proxy synchronization during multi-module releases
        print_info "    ↳ Running go mod tidy (with GOPROXY=direct)..."
        if ! GOPROXY=direct GOSUMDB=off go mod tidy; then
            exit 1
        fi
    ) || return 1
    
    # Step 2: Commit changes if any
    if ! git diff --quiet -- "$mod/go.mod" "$mod/go.sum" 2>/dev/null; then
        print_info "  → Committing dependency updates for $mod..."
        git add "$mod/go.mod" "$mod/go.sum" 2>/dev/null || true
        git commit -m "chore($mod): update dependencies to $version" || true
    else
        print_info "  → No dependency changes for $mod"
    fi
    
    # Step 3: Create and push tag
    local tag="${mod}/${version}"
    print_info "  → Creating tag: $tag"
    
    if git tag -a "$tag" -m "Release $mod $version"; then
        print_info "  → Pushing tag: $tag"
        if git push origin "$tag"; then
            print_success "  ✓ Released $mod ($tag)"
            # Add to released modules list
            RELEASED_MODULES+=("$mod")
            return 0
        else
            print_error "  ✗ Failed to push tag: $tag"
            return 1
        fi
    else
        print_error "  ✗ Failed to create tag: $tag"
        return 1
    fi
}

# ============================================================================
# Function: release_layer
# Description: Releases all modules in a specific layer
# Parameters:
#   $1 - Layer number (for display)
#   $2 - Space-separated list of modules in this layer
#   $3 - Version string
# ============================================================================
release_layer() {
    local layer_num="$1"
    local layer_modules="$2"
    local version="$3"
    
    print_section "Layer $layer_num: $layer_modules"
    
    # Release each module in this layer
    for mod in $layer_modules; do
        if ! release_single_module "$mod" "$version"; then
            exit_with_error "Failed to release module: $mod"
        fi
    done
    
    print_success "Layer $layer_num completed"
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
#   $1 - Version string
# ============================================================================
display_release_summary() {
    local version="$1"
    
    print_header "Release Summary"
    
    echo ""
    print_success "Release $version completed successfully!"
    echo ""
    
    print_info "Modules released in dependency order:"
    local layer_num=0
    for layer in "${RELEASE_LAYERS[@]}"; do
        layer_num=$((layer_num + 1))
        echo ""
        echo "  Layer $layer_num:"
        for mod in $layer; do
            echo "    ✓ ${mod}/${version}"
        done
    done
    
    echo ""
    print_info "Users can now install modules with:"
    echo ""
    for mod in "${ALL_MODULES[@]}"; do
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
    check_working_directory "$version"
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
        for mod in "${ALL_MODULES[@]}"; do
            local tag="${mod}/${version}"
            git tag -d "$tag" 2>/dev/null || true
            git push --delete origin "$tag" 2>/dev/null || true
        done
        print_success "Existing tags deleted"
    fi
    
    echo ""
    print_info "Preparing release for ${version}"
    print_info "Release strategy: Layer-by-layer (bottom-up dependency order)"
    echo ""
    
    # Release modules layer by layer
    print_header "Releasing Modules by Dependency Layers"
    echo ""
    
    local layer_num=0
    for layer in "${RELEASE_LAYERS[@]}"; do
        layer_num=$((layer_num + 1))
        release_layer "$layer_num" "$layer" "$version"
        echo ""
    done
    
    # Push any final commits
    push_final_commits
    echo ""
    
    # Display summary
    display_release_summary "$version"
}

# Entry point
main "$@"

