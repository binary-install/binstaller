// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    installSpec, err := UnmarshalInstallSpec(bytes)
//    bytes, err = installSpec.Marshal()

package spec

import "encoding/json"

func UnmarshalInstallSpec(data []byte) (InstallSpec, error) {
	var r InstallSpec
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *InstallSpec) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Configuration specification for binstaller binary installation.
//
// This is the root configuration that defines how to download, verify,
// and install binaries from GitHub releases.
//
// Minimal example:
// ```yaml
// schema: v1
// repo: owner/project
// asset:
// template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
// ```
//
// Complete example with all features:
// ```yaml
// schema: v1
// name: mytool
// repo: myorg/mytool
// default_version: latest
// default_bin_dir: ${HOME}/.local/bin
//
// # Asset configuration with platform-specific rules
// asset:
// template: "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"
// default_extension: .tar.gz
// binaries:
// - name: mytool
// path: mytool
// - name: mytool-helper
// path: bin/mytool-helper
// rules:
// # Windows gets .zip extension
// - when:
// os: windows
// ext: .zip
// # macOS uses different naming
// - when:
// os: darwin
// os: macOS
// ext: .zip
// # Special handling for M1 Macs
// - when:
// os: darwin
// arch: arm64
// template: "${NAME}_${VERSION}_${OS}_${ARCH}_signed${EXT}"
// naming_convention:
// os: lowercase
// arch_emulation:
// rosetta2: true
//
// # Security features
// checksums:
// algorithm: sha256
// template: "${NAME}_${VERSION}_checksums.txt"
// embedded_checksums:
// "1.0.0":
// - filename: "mytool_1.0.0_linux_amd64.tar.gz"
// hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
//
// # Archive handling
// unpack:
// strip_components: 1
//
// # Platform restrictions
// supported_platforms:
// - os: linux
// arch: amd64
// - os: linux
// arch: arm64
// - os: darwin
// arch: amd64
// - os: darwin
// arch: arm64
// - os: windows
// arch: amd64
// ```
type InstallSpec struct {
	// Asset download configuration
	Asset *Asset `json:"asset,omitempty"`
	// Checksum verification configuration
	Checksums *Checksums `json:"checksums,omitempty"`
	// Default binary installation directory
	DefaultBinDir *string `json:"default_bin_dir,omitempty"`
	// Default version to install
	DefaultVersion *string `json:"default_version,omitempty"`
	// Binary name (defaults to repository name if not specified)
	Name *string `json:"name,omitempty"`
	// GitHub repository in format 'owner/repo'
	Repo *string `json:"repo,omitempty"`
	// Schema version
	Schema *string `json:"schema,omitempty"`
	// List of supported OS/architecture combinations
	SupportedPlatforms []SupportedPlatformElement `json:"supported_platforms,omitempty"`
	// Archive extraction configuration
	Unpack *Unpack `json:"unpack,omitempty"`
}

// Asset download configuration
//
// Configuration for constructing download URLs and asset names.
//
// The asset configuration determines how to build the download URL for each platform.
// It uses a template system with placeholders that are replaced with actual values.
type Asset struct {
	// Architecture emulation configuration
	ArchEmulation *ArchEmulation `json:"arch_emulation,omitempty"`
	// Binary names and their paths within the asset.
	//
	// For archives: Specify the path within the extracted directory.
	//
	// If not specified, defaults to a single binary with:
	// - name: The repository name
	// - path: The repository name
	Binaries []BinaryElement `json:"binaries,omitempty"`
	// Default file extension when not specified in template.
	// This is used when the template contains ${EXT} placeholder.
	// Common values: '.tar.gz', '.zip', '.exe'
	// If not set and template uses ${EXT}, it defaults to empty string.
	DefaultExtension *string `json:"default_extension,omitempty"`
	// Controls the casing of placeholder values
	NamingConvention *NamingConvention `json:"naming_convention,omitempty"`
	// Platform-specific overrides.
	// Rules are evaluated in order, and ALL matching rules are applied cumulatively.
	// Later rules can override values set by earlier rules.
	// Use this to handle special cases for specific OS/arch combinations.
	Rules []RuleElement `json:"rules,omitempty"`
	// Filename template with placeholders.
	//
	// Available placeholders:
	// - ${NAME}: Binary name (from 'name' field or repository name)
	// - ${VERSION}: Version to install (includes 'v' prefix if present in tag)
	// - ${OS}: Operating system (e.g., 'linux', 'darwin', 'windows')
	// - ${ARCH}: Architecture (e.g., 'amd64', 'arm64', '386')
	// - ${EXT}: File extension (from 'default_extension' or rules)
	//
	// Examples:
	// - "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
	// - "${NAME}-${VERSION}-${OS}-${ARCH}${EXT}"
	// - "v${VERSION}/${NAME}_${OS}_${ARCH}.zip"
	Template *string `json:"template,omitempty"`
}

