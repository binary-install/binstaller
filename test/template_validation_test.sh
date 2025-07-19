#!/bin/bash
# Integration test for template validation security feature

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Testing Template Validation Security ==="

# Create temporary directory for test files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Test 1: Gen command should reject command substitution
echo -e "${YELLOW}Test 1: Gen command with command substitution${NC}"
cat > "$TEMP_DIR/malicious1.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}\$(rm -rf /)"
  default_extension: .tar.gz
EOF

if ! ./binst gen -c "$TEMP_DIR/malicious1.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected command substitution${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject command substitution${NC}"
    exit 1
fi

# Test 2: Gen command should reject malicious templates
echo -e "${YELLOW}Test 2: Gen command with pipe character${NC}"
cat > "$TEMP_DIR/malicious2.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}|evil"
  default_extension: .tar.gz
EOF

if ! ./binst gen -c "$TEMP_DIR/malicious2.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected pipe character${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject pipe character${NC}"
    exit 1
fi

# Test 3: Valid template should work
echo -e "${YELLOW}Test 3: Valid template should pass generation${NC}"
cat > "$TEMP_DIR/valid.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}-v\${VERSION}-\${OS}-\${ARCH}\${EXT}"
  default_extension: .tar.gz
EOF

if ./binst gen -c "$TEMP_DIR/valid.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command accepted valid template${NC}"
else
    echo -e "${RED}✗ Gen command rejected valid template${NC}"
    exit 1
fi

# Test 4: Checksum template validation
echo -e "${YELLOW}Test 4: Checksum template with dangerous pattern${NC}"
cat > "$TEMP_DIR/malicious3.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
checksums:
  template: "checksums;rm -rf /"
EOF

if ! ./binst gen -c "$TEMP_DIR/malicious3.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected malicious checksum template${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject malicious checksum template${NC}"
    exit 1
fi

# Test 5: Rule template validation
echo -e "${YELLOW}Test 5: Rule template with dangerous pattern${NC}"
cat > "$TEMP_DIR/malicious4.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
  rules:
    - when: { os: linux }
      template: "\${NAME}&&malicious"
EOF

if ! ./binst gen -c "$TEMP_DIR/malicious4.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected malicious rule template${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject malicious rule template${NC}"
    exit 1
fi

# Test 6: Verify error messages are helpful
echo -e "${YELLOW}Test 6: Error messages should be descriptive${NC}"
cat > "$TEMP_DIR/malicious5.yml" <<EOF
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: "\${NAME}\`malicious\`"
  default_extension: .tar.gz
EOF

ERROR_MSG=$(./binst gen -c "$TEMP_DIR/malicious5.yml" 2>&1 || true)
if echo "$ERROR_MSG" | grep -q "backtick"; then
    echo -e "${GREEN}✓ Error message mentions backtick specifically${NC}"
else
    echo -e "${RED}✗ Error message not specific enough${NC}"
    echo "Got: $ERROR_MSG"
    exit 1
fi

echo -e "${GREEN}=== All template validation tests passed! ===${NC}"
