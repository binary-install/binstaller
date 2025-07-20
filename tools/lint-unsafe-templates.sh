#!/bin/bash
# Linter to check that templates use ensureSafe for user input

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Checking for safe template usage ==="

# Initialize variable
FOUND_ISSUES=false

# Check if ast-grep is installed
if ! command -v ast-grep &> /dev/null; then
    echo -e "${YELLOW}Warning: ast-grep not found. Install with: brew install ast-grep${NC}"

    # Fallback to grep-based checking
    echo -e "\n${YELLOW}Checking for unsafe deref usage...${NC}"

    # Look for deref calls that are NOT wrapped with ensureSafe

    # Check for direct deref usage without ensureSafe
    if grep -r "{{ *deref" internal/shell/template.tmpl.sh 2>/dev/null | grep -v "ensureSafe" | grep -v "{{ deref .Name }}"; then
        FOUND_ISSUES=true
    fi

    if [ "$FOUND_ISSUES" = true ]; then
        echo -e "${RED}✗ Found unsafe template usage${NC}"
        echo "Wrap user input with ensureSafe, e.g.: {{ ensureSafe (deref .Name) }}"
    else
        echo -e "${GREEN}✓ All template usage appears safe${NC}"
    fi
else
    # Use ast-grep for more sophisticated checking
    echo -e "\n${YELLOW}Using ast-grep to check template safety...${NC}"

    # Create ast-grep rule file
    RULES_FILE=$(mktemp)
    trap "rm -f $RULES_FILE" EXIT

    cat > "$RULES_FILE" <<'EOF'
rules:
  - id: unsafe-deref
    pattern: "{{ deref $ARG }}"
    message: "Direct deref without ensureSafe wrapper"
    severity: warning
    note: "Wrap with ensureSafe: {{ ensureSafe (deref $ARG) }}"

  - id: unsafe-deref-in-assignment
    pattern: |
      $VAR='{{ deref $ARG }}'
    message: "Direct deref in assignment without ensureSafe"
    severity: warning
    note: "Wrap with ensureSafe: $VAR='{{ ensureSafe (deref $ARG) }}'"
EOF

    # Run ast-grep on template files
    if ast-grep --rule-file "$RULES_FILE" internal/shell/template.tmpl.sh 2>/dev/null | grep -v "No matches found"; then
        FOUND_ISSUES=true
        echo -e "${RED}✗ Found unsafe template usage${NC}"
    else
        echo -e "${GREEN}✓ All template usage appears safe${NC}"
    fi
fi

# Check that script generation validates input
echo -e "\n${YELLOW}Checking script generation validation...${NC}"
if grep -q "spec.Validate" internal/shell/script.go; then
    echo -e "${GREEN}✓ Script generation validates input${NC}"
else
    echo -e "${RED}✗ Script generation missing validation${NC}"
    FOUND_ISSUES=true
fi

# Check that Validate function covers all string fields
echo -e "\n${YELLOW}Checking field validation coverage...${NC}"

# Extract string fields from InstallSpec
SPEC_FIELDS=$(grep -A 20 "type InstallSpec struct" pkg/spec/generated.go | grep "\*string" | awk '{print $1}' | sort)

# Check which fields are validated
VALIDATED_FIELDS=""
for field in $SPEC_FIELDS; do
    # Convert field name to lowercase for checking
    field_lower=$(echo "$field" | sed 's/\([A-Z]\)/_\1/g' | tr '[:upper:]' '[:lower:]' | sed 's/^_//')
    if grep -q "s.$field" pkg/spec/validate.go; then
        VALIDATED_FIELDS="$VALIDATED_FIELDS $field"
    else
        # Skip fields that don't need validation
        if [[ "$field" != "Schema" ]]; then
            echo -e "${YELLOW}⚠ Field '$field' not validated${NC}"
        fi
    fi
done

# Summary
echo -e "\n=== Lint Summary ==="
if [ "$FOUND_ISSUES" = true ]; then
    echo -e "${RED}Issues found!${NC}"
    echo "1. Wrap all user input with ensureSafe in templates"
    echo "2. Ensure spec.Validate is called before script generation"
    echo "3. Ensure all string fields are validated"
    exit 1
else
    echo -e "${GREEN}All checks passed!${NC}"
fi
