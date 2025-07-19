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
