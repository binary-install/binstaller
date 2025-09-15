# Phase 4: Binary Installation for `binst install`

## Overview
This is the fourth implementation phase for the `binst install` command as outlined in the [design document](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md#phase-4-installation).

## Background
After extracting and verifying binaries, the `binst install` command needs to install them to the target directory with proper permissions and atomic operations.

## Prerequisites
- [ ] Phase 1 (Version Resolution) must be completed
- [ ] Phase 2 (Asset Resolution and Download) must be completed
- [ ] Phase 3 (Verification and Extraction) must be completed

## Tasks
- [ ] Implement `pkg/install` package with atomic installation
- [ ] Support default bin directory (`${HOME}/.local/bin`)
- [ ] Handle `-b` flag for custom installation directory
- [ ] Implement atomic file replacement using temp file + rename
- [ ] Set executable permissions (0755) on installed binaries
- [ ] Complete CLI command implementation in `cmd/binst/install.go`
- [ ] Implement dry-run (`-n`) support
- [ ] Add debug logging support
- [ ] Write tests for installation logic
- [ ] Test full installation flow end-to-end

## Acceptance Criteria
- [ ] Installs binaries to correct location with executable permissions
- [ ] Uses atomic operations to prevent partial installations
- [ ] Respects `-b` flag for custom install directory
- [ ] Dry-run mode shows what would be done without installing
- [ ] Overwrites existing binaries safely (atomic replacement)
- [ ] Creates target directory if it doesn't exist
- [ ] Clear error messages for permission issues
- [ ] Full command works end-to-end

## Technical Notes
- Default to `${HOME}/.local/bin` (same as generated scripts)
- Use os.Rename for atomic operations where possible
- Handle cross-device moves (copy + delete when rename fails)
- Ensure directory creation respects umask

## Related Issues
- Parent issue: #141
- Design document: [binst-install-command.md](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md)
