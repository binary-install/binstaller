# Binstaller Configuration Schema

This directory contains the TypeSpec definition and JSON Schema for binstaller configuration files.

## Overview

The binstaller configuration uses a YAML format defined by TypeSpec, which generates a JSON Schema for validation and documentation. This approach ensures:

- Type-safe configuration
- Excellent IDE support with auto-completion
- Clear documentation with descriptions for all fields
- Validation capabilities

## Files

- `main.tsp` - TypeSpec definition of the configuration schema
- `output/@typespec/json-schema/InstallSpec.json` - Generated JSON Schema with all definitions inline
- `tspconfig.yaml` - TypeSpec compiler configuration
- `package.json` - NPM scripts for schema generation

## Usage

### Generate JSON Schema

```bash
cd schema
npm install
npm run gen:schema
```

This will compile the TypeSpec definition to a complete JSON Schema with all definitions inline at `output/@typespec/json-schema/InstallSpec.json`.

### Validate Configuration Files

You can use the generated JSON Schema to validate configuration files:

```bash
# Using ajv-cli
npx ajv validate -s output/@typespec/json-schema/InstallSpec.json -d ../.config/binstaller.yml
```

### IDE Support

Many IDEs support JSON Schema validation for YAML files. Add this to your `.config/binstaller.yml`:

```yaml
# yaml-language-server: $schema=../schema/output/@typespec/json-schema/InstallSpec.json
schema: v1
repo: owner/repo
# ... rest of config
```

## Configuration Format

The configuration file (`.config/binstaller.yml`) defines how to download and install binaries from GitHub releases.

### Basic Example

```yaml
schema: v1
repo: owner/repo
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
```

### Full Example

```yaml
schema: v1
name: myapp
repo: owner/repo
default_version: latest
default_bin_dir: "${HOME}/.local/bin"

asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"
  default_extension: .tar.gz
  binaries:
    - name: myapp
      path: myapp
  naming_convention:
    os: lowercase
    arch: lowercase
  arch_emulation:
    rosetta2: true
  rules:
    - when:
        os: windows
      ext: .zip
    - when:
        os: darwin
        arch: arm64
      template: "${NAME}_${VERSION}_${OS}_${ARCH}_signed${EXT}"

checksums:
  algorithm: sha256
  template: "${NAME}_${VERSION}_checksums.txt"
  embedded_checksums:
    "v1.0.0":
      - filename: myapp_v1.0.0_linux_amd64.tar.gz
        hash: abc123...
      - filename: myapp_v1.0.0_darwin_amd64.tar.gz
        hash: def456...

attestation:
  enabled: true
  require: false
  verify_flags: "--cert-identity=https://github.com/owner/repo/.github/workflows/release.yml@refs/tags/v1.0.0"

unpack:
  strip_components: 1

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

## Schema Development

To modify the schema:

1. Edit `main.tsp`
2. Run `npm run gen:schema` to regenerate the JSON Schema
3. Test with sample configuration files

The TypeSpec language provides excellent type safety and documentation capabilities. See the [TypeSpec documentation](https://typespec.io/) for more information.

## Go Code Generation

You can generate Go structs from the JSON Schema using a forked version of quicktype that properly handles `unevaluatedProperties`:

```bash
npm run gen:go
```

This will:
1. Clone/update the forked quicktype with `unevaluatedProperties` support
2. Build quicktype if needed (cached for subsequent runs)
3. Generate Go structs with proper types (including `map[string][]EmbeddedChecksum` for embedded checksums)

The generated Go code uses JSON tags, which are compatible with the YAML library used in binstaller (gopkg.in/yaml.v3 supports JSON tags). The generated structs can be used directly in the binstaller codebase.