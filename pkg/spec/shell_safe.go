package spec

import (
	"fmt"
	"strings"
	"unicode"
)

// ShellSafeString validates that a string is safe to embed in a shell script.
// It checks for dangerous patterns that could lead to command injection.
func ShellSafeString(value string, fieldName string) error {
	if value == "" {
		return nil
	}

	// Check for command substitution patterns
	if strings.Contains(value, "$(") {
		return fmt.Errorf("%s contains dangerous command substitution '$(' pattern: %s", fieldName, value)
	}
	if strings.Contains(value, "`") {
		return fmt.Errorf("%s contains dangerous command substitution backtick '`' pattern: %s", fieldName, value)
	}

	// Check for shell metacharacters (same as ValidateAssetTemplate)
	dangerousChars := []struct {
		char string
		desc string
	}{
		// Check longer patterns first
		{">>", "append redirection"},
		{"<<", "here document"},
		{"||", "logical OR"},
		{"&&", "logical AND"},
		// Then single characters
		{";", "semicolon"},
		{"|", "pipe"},
		{"&", "ampersand"},
		{">", "output redirection"},
		{"<", "input redirection"},
		{"\n", "newline"},
		{"\r", "carriage return"},
	}

	for _, dc := range dangerousChars {
		if strings.Contains(value, dc.char) {
			return fmt.Errorf("%s contains dangerous character '%s' (%s): %s", fieldName, dc.char, dc.desc, value)
		}
	}

	// Additional check for control characters
	for _, r := range value {
		if unicode.IsControl(r) && r != '\t' {
			return fmt.Errorf("%s contains control character (code %d)", fieldName, r)
		}
	}

	return nil
}

// ValidateAllFields validates all fields in InstallSpec that will be embedded in shell scripts
func (s *InstallSpec) ValidateAllFields() error {
	// Validate name
	if s.Name != nil {
		if err := ShellSafeString(*s.Name, "name"); err != nil {
			return err
		}
	}

	// Validate repo
	if s.Repo != nil {
		if err := ShellSafeString(*s.Repo, "repo"); err != nil {
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
		if err := ShellSafeString(*s.DefaultVersion, "default_version"); err != nil {
			return err
		}
	}

	// Validate asset fields
	if s.Asset != nil {
		// Validate default_extension
		if s.Asset.DefaultExtension != nil {
			if err := ShellSafeString(*s.Asset.DefaultExtension, "asset.default_extension"); err != nil {
				return err
			}
		}

		// Validate binaries
		for i, binary := range s.Asset.Binaries {
			if binary.Name != nil {
				if err := ShellSafeString(*binary.Name, fmt.Sprintf("asset.binaries[%d].name", i)); err != nil {
					return err
				}
			}
			if binary.Path != nil {
				if err := ShellSafeString(*binary.Path, fmt.Sprintf("asset.binaries[%d].path", i)); err != nil {
					return err
				}
			}
		}

		// Validate rules
		for i, rule := range s.Asset.Rules {
			if rule.OS != nil {
				if err := ShellSafeString(*rule.OS, fmt.Sprintf("asset.rules[%d].os", i)); err != nil {
					return err
				}
			}
			if rule.Arch != nil {
				if err := ShellSafeString(*rule.Arch, fmt.Sprintf("asset.rules[%d].arch", i)); err != nil {
					return err
				}
			}
			if rule.EXT != nil {
				if err := ShellSafeString(*rule.EXT, fmt.Sprintf("asset.rules[%d].ext", i)); err != nil {
					return err
				}
			}
		}
	}

	// Call the existing Validate method for template validation
	return s.Validate()
}
