---
title: "goinstaller Documentation"
date: "2025-04-17"
author: "haya14busa"
version: "0.1.0"
status: "draft"
---

# goinstaller

> A streamlined installer for Go binaries with enhanced security features

## Overview

This project is a fork of [GoDownloader](https://github.com/goreleaser/godownloader), which was originally created as a companion tool to [GoReleaser](https://github.com/goreleaser/goreleaser). GoDownloader generates shell scripts that can download the right package and version of Go binaries released with GoReleaser, making it easy for users to install Go applications with a simple curl | bash command.

The original project has been archived since January 2022, but the core functionality remains valuable. goinstaller streamlines the codebase by removing unnecessary features, enhances security with GitHub attestation verification, and focuses on maintainability and stability.

## Key Features

- **Generate Installation Scripts**: Create shell scripts for downloading and installing Go binaries from GitHub releases
- **GoReleaser Integration**: Parse GoReleaser configuration files to understand the structure of releases
- **Cross-Platform Support**: Support various operating systems and architectures
- **Checksum Verification**: Verify checksums of downloaded binaries
- **GitHub Attestation Verification**: Verify GitHub attestations to enhance security
- **Reproducible Builds**: Generate reproducible scripts with source traceability and commit-specific builds

## Getting Started

### Installation

```bash
# Install the latest version
go install github.com/binary-install/binstaller@latest

# Or install a specific version
go install github.com/binary-install/binstaller@v0.1.0
```

### Basic Usage

```bash
# Generate an installation script for a GitHub repository
goinstaller --repo=owner/repo > install.sh

# Generate an installation script from a local GoReleaser config
goinstaller --file=.goreleaser.yml > install.sh

# Generate an installation script with attestation verification
goinstaller --repo=owner/repo --require-attestation > install.sh

# Generate an installation script from a specific commit
goinstaller --repo=owner/repo --commit=abc123 > install.sh

# Generate an installation script without source information
goinstaller --repo=owner/repo --skip-source-info > install.sh
```

### Example

For a project using GoReleaser, you can generate an installation script and add it to your repository:

```bash
# Generate the installation script
goinstaller --repo=your-username/your-project > install.sh

# Make it executable
chmod +x install.sh

# Add it to your repository
git add install.sh
git commit -m "Add installation script"
git push
```

Then, users can install your application with:

```bash
curl -sfL https://raw.githubusercontent.com/your-username/your-project/main/install.sh | sh
```

## Documentation

For more detailed documentation, see:

- [Design Overview](design/overview.md): High-level design of the project
- [Attestation Verification](design/attestation.md): Details on the GitHub attestation verification feature
- [Reproducible Builds](reproducible-builds.md): Information on reproducible builds and source traceability
- [Usage Guide](usage.md): Comprehensive guide on using the tool

## Comparison with Original GoDownloader

goinstaller differs from the original GoDownloader in several ways:

| Feature | Original GoDownloader | goinstaller |
|---------|----------------------|------------|
| GoReleaser YAML Parsing | ✅ | ✅ |
| Shell Script Generation | ✅ | ✅ |
| Checksum Verification | ✅ | ✅ |
| Equinox.io Support | ✅ | ❌ |
| Raw GitHub Releases | ✅ | ❌ |
| Tree Walking | ✅ | ❌ |
| GitHub Attestation Verification | ❌ | ✅ |
| Reproducible Builds | ❌ | ✅ |
| Commit-Specific Generation | ❌ | ✅ |

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE.md) file for details.