// Architecture emulation configuration
//
// Architecture emulation configuration.
//
// Handles cases where binaries can run on different architectures
// through emulation layers.
//
// Example:
// ```yaml
// arch_emulation:
// rosetta2: true  # Use x86_64 binaries on Apple Silicon Macs
// ```
type ArchEmulation struct {
	// Use amd64 binaries instead of arm64 when Rosetta 2 is available on macOS.
	//
	// Useful when:
	// - arm64 binaries are not available
	// - x86_64 binaries are more stable or feature-complete
	// - You need compatibility with x86_64-only dependencies
	//
	// The installer will detect Rosetta 2 and download amd64 binaries
	// on Apple Silicon Macs when this is enabled.
	Rosetta2 *bool `json:"rosetta2,omitempty"`
}

// Binary name and path configuration.
//
// Defines which binary files to install from the downloaded asset.
// For single binary releases, this is straightforward.
// For releases with multiple binaries, you can specify which ones to install.
//
// Examples:
// - Single binary in archive: {name: "mytool", path: "mytool"}
// - Binary in subdirectory: {name: "mytool", path: "bin/mytool"}
// - Multiple binaries: [{name: "tool1", path: "tool1"}, {name: "tool2", path: "tool2"}]
type BinaryElement struct {
	// Name of the binary to install.
	// This will be the filename created in the installation directory.
	Name *string `json:"name,omitempty"`
	// Path to the binary within the extracted archive.
	//
	// The path relative to the archive root.
	//
	// Examples:
	// - "mytool" - Binary at archive root
	// - "bin/mytool" - Binary in bin subdirectory
	Path *string `json:"path,omitempty"`
}

// Controls the casing of placeholder values
//
// Controls the casing of template placeholders.
//
// Some projects use different casing conventions in their release filenames.
// This provides a simpler alternative to using rules for common cases like
// titlecase OS names.
//
// Example:
// ```yaml
// naming_convention:
// os: titlecase  # "Darwin" instead of "darwin"
// arch: lowercase  # "amd64" (default)
// ```
type NamingConvention struct {
	// Casing for ${ARCH} placeholder.
	//
	// Currently only supports lowercase.
	// Values like "amd64", "arm64", "386".
	Arch *NamingConventionArch `json:"arch,omitempty"`
	// Casing for ${OS} placeholder.
	//
	// - lowercase (default): "linux", "darwin", "windows"
	// - titlecase: "Linux", "Darwin", "Windows"
	OS *NamingConventionOS `json:"os,omitempty"`
}

// Platform-specific asset configuration override.
//
// Rules are evaluated in order, and ALL matching rules are applied sequentially.
// Each matching rule's overrides are applied cumulatively, with later rules
// overriding values set by earlier rules.
//
// A rule matches when all specified conditions in 'when' are met.
//
// Example:
// ```yaml
// rules:
// # Rule 1: Windows gets .zip extension
// - when:
// os: windows
// ext: .zip
//
// # Rule 2: Darwin is renamed to macOS
// - when:
// os: darwin
// os: macOS
//
// # Rule 3: Darwin also gets .zip (cumulative with rule 2)
// - when:
// os: darwin
// ext: .zip
//
// # Rule 4: Darwin arm64 gets special template (cumulative with rules 2 & 3)
// - when:
// os: darwin
// arch: arm64
// template: "${NAME}_${VERSION}_${OS}_${ARCH}_signed${EXT}"
// ```
//
// In this example, for darwin/arm64:
// - Rule 2 changes OS to "macOS"
// - Rule 3 changes extension to ".zip"
// - Rule 4 changes the entire template
// - Final result uses all these changes
type RuleElement struct {
	// Override architecture value for matching platforms.
	// This changes the ${ARCH} placeholder value in the template.
	// Useful when the release uses different arch naming (e.g., 'x64' instead of 'amd64').
	Arch *string `json:"arch,omitempty"`
	// Override binary configuration for matching platforms.
	// This replaces the default binary configuration when the rule matches.
	// Useful when different platforms have different binary names or paths.
	Binaries []BinaryElement `json:"binaries,omitempty"`
	// Override extension for matching platforms.
	// This changes the ${EXT} placeholder value in the template.
	// Common values: '.tar.gz', '.zip', '.exe'
	EXT *string `json:"ext,omitempty"`
	// Override OS value for matching platforms.
	// This changes the ${OS} placeholder value in the template.
	// Useful when the release uses different OS naming (e.g., 'mac' instead of 'darwin').
	OS *string `json:"os,omitempty"`
	// Override template for matching platforms.
	// This completely replaces the default template when the rule matches.
	Template *string `json:"template,omitempty"`
	// Condition for applying this rule.
	// All specified fields must match for the rule to apply.
	// If a field is not specified, it matches any value.
	When *When `json:"when,omitempty"`
}

