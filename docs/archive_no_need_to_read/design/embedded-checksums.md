# Embedded Checksums in Binstaller Scripts

## Overview

This document outlines the design for embedding checksums directly within installer scripts generated by binstaller and automatically generating these checksums. This feature enhances security and reliability by allowing offline verification of downloaded binaries without requiring a separate network call to fetch checksum files.

## Goals

- Enable offline verification of downloaded artifacts
- Reduce network dependencies during installation
- Improve installation reliability in environments with restricted network access
- Make it easier to validate binaries when checksums are already known
- Automate the generation of embedded checksums to simplify configuration maintenance

## Design

### Extension to the InstallSpec Schema

The current `ChecksumConfig` structure already supports `embedded_checksums`, but this feature will enhance its usage:

```yaml
checksums:
  template: ${NAME}-${VERSION}-checksums.txt
  algorithm: sha256
  embedded_checksums:
    "v1.2.3":  # Version string as key
      - filename: example-1.2.3-linux-amd64.tar.gz
        hash: abc123...
      - filename: example-1.2.3-darwin-amd64.tar.gz
        hash: def456...
```

### Script Generation Changes

When generating the installer script, the system will:

1. Include all embedded checksums in the script as constants
2. Add logic to check for embedded checksums before attempting to download the checksum file
3. Use the embedded checksum when available, falling back to downloading the checksum file only when necessary

### Verification Flow

The following verification flow will be implemented:

1. Determine if there's an embedded checksum for the current version and asset
2. If an embedded checksum exists:
   - Extract and use it directly for verification
   - Skip downloading the checksum file
3. If no embedded checksum exists:
   - Fall back to the current behavior of downloading the checksum file
   - Verify the asset using the downloaded checksum

### Shell Script Implementation

The following changes will be made to the shell script template:

1. Add a new section to define embedded checksums as constants
2. Modify the `execute()` function to check for embedded checksums before downloading
3. Add a new function to extract the appropriate embedded checksum

## Implementation Details

### Embedded Checksum Section in Template

Add a new section to the shell script template to define embedded checksums:

```sh
# --- Embedded Checksums (Format: VERSION:FILENAME:HASH) ---
EMBEDDED_CHECKSUMS="
{{- range $version, $checksums := .Checksums.EmbeddedChecksums }}
{{- range $checksum := $checksums }}
{{ $version }}:{{ $checksum.Filename }}:{{ $checksum.Hash }}
{{- end }}
{{- end }}
"
```

### Function to Find Embedded Checksum

Add a new shell function to extract an embedded checksum for a given version and filename:

```sh
find_embedded_checksum() {
  version="$1"
  filename="$2"
  if [ -z "$EMBEDDED_CHECKSUMS" ]; then
    return 1
  fi

  echo "$EMBEDDED_CHECKSUMS" | grep -E "^${version}:${filename}:" | cut -d':' -f3
}
```

### Modify Verification Logic

Update the verification logic in the `execute()` function:

```sh
# Try to find embedded checksum first
EMBEDDED_HASH=$(find_embedded_checksum "$VERSION" "$ASSET_FILENAME")

if [ -n "$EMBEDDED_HASH" ]; then
  log_info "Using embedded checksum for verification"

  # Verify using embedded hash
  got=$(hash_sha256 "${TMPDIR}/${ASSET_FILENAME}")
  if [ "$got" != "$EMBEDDED_HASH" ]; then
    log_crit "Checksum verification failed for ${ASSET_FILENAME}"
    log_crit "Expected: ${EMBEDDED_HASH}"
    log_crit "Got: ${got}"
    return 1
  fi
  log_info "Checksum verification successful"
elif [ -n "$CHECKSUM_URL" ]; then
  # Fall back to downloading checksum file
  log_info "Downloading checksums from ${CHECKSUM_URL}"
  http_download "${TMPDIR}/${CHECKSUM_FILENAME}" "${CHECKSUM_URL}"
  log_info "Verifying checksum ..."
  hash_verify "${TMPDIR}/${ASSET_FILENAME}" "${TMPDIR}/${CHECKSUM_FILENAME}"
else
  log_info "No checksum found, skipping verification."
fi
```

## Automatic Checksum Generation

### New Command: `embed-checksums`

Add a new command to binstaller that automates the generation of embedded checksums:

```
binst embed-checksums [--version VERSION] [--output CONFIG_OUTPUT] [--mode MODE] CONFIG_FILE
```

Options:
- `--version`: The specific version to embed checksums for (default: latest release)
- `--output`: Where to write the updated config (default: overwrite input file)
- `--mode`: How to acquire checksums (options: `download`, `checksum-file`, `calculate`)

### Modes of Operation

The `embed-checksums` command will support three modes:

1. **download**: Download the checksum file from the GitHub release and extract checksums
   ```
   binst embed-checksums --mode download --version v1.2.3 example.binstaller.yml
   ```

