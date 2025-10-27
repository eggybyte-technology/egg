#!/usr/bin/env bash

# reinit-workspace.sh - Reinitialize the entire Go workspace
# This script will:
# 1. Delete all go.mod, go.sum, go.work, and go.work.sum files
# 2. Reinitialize each module with correct module path (go mod init)
# 3. Create new go.work and add all modules (go work init)
# 4. Run go mod tidy for each module (workspace auto-resolves local dependencies)
#
# Note: We don't run 'go work sync' because it tries to fetch from remote.
# Instead, we let 'go mod tidy' resolve dependencies using the workspace.

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored messages
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Get the repository root directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

log_info "Repository root: $REPO_ROOT"

# Module base path
MODULE_BASE="go.eggybyte.com/egg"

# Define modules in dependency order (layers)
# This ensures dependencies are initialized before dependents
declare -a MODULE_LAYERS=(
    "core"                                                      # L0: Zero dependencies
    "logx"                                                      # L1: Depends on core
    "configx obsx httpx"                                        # L2: Depends on L0/L1
    "runtimex connectx clientx storex k8sx testingx"           # L3: Depends on L0/L1/L2
    "servicex"                                                  # L4: Depends on all above
    "cli"                                                       # Tools
    "examples/minimal-connect-service examples/user-service examples/connect-tester"  # Examples
)

# Flatten for iteration
declare -a ALL_MODULES=(
    core logx configx obsx httpx
    runtimex connectx clientx storex k8sx testingx
    servicex cli
    examples/minimal-connect-service
    examples/user-service
    examples/connect-tester
)

# Step 1: Delete all go.mod, go.sum, go.work, and go.work.sum files
log_info "Step 1: Deleting all go.mod, go.sum, go.work, and go.work.sum files..."

# Delete go.work and go.work.sum in root
if [ -f go.work ]; then
    rm -f go.work
    log_success "Deleted go.work"
fi

if [ -f go.work.sum ]; then
    rm -f go.work.sum
    log_success "Deleted go.work.sum"
fi

# Delete go.mod and go.sum in each module
for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ]; then
        if [ -f "$module_dir/go.mod" ]; then
            rm -f "$module_dir/go.mod"
            log_success "Deleted $module/go.mod"
        fi
        if [ -f "$module_dir/go.sum" ]; then
            rm -f "$module_dir/go.sum"
            log_success "Deleted $module/go.sum"
        fi
    else
        log_warning "Module directory not found: $module_dir"
    fi
done

echo ""

# Step 2: Initialize each module with correct module path
log_info "Step 2: Initializing each module with go mod init..."

for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ]; then
        module_path="$MODULE_BASE/$module"
        log_info "Initializing $module_path..."
        cd "$module_dir"
        go mod init "$module_path" 2>&1 || {
            log_error "Failed to initialize $module"
            continue
        }
        log_success "Initialized $module"
    else
        log_warning "Skipping non-existent module: $module"
    fi
done

cd "$REPO_ROOT"
echo ""

# Step 3: Create go.work and add all modules
log_info "Step 3: Creating go.work and adding all modules..."

# Build the list of module paths for go work init
module_paths=()
for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ] && [ -f "$module_dir/go.mod" ]; then
        module_paths+=("./$module")
    else
        log_warning "Skipping $module (not found or no go.mod)"
    fi
done

# Initialize workspace with all modules
if [ ${#module_paths[@]} -gt 0 ]; then
    log_info "Initializing workspace with ${#module_paths[@]} modules..."
    go work init "${module_paths[@]}" 2>&1 || {
        log_error "Failed to initialize workspace"
        exit 1
    }
    log_success "go work init completed"
else
    log_error "No modules found to add to workspace"
    exit 1
fi

echo ""

# Step 4: Add dependencies and run go mod tidy layer by layer
log_info "Step 4: Processing modules by dependency layers..."
log_info "Note: Processing in dependency order to avoid resolution issues"

# Track processed modules for dependency injection
declare -a PROCESSED_MODULES=()

layer_num=0
for layer in "${MODULE_LAYERS[@]}"; do
    layer_num=$((layer_num + 1))
    log_info ""
    log_info "Processing Layer $layer_num: $layer"
    
    for module in $layer; do
        module_dir="$REPO_ROOT/$module"
        if [ ! -d "$module_dir" ] || [ ! -f "$module_dir/go.mod" ]; then
            log_warning "Skipping $module (not found or no go.mod)"
            continue
        fi
        
        log_info "  → Processing $module..."
        cd "$module_dir"
        
        # Add replace directives for ALL processed modules (including transitive deps)
        # This ensures that go mod tidy won't try to download fake versions
        if [ ${#PROCESSED_MODULES[@]} -gt 0 ]; then
            # Build lists of edits to apply in batch
            local -a replace_args=()
            local -a require_args=()
            
            for dep in "${PROCESSED_MODULES[@]}"; do
                # Always add replace directive for all processed modules
                replace_args+=("-replace=$MODULE_BASE/$dep=$REPO_ROOT/$dep")
                
                # Only add require directive if this module actually imports it
                if grep -r "\"$MODULE_BASE/$dep" . --include="*.go" --exclude-dir=vendor --exclude-dir=gen 2>/dev/null | head -1 > /dev/null; then
                    log_info "    ↳ Adding dependency: $dep"
                    require_args+=("-require=$MODULE_BASE/$dep@v0.0.0-00010101000000-000000000000")
                fi
            done
            
            # Apply all edits in a single go mod edit call to avoid conflicts
            if [ ${#replace_args[@]} -gt 0 ] || [ ${#require_args[@]} -gt 0 ]; then
                go mod edit "${replace_args[@]}" "${require_args[@]}" 2>/dev/null || true
            fi
        fi
        
        # Run go mod tidy with error tolerance
        log_info "    ↳ Running go mod tidy..."
        if go mod tidy 2>&1; then
            log_success "    ✓ Completed $module"
            PROCESSED_MODULES+=("$module")
        else
            log_warning "    ⚠ Tidy failed for $module (continuing anyway)"
            # Mark as processed so dependents can reference it
            PROCESSED_MODULES+=("$module")
        fi
    done
done

cd "$REPO_ROOT"
echo ""

# Final verification
log_info "Verifying workspace status..."
go work edit -json > /dev/null 2>&1 && log_success "Workspace is valid" || log_error "Workspace validation failed"

echo ""
log_success "=========================================="
log_success "Workspace reinitialization completed!"
log_success "=========================================="
log_info "Summary:"
log_info "  - All modules have been reinitialized"
log_info "  - Workspace has been recreated"
log_info "  - Local dependencies are resolved via workspace"
echo ""
log_info "Next steps:"
log_info "  1. Review any errors above"
log_info "  2. If there were errors, they may be due to missing external dependencies"
log_info "  3. Run 'go test ./...' to verify everything works"
log_info "  4. Run 'go work sync' only if you need to sync with remote versions"

