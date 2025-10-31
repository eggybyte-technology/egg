#!/usr/bin/env bash
#
# Release script for Egg Framework
#
# This script automates the release process for all Go modules in the framework.
# It follows a layer-by-layer release strategy to ensure dependency consistency.
#
# Usage:
#   ./scripts/release.sh v0.x.y              # Release a new version
#   ./scripts/release.sh --delete-all-tags   # Delete ALL tags (DANGEROUS!)
#
# Example:
#   ./scripts/release.sh v0.1.0
#   ./scripts/release.sh --delete-all-tags
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
    
    # Step 1: Clean and update dependencies
    print_info "  → Updating dependencies for $mod..."
    (
        cd "$mod_path"
        
        # Step 1a: Remove ALL replace directives (left from development/reinit)
        # This is critical for releases - go.mod must not contain local paths
        print_info "    ↳ Removing any existing replace directives..."
        existing_replaces=$(go mod edit -json | grep -o '"Replace":\[[^]]*\]' || echo "")
        if [[ -n "$existing_replaces" && "$existing_replaces" != "\"Replace\":[]" ]]; then
            # Get all replace paths from go.mod
            for dep_module in "${ALL_MODULES[@]}"; do
                go mod edit -dropreplace="$REPO_BASE/$dep_module" 2>/dev/null || true
            done
            print_success "    ↳ Cleaned replace directives"
        else
            print_info "    ↳ No replace directives to remove"
        fi
        
        # Step 1b: Update dependencies to already-released modules
        # Use array expansion that's safe for empty arrays (set -u compatible)
        if [ ${#RELEASED_MODULES[@]} -gt 0 ]; then
            for dep in "${RELEASED_MODULES[@]}"; do
                if [[ "$dep" != "$mod" ]]; then
                    # Check if this module imports from the released module (in .go files)
                    if grep -r "\"$REPO_BASE/$dep" . --include="*.go" --exclude-dir=vendor 2>/dev/null | head -1 > /dev/null; then
                        print_info "    ↳ Setting $dep@$version"
                        go mod edit -require="$REPO_BASE/$dep@$version" || true
                    fi
                fi
            done
        else
            print_info "    ↳ No dependencies to update (L0 module)"
        fi
        
        # Step 1c: Run go mod tidy with retry mechanism
        # Use GOPROXY=direct to bypass proxy cache and fetch directly from GitHub
        # Add retry logic to handle GitHub propagation delays
        print_info "    ↳ Running go mod tidy with GOPROXY=direct..."
        
        tidy_success=false
        max_retries=3
        retry_delay=5
        
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
    ) || return 1
    
    # Step 2: Update BuildTime for servicex module
    if [[ "$mod" == "servicex" ]]; then
        print_info "  → Updating BuildTime for servicex..."
        local build_time
        build_time=$(date +"%Y%m%d%H%M%S")
        local version_file="$mod_path/internal/version.go"
        
        if [ -f "$version_file" ]; then
            # Update BuildTime in version.go (works on both macOS and Linux)
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS uses BSD sed
                sed -i '' "s/var BuildTime = \".*\"/var BuildTime = \"$build_time\"/" "$version_file"
            else
                # Linux uses GNU sed
                sed -i "s/var BuildTime = \".*\"/var BuildTime = \"$build_time\"/" "$version_file"
            fi
            
            if [ $? -eq 0 ]; then
                print_success "    ↳ BuildTime updated to $build_time"
            else
                print_warning "    ↳ Failed to update BuildTime"
            fi
        else
            print_warning "    ↳ version.go not found: $version_file"
        fi
    fi
    
    # Step 3: Commit changes (go.mod, go.sum, and version.go updates)
    # Check for unstaged changes in the module directory
    cd "$PROJECT_ROOT"
    if ! git diff --quiet -- "$mod/" 2>/dev/null; then
        print_info "  → Committing dependency updates for $mod..."
        # Stage all changes in this module directory (go.mod, go.sum, and version.go if exists)
        git add "$mod/" 2>/dev/null || true
        
        local commit_msg="chore($mod): update dependencies to $version"
        if [[ "$mod" == "servicex" ]]; then
            commit_msg="chore($mod): update dependencies to $version and build time"
        fi
        
        if git commit -m "$commit_msg" 2>/dev/null; then
            print_info "    ↳ Changes committed"
        else
            print_warning "    ↳ No changes to commit or commit failed"
        fi
    else
        print_info "  → No dependency changes for $mod"
    fi
    
    # Step 4: Create and push tag
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
# Function: delete_all_tags
# Description: Deletes ALL tags for all modules (DANGEROUS OPERATION!)
#              Requires 3 confirmations to proceed
# ============================================================================
delete_all_tags() {
    print_header "Delete ALL Tags - DANGEROUS OPERATION"
    
    echo ""
    print_error "⚠️  WARNING: This will delete ALL version tags for ALL modules!"
    print_error "⚠️  This operation is IRREVERSIBLE!"
    echo ""
    
    # List all existing tags
    print_info "Finding existing tags..."
    local all_tags=()
    for mod in "${ALL_MODULES[@]}"; do
        # Find all tags for this module
        while IFS= read -r tag; do
            if [[ -n "$tag" ]]; then
                all_tags+=("$tag")
            fi
        done < <(git tag -l "${mod}/*" 2>/dev/null)
    done
    
    if [ ${#all_tags[@]} -eq 0 ]; then
        print_info "No tags found to delete"
        exit 0
    fi
    
    echo ""
    print_info "Found ${#all_tags[@]} tags:"
    for tag in "${all_tags[@]}"; do
        echo "  - $tag"
    done
    echo ""
    
    # First confirmation
    print_error "═══════════════════════════════════════════════════════════"
    print_error "  CONFIRMATION 1 OF 3"
    print_error "═══════════════════════════════════════════════════════════"
    echo ""
    read -rp "Are you ABSOLUTELY SURE you want to delete ALL ${#all_tags[@]} tags? (type 'yes' to continue): " confirm1
    
    if [[ "$confirm1" != "yes" ]]; then
        print_info "Operation cancelled"
        exit 0
    fi
    
    # Second confirmation
    echo ""
    print_error "═══════════════════════════════════════════════════════════"
    print_error "  CONFIRMATION 2 OF 3"
    print_error "═══════════════════════════════════════════════════════════"
    echo ""
    print_warning "This will delete tags both locally AND from remote (origin)"
    echo ""
    read -rp "Do you understand this will affect the remote repository? (type 'DELETE' to continue): " confirm2
    
    if [[ "$confirm2" != "DELETE" ]]; then
        print_info "Operation cancelled"
        exit 0
    fi
    
    # Third confirmation
    echo ""
    print_error "═══════════════════════════════════════════════════════════"
    print_error "  CONFIRMATION 3 OF 3 - FINAL WARNING"
    print_error "═══════════════════════════════════════════════════════════"
    echo ""
    print_error "Last chance to abort! This action CANNOT be undone!"
    echo ""
    read -rp "Type the exact text 'I UNDERSTAND THE CONSEQUENCES' to proceed: " confirm3
    
    if [[ "$confirm3" != "I UNDERSTAND THE CONSEQUENCES" ]]; then
        print_info "Operation cancelled"
        exit 0
    fi
    
    # Proceed with deletion
    echo ""
    print_section "Deleting Tags"
    
    local deleted_count=0
    local failed_count=0
    
    for tag in "${all_tags[@]}"; do
        print_info "Deleting tag: $tag"
        
        # Delete local tag
        if git tag -d "$tag" 2>/dev/null; then
            print_info "  ↳ Local tag deleted"
        else
            print_warning "  ↳ Failed to delete local tag (may not exist)"
        fi
        
        # Delete remote tag
        if git push --delete origin "$tag" 2>/dev/null; then
            print_success "  ✓ Remote tag deleted"
            deleted_count=$((deleted_count + 1))
        else
            print_error "  ✗ Failed to delete remote tag"
            failed_count=$((failed_count + 1))
        fi
    done
    
    echo ""
    print_header "Deletion Summary"
    echo ""
    print_info "Total tags processed: ${#all_tags[@]}"
    print_success "Successfully deleted: $deleted_count"
    
    if [ $failed_count -gt 0 ]; then
        print_error "Failed to delete: $failed_count"
    fi
    
    echo ""
    print_success "Tag deletion completed!"
}

# ============================================================================
# Main Execution
# ============================================================================
main() {
    local version="${1:-}"
    
    print_header "Egg Framework Release Script"
    
    # Check for special commands
    if [[ "$version" == "--delete-all-tags" ]]; then
        delete_all_tags
        exit 0
    fi
    
    # Validate input
    if [[ -z "$version" ]]; then
        print_error "Version argument is required"
        echo ""
        echo "Usage:"
        echo "  $0 v0.x.y              # Release a new version"
        echo "  $0 --delete-all-tags   # Delete ALL tags (DANGEROUS!)"
        echo ""
        echo "Example:"
        echo "  $0 v0.1.0"
        echo "  $0 v0.2.0-beta.1"
        echo "  $0 --delete-all-tags"
        exit 1
    fi
    
    # Pre-flight checks
    print_section "Pre-flight Checks"
    validate_version "$version"
    check_command "git" "Git is required but not installed"
    check_command "go" "Go is required but not installed"
    check_large_files || exit 0
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

