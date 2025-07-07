# Security Policy

## Reporting Security Vulnerabilities

We take the security of binstaller seriously. If you believe you have found a security vulnerability in this repository, please report it to us as described below.

### How to Report a Security Vulnerability

Please report security vulnerabilities using one of the following methods:
- **For sensitive security issues**: Report privately through [GitHub Security Advisories](https://github.com/binary-install/binstaller/security/advisories/new)
- **For non-sensitive security issues**: Open a public issue on the repository

### What to Include in Your Report

Please include the following information in your security report:
- Type of issue (e.g., arbitrary code execution, path traversal, command injection, etc.)
- Full paths of source file(s) related to the manifestation of the issue
- The location of the affected source code (tag/branch/commit or direct URL)
- Any special configuration required to reproduce the issue
- Step-by-step instructions to reproduce the issue
- Proof-of-concept or exploit code (if possible)
- Impact of the issue, including how an attacker might exploit the issue

## Security Considerations for binstaller

Given that binstaller generates installation scripts that download and execute binaries, please pay particular attention to:
- **Checksum verification bypass**: Issues that could allow installation of unverified binaries
- **Download source manipulation**: Vulnerabilities that could redirect downloads to malicious sources
- **Shell injection**: Issues in generated scripts that could lead to arbitrary command execution
- **Path traversal**: Vulnerabilities that could install binaries outside intended directories
- **Privilege escalation**: Issues that could lead to unintended privilege escalation during installation

## Supply Chain Security

This project uses [actionutils/trusted-go-releaser](https://github.com/actionutils/trusted-go-releaser) for releases, which is designed to comply with [SLSA Level 3](https://slsa.dev/spec/v1.0/levels#build-l3) requirements. This includes:
- Provenance generation for all release artifacts
- Signed attestations using Sigstore
- Reproducible builds
- Isolated build environments

If you discover any issues with our supply chain security or SLSA compliance, please report them following the same guidelines above.

## Response

We will respond to security reports on a best-effort basis. As this is an open-source project maintained by volunteers, we cannot guarantee specific response times, but we take security issues seriously and will address them as quickly as our resources allow.

## Disclosure Policy

We follow the principle of coordinated disclosure. We appreciate your patience as we work to ensure that vulnerabilities are properly addressed before public disclosure.

## Security Best Practices for Users

When using binstaller:
1. Always verify the checksums of downloaded binaries
2. Review generated installation scripts before execution
3. Use HTTPS URLs for binary downloads
4. Keep your binstaller configuration files in version control
5. Regularly update to the latest version of binstaller