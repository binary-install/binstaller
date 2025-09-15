# Design Document: `binst install` Command

## Overview

This document outlines the design for implementing the `binst install` command that provides a native Go implementation of the installation process, achieving script-parity with the generated shell installers.

## Goals

1. **Script Parity**: The `binst install` command must produce identical results to generated installer scripts
2. **Reusability**: Create modular packages that can be reused for future commands like `binst x`
3. **Native Implementation**: Avoid shell execution; use Go for all operations
4. **Cross-Platform**: Support the same platforms as generated installers (Linux, macOS, Windows)

## Non-Goals

1. Windows PowerShell support (this issue)
2. Alternative sources beyond GitHub (GitLab, S3, etc.)
3. Additional verification methods beyond current installer capabilities
4. UX improvements beyond script parity

## Architecture

### Package Structure

```
pkg/
├── config/          # Config loading and discovery
├── resolve/         # Asset resolution logic
├── fetch/           # Download with retry logic
├── verify/          # Checksum verification
├── archive/         # Archive extraction
└── install/         # Binary installation

cmd/binst/
└── install.go       # CLI command implementation
```

### Existing Packages We Can Reuse

Based on the codebase analysis, we can leverage:

1. **`pkg/spec`**: Already defines the `InstallSpec` structure and validation
2. **`pkg/asset`**: Contains `FilenameGenerator` for asset name resolution
3. **`pkg/checksums`**: Has checksum calculation and verification logic
4. **`pkg/httpclient`**: Provides GitHub API client with authentication
5. **`cmd/shared.go`**: Contains `loadInstallSpec` and config resolution logic

### New Packages to Implement

#### 1. `pkg/config` - Configuration Management
```go
package config

// Loader handles config file discovery and loading
type Loader struct {
    configPath string
}

// Load reads and validates the InstallSpec
func (l *Loader) Load() (*spec.InstallSpec, error)

// DiscoverConfig finds the config file using default paths
func DiscoverConfig() (string, error)
```

**Implementation Notes:**
- Reuse `resolveConfigFile` from `cmd/root.go`
- Reuse `loadInstallSpec` from `cmd/shared.go`

#### 2. `pkg/resolve` - Asset Resolution
```go
package resolve

// Resolver determines the correct asset to download
type Resolver struct {
    spec    *spec.InstallSpec
    version string
}

// ResolveAsset returns the asset filename for given OS/arch
func (r *Resolver) ResolveAsset(os, arch string) (string, error)

// ResolveVersion determines the actual version to install
func (r *Resolver) ResolveVersion() (string, error)

// GetDownloadURL constructs the full download URL
func (r *Resolver) GetDownloadURL(assetName string) (string, error)
```

**Implementation Notes:**
- Use `pkg/asset.FilenameGenerator` for asset name generation
- Handle "latest" version resolution via GitHub API
- Apply OS/arch detection including Rosetta 2 support

#### 3. `pkg/fetch` - Download Management
```go
package fetch

// Downloader handles file downloads with retry logic
type Downloader struct {
    client *http.Client
    token  string
}

// Download fetches a file to a temporary location
func (d *Downloader) Download(url string) (string, error)

// DownloadWithRetry implements retry logic matching shell script
func (d *Downloader) DownloadWithRetry(url string, maxRetries int) (string, error)
```

**Implementation Notes:**
- Use `pkg/httpclient` for GitHub API access
- Implement same retry logic as shell script (3 retries by default)
- Support `GITHUB_TOKEN` environment variable

#### 4. `pkg/verify` - Checksum Verification
```go
package verify

// Verifier handles checksum verification
type Verifier struct {
    spec      *spec.InstallSpec
    algorithm string
}

// Verify checks file integrity against checksums
func (v *Verifier) Verify(filepath, version, assetName string) error

// GetExpectedChecksum retrieves checksum from embedded or downloaded source
func (v *Verifier) GetExpectedChecksum(version, assetName string) (string, error)
```

**Implementation Notes:**
- Reuse logic from `pkg/checksums`
- Support embedded checksums from config
- Download and parse checksum files when needed

#### 5. `pkg/archive` - Archive Extraction
```go
package archive

// Extractor handles archive extraction
type Extractor struct {
    stripComponents int
}

// Extract unpacks archive and returns binary paths
func (e *Extractor) Extract(archivePath, destDir string) ([]string, error)

// DetectFormat determines archive type from filename
func (e *Extractor) DetectFormat(filename string) (Format, error)

// SelectBinary picks the correct binary from extracted files
func (e *Extractor) SelectBinary(files []string, spec *spec.InstallSpec) (string, error)
```

