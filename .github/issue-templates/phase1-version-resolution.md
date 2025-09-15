# Phase 1: Version Resolution for `binst install`

## Overview
This is the first implementation phase for the `binst install` command as outlined in the [design document](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md#phase-1-version-resolution).

## Background
The `binst install` command needs to resolve version strings (including "latest") to actual release versions from GitHub, maintaining parity with how generated installer scripts handle version resolution.

## Tasks
- [ ] Implement version resolution logic to resolve "latest" to actual version using GitHub API
- [ ] Add basic CLI command skeleton for `binst install` in `cmd/binst/install.go`
- [ ] Support positional VERSION argument
- [ ] Use existing `pkg/httpclient` for GitHub API calls
- [ ] Handle GITHUB_TOKEN environment variable for authentication
- [ ] Write unit tests for version resolution logic

## Acceptance Criteria
- [ ] `binst install latest` resolves to the most recent GitHub release
- [ ] `binst install v1.2.3` validates and accepts explicit version strings
- [ ] Version resolution respects GITHUB_TOKEN when provided
- [ ] CLI command skeleton is in place with proper flag parsing
- [ ] Unit tests cover version resolution edge cases

## Technical Notes
- Use existing `pkg/httpclient` package for GitHub API interaction
- Follow the same version resolution logic as generated installers
- Ensure error messages are clear when version cannot be resolved

## Related Issues
- Parent issue: #141
- Design document: [binst-install-command.md](https://github.com/binary-install/binstaller/blob/main/doc/design/binst-install-command.md)
