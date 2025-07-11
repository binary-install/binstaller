#!/bin/bash
set -e

echo "=== Testing runner mode functionality ==="

# Test directory
TEST_DIR=$(mktemp -d)
trap 'rm -rf -- "$TEST_DIR"' EXIT

echo "=== Test 1: Generate runner script for binst ==="
./binst gen --config .config/binstaller.yml --type runner -o "$TEST_DIR/run-binst.sh"

# Verify runner script was generated
if [ ! -f "$TEST_DIR/run-binst.sh" ]; then
    echo "ERROR: Runner script was not generated"
    exit 1
fi

# Verify it's a runner script (not installer)
if ! grep -q "download and run" "$TEST_DIR/run-binst.sh"; then
    echo "ERROR: Generated script should contain 'download and run' in usage"
    exit 1
fi

# Verify it has runner-specific usage patterns
if ! grep -q -- "-- \[binary arguments\]" "$TEST_DIR/run-binst.sh"; then
    echo "ERROR: Generated script should show binary arguments pattern"
    exit 1
fi

# Verify it has example usage with --
if ! grep -q "Example.*-- --help" "$TEST_DIR/run-binst.sh"; then
    echo "ERROR: Generated script should have example with -- --help"
    exit 1
fi

# Verify it doesn't have installer-specific options like -b (bindir)
if grep -q "\-b.*bindir" "$TEST_DIR/run-binst.sh"; then
    echo "ERROR: Runner script should not have -b bindir option"
    exit 1
fi

echo "✓ Runner script generated correctly"

echo ""
echo "=== Test 2: Test runner script with help command ==="
# Set environment variables to avoid actual GitHub API calls
export BINSTALLER_OS=linux
export BINSTALLER_ARCH=amd64

# Generate a runner script that will work offline for testing
# Create a test config that uses embedded checksums to avoid GitHub API calls
cat > "$TEST_DIR/test-binst.binstaller.yml" << 'EOF'
schema: v1
name: binst
repo: binary-install/binstaller
asset:
  template: ${NAME}_${OS}_${ARCH}${EXT}
  default_extension: .tar.gz
  rules:
    - when:
        arch: amd64
      arch: x86_64
    - when:
        os: windows
      ext: .zip
  naming_convention:
    os: titlecase
    arch: lowercase
checksums:
  algorithm: sha256
  template: checksums.txt
  embedded_checksums:
    v0.2.0:
      - filename: binst_Linux_x86_64.tar.gz
        hash: d404401c8c6495b206fc35c95e55a6e65edf2655e4eeda3d1a4b0cfd7c89b3ce
supported_platforms:
  - os: linux
    arch: amd64
EOF

./binst gen --config "$TEST_DIR/test-binst.binstaller.yml" --type runner --target-version v0.2.0 -o "$TEST_DIR/run-binst-test.sh"

echo "✓ Runner script for testing generated"

echo ""
echo "=== Test 3: Test runner script argument parsing ==="
# Test that runner script shows proper usage when called without --
if ! timeout 10s bash "$TEST_DIR/run-binst-test.sh" --help 2>&1 | grep -q "download and run"; then
    echo "ERROR: Runner script --help should show usage"
    exit 1
fi

echo "✓ Runner script argument parsing works"

echo ""
echo "=== Test 4: Generate runner script with target version ==="
./binst gen --config "$TEST_DIR/test-binst.binstaller.yml" --type runner --target-version v0.2.0 -o "$TEST_DIR/run-binst-v0.2.0.sh"

# Verify target version is embedded
if ! grep -q 'TAG="v0.2.0"' "$TEST_DIR/run-binst-v0.2.0.sh"; then
    echo "ERROR: Generated runner script should contain fixed TAG=v0.2.0"
    exit 1
fi

# Verify runner-specific message with version
if ! grep -q "Running \${NAME} version \${VERSION}" "$TEST_DIR/run-binst-v0.2.0.sh"; then
    echo "ERROR: Generated runner script should contain 'Running' message with version"
    exit 1
fi

# Verify usage message mentions the fixed version
if ! grep -q "This script is configured for v0.2.0 only" "$TEST_DIR/run-binst-v0.2.0.sh"; then
    echo "ERROR: Generated runner script should mention fixed version in usage"
    exit 1
fi

echo "✓ Runner script with target version generated correctly"

echo ""
echo "=== Test 5: Verify --type flag in help ==="
if ! ./binst gen --help | grep -q -- "--type"; then
    echo "ERROR: gen --help should show --type flag"
    exit 1
fi

if ! ./binst gen --help | grep -q "Type of script to generate.*runner"; then
    echo "ERROR: gen --help should describe runner type"
    exit 1
fi

echo "✓ --type flag is documented in help"

echo ""
echo "=== All runner mode tests passed ==="