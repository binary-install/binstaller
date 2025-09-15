#!/bin/bash
# Run all end-to-end tests for binst install command

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Test suites
TEST_SUITES=(
    "parity_test.sh"
    "flags_test.sh"
    "env_test.sh"
    "error_test.sh"
    "platform_test.sh"
)

# Test result tracking
TOTAL_SUITES="${#TEST_SUITES[@]}"
PASSED_SUITES=0
FAILED_SUITES=0
FAILED_SUITE_NAMES=()

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_suite() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}Running: $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# Check prerequisites
check_prerequisites() {
    if [ ! -f "$ROOT_DIR/binst" ]; then
        log_error "binst binary not found at $ROOT_DIR/binst"
        log_info "Building binst..."
        if ! make -C "$ROOT_DIR" build; then
            log_error "Failed to build binst"
            exit 1
        fi
    fi

    # Ensure all test scripts are executable
    for suite in "${TEST_SUITES[@]}"; do
        if [ ! -x "$SCRIPT_DIR/$suite" ]; then
            chmod +x "$SCRIPT_DIR/$suite"
        fi
    done
}

# Run a single test suite
run_test_suite() {
    local suite="$1"
    local suite_path="$SCRIPT_DIR/$suite"

    if [ ! -f "$suite_path" ]; then
        log_error "Test suite not found: $suite"
        return 1
    fi

    log_suite "$suite"

    if "$suite_path"; then
        log_info "✓ $suite PASSED"
        ((PASSED_SUITES++))
        return 0
    else
        log_error "✗ $suite FAILED"
        ((FAILED_SUITES++))
        FAILED_SUITE_NAMES+=("$suite")
        return 1
    fi
}

# Main execution
main() {
    local start_time
    start_time=$(date +%s)

    log_info "Starting binst install end-to-end test suite..."
    log_info "Running $TOTAL_SUITES test suites"

    check_prerequisites

    # Run all test suites
    for suite in "${TEST_SUITES[@]}"; do
        run_test_suite "$suite" || true
    done

    # Calculate elapsed time
    local end_time elapsed
    end_time=$(date +%s)
    elapsed=$((end_time - start_time))

    # Final summary
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}End-to-End Test Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo
    log_info "Total test suites: $TOTAL_SUITES"
    log_info "Passed: $PASSED_SUITES"
    log_info "Failed: $FAILED_SUITES"
    log_info "Time elapsed: ${elapsed}s"

    if [ "$FAILED_SUITES" -gt 0 ]; then
        echo
        log_error "Failed test suites:"
        for suite in "${FAILED_SUITE_NAMES[@]}"; do
            echo "  - $suite"
        done
        echo
        log_error "Some test suites failed!"
        exit 1
    else
        echo
        log_info "All test suites passed! 🎉"
        exit 0
    fi
}

# Run main if not sourced
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