// Condition for applying this rule.
// All specified fields must match for the rule to apply.
// If a field is not specified, it matches any value.
//
// Condition for matching specific platforms in rules.
//
// Used in the 'when' clause of asset rules to specify which
// platforms the rule should apply to. Note that matching uses
// the original OS and architecture values, not any overridden
// values from previous rules.
//
// Example:
// ```yaml
// when:
// os: darwin
// arch: arm64
// ```
type When struct {
	// Match specific architecture.
	//
	// If specified, the rule only applies when the runtime architecture matches.
	// If omitted, the rule matches any architecture.
	//
	// Can be any string value to support custom architecture identifiers.
	Arch *string `json:"arch,omitempty"`
	// Match specific operating system.
	//
	// If specified, the rule only applies when the runtime OS matches.
	// If omitted, the rule matches any OS.
	//
	// Can be any string value to support custom OS identifiers.
	OS *string `json:"os,omitempty"`
}

// Checksum verification configuration
//
// Checksum verification configuration.
//
// Binstaller verifies downloaded files using checksums to ensure integrity.
// It can either download checksum files from the release or use pre-verified
// checksums embedded in the configuration.
//
// Example:
// ```yaml
// checksums:
// algorithm: sha256
// template: "${NAME}_${VERSION}_checksums.txt"
// embedded_checksums:
// "1.0.0":
// - filename: "mytool_1.0.0_linux_amd64.tar.gz"
// hash: "abc123..."
// - filename: "mytool_1.0.0_darwin_amd64.tar.gz"
// hash: "def456..."
// ```
type Checksums struct {
	// Hash algorithm used for checksums.
	// Must match the algorithm used by the project's checksum files.
	// Most projects use sha256.
	Algorithm *Algorithm `json:"algorithm,omitempty"`
	// Pre-verified checksums organized by version.
	//
	// Use 'binst embed-checksums' command to automatically populate this.
	// The key is the version string (includes 'v' prefix if present in tag).
	// The value is an array of filename/hash pairs.
	//
	// This allows offline installation and protects against
	// compromised checksum files.
	EmbeddedChecksums map[string][]EmbeddedChecksumElement `json:"embedded_checksums,omitempty"`
	// Template for checksum filename.
	//
	// If specified, binstaller will download this file to verify checksums.
	// Uses the same placeholders as asset templates.
	//
	// Common patterns:
	// - "${NAME}_${VERSION}_checksums.txt"
	// - "checksums.txt"
	// - "${NAME}-${VERSION}-SHA256SUMS"
	//
	// Leave empty to rely only on embedded checksums.
	Template *string `json:"template,omitempty"`
}

// Pre-verified checksum for a specific asset.
//
// Stores the checksum hash for a specific file.
// These are typically populated using 'binst embed-checksums' command.
//
// Example:
// ```yaml
// filename: "mytool_1.0.0_linux_amd64.tar.gz"
// hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
// ```
type EmbeddedChecksumElement struct {
	// Asset filename exactly as it appears in the release.
	// This must match the filename generated by the asset template.
	Filename *string `json:"filename,omitempty"`
	// Checksum hash value in hexadecimal format.
	// The format depends on the algorithm specified in ChecksumConfig.
	// For sha256: 64 hex characters, for sha512: 128 hex characters, etc.
	Hash *string `json:"hash,omitempty"`
}

