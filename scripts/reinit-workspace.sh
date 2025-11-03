#!/usr/bin/env bash

# reinit-workspace.sh - Reinitialize the entire Go workspace
# This script will:
# 1. Delete all go.mod, go.sum, go.work, and go.work.sum files
# 2. Reinitialize each module with correct module path (go mod init)
# 3. Create new go.work and add all modules (go work init)
# 4. Add replace directives for all internal module dependencies (kept in go.mod)
# 5. Run go mod tidy for each module with replace directives in place
#
# Important: Replace directives are KEPT in go.mod after this script runs.
# - They use RELATIVE paths (e.g., ../core, ../../core) for portability
# - They are essential for local development (prevent fetching non-existent remote versions)
# - They should be committed to the repository
# - They will be automatically removed during release by release.sh
#
# Note: We don't run 'go work sync' because it tries to fetch from remote.
# Instead, we let 'go mod tidy' resolve dependencies using the workspace + replace.

set -euo pipefail

# Source logger script for unified output
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/logger.sh"

# Get the repository root directory
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

print_info "Repository root: $REPO_ROOT"

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
print_section "Step 1: Deleting all go.mod, go.sum, go.work, and go.work.sum files"

# Delete go.work and go.work.sum in root
if [ -f go.work ]; then
    rm -f go.work
    print_success "Deleted go.work"
fi

if [ -f go.work.sum ]; then
    rm -f go.work.sum
    print_success "Deleted go.work.sum"
fi

# Delete go.mod and go.sum in each module
for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ]; then
        if [ -f "$module_dir/go.mod" ]; then
            rm -f "$module_dir/go.mod"
            print_success "Deleted $module/go.mod"
        fi
        if [ -f "$module_dir/go.sum" ]; then
            rm -f "$module_dir/go.sum"
            print_success "Deleted $module/go.sum"
        fi
    else
        print_warning "Module directory not found: $module_dir"
    fi
done

echo ""

# Step 2: Initialize each module with correct module path
print_section "Step 2: Initializing each module with go mod init"

for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ]; then
        module_path="$MODULE_BASE/$module"
        print_info "Initializing $module_path..."
        cd "$module_dir"
        go mod init "$module_path" 2>&1 || {
            print_error "Failed to initialize $module"
            continue
        }
        print_success "Initialized $module"
    else
        print_warning "Skipping non-existent module: $module"
    fi
done

cd "$REPO_ROOT"
echo ""

# Step 3: Create go.work and add all modules
print_section "Step 3: Creating go.work and adding all modules"

# Build the list of module paths for go work init
module_paths=()
for module in "${ALL_MODULES[@]}"; do
    module_dir="$REPO_ROOT/$module"
    if [ -d "$module_dir" ] && [ -f "$module_dir/go.mod" ]; then
        module_paths+=("./$module")
    else
        print_warning "Skipping $module (not found or no go.mod)"
    fi
done

