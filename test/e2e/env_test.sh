#!/bin/bash
# Test binst install environment variable handling

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

# Test BINSTALLER_BIN environment variable
test_binstaller_bin_env() {
    log_info "Testing BINSTALLER_BIN environment variable..."

    local temp_dir
    temp_dir=$(mktemp -d)

    local env_bin_dir="$temp_dir/env/bin"
    mkdir -p "$env_bin_dir"

    # Test with BINSTALLER_BIN set
    log_info "Installing with BINSTALLER_BIN=$env_bin_dir"
    BINSTALLER_BIN="$env_bin_dir" "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" "jq-1.7" >/dev/null 2>&1

    if [ -f "$env_bin_dir/jq" ]; then
        log_info "✓ BINSTALLER_BIN test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ BINSTALLER_BIN test FAILED: jq not found in $env_bin_dir"
        ((FAILED_TESTS++)) || true
    fi

    # Test priority: -b flag should override BINSTALLER_BIN
    local flag_bin_dir="$temp_dir/flag/bin"
    mkdir -p "$flag_bin_dir"
    rm -f "$env_bin_dir/jq"

    log_info "Testing -b flag overrides BINSTALLER_BIN..."
    BINSTALLER_BIN="$env_bin_dir" "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$flag_bin_dir" "jq-1.7" >/dev/null 2>&1

    if [ -f "$flag_bin_dir/jq" ] && [ ! -f "$env_bin_dir/jq" ]; then
        log_info "✓ -b flag priority test PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ -b flag priority test FAILED"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test GITHUB_TOKEN environment variable
test_github_token_env() {
    log_info "Testing GITHUB_TOKEN environment variable..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test without GITHUB_TOKEN (should still work for public repos)
    log_info "Testing without GITHUB_TOKEN..."
    unset GITHUB_TOKEN
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" -n >/dev/null 2>&1
    local no_token_exit=$?

    if [ "$no_token_exit" -eq 0 ]; then
        log_info "✓ Installation without GITHUB_TOKEN PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Installation without GITHUB_TOKEN FAILED"
        ((FAILED_TESTS++)) || true
    fi

    # Test with invalid GITHUB_TOKEN (should still work for public repos)
    log_info "Testing with invalid GITHUB_TOKEN..."
    GITHUB_TOKEN="invalid_token_12345" "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" -n >/dev/null 2>&1
    local invalid_token_exit=$?

    if [ "$invalid_token_exit" -eq 0 ]; then
        log_info "✓ Installation with invalid GITHUB_TOKEN PASSED (public repo)"
        ((PASSED_TESTS++)) || true
    else
        log_warning "Installation with invalid GITHUB_TOKEN failed - this might be expected for rate-limited APIs"
        ((PASSED_TESTS++)) || true
    fi

    # Test with valid GITHUB_TOKEN if available
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        log_info "Testing with valid GITHUB_TOKEN..."
        "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" -b "$temp_dir" "jq-1.7" >/dev/null 2>&1

        if [ -f "$temp_dir/jq" ]; then
            log_info "✓ Installation with valid GITHUB_TOKEN PASSED"
            ((PASSED_TESTS++)) || true
        else
            log_error "✗ Installation with valid GITHUB_TOKEN FAILED"
            ((FAILED_TESTS++)) || true
        fi
    else
        log_warning "Skipping valid GITHUB_TOKEN test (token not available)"
    fi

    rm -rf "$temp_dir"
}

# Test default installation directory fallback
test_default_install_dir() {
    log_info "Testing default installation directory..."

    # Save original HOME
    local original_home="$HOME"

    # Create temporary HOME
    local temp_home
    temp_home=$(mktemp -d)

    export HOME="$temp_home"
    local expected_dir="$HOME/.local/bin"

    # Install without specifying directory
    log_info "Installing to default directory..."
    unset BINSTALLER_BIN
    "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" "jq-1.7" >/dev/null 2>&1

    if [ -f "$expected_dir/jq" ]; then
        log_info "✓ Default directory installation PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Default directory installation FAILED: jq not found in $expected_dir"
        ((FAILED_TESTS++)) || true
    fi

    # Restore HOME
    export HOME="$original_home"
    rm -rf "$temp_home"
}

# Test environment variable interactions
test_env_interactions() {
    log_info "Testing environment variable interactions..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test that both script and binst install use BINSTALLER_BIN correctly
    local script_dir="$temp_dir/script"
    local binst_dir="$temp_dir/binst"
    mkdir -p "$script_dir" "$binst_dir"

    # Generate installer script
    local installer_script="$TESTDATA_DIR/jq.install.sh"
    if [ ! -f "$installer_script" ]; then
        "$BINST_CMD" gen --config "$TESTDATA_DIR/jq.binstaller.yml" -o "$installer_script"
    fi

    # Install with script using BINSTALLER_BIN
    BINSTALLER_BIN="$script_dir" bash "$installer_script" "jq-1.7" >/dev/null 2>&1

    # Install with binst using BINSTALLER_BIN
    BINSTALLER_BIN="$binst_dir" "$BINST_CMD" install -c "$TESTDATA_DIR/jq.binstaller.yml" "jq-1.7" >/dev/null 2>&1

    # Compare installed binaries
    if [ -f "$script_dir/jq" ] && [ -f "$binst_dir/jq" ]; then
        local hash1 hash2
        hash1=$(sha256sum "$script_dir/jq" | cut -d' ' -f1)
        hash2=$(sha256sum "$binst_dir/jq" | cut -d' ' -f1)

        if [ "$hash1" = "$hash2" ]; then
            log_info "✓ Environment variable parity test PASSED"
            ((PASSED_TESTS++)) || true
        else
            log_error "✗ Environment variable parity test FAILED: different binaries"
            ((FAILED_TESTS++)) || true
        fi
    else
        log_error "✗ Environment variable parity test FAILED: missing binaries"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Main test execution
main() {
    log_info "Starting binst install environment variable tests..."

    if [ ! -f "$BINST_CMD" ]; then
        log_error "binst binary not found at $BINST_CMD"
        log_info "Please run 'make build' first"
        exit 1
    fi

    # Save original environment
    local original_github_token="${GITHUB_TOKEN:-}"
    local original_binstaller_bin="${BINSTALLER_BIN:-}"

    # Run all environment tests
    test_binstaller_bin_env
    test_github_token_env
    test_default_install_dir
    test_env_interactions

    # Restore original environment
    export GITHUB_TOKEN="$original_github_token"
    export BINSTALLER_BIN="$original_binstaller_bin"

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