2. **checksum-file**: Use a local checksum file to extract checksums
   ```
   binst embed-checksums --mode checksum-file --version v1.2.3 --file checksums.txt example.binstaller.yml
   ```

3. **calculate**: Download the release assets and calculate checksums directly
   ```
   binst embed-checksums --mode calculate --version v1.2.3 example.binstaller.yml
   ```

### Implementation Structure

The implementation will follow these steps:

1. Parse the InstallSpec from the input file
2. Resolve the target version (latest or specified)
3. Based on the mode:
   - `download`: Fetch the checksum file from GitHub releases
   - `checksum-file`: Parse the provided checksum file
   - `calculate`: Fetch each asset for the target platforms, calculate checksums
4. Update the InstallSpec with the new embedded checksums
5. Write the updated InstallSpec to the output file

### Asset Resolution for Calculation Mode

In `calculate` mode, the tool will:

1. Use the InstallSpec to determine all possible combinations of OS/ARCH
2. For each combination:
   - Generate the expected asset filename
   - Download the asset from GitHub releases
   - Calculate its checksum
   - Add the checksum to the embedded_checksums map
3. Handle platforms in parallel to improve performance

### Code Structure

```go
// cmd/binst/embed_checksums.go
package main

import (
    "github.com/binary-install/binstaller/pkg/checksums"
    "github.com/spf13/cobra"
)

var embedChecksumsCmd = &cobra.Command{
    Use:   "embed-checksums",
    Short: "Embed checksums for release assets into a binstaller configuration",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Implementation
    },
}

func init() {
    rootCmd.AddCommand(embedChecksumsCmd)
    // Add flags for version, output, mode, etc.
}

// pkg/checksums/checksums.go
package checksums

// EmbedMode represents the checksum acquisition mode
type EmbedMode string

const (
    EmbedModeDownload     EmbedMode = "download"
    EmbedModeChecksumFile EmbedMode = "checksum-file"
    EmbedModeCalculate    EmbedMode = "calculate"
)

// Embedder manages the process of embedding checksums
type Embedder struct {
    Mode      EmbedMode
    Version   string
    InputFile string
    OutputFile string
    ChecksumFile string
}

// Embed performs the checksum embedding process
func (e *Embedder) Embed() error {
    // Implementation
}

// downloadAndParseChecksumFile downloads a checksum file and extracts checksums
func downloadAndParseChecksumFile(repo, version, checksumTemplate string) (map[string]string, error) {
    // Implementation
}

// calculateChecksums downloads assets and calculates checksums
func calculateChecksums(spec *spec.InstallSpec, version string) (map[string]string, error) {
    // Implementation
}
```

## Backward Compatibility

This design maintains full backward compatibility:

1. Existing InstallSpec configurations without embedded checksums will continue to work as before
2. When no embedded checksums are defined, the script will still attempt to download checksum files
3. The only behavior change is to prefer embedded checksums when available

## Security Considerations

1. Embedded checksums provide the same level of security as downloaded checksums since they both verify the integrity of the downloaded asset
2. The embedded checksums will be present in the installer script, making them as transparent as downloaded checksum files
3. Users should only add checksums they have verified themselves or that come from trusted sources
4. The `calculate` mode introduces a potential security risk since it trusts the checksums it calculates at that point in time. Users should verify these checksums match the official ones

## Example Usage

### Basic Configuration with Embedded Checksums

```yaml
name: example-tool
repo: owner/example-tool
asset:
  template: ${NAME}-${VERSION}-${OS}-${ARCH}${EXT}
  default_extension: .tar.gz
checksums:
  template: ${NAME}-${VERSION}-checksums.txt
  algorithm: sha256
  embedded_checksums:
    "1.2.3":
      - filename: example-tool-1.2.3-linux-amd64.tar.gz
        hash: abc123def456...
      - filename: example-tool-1.2.3-darwin-amd64.tar.gz
        hash: 789abc...
```

### Auto-generating Embedded Checksums

```bash
# Add checksums for the latest release by downloading the checksum file
binst embed-checksums --mode download example.binstaller.yml

# Add checksums for a specific version by calculating them from the assets
binst embed-checksums --mode calculate --version v1.2.3 example.binstaller.yml

# Add checksums from a local checksum file
binst embed-checksums --mode checksum-file --file checksums.txt --version v1.2.3 example.binstaller.yml

# Generate checksums for all specified platforms
binst embed-checksums --mode calculate --all-platforms example.binstaller.yml
```

## Implementation Plan

1. Update the `shell/script.go` to modify the template data structure
2. Add new shell functions to handle embedded checksum verification
3. Update the template to include embedded checksums and modified verification logic
4. Create the new `embed-checksums` command and its implementation
5. Implement the different checksum acquisition modes
6. Add test cases to ensure proper behavior with and without embedded checksums
7. Update documentation to explain the new features
