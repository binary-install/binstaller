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

# Run before committing
make fmt
make test-integration
```

## Project Structure

- `cmd/binst/main.go` - Entry point
- `cmd/` - CLI commands (init.go, gen.go, embed_checksums.go)
- `pkg/spec/` - Configuration spec (`.config/binstaller.yml` format)
- `pkg/datasource/` - Adapters for GitHub, GoReleaser, Aqua
- `internal/shell/` - Shell script templates
- `pkg/checksums/` - Checksum handling

## Important Notes

1. Default config path is `.config/binstaller.yml`
2. Generated scripts must be POSIX-compliant
3. Run `make test-integration` for major changes
4. Template variables: `${NAME}`, `${VERSION}`, `${OS}`, `${ARCH}`, `${EXT}`