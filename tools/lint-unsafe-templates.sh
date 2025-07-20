#!/bin/bash
# Linter to check that templates use ensureSafe for user input

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Checking for safe template usage ==="

# Check that deref calls are wrapped with ensureSafe
echo -e "\n${YELLOW}Checking for unsafe deref usage...${NC}"

# Look for deref calls that are NOT wrapped with ensureSafe
UNSAFE_PATTERNS=(
    # Direct deref without ensureSafe
    "{{ *deref [^}]* }}"
    # Assignment without ensureSafe
    "=['\"]?{{ *deref"
)

FOUND_ISSUES=false
for pattern in "${UNSAFE_PATTERNS[@]}"; do
    # Skip patterns that include ensureSafe
    if grep -r -E "$pattern" internal/shell/template.tmpl.sh 2>/dev/null | grep -v "ensureSafe" | grep -v "{{ deref .Name }}"; then
        FOUND_ISSUES=true
    fi
done

if [ "$FOUND_ISSUES" = true ]; then
    echo -e "${RED}✗ Found unsafe template usage${NC}"
    echo "Wrap user input with ensureSafe, e.g.: {{ ensureSafe (deref .Name) }}"
else
    echo -e "${GREEN}✓ All template usage appears safe${NC}"
fi

# Check that script generation validates input
echo -e "\n${YELLOW}Checking script generation validation...${NC}"
if grep -q "spec.Validate" internal/shell/script.go; then
    echo -e "${GREEN}✓ Script generation validates input${NC}"
else
    echo -e "${RED}✗ Script generation missing validation${NC}"
    FOUND_ISSUES=true
fi

# Summary
echo -e "\n=== Lint Summary ==="
if [ "$FOUND_ISSUES" = true ]; then
    echo -e "${RED}Issues found!${NC}"
    echo "1. Wrap all user input with ensureSafe in templates"
    echo "2. Ensure spec.Validate is called before script generation"
    exit 1
else
    echo -e "${GREEN}All checks passed!${NC}"
fi
