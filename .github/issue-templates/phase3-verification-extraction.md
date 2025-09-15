# Phase 3: Checksum Verification and Archive Extraction for `binst install`

## Overview
This is the third implementation phase for the `binst install` command as outlined in the [design document](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md#phase-3-verification-and-extraction).

## Background
After downloading assets, the `binst install` command needs to verify checksums and extract binaries from archives, maintaining parity with generated scripts.

## Prerequisites
- [ ] Phase 1 (Version Resolution) must be completed
- [ ] Phase 2 (Asset Resolution and Download) must be completed

## Tasks
- [ ] Use `pkg/checksums` directly for checksum verification
- [ ] Support embedded checksums from config
- [ ] Support downloading and parsing checksum files
- [ ] Implement `pkg/archive` package for archive extraction
- [ ] Support tar.gz format extraction
- [ ] Support zip format extraction
- [ ] Implement strip_components logic
- [ ] Handle binary selection from extracted files
- [ ] Write comprehensive tests for checksum verification
- [ ] Write tests for archive extraction

## Acceptance Criteria
- [ ] Verifies checksums using SHA256 (matching script behavior)
- [ ] Supports both embedded and downloaded checksums
- [ ] Extracts tar.gz archives correctly
- [ ] Extracts zip archives correctly
- [ ] Applies strip_components setting from spec
- [ ] Selects correct binary from spec.Asset.Binaries list
- [ ] Fails fast on checksum mismatch with clear error
- [ ] Tests cover various archive formats and edge cases

## Technical Notes
- Use existing `pkg/checksums` package
- Archive package should detect format from file extension
- Binary selection should match script logic (use spec.Asset.Binaries)
- Consider using archive/tar and archive/zip from stdlib

## Related Issues
- Parent issue: #141
- Design document: [binst-install-command.md](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md)
