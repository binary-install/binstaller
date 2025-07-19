#!/bin/bash
# Integration test for comprehensive field validation

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=== Testing Comprehensive Field Validation ==="

# Create temporary directory for test files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Test 1: Dangerous name field
echo -e "${YELLOW}Test 1: Gen command should reject dangerous name${NC}"
cat > "$TEMP_DIR/dangerous_name.yml" <<EOF
schema: v1
name: "mytool\$(whoami)"
repo: owner/repo
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_name.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous name${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous name${NC}"
    exit 1
fi

# Test 2: Dangerous repo field
echo -e "${YELLOW}Test 2: Gen command should reject dangerous repo${NC}"
cat > "$TEMP_DIR/dangerous_repo.yml" <<EOF
schema: v1
name: mytool
repo: "owner/repo;echo hacked"
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_repo.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous repo${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous repo${NC}"
    exit 1
fi

# Test 3: Dangerous binary name
echo -e "${YELLOW}Test 3: Gen command should reject dangerous binary name${NC}"
cat > "$TEMP_DIR/dangerous_binary.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
  binaries:
    - name: "tool\`malicious\`"
      path: tool
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_binary.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous binary name${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous binary name${NC}"
    exit 1
fi

# Test 4: Dangerous rule OS override
echo -e "${YELLOW}Test 4: Gen command should reject dangerous rule OS${NC}"
cat > "$TEMP_DIR/dangerous_rule_os.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
asset:
  template: "\${NAME}-\${VERSION}-\${OS}"
  default_extension: .tar.gz
  rules:
    - when: { os: linux }
      os: "linux|uname -a"
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_rule_os.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous rule OS${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous rule OS${NC}"
    exit 1
fi

# Test 5: Valid default_bin_dir with shell variables
echo -e "${YELLOW}Test 5: Valid default_bin_dir with shell variables should pass${NC}"
cat > "$TEMP_DIR/valid_bindir.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
default_bin_dir: "\${HOME}/.local/bin"
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
EOF

if ./binst gen -c "$TEMP_DIR/valid_bindir.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command accepted valid default_bin_dir${NC}"
else
    echo -e "${RED}✗ Gen command rejected valid default_bin_dir${NC}"
    exit 1
fi

# Test 6: Dangerous default_bin_dir with command substitution
echo -e "${YELLOW}Test 6: Dangerous default_bin_dir should be rejected${NC}"
cat > "$TEMP_DIR/dangerous_bindir.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
default_bin_dir: "\$(echo /tmp)/bin"
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_bindir.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous default_bin_dir${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous default_bin_dir${NC}"
    exit 1
fi

# Test 7: Check command also validates all fields
echo -e "${YELLOW}Test 7: Check command should validate all fields${NC}"
cat > "$TEMP_DIR/dangerous_check.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
default_version: "v1.0.0;bad"
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: .tar.gz
EOF

if ! ./binst check -c "$TEMP_DIR/dangerous_check.yml" --check-assets=false >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Check command correctly rejected dangerous default_version${NC}"
else
    echo -e "${RED}✗ Check command failed to reject dangerous default_version${NC}"
    exit 1
fi

# Test 8: Extension with dangerous pattern
echo -e "${YELLOW}Test 8: Extension with dangerous pattern${NC}"
cat > "$TEMP_DIR/dangerous_ext.yml" <<EOF
schema: v1
name: mytool
repo: owner/repo
asset:
  template: "\${NAME}-\${VERSION}"
  default_extension: ".tar.gz && echo hacked"
EOF

if ! ./binst gen -c "$TEMP_DIR/dangerous_ext.yml" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Gen command correctly rejected dangerous extension${NC}"
else
    echo -e "${RED}✗ Gen command failed to reject dangerous extension${NC}"
    exit 1
fi

echo -e "${GREEN}=== All comprehensive validation tests passed! ===${NC}"