// Supported OS and architecture combination.
//
// Defines a specific platform that the binary supports.
// Used to restrict installation to known-working platforms.
//
// Example:
// ```yaml
// supported_platforms:
// - os: linux
// arch: amd64
// - os: linux
// arch: arm64
// - os: darwin
// arch: amd64
// - os: darwin
// arch: arm64
// - os: windows
// arch: amd64
// ```
type SupportedPlatformElement struct {
	// CPU architecture identifier.
	//
	// Common values:
	// - "amd64 (x86_64)" - 64-bit x86
	// - "arm64 (aarch64)" - 64-bit ARM
	// - "386" - 32-bit x86
	// - "arm" - 32-bit ARM
	//
	// Less common:
	// - "ppc64", "ppc64le" - PowerPC 64-bit
	// - "mips", "mipsle", "mips64", "mips64le" - MIPS architectures
	// - "s390x" - IBM Z architecture
	// - "riscv64" - RISC-V 64-bit
	Arch *SupportedPlatformArch `json:"arch,omitempty"`
	// Operating system identifier.
	//
	// Common values:
	// - "linux" - Linux distributions
	// - "darwin" - macOS
	// - "windows" - Windows
	// - "freebsd", "openbsd", "netbsd" - BSD variants
	// - "android" - Android
	OS *SupportedPlatformOS `json:"os,omitempty"`
}

// Archive extraction configuration
//
// Archive extraction configuration.
//
// Controls how archives are extracted during installation.
// Primarily used to handle archives with unnecessary directory nesting.
//
// Example:
// ```yaml
// # Archive structure: mytool-v1.0.0/bin/mytool
// # We want just: bin/mytool
// unpack:
// strip_components: 1
// ```
type Unpack struct {
	// Number of leading path components to strip when extracting.
	//
	// Similar to tar's --strip-components option.
	// Useful when archives have an extra top-level directory.
	//
	// Examples:
	// - 0 (default): Extract as-is
	// - 1: Remove first directory level (e.g., "mytool-v1.0.0/bin/mytool" → "bin/mytool")
	// - 2: Remove first two directory levels
	StripComponents *int64 `json:"strip_components,omitempty"`
}

type NamingConventionArch string

const (
	ArchLowercase NamingConventionArch = "lowercase"
)

// Casing for ${OS} placeholder.
//
// - lowercase (default): "linux", "darwin", "windows"
// - titlecase: "Linux", "Darwin", "Windows"
type NamingConventionOS string

const (
	OSLowercase NamingConventionOS = "lowercase"
	Titlecase   NamingConventionOS = "titlecase"
)

// Hash algorithm used for checksums.
// Must match the algorithm used by the project's checksum files.
// Most projects use sha256.
type Algorithm string

const (
	Md5    Algorithm = "md5"
	Sha1   Algorithm = "sha1"
	Sha256 Algorithm = "sha256"
	Sha512 Algorithm = "sha512"
)

// CPU architecture identifier.
//
// Common values:
// - "amd64 (x86_64)" - 64-bit x86
// - "arm64 (aarch64)" - 64-bit ARM
// - "386" - 32-bit x86
// - "arm" - 32-bit ARM
//
// Less common:
// - "ppc64", "ppc64le" - PowerPC 64-bit
// - "mips", "mipsle", "mips64", "mips64le" - MIPS architectures
// - "s390x" - IBM Z architecture
// - "riscv64" - RISC-V 64-bit
type SupportedPlatformArch string

const (
	Amd64    SupportedPlatformArch = "amd64"
	Arm      SupportedPlatformArch = "arm"
	Arm64    SupportedPlatformArch = "arm64"
	MIPS     SupportedPlatformArch = "mips"
	Mips64   SupportedPlatformArch = "mips64"
	Mips64LE SupportedPlatformArch = "mips64le"
	Mipsle   SupportedPlatformArch = "mipsle"
	Ppc64    SupportedPlatformArch = "ppc64"
	Ppc64LE  SupportedPlatformArch = "ppc64le"
	Riscv64  SupportedPlatformArch = "riscv64"
	S390X    SupportedPlatformArch = "s390x"
	The386   SupportedPlatformArch = "386"
)

// Operating system identifier.
//
// Common values:
// - "linux" - Linux distributions
// - "darwin" - macOS
// - "windows" - Windows
// - "freebsd", "openbsd", "netbsd" - BSD variants
// - "android" - Android
type SupportedPlatformOS string

const (
	Android   SupportedPlatformOS = "android"
	Darwin    SupportedPlatformOS = "darwin"
	Dragonfly SupportedPlatformOS = "dragonfly"
	Freebsd   SupportedPlatformOS = "freebsd"
	Linux     SupportedPlatformOS = "linux"
	Netbsd    SupportedPlatformOS = "netbsd"
	Openbsd   SupportedPlatformOS = "openbsd"
	Solaris   SupportedPlatformOS = "solaris"
	Windows   SupportedPlatformOS = "windows"
)
