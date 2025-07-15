# Embedded Checksums

Binstaller supports embedding checksums directly in generated installer scripts for enhanced security and offline verification.

## Overview

When distributing binary installers, it's important to verify the integrity of downloaded files to ensure they haven't been tampered with or corrupted. Binstaller provides two methods for this verification:

1. **Remote checksum verification**: Downloads a checksum file from the GitHub release to verify the downloaded binary.
2. **Embedded checksum verification**: Embeds checksums directly in the installer script itself, enabling offline verification.

## Benefits of Embedded Checksums

- **Offline verification**: No need to download a separate checksum file
- **Reduced network dependencies**: More reliable installation in environments with limited connectivity
- **Enhanced security**: Verification still occurs even when checksum files are unavailable

## Configuration

Embedded checksums are configured in the `.binstaller.yml` file under the `checksums` section:

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

## Verification Process

When an installer script is run, it follows this process:

1. First, it checks if an embedded checksum exists for the current version and asset filename
2. If an embedded checksum is found, it uses that for verification without downloading the checksum file
3. If no embedded checksum exists, it falls back to downloading the checksum file from the GitHub release
4. If neither option is available, verification is skipped

## Generating Embedded Checksums

Binstaller provides a command to automatically generate and embed checksums:

```bash
binst embed-checksums [options] [config-file]
```

### Options

- `--version, -v`: The version to embed checksums for (default: latest)
- `--output, -o`: Output path for the updated config (default: overwrite input file)
- `--mode, -m`: How to acquire checksums:
  - `download`: Download checksum file from GitHub release (default)
  - `checksum-file`: Parse a local checksum file
  - `calculate`: Download assets and calculate checksums directly
- `--file, -f`: Path to local checksum file (required for `checksum-file` mode)
- `--all-platforms`: Generate checksums for all platforms in `supported_platforms`
  (only applicable for `calculate` mode)

### Examples

```bash
# Add checksums for latest release by downloading the checksum file
binst embed-checksums --mode download example.binstaller.yml

# Add checksums for v1.2.3 by calculating them from the assets
binst embed-checksums --mode calculate --version v1.2.3 example.binstaller.yml

# Add checksums from a local checksum file
binst embed-checksums --mode checksum-file --file checksums.txt --version v1.2.3 example.binstaller.yml

# Generate checksums for all supported platforms
binst embed-checksums --mode calculate --all-platforms example.binstaller.yml
```

## Security Considerations

1. Always verify embedded checksums match the official checksums published by the tool developer
2. When using the `calculate` mode, remember that it trusts the checksums it calculates at that point in time
3. When adding checksums manually to configuration files, ensure they come from trusted sources

## Future Improvements

### Advanced Verification at Calculation Time

In future versions, binstaller plans to enhance the checksum calculation process with advanced verification mechanisms:

1. **Supply Chain Verification**: When calculating checksums in `calculate` mode, binstaller could also verify:
   - GitHub attestations using `gh attestation verify`
   - Signatures using `cosign verify`
   - SLSA provenance using `slsa-verifier`

2. **Pre-verification Benefits**:
   - Moves verification from installation time to build time
   - Allows users to trust just the installer script itself
   - Enables fully offline installations with high security guarantees
   - Reduces complexity for end users who may not have verification tools

3. **Metadata Embedding**:
   - Along with checksums, verification metadata could be embedded
   - Proof that verification was performed during checksum calculation
   - Record of which verification methods were used

This approach would provide a more streamlined and secure experience, as binstaller would pre-verify the assets during checksum calculation rather than requiring users to verify them at installation time.
