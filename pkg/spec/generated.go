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

// Configuration specification for binstaller binary installation
type InstallSpec struct {
	// Asset download configuration                                                         
	Asset                                                        *Asset                     `json:"asset,omitempty"`
	// Attestation verification configuration                                               
	Attestation                                                  *Attestation               `json:"attestation,omitempty"`
	// Checksum verification configuration                                                  
	Checksums                                                    *Checksums                 `json:"checksums,omitempty"`
	// Default binary installation directory                                                
	DefaultBinDir                                                *string                    `json:"default_bin_dir,omitempty"`
	// Default version to install                                                           
	DefaultVersion                                               *string                    `json:"default_version,omitempty"`
	// Binary name (defaults to repository name if not specified)                           
	Name                                                         *string                    `json:"name,omitempty"`
	// GitHub repository in format 'owner/repo'                                             
	Repo                                                         *string                    `json:"repo,omitempty"`
	// Schema version                                                                       
	Schema                                                       *string                    `json:"schema,omitempty"`
	// List of supported OS/architecture combinations                                       
	SupportedPlatforms                                           []SupportedPlatformElement `json:"supported_platforms,omitempty"`
	// Archive extraction configuration                                                     
	Unpack                                                       *Unpack                    `json:"unpack,omitempty"`
}

// Asset download configuration
//
// Configuration for constructing download URLs and asset names
type Asset struct {
	// Architecture emulation configuration                                                              
	ArchEmulation                                                                      *ArchEmulation    `json:"arch_emulation,omitempty"`
	// Binary names and their paths within the asset                                                     
	Binaries                                                                           []BinaryElement   `json:"binaries,omitempty"`
	// Default file extension when not specified in template                                             
	DefaultExtension                                                                   *string           `json:"default_extension,omitempty"`
	// Controls the casing of placeholder values                                                         
	NamingConvention                                                                   *NamingConvention `json:"naming_convention,omitempty"`
	// Platform-specific overrides                                                                       
	Rules                                                                              []RuleElement     `json:"rules,omitempty"`
	// Filename template with placeholders: ${NAME}, ${VERSION}, ${OS}, ${ARCH}, ${EXT}                  
	Template                                                                           *string           `json:"template,omitempty"`
}

// Architecture emulation configuration
type ArchEmulation struct {
	// Use amd64 instead of arm64 when Rosetta 2 is available on macOS      
	Rosetta2                                                          *bool `json:"rosetta2,omitempty"`
}

// Binary name and path configuration
type BinaryElement struct {
	// Name of the binary to install                                                                 
	Name                                                                                     *string `json:"name,omitempty"`
	// Path to the binary within the extracted archive (use ${ASSET_FILENAME} for non-archive        
	// assets)                                                                                       
	Path                                                                                     *string `json:"path,omitempty"`
}

// Controls the casing of placeholder values
//
// Controls the casing of template placeholders
type NamingConvention struct {
	// Casing for ${ARCH} placeholder                      
	Arch                             *NamingConventionArch `json:"arch,omitempty"`
	// Casing for ${OS} placeholder                        
	OS                               *NamingConventionOS   `json:"os,omitempty"`
}

// Platform-specific asset configuration override
type RuleElement struct {
	// Override architecture value for matching platforms                  
	Arch                                                   *string         `json:"arch,omitempty"`
	// Override binary configuration for matching platforms                
	Binaries                                               []BinaryElement `json:"binaries,omitempty"`
	// Override extension for matching platforms                           
	EXT                                                    *string         `json:"ext,omitempty"`
	// Override OS value for matching platforms                            
	OS                                                     *string         `json:"os,omitempty"`
	// Override template for matching platforms                            
	Template                                               *string         `json:"template,omitempty"`
	// Condition for applying this rule                                    
	When                                                   *When           `json:"when,omitempty"`
}

// Condition for applying this rule
//
// Condition for matching specific platforms
type When struct {
	// Match specific architecture            
	Arch                              *string `json:"arch,omitempty"`
	// Match specific operating system        
	OS                                *string `json:"os,omitempty"`
}

// Attestation verification configuration
//
// Attestation verification using GitHub's attestation feature
type Attestation struct {
	// Enable attestation verification                             
	Enabled                                                *bool   `json:"enabled,omitempty"`
	// Require attestation to pass                                 
	Require                                                *bool   `json:"require,omitempty"`
	// Additional flags for 'gh attestation verify' command        
	VerifyFlags                                            *string `json:"verify_flags,omitempty"`
}

// Checksum verification configuration
type Checksums struct {
	// Hash algorithm                                                              
	Algorithm                                 *Algorithm                           `json:"algorithm,omitempty"`
	// Pre-verified checksums keyed by version                                     
	EmbeddedChecksums                         map[string][]EmbeddedChecksumElement `json:"embedded_checksums,omitempty"`
	// Checksum filename template                                                  
	Template                                  *string                              `json:"template,omitempty"`
}

// Pre-verified checksum for a specific asset
type EmbeddedChecksumElement struct {
	// Asset filename             
	Filename              *string `json:"filename,omitempty"`
	// Checksum hash value        
	Hash                  *string `json:"hash,omitempty"`
}

// Supported OS and architecture combination
type SupportedPlatformElement struct {
	// Architecture (e.g., amd64, arm64, 386)                                
	Arch                                              *SupportedPlatformArch `json:"arch,omitempty"`
	// Operating system (e.g., linux, darwin, windows)                       
	OS                                                *SupportedPlatformOS   `json:"os,omitempty"`
}

// Archive extraction configuration
type Unpack struct {
	// Number of leading path components to strip when extracting       
	StripComponents                                              *int64 `json:"strip_components,omitempty"`
}

type NamingConventionArch string

const (
	ArchLowercase NamingConventionArch = "lowercase"
)

// Casing for ${OS} placeholder
type NamingConventionOS string

const (
	OSLowercase NamingConventionOS = "lowercase"
	Titlecase   NamingConventionOS = "titlecase"
)

// Hash algorithm
type Algorithm string

const (
	Md5    Algorithm = "md5"
	Sha1   Algorithm = "sha1"
	Sha256 Algorithm = "sha256"
	Sha512 Algorithm = "sha512"
)

// Architecture (e.g., amd64, arm64, 386)
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

// Operating system (e.g., linux, darwin, windows)
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
