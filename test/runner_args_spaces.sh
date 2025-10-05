#!/bin/sh
# Test that runner scripts correctly handle arguments with spaces
# This tests the fix for issue #212

set -e

# Setup
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINST="${PROJECT_ROOT}/binst"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf -- "$TEMP_DIR"' EXIT

cd "$PROJECT_ROOT"

# Disable progress OSC sequences to avoid polluting stdout
export BINSTALLER_NO_PROGRESS=1

echo "=== Testing runner script with arguments containing spaces ==="
echo ""

# Generate jq runner script
echo "Generating jq runner script..."
"$BINST" gen --config testdata/jq.binstaller.yml --type runner -o "$TEMP_DIR/jq-runner.sh"
chmod +x "$TEMP_DIR/jq-runner.sh"

# Test 1: Simple jq filter with spaces in string literal
echo "Test 1: Filter with spaces in string literal"
result=$(echo '{"name": "John Doe", "age": 30}' | "$TEMP_DIR/jq-runner.sh" '.name == "John Doe"')
if [ "$result" = "true" ]; then
    echo "  ✓ Passed: String with spaces preserved"
else
    echo "  ✗ Failed: Expected 'true', got '$result'"
    exit 1
fi

# Test 2: jq with --arg containing spaces
echo "Test 2: --arg parameter with spaces"
result=$(echo '{"greeting": "Hello World"}' | "$TEMP_DIR/jq-runner.sh" --arg msg 'Hello World' '.greeting == $msg')
if [ "$result" = "true" ]; then
    echo "  ✓ Passed: --arg with spaces preserved"
else
    echo "  ✗ Failed: Expected 'true', got '$result'"
    exit 1
fi

# Test 3: Complex jq expression with spaces and pipes
echo "Test 3: Complex expression with pipes and spaces"
result=$(echo '[{"name": "Alice Smith"}, {"name": "Bob Jones"}]' | \
    "$TEMP_DIR/jq-runner.sh" '.[] | select(.name | contains("Smith"))' | \
    jq -r '.name')
if [ "$result" = "Alice Smith" ]; then
    echo "  ✓ Passed: Complex expression preserved"
else
    echo "  ✗ Failed: Expected 'Alice Smith', got '$result'"
    exit 1
fi

# Test 4: Template-like expression (simulating github-comment case)
echo "Test 4: Template-like expression with special characters"
result=$(echo '{"content": "test data"}' | \
    "$TEMP_DIR/jq-runner.sh" --arg template '{{ .Vars.content | AvoidHTMLEscape }}' \
    '{template: $template}' | jq -r '.template')
if [ "$result" = "{{ .Vars.content | AvoidHTMLEscape }}" ]; then
    echo "  ✓ Passed: Template expression preserved"
else
    echo "  ✗ Failed: Template expression was mangled"
    echo "    Expected: '{{ .Vars.content | AvoidHTMLEscape }}'"
    echo "    Got: '$result'"
    exit 1
fi

# Test 5: Multiple arguments with varying content
echo "Test 5: Multiple mixed arguments"
result=$(echo '{}' | \
    "$TEMP_DIR/jq-runner.sh" \
    --arg a 'first arg' \
    --arg b 'second arg with more spaces' \
    --arg c 'third' \
    '{a: $a, b: $b, c: $c}' | jq -r '.b')
if [ "$result" = "second arg with more spaces" ]; then
    echo "  ✓ Passed: Multiple arguments preserved"
else
    echo "  ✗ Failed: Multiple arguments were not preserved correctly"
    echo "    Expected: 'second arg with more spaces'"
    echo "    Got: '$result'"
    exit 1
fi

# Test 6: Arguments with special shell characters
echo "Test 6: Arguments with special shell characters"
result=$(echo '{}' | \
    "$TEMP_DIR/jq-runner.sh" \
    --arg special '$HOME/*.txt' \
    '{special: $special}' | jq -r '.special')
if [ "$result" = '$HOME/*.txt' ]; then
    echo "  ✓ Passed: Special characters not expanded"
else
    echo "  ✗ Failed: Special characters were incorrectly expanded"
    echo "    Expected: '\$HOME/*.txt'"
    echo "    Got: '$result'"
    exit 1
fi

# Test 7: Empty string and whitespace-only arguments
echo "Test 7: Empty and whitespace-only arguments"
result=$(echo '{}' | \
    "$TEMP_DIR/jq-runner.sh" \
    --arg empty '' \
    --arg spaces '   ' \
    '{empty: $empty, spaces: $spaces, empty_len: ($empty | length), spaces_len: ($spaces | length)}' | \
    jq -r '.spaces_len')
if [ "$result" = "3" ]; then
    echo "  ✓ Passed: Whitespace preserved correctly"
else
    echo "  ✗ Failed: Whitespace not preserved"
    echo "    Expected length: 3"
    echo "    Got length: '$result'"
    exit 1
fi

echo ""
echo "=== All tests passed! ==="
echo "Runner scripts correctly preserve arguments with spaces."
