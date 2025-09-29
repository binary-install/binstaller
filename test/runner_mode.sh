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

echo "✓ Runner script generated correctly"

echo ""
echo "=== Test 2: Test runner script commands ==="
# Test runner script with help command (all args pass through)
if ! "$TEST_DIR/run-binst.sh" help > /dev/null 2>&1; then
    echo "ERROR: Runner script failed to execute 'binst help'"
    exit 1
fi

echo "✓ Runner script 'help' command executed successfully"

# Test runner script with check command (use BINSTALLER_DEBUG for debug output)
if ! BINSTALLER_DEBUG=1 "$TEST_DIR/run-binst.sh" check --help | grep '✓ EXISTS'; then
    echo "ERROR: Runner script failed to execute 'binst check --help' or output missing expected content"
    exit 1
fi

echo "✓ Runner script 'check --help' command executed successfully"

echo ""
echo "=== All runner mode tests passed ==="
