---
title: "InstallSpec‑Driven Installer Architecture"
date: "2025-04-19"
author: "binstaller Team"
version: "0.2.0"
status: "draft"
---

# Generic Config‑Driven Installer Architecture

> 📄 **Document series overview**
> This file lays out the *architecture* of a generic, config‑driven installer
> pipeline.  The concrete specification of the on‑disk schema referred to in
> this document lives in **[InstallSpec v1](install-spec-v1.md)** which should be
> read together with this file.

> 🛈 **Naming note** — The prototype implementation was called *goinstaller* but
> the scope has since expanded beyond Go projects.  To avoid confusion and to
> reflect its language‑agnostic mission the tool will be renamed **binstaller**
> (CLI binary `binst`).  Throughout this document we use the new name; any
> lingering references to *goinstaller* denote historic context only.

## 1. Background & Motivation
Today, the pre‑rename **goinstaller** tool only supports reading a GoReleaser YAML (`.goreleaser.yml`) to generate a shell installer script.  As we transition to the **binstaller** name (removing the "Go" coupling) this document generalises the design to be language‑agnostic.
Many projects either do not use GoReleaser or have custom asset naming conventions and release workflows.
We need a pluggable, data‑source‑agnostic architecture that—given minimal inputs via CLI flags, manual config, GitHub API, etc.—can generate the same installer logic without being tightly coupled to GoReleaser.

## 2. Goals
- Extract a **pure InstallSpec** that fully describes name, version placeholder, supported platforms, asset templates, checksum/verification settings, etc.
- Define a **SourceAdapter** interface to populate that InstallSpec from any origin.
- Maintain a single **ScriptGenerator** component that transforms InstallSpec → POSIX `sh` installer scripts.
- Preserve existing GoReleaser flow as one SourceAdapter implementation.

## 3. High‑Level Architecture

```text
┌──────────────┐    ┌───────────────┐    ┌───────────────────┐
│ SourceAdapter│ ─> │  InstallSpec │ ─> │  ScriptBuilder   │ ─> install.sh
└──────────────┘    └───────────────┘    └───────────────────┘
```

- **SourceAdapter**: interface `Detect(ctx, in DetectInput) (InstallSpec, error)`
  - goreleaserAdapter (existing `.goreleaser.yml`)
  - githubProbeAdapter (GitHub Releases API, asset inspection)
  - flagsAdapter (CLI flags for name, patterns, etc.)
  - fileAdapter (user‑supplied `.binstaller.yml`)
- **InstallSpec**: Go struct/YAML schema that holds:
  - `name`, `repo`, `version` placeholder
  - per‑platform archives (`os`, `arch`, `asset` template, `bin`)
  - checksum definition (file, algorithm, embedded checksums)
  - attestation verification settings
  - unpack options (strip components)
  - For detailed schema definition, see [InstallSpec v1](install-spec-v1.md)
- **ScriptBuilder**: generates installer scripts
  - powered by Go `text/template`
  - outputs POSIX `sh` installer scripts
  - injects download, checksum verify, attestation, retry, flags

## 4. Two-Step Workflow

The new architecture introduces a two-step workflow to simplify the process:

1. **Config Generation**: First generate the InstallSpec‑compatible config
   ```bash
   binst init --source [goreleaser|github|cli] [options]
   ```

2. **Script Generation**: Then generate the installer script from that config
   ```bash
   binst gen --config .binstaller.yml [options]
   ```

Additionally, a utility command is provided to embed checksums into an existing config:

```bash
binst embed --config .binstaller.yml --checksum-file SHA256SUMS
```

This separation allows for:
- Better validation and inspection of the config before script generation
- Reuse of configs across multiple script generations
- Simplified script generation logic
- Easier testing and debugging
- Ability to add checksums to existing configs from external checksum files

## 5. Command‑line Interface (Design Draft)

This section consolidates the CLI discussion for the **binstaller** program (binary name `binst`).  The goal is to keep the surface small but expressive.

### 5.1 Top‑level grammar

```
binst <command> [global‑flags] [command‑flags] [args]
```

Canonical commands (only four):

| Command | Purpose |
|---------|---------|
| `init`     | Create an InstallSpec from various sources (0 → 1) |
| `gen`      | Generate an installer script from an InstallSpec |
| `embed`    | Embed checksums or extra metadata into an InstallSpec |
| `install`  | One‑shot install (internally runs *init* + *gen* and executes) |

`embed` may be invoked via aliases (`embed‑hash`, `embed‑checksum`, `hash`).  Legacy longer names such as `init‑config`, `generate` are provided as hidden aliases.

### 5.2 Global flags (available to every command)

```
  -c, --config <file>   Path to InstallSpec (default: auto‑detect ./.binstaller.yml)
      --dry-run         Print actions without performing network or FS writes
      --verbose|--debug Increase log verbosity
      --quiet           Suppress progress output
  -y, --yes             Assume "yes" on interactive prompts
      --timeout <dur>   HTTP / process timeout (e.g. 30s, 2m)
```

### 5.3 Command details & flags

#### A) `binst init`

Generate an InstallSpec.

Required flag `--source` (`goreleaser|github|cli|file|…`).  Other important flags:

```
  --file <path>              Path to .goreleaser.yml / other source file
  --repo <owner/repo>        GitHub repository
  --tag <vX.Y.Z>             Release tag / ref
  --asset-pattern <tmpl>     Template for asset file names
  -o, --output <file>        Write spec to file (default: stdout)
```

