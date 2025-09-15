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
├── verify/          # Checksum verification (uses pkg/checksums)
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
5. **`cmd/shared.go`**: Contains `loadInstallSpec` for loading config files
6. **`cmd/root.go`**: Contains `resolveConfigFile` for config path discovery

### New Packages to Implement

#### 1. `pkg/verify` - Checksum Verification
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
- Use `pkg/checksums` package for all checksum operations
- Support embedded checksums from config
- Download and parse checksum files when needed

#### 2. `pkg/archive` - Archive Extraction
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

#### 3. `pkg/install` - Installation Logic
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
    installCmd.Flags().StringVarP(&binDir, "bin-dir", "b", "", "Installation directory")
    installCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Dry run mode")
    // Config file flag is inherited from root command
}

func runInstall(cmd *cobra.Command, args []string) error {
    // 1. Resolve config file path using resolveConfigFile from root.go
    cfgPath, err := resolveConfigFile(configFile)
    if err != nil {
        return err
    }

    // 2. Load config using loadInstallSpec from shared.go
    spec, err := loadInstallSpec(cfgPath)
    if err != nil {
        return err
    }

    // 3. Resolve version (latest if not specified)
    // 4. Use pkg/asset.FilenameGenerator for asset resolution
    // 5. Use pkg/httpclient.NewGitHubClient() for downloading
    // 6. Verify using pkg/checksums, extract, and install

    // For dry-run: validate URLs/versions but skip installation
    if dryRun {
        // Perform version resolution
        // Validate URLs exist
        // Skip the actual download and installation
    }

    return nil
}
```

## Implementation Plan

### Phase 1: Core Infrastructure (Foundation)
1. Implement version resolution logic
2. Add basic CLI command skeleton using existing functions from cmd/shared.go and cmd/root.go

### Phase 2: Resolution and Download
1. Use `pkg/asset.FilenameGenerator` for asset resolution
2. Use `pkg/httpclient` for downloading files
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

### Phase 5: End-to-End Testing
1. Create e2e test suite with scripts and make tasks
2. Test on multiple platforms (Linux, macOS)
3. Test with various projects from testdata/
4. Ensure behavioral compatibility with generated scripts

## Testing Strategy

### Unit Tests
- Each package gets comprehensive unit tests
- Mock external dependencies (GitHub API, filesystem)
- Test error conditions and edge cases
- Use go-cmp for comparison instead of testify

### Integration Tests
- Test full installation flow
- Use testdata configurations
- Compare results with generated installers

### End-to-End Tests
- Create e2e tests with scripts and make tasks
- Test full installation flow on different platforms
- Compare results with generated installers
- Use GitHub Actions for CI testing

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
1. Network errors: Fail immediately (no retry logic)
2. Checksum mismatch: Fail immediately
3. Missing assets: Clear error message
4. Permission errors: Suggest fixes

## Environment Variables

Support the same variables as generated scripts:
- `GITHUB_TOKEN`: Optional for API authentication (helps with rate limits)
- `BINSTALLER_BIN`: Override default bin directory
- Standard proxy variables: `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`

## Future Extensibility

This design enables future `binst x` command by:
1. Reusing verify/archive/install packages
2. Using existing asset and httpclient packages
3. Adding caching layer on top
4. Executing in temporary directory instead of installing

## Open Questions

1. ~Should we support parallel asset downloads for faster installation?~ No, download only one asset (exception: checksum files)
2. ~How should we handle GitHub API rate limits?~ Support optional GITHUB_TOKEN environment variable
3. ~Should dry-run mode make network requests?~ Yes, validate URLs/versions but skip actual installation

## References

- Issue #141: Original feature request
- `internal/shell/template.tmpl.sh`: Shell script template
- `pkg/asset/filename.go`: Asset name generation
- `testdata/`: Example configurations for testing
