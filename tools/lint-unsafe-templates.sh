#!/bin/bash
# Linter to check for potentially unsafe template usage in Go code
# This script uses ast-grep to find patterns that might bypass validation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Checking for unsafe template usage ==="

# Create temporary file for ast-grep rules
RULES_FILE=$(mktemp)
trap "rm -f $RULES_FILE" EXIT

# Rule 1: Direct use of deref without validation in shell templates
cat > "$RULES_FILE" <<'EOF'
rules:
  - id: direct-deref-in-template
    pattern: |
      {{ deref $ARG }}
    message: "Direct use of deref in template without explicit validation"
    severity: warning
    note: |
      The deref function is being used directly in a template.
      Ensure that the dereferenced value has been validated for shell safety.
      Consider using ValidateAllFields() before template rendering.
EOF

# Check for ast-grep
if ! command -v ast-grep &> /dev/null; then
    echo -e "${YELLOW}Warning: ast-grep not found. Install with: brew install ast-grep${NC}"
    echo "Falling back to grep-based checks..."

    # Fallback: Use grep to find potential issues
    echo -e "\n${YELLOW}Checking for direct template embedding patterns...${NC}"

    # Look for patterns that might embed user input directly
    PATTERNS=(
        "{{ *deref[^}]*}}"
        "BINARY_NAME_[0-9]="
        "BINARY_PATH_[0-9]="
        "OS=.*deref"
        "ARCH=.*deref"
        "NAME=.*deref"
        "REPO=.*deref"
    )

    FOUND_ISSUES=false
    for pattern in "${PATTERNS[@]}"; do
        echo -e "\nChecking for pattern: $pattern"
        if grep -r -n -E "$pattern" internal/shell/template.tmpl.sh 2>/dev/null; then
            FOUND_ISSUES=true
        fi
    done

    if [ "$FOUND_ISSUES" = true ]; then
        echo -e "\n${YELLOW}⚠ Found potential unsafe template patterns${NC}"
        echo "These patterns embed user input directly into shell scripts."
        echo "Ensure ValidateAllFields() is called before template rendering."
    else
        echo -e "\n${GREEN}✓ No unsafe template patterns found${NC}"
    fi
else
    # Use ast-grep for more sophisticated checking
    echo "Running ast-grep checks..."

    # Check Go template files
    if ast-grep --rule-file "$RULES_FILE" internal/shell/template.tmpl.sh 2>/dev/null; then
        echo -e "${YELLOW}⚠ Found potential unsafe template usage${NC}"
    fi
fi

# Additional checks using standard tools
echo -e "\n${YELLOW}Checking for validation bypass patterns...${NC}"

# Check if any file generates scripts without calling ValidateAllFields
echo "Checking for script generation without validation..."
if grep -r "Generate\|GenerateWithVersion\|GenerateRunner" cmd/ internal/ --include="*.go" | \
   grep -v "ValidateAllFields\|test\|Test" | \
   grep -v "internal/shell/script.go"; then
    echo -e "${RED}✗ Found script generation that might bypass validation${NC}"
    echo "Ensure all paths to script generation call ValidateAllFields()"
else
    echo -e "${GREEN}✓ No validation bypass found${NC}"
fi

# Check for new template functions that might not be safe
echo -e "\n${YELLOW}Checking for custom template functions...${NC}"
if grep -r "FuncMap\[" internal/shell/script.go | grep -v "shellQuote\|escapeShellTemplate\|deref\|default\|hasBinaryOverride\|trimPrefix"; then
    echo -e "${YELLOW}⚠ Found custom template functions that need review${NC}"
else
    echo -e "${GREEN}✓ All template functions are known and reviewed${NC}"
fi

# Summary
echo -e "\n=== Lint Summary ==="
echo "1. Always call ValidateAllFields() before generating scripts"
echo "2. Never embed user input directly without validation"
echo "3. Use shellQuote for dynamic values when necessary"
echo "4. Review any new template functions for security"
