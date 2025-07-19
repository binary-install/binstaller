package spec

import (
	"fmt"
	"strings"
)

// Template validation constants
const (
	errMsgCommandSubstitution = "asset template contains dangerous command substitution %s: %s"
	errMsgDangerousChar       = "asset template contains dangerous character '%s' (%s): %s"
)

// Dangerous patterns that can lead to command injection
var (
	commandSubstitutionPatterns = []struct {
		pattern string
		desc    string
	}{
		{"$(", "'$(' pattern"},
		{"`", "backtick '`' pattern"},
	}

	dangerousCharacters = []struct {
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
	}
)

// ValidateAssetTemplate validates that an asset template doesn't contain
// dangerous shell metacharacters that could lead to command injection.
// Valid templates should only contain:
// - Template variables: ${NAME}, ${VERSION}, ${OS}, ${ARCH}, ${EXT}
// - Safe characters: alphanumeric, dash, underscore, dot
// - Static text
func ValidateAssetTemplate(template string) error {
	if template == "" {
		return nil // Empty template is valid
	}

	// Check for command substitution patterns
	for _, pattern := range commandSubstitutionPatterns {
		if strings.Contains(template, pattern.pattern) {
			return fmt.Errorf(errMsgCommandSubstitution, pattern.desc, template)
		}
	}

	// Check for shell metacharacters
	// Note: We check longer patterns first (e.g., ">>" before ">")
	// to provide more specific error messages
	for _, dc := range dangerousCharacters {
		if strings.Contains(template, dc.char) {
			return fmt.Errorf(errMsgDangerousChar, dc.char, dc.desc, template)
		}
	}

	return nil
}

// Validate validates the InstallSpec for security issues.
// It checks all templates (asset, checksum, and rule templates) for
// dangerous shell metacharacters that could lead to command injection.
func (s *InstallSpec) Validate() error {
	// Validate main asset template
	if s.Asset != nil && s.Asset.Template != nil {
		if err := ValidateAssetTemplate(*s.Asset.Template); err != nil {
			return fmt.Errorf("invalid asset template: %w", err)
		}

		// Validate rule templates
		for i, rule := range s.Asset.Rules {
			if rule.Template != nil {
				if err := ValidateAssetTemplate(*rule.Template); err != nil {
					return fmt.Errorf("invalid rule template at index %d: %w", i, err)
				}
			}
		}
	}

	// Validate checksum template
	if s.Checksums != nil && s.Checksums.Template != nil {
		if err := ValidateAssetTemplate(*s.Checksums.Template); err != nil {
			return fmt.Errorf("invalid checksum template: %w", err)
		}
	}

	return nil
}
