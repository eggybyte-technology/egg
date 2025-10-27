#!/bin/bash
# Quick verification script for CLI v0.2.0 key features
# Usage: ./scripts/quick-verify.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

echo "=== Quick CLI Verification ==="
echo ""

# 1. Check CLI binary exists
if [ -f "$PROJECT_ROOT/cli/egg" ]; then
    echo "✓ CLI binary exists"
else
    echo "✗ CLI binary missing - building..."
    cd "$PROJECT_ROOT/cli" && go build -o egg ./cmd/egg
    echo "✓ CLI binary built"
fi

# 2. Check CLI version/help
echo ""
echo "--- CLI Help Output ---"
"$PROJECT_ROOT/cli/egg" --help | head -15
echo ""

# 3. Verify template files exist
echo "--- Template Files Check ---"
TEMPLATES=(
    "cli/internal/templates/templates/backend/main.go.tmpl"
    "cli/internal/templates/templates/backend/handler.go.tmpl"
    "cli/internal/templates/templates/backend/service.go.tmpl"
    "cli/internal/templates/templates/backend/repository.go.tmpl"
    "cli/internal/templates/templates/backend/model.go.tmpl"
    "cli/internal/templates/templates/backend/errors.go.tmpl"
    "cli/internal/templates/templates/backend/app_config.go.tmpl"
    "cli/internal/templates/templates/api/proto_echo.tmpl"
    "cli/internal/templates/templates/api/proto_crud.tmpl"
    "cli/internal/templates/templates/build/Makefile.backend.tmpl"
)

for tmpl in "${TEMPLATES[@]}"; do
    if [ -f "$PROJECT_ROOT/$tmpl" ]; then
        echo "✓ $(basename $tmpl)"
    else
        echo "✗ Missing: $tmpl"
        exit 1
    fi
done

echo ""
echo "--- Key Implementation Checks ---"

# 4. Check configschema has GetImageName
if grep -q "func.*GetImageName" "$PROJECT_ROOT/cli/internal/configschema/config.go"; then
    echo "✓ GetImageName method implemented"
else
    echo "✗ GetImageName method missing"
    exit 1
fi

# 5. Check configschema has ValidateServiceName
if grep -q "func ValidateServiceName" "$PROJECT_ROOT/cli/internal/configschema/config.go"; then
    echo "✓ ValidateServiceName function implemented"
else
    echo "✗ ValidateServiceName function missing"
    exit 1
fi

# 6. Check create.go has protoTemplate flag
if grep -q "protoTemplate" "$PROJECT_ROOT/cli/cmd/egg/create.go"; then
    echo "✓ --proto flag implemented"
else
    echo "✗ --proto flag missing"
    exit 1
fi

# 7. Check compose.go uses correct path
if grep -q 'deploy/compose/compose.yaml' "$PROJECT_ROOT/cli/cmd/egg/compose.go"; then
    echo "✓ compose.yaml path updated to deploy/compose/"
else
    echo "✗ compose.yaml path not updated"
    exit 1
fi

# 8. Check lint.go removed ImageName checks
if grep -q "service.ImageName" "$PROJECT_ROOT/cli/internal/lint/lint.go"; then
    echo "✗ lint.go still references service.ImageName"
    exit 1
else
    echo "✓ lint.go ImageName references removed"
fi

# 9. Check helm.go uses GetImageName
if grep -q "GetImageName" "$PROJECT_ROOT/cli/internal/render/helm/helm.go"; then
    echo "✓ helm.go uses GetImageName"
else
    echo "✗ helm.go does not use GetImageName"
    exit 1
fi

# 10. Check test script updates
if grep -q "deploy/compose/compose.yaml" "$PROJECT_ROOT/scripts/test-cli.sh"; then
    echo "✓ test-cli.sh updated with new paths"
else
    echo "✗ test-cli.sh not updated"
    exit 1
fi

if grep -q "Proto template generation" "$PROJECT_ROOT/scripts/test-cli.sh"; then
    echo "✓ test-cli.sh includes new feature tests"
else
    echo "✗ test-cli.sh missing new feature tests"
    exit 1
fi

echo ""
echo "=== ✓ All Verification Checks Passed ==="
echo ""
echo "To run full integration test:"
echo "  cd $PROJECT_ROOT && ./scripts/test-cli.sh"
echo ""

