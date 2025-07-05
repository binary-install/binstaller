import "@typespec/json-schema";

using TypeSpec.JsonSchema;

@doc("Configuration specification for binstaller binary installation")
@jsonSchema
model InstallSpec {
  @doc("Schema version")
  schema?: string = "v1";
  
  @doc("Binary name (defaults to repository name if not specified)")
  name?: string;
  
  @doc("GitHub repository in format 'owner/repo'")
  @pattern("^[^/]+/[^/]+$")
  repo: string;
  
  @doc("Default version to install")
  default_version?: string = "latest";
  
  @doc("Default binary installation directory")
  default_bin_dir?: string = "\${BINSTALLER_BIN:-\${HOME}/.local/bin}";
  
  @doc("Asset download configuration")
  asset: AssetConfig;
  
  @doc("Checksum verification configuration")
  checksums?: ChecksumConfig;
  
  @doc("Attestation verification configuration")
  attestation?: AttestationConfig;
  
  @doc("Archive extraction configuration")
  unpack?: UnpackConfig;
  
  @doc("List of supported OS/architecture combinations")
  supported_platforms?: Platform[];
}

@doc("Supported OS and architecture combination")
model Platform {
  @doc("Operating system (e.g., linux, darwin, windows)")
  os: "linux" | "darwin" | "windows" | "freebsd" | "openbsd" | "netbsd" | "dragonfly" | "solaris" | "android";
  
  @doc("Architecture (e.g., amd64, arm64, 386)")
  arch: "amd64" | "arm64" | "386" | "arm" | "ppc64" | "ppc64le" | "mips" | "mipsle" | "mips64" | "mips64le" | "s390x" | "riscv64";
}

@doc("Configuration for constructing download URLs and asset names")
model AssetConfig {
  @doc("Filename template with placeholders: \${NAME}, \${VERSION}, \${OS}, \${ARCH}, \${EXT}")
  template: string;
  
  @doc("Default file extension when not specified in template")
  default_extension?: string;
  
  @doc("Binary names and their paths within the asset")
  binaries?: Binary[];
  
  @doc("Platform-specific overrides")
  rules?: AssetRule[];
  
  @doc("Controls the casing of placeholder values")
  naming_convention?: NamingConvention;
  
  @doc("Architecture emulation configuration")
  arch_emulation?: ArchEmulation;
}

@doc("Platform-specific asset configuration override")
model AssetRule {
  @doc("Condition for applying this rule")
  when: PlatformCondition;
  
  @doc("Override template for matching platforms")
  template?: string;
  
  @doc("Override OS value for matching platforms")
  os?: string;
  
  @doc("Override architecture value for matching platforms")
  arch?: string;
  
  @doc("Override extension for matching platforms")
  ext?: string;
  
  @doc("Override binary configuration for matching platforms")
  binaries?: Binary[];
}

@doc("Binary name and path configuration")
model Binary {
  @doc("Name of the binary to install")
  name: string;
  
  @doc("Path to the binary within the extracted archive (use \${ASSET_FILENAME} for non-archive assets)")
  path: string;
}

@doc("Condition for matching specific platforms")
model PlatformCondition {
  @doc("Match specific operating system")
  os?: string;
  
  @doc("Match specific architecture")
  arch?: string;
}

@doc("Controls the casing of template placeholders")
model NamingConvention {
  @doc("Casing for \${OS} placeholder")
  os?: "lowercase" | "titlecase" = "lowercase";
  
  @doc("Casing for \${ARCH} placeholder")
  arch?: "lowercase" = "lowercase";
}

@doc("Architecture emulation configuration")
model ArchEmulation {
  @doc("Use amd64 instead of arm64 when Rosetta 2 is available on macOS")
  rosetta2?: boolean = false;
}

@doc("Checksum verification configuration")
model ChecksumConfig {
  @doc("Hash algorithm")
  algorithm?: "sha256" | "sha512" | "sha1" | "md5" = "sha256";
  
  @doc("Checksum filename template")
  template?: string;
  
  @doc("Pre-verified checksums keyed by version")
  embedded_checksums?: Record<EmbeddedChecksum[]>;
}

@doc("Pre-verified checksum for a specific asset")
model EmbeddedChecksum {
  @doc("Asset filename")
  filename: string;
  
  @doc("Checksum hash value")
  hash: string;
}

@doc("Attestation verification using GitHub's attestation feature")
model AttestationConfig {
  @doc("Enable attestation verification")
  enabled?: boolean = false;
  
  @doc("Require attestation to pass")
  require?: boolean = false;
  
  @doc("Additional flags for 'gh attestation verify' command")
  verify_flags?: string;
}

@doc("Archive extraction configuration")
model UnpackConfig {
  @doc("Number of leading path components to strip when extracting")
  @minValue(0)
  strip_components?: int32 = 0;
}