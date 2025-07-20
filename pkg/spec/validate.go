package spec

import (
	"fmt"
	"strings"
	"unicode"
)

// dangerousPatterns defines shell patterns that could lead to command injection
var dangerousPatterns = []struct {
	pattern string
	desc    string
}{
	// Command substitution
	{"$(", "command substitution"},
	{"`", "command substitution"},
	// Multi-char operators (check before single chars)
	{">>", "append redirection"},
	{"<<", "here document"},
	{"||", "logical OR"},
	{"&&", "logical AND"},
	// Single char operators
	{";", "semicolon"},
	{"|", "pipe"},
	{"&", "ampersand"},
	{">", "output redirection"},
	{"<", "input redirection"},
	{"\n", "newline"},
	{"\r", "carriage return"},
}

// ValidateShellSafe checks if a string is safe to embed in shell scripts
func ValidateShellSafe(value, fieldName string) error {
	if value == "" {
		return nil
	}

	// Check dangerous patterns
	for _, p := range dangerousPatterns {
		if strings.Contains(value, p.pattern) {
			return fmt.Errorf("%s contains dangerous %s: %s", fieldName, p.desc, value)
		}
	}

	// Check control characters
	for _, r := range value {
		if unicode.IsControl(r) && r != '\t' {
			return fmt.Errorf("%s contains control character (code %d)", fieldName, r)
		}
	}

	return nil
}

// Validate validates all fields in InstallSpec that will be embedded in shell scripts
func Validate(s *InstallSpec) error {
	if s == nil {
		return fmt.Errorf("InstallSpec is nil")
	}
	// Validate name
	if s.Name != nil {
		if err := ValidateShellSafe(*s.Name, "name"); err != nil {
			return err
		}
	}

	// Validate repo
	if s.Repo != nil {
		if err := ValidateShellSafe(*s.Repo, "repo"); err != nil {
			return err
		}
	}

	// Validate default_bin_dir - special handling as it can contain shell variables
	if s.DefaultBinDir != nil {
		// Allow ${...} patterns in default_bin_dir as they are expected
		// But still check for command substitution and other dangerous patterns
		binDir := *s.DefaultBinDir
		if strings.Contains(binDir, "$(") || strings.Contains(binDir, "`") {
			return fmt.Errorf("default_bin_dir contains dangerous command substitution: %s", binDir)
		}
		// Check for dangerous characters except $ which is allowed for variables
		dangerousInBinDir := []string{";", "|", "&", ">", "<", ">>", "<<", "||", "&&", "\n", "\r"}
		for _, char := range dangerousInBinDir {
			if strings.Contains(binDir, char) {
				return fmt.Errorf("default_bin_dir contains dangerous character '%s': %s", char, binDir)
			}
		}
	}

	// Validate default_version
	if s.DefaultVersion != nil {
		if err := ValidateShellSafe(*s.DefaultVersion, "default_version"); err != nil {
			return err
		}
	}

	// Validate asset fields
	if s.Asset != nil {
		// Validate default_extension
		if s.Asset.DefaultExtension != nil {
			if err := ValidateShellSafe(*s.Asset.DefaultExtension, "asset.default_extension"); err != nil {
				return err
			}
		}

		// Validate main asset template
		if s.Asset.Template != nil {
			if err := ValidateShellSafe(*s.Asset.Template, "asset.template"); err != nil {
				return err
			}
		}

		// Validate binaries
		for i, binary := range s.Asset.Binaries {
			if binary.Name != nil {
				if err := ValidateShellSafe(*binary.Name, fmt.Sprintf("asset.binaries[%d].name", i)); err != nil {
					return err
				}
			}
			if binary.Path != nil {
				if err := ValidateShellSafe(*binary.Path, fmt.Sprintf("asset.binaries[%d].path", i)); err != nil {
					return err
				}
			}
		}

		// Validate rules
		for i, rule := range s.Asset.Rules {
			if rule.OS != nil {
				if err := ValidateShellSafe(*rule.OS, fmt.Sprintf("asset.rules[%d].os", i)); err != nil {
					return err
				}
			}
			if rule.Arch != nil {
				if err := ValidateShellSafe(*rule.Arch, fmt.Sprintf("asset.rules[%d].arch", i)); err != nil {
					return err
				}
			}
			if rule.EXT != nil {
				if err := ValidateShellSafe(*rule.EXT, fmt.Sprintf("asset.rules[%d].ext", i)); err != nil {
					return err
				}
			}
			// Validate rule template
			if rule.Template != nil {
				if err := ValidateShellSafe(*rule.Template, fmt.Sprintf("asset.rules[%d].template", i)); err != nil {
					return err
				}
			}
		}
	}

	// Validate checksum template
	if s.Checksums != nil && s.Checksums.Template != nil {
		if err := ValidateShellSafe(*s.Checksums.Template, "checksums.template"); err != nil {
			return err
		}
	}

	return nil
}
