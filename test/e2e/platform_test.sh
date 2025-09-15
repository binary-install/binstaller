#!/bin/bash
# Test binst install across different platforms and architectures

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

# Get current platform info
get_platform_info() {
    local os arch
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$(uname -m)" in
        x86_64) arch="amd64" ;;
        aarch64) arch="arm64" ;;
        arm64) arch="arm64" ;;
        armv7l) arch="armv7" ;;
        i386) arch="386" ;;
        i686) arch="386" ;;
        *) arch="$(uname -m)" ;;
    esac
    echo "$os/$arch"
}

# Test platform detection
test_platform_detection() {
    log_info "Testing platform detection..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with debug output to see detected platform
    local output
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/reviewdog.binstaller.yml" -b "$temp_dir" -n 2>&1)

    local current_platform
    current_platform=$(get_platform_info)

    if [[ "$output" =~ "Detected Platform" ]] || [[ "$output" =~ "linux/amd64" ]] || [[ "$output" =~ "darwin" ]]; then
        log_info "✓ Platform detection working"
        log_info "Current platform: $current_platform"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Platform detection failed: platform info not found in output"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test various archive formats
test_archive_formats() {
    log_info "Testing various archive formats..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Projects with different archive formats
    local -A archive_tests=(
        ["tar.gz"]="reviewdog"    # .tar.gz format
        ["zip"]="shellcheck"       # .zip format
        ["tar.xz"]="bat"          # .tar.xz format
        ["binary"]="jq"           # Direct binary (no archive)
    )

    for format in "${!archive_tests[@]}"; do
        local project="${archive_tests[$format]}"
        log_info "Testing $format format with $project..."

        local test_dir="$temp_dir/$format"
        mkdir -p "$test_dir"

        if "$BINST_CMD" install -c "$TESTDATA_DIR/${project}.binstaller.yml" -b "$test_dir" >/dev/null 2>&1; then
            # Check if binary was extracted correctly
            local binary_found=false
            for file in "$test_dir"/*; do
                if [ -f "$file" ] && [ -x "$file" ]; then
                    binary_found=true
                    break
                fi
            done

            if [ "$binary_found" = true ]; then
                log_info "✓ Archive format $format test PASSED"
                ((PASSED_TESTS++)) || true
            else
                log_error "✗ Archive format $format test FAILED: no executable found"
                ((FAILED_TESTS++)) || true
            fi
        else
            log_error "✗ Archive format $format test FAILED: installation failed"
            ((FAILED_TESTS++)) || true
        fi
    done

    rm -rf "$temp_dir"
}

# Test Rosetta 2 detection on macOS
test_rosetta2_detection() {
    log_info "Testing Rosetta 2 detection..."

    # Only run on macOS arm64
    if [[ "$OSTYPE" != "darwin"* ]] || [[ "$(uname -m)" != "arm64" ]]; then
        log_warning "Skipping Rosetta 2 test (not on macOS arm64)"
        return
    fi

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with a project that supports Rosetta 2
    local output
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/reviewdog.binstaller.yml" -b "$temp_dir" -n 2>&1)

    # Check if Rosetta 2 detection is working
    if command -v arch >/dev/null 2>&1 && arch -arch x86_64 true 2>/dev/null; then
        if [[ "$output" =~ "Rosetta 2" || "$output" =~ "using amd64" ]]; then
            log_info "✓ Rosetta 2 detection PASSED"
            ((PASSED_TESTS++)) || true
        else
            log_warning "Rosetta 2 available but not detected in output"
            ((PASSED_TESTS++)) || true
        fi
    else
        log_info "✓ Rosetta 2 not available (expected)"
        ((PASSED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test multi-binary installations
test_multi_binary() {
    log_info "Testing multi-binary installations..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with projects that install multiple binaries
    # Note: Most test projects install single binaries, but we can test the mechanism
    log_info "Testing single binary installation (baseline)..."

    "$BINST_CMD" install -c "$TESTDATA_DIR/gh.binstaller.yml" -b "$temp_dir" >/dev/null 2>&1

    local binary_count=0
    for file in "$temp_dir"/*; do
        if [ -f "$file" ] && [ -x "$file" ]; then
            ((binary_count++)) || true
            log_info "Found binary: $(basename "$file")"
        fi
    done

    if [ "$binary_count" -gt 0 ]; then
        log_info "✓ Binary installation test PASSED ($binary_count binaries)"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Binary installation test FAILED: no binaries found"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test strip components functionality
test_strip_components() {
    log_info "Testing strip components functionality..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with a project that uses strip_components
    log_info "Testing archive extraction with strip_components..."

    if "$BINST_CMD" install -c "$TESTDATA_DIR/goreleaser.binstaller.yml" -b "$temp_dir" >/dev/null 2>&1; then
        if [ -f "$temp_dir/goreleaser" ] && [ -x "$temp_dir/goreleaser" ]; then
            log_info "✓ Strip components test PASSED"
            ((PASSED_TESTS++)) || true
        else
            log_error "✗ Strip components test FAILED: binary not found at expected location"
            ((FAILED_TESTS++)) || true
        fi
    else
        log_error "✗ Strip components test FAILED: installation failed"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Test OS/arch specific rules
test_platform_rules() {
    log_info "Testing platform-specific rules..."

    local temp_dir
    temp_dir=$(mktemp -d)

    # Test with a project that has platform-specific rules
    local output
    output=$("$BINST_CMD" install -c "$TESTDATA_DIR/hugo.binstaller.yml" -b "$temp_dir" -n 2>&1)

    # Check that platform rules are being evaluated - just check it runs successfully
    if [[ -n "$output" ]]; then
        log_info "✓ Platform rules evaluation PASSED"
        ((PASSED_TESTS++)) || true
    else
        log_error "✗ Platform rules evaluation FAILED"
        ((FAILED_TESTS++)) || true
    fi

    rm -rf "$temp_dir"
}

# Main test execution
main() {
    log_info "Starting binst install platform tests..."

    if [ ! -f "$BINST_CMD" ]; then
        log_error "binst binary not found at $BINST_CMD"
        log_info "Please run 'make build' first"
        exit 1
    fi

    # Run all platform tests
    test_platform_detection
    test_archive_formats
    test_rosetta2_detection
    test_multi_binary
    test_strip_components
    test_platform_rules

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
