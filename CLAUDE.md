# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**binstaller** (binst) is a binary installer generator that creates secure, reproducible installation scripts for static binaries distributed via GitHub releases.

## Key Commands

```bash
# Build and test
make build
make test
make lint

# Run CI checks (no external API calls, safe for offline use)
make ci

# Run full integration tests (accesses GitHub API)
# Optionally set GITHUB_TOKEN to avoid rate limits:
GITHUB_TOKEN=your_token make test-integration

# Run before committing (for development)
make fmt

# Get comprehensive help for all binst commands
./binst helpful
```

### Testing Notes

- `make ci`: Runs build, tests, linting, generation, and formatting without any external API calls. Safe for offline development.
- `make test-integration`: Runs the full integration test suite which includes:
  - Generating test configurations and installers
  - Accessing GitHub APIs to verify releases and assets
  - Running generated installers
  
  **Important**: Run `make test-integration` before creating commits or PRs when changes might affect generated output (e.g., modifying templates, asset rules, or configuration handling). The command works without GITHUB_TOKEN but setting it helps avoid rate limits.

## Project Structure

- `cmd/binst/main.go` - Entry point
- `cmd/` - CLI commands:
  - `init.go` - Initialize new binstaller configuration
  - `gen.go` - Generate installation scripts
  - `embed_checksums.go` - Embed checksums into scripts
  - `check.go` - Check and validate configurations
  - `helpful.go` - Display comprehensive help
- `pkg/spec/` - Configuration spec (`.config/binstaller.yml` format)
- `pkg/datasource/` - Adapters for GitHub, GoReleaser, Aqua registry
- `pkg/asset/` - Asset filename parsing and platform handling
- `pkg/checksums/` - Checksum calculation and verification
- `internal/shell/` - Shell script templates and generation
- `schema/` - TypeSpec definitions for JSON schema generation
- `testdata/` - Example configurations and generated installers

## Important Notes

1. Default config path is `.config/binstaller.yml`
2. Generated scripts must be POSIX-compliant
3. Run `make test-integration` for major changes
4. Template variables: `${NAME}`, `${VERSION}`, `${OS}`, `${ARCH}`, `${EXT}`