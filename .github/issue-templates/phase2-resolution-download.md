# Phase 2: Asset Resolution and Download for `binst install`

## Overview
This is the second implementation phase for the `binst install` command as outlined in the [design document](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md#phase-2-resolution-and-download).

## Background
After resolving the version, the `binst install` command needs to determine the correct asset filename based on OS/arch and download it from GitHub releases.

## Prerequisites
- [ ] Phase 1 (Version Resolution) must be completed

## Tasks
- [ ] Use `pkg/asset.FilenameGenerator` for asset resolution based on OS/arch
- [ ] Implement OS/arch detection logic (including Rosetta 2 on Apple Silicon)
- [ ] Use `pkg/httpclient` for downloading release assets
- [ ] Handle asset filename templates from spec
- [ ] Apply asset selection rules from spec
- [ ] Download assets to temporary files
- [ ] Add progress reporting for downloads
- [ ] Write tests for asset resolution logic
- [ ] Write tests for download functionality

## Acceptance Criteria
- [ ] Correctly resolves asset names using templates like `${NAME}_${VERSION}_${OS}_${ARCH}${EXT}`
- [ ] Detects current OS/arch including special cases (darwin_arm64 with Rosetta 2)
- [ ] Downloads assets to secure temporary locations
- [ ] Handles missing assets with clear error messages
- [ ] Respects GITHUB_TOKEN for authenticated downloads
- [ ] Tests cover various OS/arch combinations

## Technical Notes
- Leverage existing `pkg/asset.FilenameGenerator`
- Use `pkg/spec.InstallSpec.Asset` for configuration
- Match OS/arch detection logic from generated scripts
- Use io.TeeReader for progress reporting during downloads

## Related Issues
- Parent issue: #141
- Design document: [binst-install-command.md](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md)