#### B) `binst gen`

Transforms an InstallSpec into an installer script.

```
  -o, --output <file>   Output path (default: stdout)
```

Typical usage:

```bash
binst gen -c .binstaller.yml > install.sh
```

#### C) `binst embed` (aliases: `embed-hash`, `embed-checksum`, `hash`)

Embed checksums or additional metadata into a spec.

```
  --checksum-file <SHA256SUMS>   Path to checksum file
  --version <vX.Y.Z>             Version being embedded (optional)
  --algo <sha256|sha512>         Hash algorithm (default: sha256)
  -o, --output <file>            If omitted, overwrite original spec
```

#### D) `binst install`

Sugar command that performs *init* → *gen* → *execute* in one go.

Examples:

```bash
# From existing spec
binst install -c .binstaller.yml

# Ad‑hoc install from GitHub release (no spec file on disk)
binst install --repo cli/cli --tag v2.45.0
```

Implementation detail: the generated script is piped to `sh` via a temporary file or stdin.

### 5.4 Cheat‑sheet

```bash
# 1) Generate spec from GoReleaser YAML
binst init --source goreleaser --file .goreleaser.yml -o .binstaller.yml

# 2) Inspect & generate installer script
binst gen -c .binstaller.yml -o install.sh

# 3) Embed checksums
binst embed -c .binstaller.yml --checksum-file SHA256SUMS

# 4) Direct install using a spec
binst install -c .binstaller.yml

# 5) Direct install from GitHub without local spec
binst install --repo cli/cli --tag v2.45.0
```

> The traditional pipeline `binst gen … | sh` continues to work; `binst install` is merely a convenience wrapper.

## 6. Code Layout (proposed)

The project follows a *public API vs internal* split:

```
binstaller/
├── cmd/binst/main.go      # CLI entry‑point (cobra / urfave‑cli, etc.)
│
├── pkg/                   # Public, import‑able Go packages
│   ├── datasource/        # SourceAdapter interfaces + built‑in adapters
│   │   ├── goreleaser.go  # GoReleaser YAML adapter
│   │   └── github.go      # GitHub Releases probing adapter
│   └── spec/              # InstallSpec struct, validation helpers
│
├── internal/              # No compatibility promise
│   ├── shell/             # sh script generator & templates
│   └── checksum/          # crypto helpers (used by embed‑hash)
│
└── tools/                 # cmd/scripts used only at build‑time (cue vet, etc.)
```

Rationale:

* `cmd/binst` reflects the final binary name.
* `pkg/datasource` is exported so that external ecosystems can contribute new adapters without modifying core.
* InstallSpec lives in `pkg/spec` because it is the primary user‑facing type; downstream tools may wish to generate or manipulate specs.
* Script generation is implementation detail → `internal/shell`.
* No standalone `pkg/verify` – checksum logic is small and kept internal.

## 7. Embedded Checksums Benefits

The embedded checksums feature provides several significant advantages:

### 7.1 Performance Improvements
- **Reduced HTTP Requests**: Eliminates the need to download separate checksum files during installation, reducing the number of HTTP requests by at least one per installation.
- **Faster Installations**: Installation completes more quickly, especially on slower networks, as there's no need to wait for additional checksum file downloads.
- **Optimized Verification Flow**: When the installer script itself is verified with attestation, the embedded checksums can be trusted implicitly, allowing the binary verification process to be streamlined or even skipped in certain scenarios, further accelerating the installation process.

### 7.2 Reliability Enhancements
- **Offline Installation Support**: Enables completely offline installations once the installer script and binary are downloaded, as no additional network requests for checksum files are needed.
- **Reduced Network Dependency**: Less susceptible to temporary network issues or checksum file server unavailability.
- **Consistent Verification**: Ensures the same checksums are used for verification regardless of network conditions or changes to remote checksum files.

### 7.3 Security Considerations
- **Pre-verified Integrity**: Checksums can be pre-verified by the script generator using `gh attestation verify` or other secure methods before embedding.
- **Tamper Resistance**: Makes it harder for attackers to substitute malicious checksums, as they would need to modify the installer script itself (which could also be signed or verified).
- **Audit Trail**: Provides a clear record of which checksums were used for verification at the time the installer was generated.
- **Trust Chain**: When the installer script is verified with attestation, the embedded checksums inherit this trust, creating a stronger end-to-end security model.

### 7.4 User Experience
- **Simplified Installation**: Users don't need to worry about checksum file availability or format.
- **Consistent Behavior**: Installation process behaves the same way across different environments and network conditions.
- **Transparent Verification**: Users can inspect the embedded checksums in the installer script before running it.

This feature is particularly valuable for enterprise environments with strict security policies, air-gapped systems, or deployments in regions with unreliable internet connectivity.

## 8. Implementation Roadmap
1. Phase 1: Extract InstallSpec & SourceAdapter interface; migrate current GoReleaser code
2. Phase 2: Implement two-step workflow with init-config and generate-script commands
3. Phase 3: Implement GitHub Probe adapter (API calls, naming heuristics)
4. Phase 4: Add File & Flags adapters
5. Phase 5: Implement `binst embed` command for adding checksums to existing configs
6. Phase 6: Update templates, tests, examples, docs

## 9. Risks & Mitigations
- Naming inference may require pattern flags for edge cases
- GitHub rate limits: support unauthenticated vs token flows
- Template versioning: allow per‑project overrides and locking
- Embedded checksums: ensure proper validation of pre-verified checksums
