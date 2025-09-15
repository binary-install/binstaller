#!/bin/bash
# End-to-end parity test for binst install command
# Compares behavior of binst install with generated installer scripts

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
TEST_PROJECTS=(
    "reviewdog"
    "gh"
    "jq"
    "shellcheck"
    "ripgrep"
    "bat"
)

# Test result tracking
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
TESTDATA_DIR="$ROOT_DIR/testdata"
BINST_CMD="$ROOT_DIR/binst"

# Test environment setup
export GITHUB_TOKEN="${GITHUB_TOKEN:-}"

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

check_prerequisites() {
    if [ ! -f "$BINST_CMD" ]; then
        log_error "binst binary not found at $BINST_CMD"
        log_info "Please run 'make build' first"
        exit 1
    fi

    # Check if all test configs exist
    for project in "${TEST_PROJECTS[@]}"; do
        if [ ! -f "$TESTDATA_DIR/${project}.binstaller.yml" ]; then
            log_error "Test config not found: $TESTDATA_DIR/${project}.binstaller.yml"
            exit 1
        fi
    done
}

# Compare two binaries by checking their SHA256 checksums
compare_binaries() {
    local bin1="$1"
    local bin2="$2"

    if [ ! -f "$bin1" ] || [ ! -f "$bin2" ]; then
        return 1
    fi

    local hash1 hash2
    hash1=$(sha256sum "$bin1" | cut -d' ' -f1)
    hash2=$(sha256sum "$bin2" | cut -d' ' -f1)

    [ "$hash1" = "$hash2" ]
}

# Run parity test for a single project
run_parity_test() {
    local project="$1"
    local version="${2:-}"
    local test_name="$project${version:+-$version}"

    log_info "Testing parity for $test_name..."

    # Create temporary directories for installation
    local temp_dir
    temp_dir=$(mktemp -d)

    local script_install_dir="$temp_dir/script"
    local binst_install_dir="$temp_dir/binst"
    mkdir -p "$script_install_dir" "$binst_install_dir"

    # Generate installer script if it doesn't exist
    local installer_script="$TESTDATA_DIR/${project}.install.sh"
    if [ ! -f "$installer_script" ]; then
        log_info "Generating installer script for $project..."
        "$BINST_CMD" gen --config "$TESTDATA_DIR/${project}.binstaller.yml" -o "$installer_script"
    fi

    # Install using generated script
    log_info "Installing with generated script..."
    local script_exit_code=0
    if [ -n "$version" ]; then
        BINSTALLER_BIN="$script_install_dir" bash "$installer_script" "$version" >/dev/null 2>&1 || script_exit_code=$?
    else
        BINSTALLER_BIN="$script_install_dir" bash "$installer_script" >/dev/null 2>&1 || script_exit_code=$?
    fi

    # Install using binst install
    log_info "Installing with binst install..."
    local binst_exit_code=0
    if [ -n "$version" ]; then
        "$BINST_CMD" install -c "$TESTDATA_DIR/${project}.binstaller.yml" -b "$binst_install_dir" "$version" >/dev/null 2>&1 || binst_exit_code=$?
    else
        "$BINST_CMD" install -c "$TESTDATA_DIR/${project}.binstaller.yml" -b "$binst_install_dir" >/dev/null 2>&1 || binst_exit_code=$?
    fi

    # Compare exit codes
    if [ "$script_exit_code" -ne "$binst_exit_code" ]; then
        log_error "Exit code mismatch for $test_name: script=$script_exit_code, binst=$binst_exit_code"
        ((FAILED_TESTS++)) || true
        return 1
    fi

    # If both failed, that's still parity
    if [ "$script_exit_code" -ne 0 ]; then
        log_warning "Both methods failed for $test_name with exit code $script_exit_code (parity maintained)"
        ((PASSED_TESTS++)) || true
        return 0
    fi

    # Compare installed binaries
    local installed_files=()
    for file in "$script_install_dir"/*; do
        if [ -f "$file" ]; then
            installed_files+=("$(basename "$file")")
        fi
    done

    # Check each installed file
    local all_match=true
    for file in "${installed_files[@]}"; do
        if [ ! -f "$binst_install_dir/$file" ]; then
            log_error "File $file installed by script but not by binst"
            all_match=false
        elif ! compare_binaries "$script_install_dir/$file" "$binst_install_dir/$file"; then
            log_error "Binary $file differs between script and binst installation"
            all_match=false
        else
            log_info "Binary $file matches (parity confirmed)"
        fi
    done

    # Check if binst installed any extra files
    for file in "$binst_install_dir"/*; do
        if [ -f "$file" ]; then
            local basename
            basename=$(basename "$file")
            if [[ ! " ${installed_files[*]} " =~ " ${basename} " ]]; then
                log_error "File $basename installed by binst but not by script"
                all_match=false
            fi
        fi
    done

    if [ "$all_match" = true ]; then
        log_info "✓ Parity test PASSED for $test_name"
        ((PASSED_TESTS++)) || true
        return 0
    else
        log_error "✗ Parity test FAILED for $test_name"
        ((FAILED_TESTS++)) || true
        return 1
    fi
}

# Main test execution
main() {
    log_info "Starting binst install parity tests..."
    check_prerequisites

    # Run tests for each project
    for project in "${TEST_PROJECTS[@]}"; do
        # Test latest version
        run_parity_test "$project" || true

        # Test specific version (if available)
        case "$project" in
            "reviewdog")
                run_parity_test "$project" "v0.20.0" || true
                ;;
            "gh")
                run_parity_test "$project" "v2.50.0" || true
                ;;
            "jq")
                run_parity_test "$project" "jq-1.7" || true
                ;;
        esac
    done

    # Summary
    echo
    log_info "Test Summary:"
    log_info "  Passed: $PASSED_TESTS"
    log_info "  Failed: $FAILED_TESTS"
    log_info "  Skipped: $SKIPPED_TESTS"

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
