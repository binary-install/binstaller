# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**binstaller** (binst) is a modern binary installer generator that creates secure, reproducible installation scripts for static binaries distributed via GitHub releases. It generates POSIX-compliant shell scripts that can download, verify, and install binaries from GitHub releases.

## Key Development Commands

### Building and Testing
```bash
# Build the binary
make build

# Run all tests with race detection
make test

# Run linting
make lint

# Format code
make fmt

# Run full integration test suite
make test-integration

# Generate test configurations and installers
make test-gen-configs
make test-gen-installers

# Run tests with coverage
make cover
```

### Common Development Workflows

1. **Adding a new data source adapter**:
   - Create new adapter in `pkg/datasource/`
   - Implement the adapter interface to convert to `spec.Config`
   - Add tests in the same package
   - Register in `cmd/binst/init.go`

2. **Modifying shell script generation**:
   - Templates are in `internal/shell/`
   - Hash functions are embedded from `internal/shell/hash_*.sh`
   - Test generated scripts with `make test-run-installers`

3. **Working with checksums**:
   - Checksum logic is in `pkg/checksums/`
   - Embedded checksums use `cmd/binst/embed_checksums.go`
   - Always test with `make test-integration` after changes

## Architecture and Code Structure

### Command Pattern
The CLI uses Cobra with three main commands:
- `init`: Initializes config from various sources (GitHub, GoReleaser, Aqua)
- `gen`: Generates installer scripts from config
- `embed-checksums`: Embeds checksums into config for offline verification

### Data Flow
```
Source (GitHub/GoReleaser/Aqua) → Adapter → spec.Config → Template → Shell Script
```

### Key Packages
- `pkg/spec/`: Configuration specification and types
- `pkg/datasource/`: Adapters for different config sources
- `pkg/checksums/`: Checksum calculation and verification
- `internal/shell/`: Shell script templates and utilities

### Security Model
The project emphasizes security through:
- Mandatory HTTPS downloads
- SHA256 checksum verification
- Optional GitHub attestation verification
- Embedded checksums for offline verification
- Configurable security policies (deny self-hosted runners, require attestation)

## Testing Guidelines

1. **Unit Tests**: Standard Go testing, aim for 80%+ coverage
2. **Integration Tests**: Use `make test-integration` to test full workflows
3. **Shell Script Tests**: Generated scripts are tested with `make test-run-installers`
4. **Parallel Testing**: Tests use `rush` for parallel execution

## Important Implementation Notes

1. **Error Handling**: Always wrap errors with context, avoid panicking
2. **Platform Support**: Test changes across Linux, macOS, and Windows
3. **Shell Compatibility**: Generated scripts must be POSIX-compliant
4. **Template Variables**: Support ${NAME}, ${VERSION}, ${OS}, ${ARCH}, ${EXT}
5. **GitHub API**: Use official GitHub API client, handle rate limiting

## Configuration Schema

The `.binstaller.yml` format follows schema v1:
```yaml
schema: v1
name: project-name
url: https://github.com/owner/repo
description: Project description
systems:
  - os: linux
    arch: amd64
    url: https://...
    sha256: ...
checksums:
  - name: checksums.txt
    url: https://...
```

## Debugging Tips

1. Use `-v` flag for verbose output during `binst` commands
2. Generated scripts have debug mode when `TAR_VERBOSE=1`
3. Check `testdata/` for example configurations
4. Integration tests log to `test/integration_test.log`