# Initialize workspace with all modules
if [ ${#module_paths[@]} -gt 0 ]; then
    print_info "Initializing workspace with ${#module_paths[@]} modules..."
    go work init "${module_paths[@]}" 2>&1 || {
        print_error "Failed to initialize workspace"
        exit 1
    }
    print_success "go work init completed"
else
    print_error "No modules found to add to workspace"
    exit 1
fi

echo ""

# Step 4: Add dependencies and run go mod tidy layer by layer
print_section "Step 4: Processing modules by dependency layers"
print_info "Processing in dependency order to avoid resolution issues"

# Track processed modules for dependency injection
declare -a PROCESSED_MODULES=()

layer_num=0
for layer in "${MODULE_LAYERS[@]}"; do
    layer_num=$((layer_num + 1))
    print_info ""
    print_section "Processing Layer $layer_num: $layer"
    
    for module in $layer; do
        module_dir="$REPO_ROOT/$module"
        if [ ! -d "$module_dir" ] || [ ! -f "$module_dir/go.mod" ]; then
            print_warning "Skipping $module (not found or no go.mod)"
            continue
        fi
        
        print_info "Processing $module..."
        cd "$module_dir"
        
        # Step 1: Add replace directives for ALL processed modules (kept for development)
        # These replace directives use RELATIVE paths and are permanent for development:
        # - They use relative paths (e.g., ../core) for portability across developers
        # - They allow go.work to resolve internal module dependencies correctly
        # - They prevent go mod tidy from trying to fetch non-existent remote versions
        # - They will be automatically removed during release by release.sh
        declare -a replace_args=()
        declare -a require_args=()
        
        if [ ${#PROCESSED_MODULES[@]} -gt 0 ]; then
            for dep in "${PROCESSED_MODULES[@]}"; do
                # Calculate relative path from current module to dependency module
                dep_dir="$REPO_ROOT/$dep"
                if [ -d "$dep_dir" ]; then
                    # Calculate relative path using Python (most reliable across platforms)
                    # CRITICAL: Must use relative paths, never absolute paths
                    relative_path=""
                    
                    if command -v python3 >/dev/null 2>&1; then
                        relative_path=$(python3 -c "import os; print(os.path.relpath('$dep_dir', '$module_dir'))")
                    elif command -v python >/dev/null 2>&1; then
                        relative_path=$(python -c "import os; print(os.path.relpath('$dep_dir', '$module_dir'))")
                    elif command -v realpath >/dev/null 2>&1; then
                        # Try realpath (GNU coreutils version, may not work on macOS)
                        relative_path=$(realpath --relative-to="$module_dir" "$dep_dir" 2>/dev/null || echo "")
                        if [ -z "$relative_path" ]; then
                            # Fallback to manual calculation if realpath fails
                            module_depth=$(echo "$module" | awk -F'/' '{print NF}')
                            relative_path=""
                            for ((i=0; i<module_depth; i++)); do
                                relative_path="../$relative_path"
                            done
                            relative_path="${relative_path}${dep}"
                        fi
                    else
                        # Fallback: manual calculation
                        # Count directory depth (number of slashes + 1)
                        module_depth=$(echo "$module" | awk -F'/' '{print NF}')
                        # Build relative path: go up (module_depth) levels, then append dep path
                        relative_path=""
                        for ((i=0; i<module_depth; i++)); do
                            relative_path="../$relative_path"
                        done
                        relative_path="${relative_path}${dep}"
                    fi
                    
                    # Verify that relative_path is actually relative (not absolute)
                    # Absolute paths start with / on Unix or C:\ on Windows
                    if [[ "$relative_path" =~ ^/ ]] || [[ "$relative_path" =~ ^[A-Za-z]: ]]; then
                        print_error "ERROR: Calculated path is absolute: $relative_path"
                        print_error "This should never happen. Please check path calculation logic."
                        exit 1
                    fi
                    
                    # Ensure relative_path is not empty
                    if [ -z "$relative_path" ]; then
                        print_error "ERROR: Failed to calculate relative path from $module_dir to $dep_dir"
                        exit 1
                    fi
                    
                    # Always add replace directive for all processed modules using relative path
                    replace_args+=("-replace=$MODULE_BASE/$dep=$relative_path")
                else
                    print_warning "Dependency directory not found: $dep_dir"
                fi
                
                # Only add require directive if this module actually imports it
                if grep -r "\"$MODULE_BASE/$dep" . --include="*.go" --exclude-dir=vendor --exclude-dir=gen 2>/dev/null | head -1 > /dev/null; then
                    print_info "Adding development dependency: $dep"
                    require_args+=("-require=$MODULE_BASE/$dep@v0.0.0-00010101000000-000000000000")
                fi
            done
            
            # Apply all edits in a single go mod edit call
            declare -a all_args=()
            if [ ${#replace_args[@]} -gt 0 ]; then
                print_info "Adding ${#replace_args[@]} replace directive(s) for development..."
                for replace_arg in "${replace_args[@]}"; do
                    print_info "  ↳ $replace_arg"
                done
                all_args+=("${replace_args[@]}")
            else
                print_warning "No replace directives to add for $module"
            fi
            if [ ${#require_args[@]} -gt 0 ]; then
                all_args+=("${require_args[@]}")
            fi
            
            if [ ${#all_args[@]} -gt 0 ]; then
                if ! go mod edit "${all_args[@]}" 2>&1; then
                    print_error "Failed to add replace/require directives for $module"
                    print_error "Command: go mod edit ${all_args[*]}"
                    exit 1
                fi
                print_success "Successfully added replace directives"
            fi
        fi
        
        # Step 2: Run go mod tidy with replace directives in place
        print_info "Running go mod tidy..."
        tidy_success=false
        if go mod tidy 2>&1; then
            tidy_success=true
        else
            print_warning "Tidy failed for $module (continuing anyway)"
        fi
        
        # Step 3: Keep replace directives for development (DO NOT REMOVE!)
        # These replace directives use RELATIVE paths and are essential for local development:
        # - They use relative paths (e.g., ../core) for portability across developers
        # - They allow go.work to resolve internal module dependencies correctly
        # - They prevent go mod tidy from trying to fetch non-existent remote versions
        # - They will be automatically removed during release by release.sh
        print_info "Keeping replace directives for development workspace"
        
        # Step 4: Mark as processed
        if [ "$tidy_success" = true ]; then
            print_success "Completed $module"
        fi
        PROCESSED_MODULES+=("$module")
    done
done

cd "$REPO_ROOT"
echo ""

# Final verification
print_section "Final verification"
print_info "Verifying workspace status..."
go work edit -json > /dev/null 2>&1 && print_success "Workspace is valid" || print_error "Workspace validation failed"

echo ""
print_header "Workspace Reinitialization Completed"
print_info "Summary:"
print_info "  - All modules have been reinitialized"
print_info "  - Workspace has been recreated"
print_info "  - Replace directives have been added to all go.mod files"
print_info "  - Local dependencies are resolved via workspace + replace"
echo ""
print_info "Important notes:"
print_info "  1. Replace directives are KEPT in go.mod (they should be committed)"
print_info "  2. Replace directives use RELATIVE paths (e.g., ../core) for portability"
print_info "  3. These replace directives are essential for local development"
print_info "  4. They will be automatically removed during release by release.sh"
echo ""
print_info "Next steps:"
print_info "  1. Review any errors above"
print_info "  2. Run 'go test ./...' to verify everything works"
print_info "  3. Commit the go.mod files with their replace directives"
print_info "  4. Never manually remove replace directives (let release.sh handle it)"

