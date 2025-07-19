package spec

import (
	"fmt"
	"strings"
)

// ValidateAssetTemplate validates that an asset template doesn't contain
// dangerous shell metacharacters that could lead to command injection.
func ValidateAssetTemplate(template string) error {
	// Check for command substitution patterns
	if strings.Contains(template, "$(") {
		return fmt.Errorf("asset template contains dangerous command substitution '$(' pattern: %s", template)
	}
	if strings.Contains(template, "`") {
		return fmt.Errorf("asset template contains dangerous command substitution '`' pattern: %s", template)
	}

	// Check for shell metacharacters
	dangerousChars := []struct {
		char string
		desc string
	}{
		{";", "semicolon"},
		{"|", "pipe"},
		{"&", "ampersand"},
		{">", "output redirection"},
		{"<", "input redirection"},
	}

	for _, dc := range dangerousChars {
		if strings.Contains(template, dc.char) {
			return fmt.Errorf("asset template contains dangerous character '%s' (%s): %s", dc.char, dc.desc, template)
		}
	}

	return nil
}
