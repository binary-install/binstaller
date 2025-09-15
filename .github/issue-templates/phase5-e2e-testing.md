# Phase 5: End-to-End Testing for `binst install`

## Overview
This is the fifth and final implementation phase for the `binst install` command as outlined in the [design document](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md#phase-5-end-to-end-testing).

## Background
With all components implemented, we need comprehensive end-to-end testing to ensure the `binst install` command achieves true parity with generated installer scripts.

## Prerequisites
- [ ] Phase 1-4 must be completed
- [ ] `binst install` command must be fully functional

## Tasks
- [ ] Create e2e test suite with scripts and make tasks
- [ ] Add parity tests comparing `binst install` with generated scripts
- [ ] Test on Linux (amd64, arm64) using GitHub Actions
- [ ] Test on macOS (amd64, arm64) including Rosetta 2 detection
- [ ] Test with various real projects from testdata/
- [ ] Test all command-line flags (-b, -n, --debug)
- [ ] Test environment variable handling (GITHUB_TOKEN, BINSTALLER_BIN)
- [ ] Test error scenarios (network failures, missing assets, bad checksums)
- [ ] Ensure behavioral compatibility with generated scripts
- [ ] Add make targets for running e2e tests
- [ ] Update CI workflow to include e2e tests

## Acceptance Criteria
- [ ] Parity tests prove identical results between `binst install` and scripts
- [ ] Tests pass on all supported platforms
- [ ] Tests cover projects with different archive formats
- [ ] Tests verify checksum verification works correctly
- [ ] Tests confirm atomic installation behavior
- [ ] CI runs e2e tests on every PR
- [ ] Documentation updated with test instructions

## Technical Notes
- Use shell scripts to compare outcomes
- Test should verify: installed binary location, permissions, and content hash
- Consider using containers for Linux testing consistency
- May need to mock GitHub API for some error scenarios

## Related Issues
- Parent issue: #141
- Design document: [binst-install-command.md](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md)
