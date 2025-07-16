# Binstaller Configuration Schema

This directory contains the TypeSpec definition for binstaller's configuration format and tools to generate JSON Schema and Go types.

## Configuration Format

Binstaller uses YAML configuration files (typically `.config/binstaller.yml`) to define how to download, verify, and install binaries from GitHub releases.

### Quick Start

Here's a minimal configuration:

```yaml
schema: v1
repo: owner/project
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
```

### Complete Example

```yaml
schema: v1
name: mytool
repo: myorg/mytool
default_version: latest
default_bin_dir: ${HOME}/.local/bin

# Define how to construct download URLs
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"
  default_extension: .tar.gz

  # Multiple binaries in the archive
  binaries:
    - name: mytool
      path: mytool
    - name: mytool-helper
      path: bin/mytool-helper

  # Platform-specific rules (applied cumulatively)
  rules:
    # Windows uses .zip files
    - when:
        os: windows
      ext: .zip

    # macOS uses different naming
    - when:
        os: darwin
      os: macOS  # Changes ${OS} to "macOS"

    # macOS also uses .zip
    - when:
        os: darwin
      ext: .zip

    # M1 Macs need signed binaries
    - when:
        os: darwin
        arch: arm64
      template: "${NAME}_${VERSION}_${OS}_${ARCH}_signed${EXT}"

# Security features
checksums:
  algorithm: sha256
  template: "${NAME}_${VERSION}_checksums.txt"

# Archive extraction
unpack:
  strip_components: 1

# Platform restrictions
supported_platforms:
  - os: linux
    arch: amd64
  - os: linux
    arch: arm64
  - os: darwin
    arch: amd64
  - os: darwin
    arch: arm64
  - os: windows
    arch: amd64
```

## Key Concepts

### Template Placeholders

The asset template uses these placeholders:

- `${NAME}` - Binary name (from `name` field or repository name)
- `${VERSION}` - Version to install (without 'v' prefix)
- `${OS}` - Operating system (e.g., 'linux', 'darwin', 'windows')
- `${ARCH}` - Architecture (e.g., 'amd64', 'arm64', '386')
- `${EXT}` - File extension (from `default_extension` or rules)

### Rules System

Rules are evaluated **sequentially** and **all matching rules are applied**:

1. Each rule's `when` condition is checked
2. If all conditions match, the rule's overrides are applied
3. Later rules can override values from earlier rules

Example flow for `darwin/arm64`:
```yaml
rules:
  - when: { os: darwin }
    os: macOS      # ${OS} becomes "macOS"
  - when: { os: darwin }
    ext: .zip      # ${EXT} becomes ".zip"
  - when: { os: darwin, arch: arm64 }
    template: "special_${OS}_${ARCH}${EXT}"  # Uses "macOS" and ".zip" from above
```

### Security Features

#### Checksums
- Download checksum files from releases
- Or embed pre-verified checksums using `binst embed-checksums`

## Common Patterns

### Single Binary with Archive Per Platform

```yaml
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  rules:
    - when: { os: windows }
      ext: .zip
```

### Direct Binary Download (No Archive)

```yaml
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"
  default_extension: ""
  binaries:
    - name: mytool
      path: ${ASSET_FILENAME}  # The downloaded file IS the binary
  rules:
    - when: { os: windows }
      ext: .exe
```

### Multiple Architectures with Emulation

```yaml
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  arch_emulation:
    rosetta2: true  # Use x86_64 on Apple Silicon when available
```

### Custom OS/Arch Names

Some projects use non-standard naming:

```yaml
asset:
  template: "${NAME}-${OS}-${ARCH}"
  rules:
    - when: { os: darwin }
      os: macos
    - when: { arch: amd64 }
      arch: x64
    - when: { arch: 386 }
      arch: x86
```

## Schema Development

The schema is defined using [TypeSpec](https://typespec.io/):

```bash
# Install dependencies
cd schema
npm install

# Generate JSON Schema and Go types
make gen

# Or individually:
make gen-schema  # Generate JSON Schema
make gen-go      # Generate Go types
```

The generation pipeline:
1. TypeSpec → JSON Schema (via TypeSpec compiler)
2. JSON Schema → Go structs (via customized quicktype)

## Files

- `main.tsp` - TypeSpec definition of the configuration schema (source)
- `InstallSpec.json` - Generated JSON Schema with all definitions inline
- `InstallSpec.yaml` - YAML version of the JSON Schema for easier reading
- `../pkg/spec/generated.go` - Generated Go structs
- `tspconfig.yaml` - TypeSpec compiler configuration
- `package.json` - NPM scripts for schema generation
- `gen-go-with-fork.sh` - Script to use forked quicktype with unevaluatedProperties support
- `output/` - TypeSpec compiler output directory (gitignored)

## Validation and Testing

### Configuration Validation

Use `binst check` to validate your configuration and verify assets:

```bash
# Check the default config file
binst check

# Check with a specific version
binst check --version v1.2.3

# Check without verifying GitHub assets (faster)
binst check --check-assets=false

# Ignore specific file patterns
binst check --ignore "\.deb$" --ignore ".*-musl.*"
```

### Testing Generated Installers

Test your installer script with dry-run mode:

```bash
# Generate and test installer without actually installing
binst gen | sh -s -- -n

# Test with a specific version
binst gen | sh -s -- -n v1.2.3

# Test installation to a custom directory
binst gen | sh -s -- -n -b /tmp/test-install
```

### Schema Validation

For direct schema validation:

```bash
# Using any JSON Schema validator
npx ajv validate -s schema/InstallSpec.json -d .config/binstaller.yml
```

### IDE Support

Many IDEs support JSON Schema validation for YAML files. The `binst init` command automatically adds this directive:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/binary-install/binstaller/main/schema/InstallSpec.json
schema: v1
repo: owner/repo
# ... rest of config
```

## Tips and Best Practices

### Workflow Tips

1. **Start with init**: Use `binst init` to generate a base configuration from existing sources:
   ```bash
   # From GitHub releases
   binst init --source=github --repo=junegunn/fzf

   # From GoReleaser config
   binst init --source=goreleaser --repo=owner/repo

   # From Aqua registry
   binst init --source=aqua --repo=junegunn/fzf
   ```

2. **Validate early and often**: Use `binst check` after each configuration change:
   ```bash
   binst check  # Validates config and checks assets exist
   ```

3. **Test incrementally**: Use dry-run mode to test without installing:
   ```bash
   binst gen | sh -s -- -n  # Test the installer script
   ```

### Configuration Tips

4. **Start simple**: Begin with minimal configuration and add complexity as needed
5. **Use rules sparingly**: Only add rules for actual platform differences
6. **Document your choices**: Add comments explaining non-obvious configurations

### Security Tips

7. **Always use checksums**: Configure checksums for security:
   ```yaml
   checksums:
     algorithm: sha256
     template: "${NAME}_${VERSION}_checksums.txt"
   ```

8. **Embed checksums for releases**: For published releases, embed checksums:
   ```bash
   # Download checksum file from GitHub
   binst embed-checksums --version v1.0.0 --mode download

   # Or calculate directly (requires GITHUB_TOKEN)
   export GITHUB_TOKEN=$(gh auth token)
   binst embed-checksums --version v1.0.0 --mode calculate
   ```

### Advanced Usage

9. **Generate version-specific installers**: For CI/CD or specific deployments:
   ```bash
   binst gen --target-version v1.2.3 -o install-v1.2.3.sh
   ```

10. **Use runner mode**: For temporary execution without installation:
    ```bash
    binst gen --type=runner -o run.sh
    ./run.sh -- --help  # Run the binary directly
    ```
