#!/bin/bash
set -e

echo "=== Testing target version functionality ==="

# Test directory
TEST_DIR=$(mktemp -d)
trap 'rm -rf -- "$TEST_DIR"' EXIT

# Create test config with embedded checksums
cat > "$TEST_DIR/test-tool.binstaller.yml" << 'EOF'
schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: ${NAME}-${VERSION}-${OS}_${ARCH}${EXT}
  default_extension: .tar.gz
checksums:
  algorithm: sha256
  template: ${NAME}_${VERSION}_checksums.txt
  embedded_checksums:
    v1.2.3:
      - filename: test-tool-1.2.3-linux_amd64.tar.gz
        hash: abc123def456abc123def456abc123def456abc123def456abc123def456abc1
    v1.2.4:
      - filename: test-tool-1.2.4-linux_amd64.tar.gz
        hash: def456ghi789def456ghi789def456ghi789def456ghi789def456ghi789def4
supported_platforms:
  - os: linux
    arch: amd64
  - os: darwin
    arch: amd64
EOF

echo "=== Test 1: Generate installer with target version v1.2.3 ==="
./binst gen --config "$TEST_DIR/test-tool.binstaller.yml" --target-version v1.2.3 -o "$TEST_DIR/install-v1.2.3.sh"

# Verify target version is embedded
if ! grep -q 'TAG="v1.2.3"' "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should contain fixed TAG=v1.2.3"
    exit 1
fi

# Verify installing message with version
if ! grep -q "Installing \${NAME} version \${VERSION}" "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should contain 'Installing' message with version"
    exit 1
fi

# Verify usage message mentions the fixed version
if ! grep -q "This installer is configured for v1.2.3 only" "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should mention fixed version in usage"
    exit 1
fi

# Verify no dynamic version logic
if grep -q 'TAG="${1:-latest}"' "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should not contain dynamic TAG assignment"
    exit 1
fi

if grep -q "checking GitHub for latest tag" "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should not contain GitHub API calls"
    exit 1
fi

# Verify only target version checksums are included
if ! grep -q "1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123" "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should contain v1.2.3 checksums"
    exit 1
fi

if grep -q "1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:def456" "$TEST_DIR/install-v1.2.3.sh"; then
    echo "ERROR: Generated script should not contain v1.2.4 checksums"
    exit 1
fi

echo "✓ Target version v1.2.3 script generated correctly"

echo ""
echo "=== Test 2: Generate normal installer without target version ==="
./binst gen --config "$TEST_DIR/test-tool.binstaller.yml" -o "$TEST_DIR/install-normal.sh"

# Verify dynamic version logic is present
if ! grep -q 'TAG="${1:-latest}"' "$TEST_DIR/install-normal.sh"; then
    echo "ERROR: Normal script should contain dynamic TAG assignment"
    exit 1
fi

# Verify all checksums are included
if ! grep -q "1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123" "$TEST_DIR/install-normal.sh"; then
    echo "ERROR: Normal script should contain v1.2.3 checksums"
    exit 1
fi

if ! grep -q "1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:def456" "$TEST_DIR/install-normal.sh"; then
    echo "ERROR: Normal script should contain v1.2.4 checksums"
    exit 1
fi

# Verify no fixed version installing message
if grep -q "Installing \${NAME} version \${VERSION}" "$TEST_DIR/install-normal.sh"; then
    echo "ERROR: Normal script should not contain fixed version installing message"
    exit 1
fi

echo "✓ Normal dynamic script generated correctly"

echo ""
echo "=== Test 3: Verify --target-version flag in help ==="
if ! ./binst gen --help | grep -q -- "--target-version"; then
    echo "ERROR: gen --help should show --target-version flag"
    exit 1
fi

if ! ./binst gen --help | grep -q "Generate script for specific version only"; then
    echo "ERROR: gen --help should describe --target-version flag"
    exit 1
fi

echo "✓ --target-version flag is documented in help"

echo ""
echo "=== All target version tests passed ==="
