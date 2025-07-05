package spec

import "strings"

// SetDefaults sets default values for the InstallSpec
func (s *InstallSpec) SetDefaults() {
	if s.Schema == nil || *s.Schema == "" {
		schema := "v1"
		s.Schema = &schema
	}
	if s.DefaultVersion == nil || *s.DefaultVersion == "" {
		version := "latest"
		s.DefaultVersion = &version
	}
	if s.DefaultBinDir == nil || *s.DefaultBinDir == "" {
		binDir := "${BINSTALLER_BIN:-${HOME}/.local/bin}"
		s.DefaultBinDir = &binDir
	}
	if s.Asset != nil {
		if s.Asset.NamingConvention == nil {
			s.Asset.NamingConvention = &NamingConvention{}
		}
		if s.Asset.NamingConvention.OS == nil {
			os := OSLowercase
			s.Asset.NamingConvention.OS = &os
		}
		if s.Asset.NamingConvention.Arch == nil {
			arch := ArchLowercase
			s.Asset.NamingConvention.Arch = &arch
		}
	}
	if s.Name == nil && s.Repo != nil && *s.Repo != "" {
		sp := strings.SplitN(*s.Repo, "/", 2)
		if len(sp) == 2 {
			s.Name = &sp[1]
		}
	}
	if s.Asset != nil && (s.Asset.Binaries == nil || len(s.Asset.Binaries) == 0) && s.Name != nil && *s.Name != "" {
		if s.Asset.DefaultExtension != nil && *s.Asset.DefaultExtension != "" {
			s.Asset.Binaries = []BinaryElement{
				{Name: s.Name, Path: s.Name},
			}
		} else {
			assetFilename := "${ASSET_FILENAME}"
			s.Asset.Binaries = []BinaryElement{
				{Name: s.Name, Path: &assetFilename},
			}
		}
	}
	if s.Checksums != nil {
		if s.Checksums.Algorithm == nil {
			algo := Sha256
			s.Checksums.Algorithm = &algo
		}
	}
}

// Type aliases for backward compatibility
type Platform = SupportedPlatformElement
type AssetConfig = Asset
type ChecksumConfig = Checksums
type UnpackConfig = Unpack
type AssetRule = RuleElement
type Binary = BinaryElement
type PlatformCondition = When
type EmbeddedChecksum = EmbeddedChecksumElement

// Helper function to get Ext field (generated code uses EXT)
func (r *RuleElement) GetExt() *string {
	return r.EXT
}

// Helper function to set Ext field
func (r *RuleElement) SetExt(ext *string) {
	r.EXT = ext
}

// Helper functions for string conversion

// StringPtr returns a pointer to the string
func StringPtr(s string) *string {
	return &s
}

// StringValue safely dereferences a string pointer
func StringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// AlgorithmString converts Algorithm to string
func AlgorithmString(a *Algorithm) string {
	if a == nil {
		return ""
	}
	return string(*a)
}

// AlgorithmPtr converts string to Algorithm pointer
func AlgorithmPtr(s string) *Algorithm {
	a := Algorithm(s)
	return &a
}

// PlatformOSString converts SupportedPlatformOS to string
func PlatformOSString(os *SupportedPlatformOS) string {
	if os == nil {
		return ""
	}
	return string(*os)
}

// PlatformArchString converts SupportedPlatformArch to string
func PlatformArchString(arch *SupportedPlatformArch) string {
	if arch == nil {
		return ""
	}
	return string(*arch)
}

// NamingConventionOSString converts NamingConventionOS to string
func NamingConventionOSString(os *NamingConventionOS) string {
	if os == nil {
		return ""
	}
	return string(*os)
}

// NamingConventionOSPtr converts string to NamingConventionOS pointer
func NamingConventionOSPtr(s string) *NamingConventionOS {
	n := NamingConventionOS(s)
	return &n
}