**Implementation Notes:**
- Support tar.gz, zip formats (matching shell script)
- Implement strip_components logic
- Handle binary selection from `spec.Asset.Binaries`

#### 6. `pkg/install` - Installation Logic
```go
package install

// Installer handles binary installation
type Installer struct {
    binDir  string
    dryRun  bool
}

// Install places binary in target directory
func (i *Installer) Install(binaryPath, targetName string) error

// MakeExecutable sets appropriate permissions
func (i *Installer) MakeExecutable(path string) error

// AtomicInstall performs atomic file replacement
func (i *Installer) AtomicInstall(src, dst string) error
```

**Implementation Notes:**
- Default to `${HOME}/.local/bin` (matching script)
- Implement atomic rename for safe updates
- Set executable permissions (0755)

### CLI Command Implementation

```go
// cmd/binst/install.go
package cmd

var installCmd = &cobra.Command{
    Use:   "install [VERSION]",
    Short: "Install a binary directly from GitHub releases",
    RunE:  runInstall,
}

func init() {
    installCmd.Flags().StringVarP(&configFile, "config", "c", "", "Config file path")
    installCmd.Flags().StringVarP(&binDir, "bin-dir", "b", "", "Installation directory")
    installCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Dry run mode")
    installCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
}
```

## Implementation Plan

### Phase 1: Core Infrastructure (Foundation)
1. Create package structure
2. Implement `pkg/config` using existing code
3. Add basic CLI command skeleton
4. Write integration test framework

### Phase 2: Resolution and Download
1. Implement `pkg/resolve` with OS/arch detection
2. Implement `pkg/fetch` with retry logic
3. Add tests for version resolution
4. Test download functionality

### Phase 3: Verification and Extraction
1. Implement `pkg/verify` using existing checksum code
2. Implement `pkg/archive` for tar.gz and zip
3. Add tests for checksum verification
4. Test archive extraction

### Phase 4: Installation
1. Implement `pkg/install` with atomic operations
2. Complete CLI command implementation
3. Add dry-run support
4. Test full installation flow

### Phase 5: Parity Testing
1. Create parity test suite comparing with generated scripts
2. Test on multiple platforms (Linux, macOS)
3. Test with various projects from testdata/
4. Fix any behavioral differences

## Testing Strategy

### Unit Tests
- Each package gets comprehensive unit tests
- Mock external dependencies (GitHub API, filesystem)
- Test error conditions and edge cases

### Integration Tests
- Test full installation flow
- Use testdata configurations
- Compare results with generated installers

### Parity Tests
```go
func TestInstallParity(t *testing.T) {
    // 1. Generate installer script
    // 2. Run installer script in container
    // 3. Run binst install in same container
    // 4. Compare installed binaries (path, permissions, content)
}
```

### Platform Testing
- Linux: amd64, arm64
- macOS: amd64, arm64 (including Rosetta 2)
- Use GitHub Actions matrix builds

## Security Considerations

1. **Checksum Verification**: Always verify checksums before installation
2. **TLS**: Use HTTPS for all downloads
3. **Atomic Operations**: Prevent partial installations
4. **Permission Handling**: Set appropriate file permissions
5. **Path Validation**: Prevent directory traversal attacks

## Error Handling

Match the shell script's error behavior:
1. Network errors: Retry with backoff
2. Checksum mismatch: Fail immediately
3. Missing assets: Clear error message
4. Permission errors: Suggest fixes

## Environment Variables

Support the same variables as generated scripts:
- `GITHUB_TOKEN`: For API authentication
- `BINSTALLER_BIN`: Override default bin directory
- Standard proxy variables: `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`

## Future Extensibility

This design enables future `binst x` command by:
1. Reusing resolution/fetch/verify packages
2. Adding caching layer on top
3. Executing in temporary directory instead of installing

## Open Questions

1. Should we support parallel asset downloads for faster installation?
2. How should we handle GitHub API rate limits?
3. Should dry-run mode make network requests?

## References

- Issue #141: Original feature request
- `internal/shell/template.tmpl.sh`: Shell script template
- `pkg/asset/filename.go`: Asset name generation
- `testdata/`: Example configurations for testing
