#!/bin/bash
# Test binst install error scenarios

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

# Test invalid version
test_invalid_version() {
    log_info "Testing invalid version handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with non-existent version
    local output exit_code
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" "v99.99.99" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "404" || "$output" =~ "Not Found" || "$output" =~ "failed" ]]; then
        log_info "✓ Invalid version error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Invalid version error handling FAILED: expected error for non-existent version"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test missing config file
test_missing_config() {
    log_info "Testing missing config file handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with non-existent config
    local output exit_code
    output=$("$BINST_CMD" install -c "$temp_dir/non-existent.yml" -b "$temp_dir" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "no such file" || "$output" =~ "not found" || "$output" =~ "failed to read" ]]; then
        log_info "✓ Missing config file error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Missing config file error handling FAILED: expected error for missing config"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test invalid config file
test_invalid_config() {
    log_info "Testing invalid config file handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Create invalid config
    cat > "$temp_dir/invalid.yml" <<EOF
this is not valid yaml: [
  incomplete
EOF

    # Test with invalid config
    local output exit_code
    output=$("$BINST_CMD" install -c "$temp_dir/invalid.yml" -b "$temp_dir" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "yaml" || "$output" =~ "parse" || "$output" =~ "invalid" ]]; then
        log_info "✓ Invalid config file error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Invalid config file error handling FAILED: expected error for invalid YAML"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test network failure simulation
test_network_failure() {
    log_info "Testing network failure handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Create a config pointing to invalid GitHub repo
    cat > "$temp_dir/network-test.yml" <<EOF
version: 1
name: test-tool
repo: invalid-org-12345/non-existent-repo-67890
asset:
  filename: test-\${VERSION}-\${OS}-\${ARCH}.tar.gz
EOF

    # Test with invalid repo (network failure)
    local output exit_code
    output=$("$BINST_CMD" install -c "$temp_dir/network-test.yml" -b "$temp_dir" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "404" || "$output" =~ "failed" || "$output" =~ "error" ]]; then
        log_info "✓ Network failure error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Network failure error handling FAILED: expected error for invalid repo"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test checksum mismatch
test_checksum_mismatch() {
    log_info "Testing checksum mismatch handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Create a config with wrong checksum
    cat > "$temp_dir/checksum-test.yml" <<EOF
schema: v1
name: jq
repo: jqlang/jq
asset:
  template: \${NAME}-\${OS}-\${ARCH}
  rules:
    - when: { os: darwin }
      os: macos
    - when:
        arch: "386"
      arch: i386
checksums:
  algorithm: sha256
  pattern: inline
  values:
    jq-linux-amd64: "0000000000000000000000000000000000000000000000000000000000000000"
EOF

    # Test with wrong checksum
    local output exit_code
    output=$("$BINST_CMD" install -c "$temp_dir/checksum-test.yml" -b "$temp_dir" "jq-1.7" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "checksum" || "$output" =~ "verification failed" || "$output" =~ "mismatch" ]]; then
        log_info "✓ Checksum mismatch error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Checksum mismatch error handling FAILED: expected error for wrong checksum"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test permission denied scenario
test_permission_denied() {
    log_info "Testing permission denied handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Create read-only directory
    local readonly_dir="$temp_dir/readonly"
    mkdir -p "$readonly_dir"
    chmod 555 "$readonly_dir"

    # Test installation to read-only directory
    local output exit_code
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$readonly_dir" "jq-1.7" 2>&1) || exit_code=$?

    if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "permission denied" || "$output" =~ "Permission denied" || "$output" =~ "failed to" ]]; then
        log_info "✓ Permission denied error handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Permission denied error handling FAILED: expected error for read-only directory"
        ((FAILED_TESTS++)) || true
    fi

    chmod 755 "$readonly_dir"
    rm -rf "$temp_dir"
}

# Test disk space simulation (using small tmpfs)
test_disk_space() {
    log_info "Testing disk space error handling..."

    # Skip if not on Linux or if running in CI without privileges
    if [[ "$OSTYPE" != "linux"* ]] || [ -n "${CI:-}" ]; then
        log_warning "Skipping disk space test (requires Linux with mount privileges)"
        return
    fi

    local temp_dir
    temp_dir=$(mktemp -d)

    # Try to create small tmpfs (may fail without privileges)
    mkdir -p "$temp_dir/small"
    if mount -t tmpfs -o size=1M tmpfs "$temp_dir/small" 2>/dev/null; then
        # Test installation to small filesystem
        local output exit_code
        output=$("$BINST_CMD" install -c "$TESTDATA_DIR/goreleaser.binstaller.yml" -b "$temp_dir/small" 2>&1) || exit_code=$?

        if [ "${exit_code:-0}" -ne 0 ] && [[ "$output" =~ "space" || "$output" =~ "full" || "$output" =~ "failed" ]]; then
            log_info "✓ Disk space error handling PASSED"
            ((PASSED_TESTS++)) || true
        else
            log_error "✗ Disk space error handling FAILED: expected error for full filesystem"
            ((FAILED_TESTS++)) || true
        fi

        umount "$temp_dir/small" 2>/dev/null || true
    else
        log_warning "Skipping disk space test (mount failed - need privileges)"
    fi

    rm -rf "$temp_dir"
}

# Test concurrent installation (race condition)
test_concurrent_installation() {
    log_info "Testing concurrent installation handling..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Run two installations concurrently to same location
    log_info "Running concurrent installations..."
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" "jq-1.7" >/dev/null 2>&1 &
    local pid1=$!

    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" "jq-1.7" >/dev/null 2>&1 &
    local pid2=$!

    # Wait for both to complete
    local exit1=0 exit2=0
    wait $pid1 || exit1=$?
    wait $pid2 || exit2=$?

    # At least one should succeed, and binary should be valid
    if { [ "$exit1" -eq 0 ] || [ "$exit2" -eq 0 ]; } && [ -f "$temp_dir/jq" ] && [ -x "$temp_dir/jq" ]; then
        log_info "✓ Concurrent installation handling PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Concurrent installation handling FAILED: both installations failed or binary invalid"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Main test execution
main() {
    log_info "Starting binst install error scenario tests..."

    if [ ! -f "$BINST_CMD" ]; then
        log_error "binst binary not found at $BINST_CMD"
        log_info "Please run 'make build' first"
        exit 1
    fi

    # Run all error tests
    test_invalid_version
    test_missing_config
    test_invalid_config
    test_network_failure
    test_checksum_mismatch
    test_permission_denied
    test_disk_space
    test_concurrent_installation

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
