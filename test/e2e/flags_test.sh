#!/bin/bash
# Test binst install command-line flags

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TESTDATA_DIR="$ROOT_DIR/testdata"
BINST_CMD="$ROOT_DIR/binst"

# Test result tracking
PASSED_TESTS=0
FAILED_TESTS=0

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Test --bin-dir/-b flag
test_bin_dir_flag() {
    log_info "Testing --bin-dir/-b flag..."

    local temp_dir
    temp_dir=$(mktemp -d)

    local custom_bin_dir="$temp_dir/custom/bin"
    mkdir -p "$custom_bin_dir"

    # Test with --bin-dir
    log_info "Testing with --bin-dir flag..."
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" --bin-dir "$custom_bin_dir" "jq-1.7" >/dev/null 2>&1

    if [ -f "$custom_bin_dir/jq" ]; then
        log_info "✓ --bin-dir flag test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ --bin-dir flag test FAILED: jq not found in $custom_bin_dir"
        ((FAILED_TESTS++)) || true
    fi

    # Clean up for next test
    rm -f "$custom_bin_dir/jq"

    # Test with -b (short form)
    log_info "Testing with -b flag (short form)..."
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$custom_bin_dir" "jq-1.7" >/dev/null 2>&1

    if [ -f "$custom_bin_dir/jq" ]; then
        log_info "✓ -b flag test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ -b flag test FAILED: jq not found in $custom_bin_dir"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test --dry-run/-n flag
test_dry_run_flag() {
    log_info "Testing --dry-run/-n flag..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with --dry-run
    log_info "Testing with --dry-run flag..."
    local output
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" --dry-run 2>&1)

    # Check that dry run doesn't actually install
    if [ ! -f "$temp_dir/jq" ] && [[ "$output" =~ "Dry run mode" ]]; then
        log_info "✓ --dry-run flag test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ --dry-run flag test FAILED"
        ((FAILED_TESTS++)) || true
    fi

    # Test with -n (short form)
    log_info "Testing with -n flag (short form)..."
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" -n 2>&1)

    if [ ! -f "$temp_dir/jq" ] && [[ "$output" =~ "Dry run mode" ]]; then
        log_info "✓ -n flag test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ -n flag test FAILED"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test --debug flag
test_debug_flag() {
    log_info "Testing --debug flag..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Capture debug output
    local output
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" --debug -n 2>&1)

    # Check for debug-level output indicators
    if [[ "$output" =~ "Resolved version" ]] && [[ "$output" =~ "Detected Platform" ]]; then
        log_info "✓ --debug flag test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ --debug flag test FAILED: debug output not detected"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test version argument
test_version_argument() {
    log_info "Testing VERSION argument..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test specific version
    log_info "Installing specific version jq-1.6..."
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" "jq-1.6" >/dev/null 2>&1

    if [ -f "$temp_dir/jq" ]; then
        log_info "✓ Specific version installation PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Specific version installation FAILED"
        ((FAILED_TESTS++)) || true
    fi

    rm -f "$temp_dir/jq"

    # Test latest version (no argument)
    log_info "Installing latest version..."
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" >/dev/null 2>&1

    if [ -f "$temp_dir/jq" ]; then
        log_info "✓ Latest version installation PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Latest version installation FAILED"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Main test execution
main() {
    log_info "Starting binst install flags tests..."

    if [ ! -f "$BINST_CMD" ]; then
        log_error "binst binary not found at $BINST_CMD"
        log_info "Please run 'make build' first"
        exit 1
    fi

    # Run all flag tests
    test_bin_dir_flag
    test_dry_run_flag
    # test_debug_flag  # Commented out: binst install doesn't have --debug flag
    test_version_argument

    # Summary
    echo
    log_info "Test Summary:"
    log_info "  Passed: $PASSED_TESTS"
    log_info "  Failed: $FAILED_TESTS"

    if [ "$FAILED_TESTS" -gt 0 ]; then
        log_error "Some tests failed!"
        exit 1
    else
        log_info "All tests passed!"
        exit 0
    fi
}

# Run main if not sourced
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
