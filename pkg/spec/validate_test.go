package spec

import (
	"testing"
)

func TestValidateAssetTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  bool
		errMsg   string
	}{
		// Valid templates
		{
			name:     "valid template with all variables",
			template: "${NAME}-v${VERSION}-${OS}-${ARCH}${EXT}",
			wantErr:  false,
		},
		{
			name:     "valid template with underscores",
			template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz",
			wantErr:  false,
		},
		{
			name:     "valid template with dots",
			template: "${NAME}-${VERSION}.${OS}-${ARCH}",
			wantErr:  false,
		},
		{
			name:     "valid template with fixed text",
			template: "my-app-${VERSION}-${OS}-${ARCH}.zip",
			wantErr:  false,
		},
		// Dangerous templates - command substitution
		{
			name:     "reject command substitution with $()",
			template: "${NAME}$(malicious command)",
			wantErr:  true,
			errMsg:   "command substitution",
		},
		{
			name:     "reject command substitution with backticks",
			template: "${NAME}`evil`",
			wantErr:  true,
			errMsg:   "command substitution",
		},
		{
			name:     "reject nested command substitution",
			template: "${NAME}-v${VERSION}-$(rm -rf /)",
			wantErr:  true,
			errMsg:   "command substitution",
		},
		// Dangerous templates - shell metacharacters
		{
			name:     "reject semicolon",
			template: "${NAME};rm -rf /",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		{
			name:     "reject pipe",
			template: "${NAME}|cat /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		{
			name:     "reject ampersand",
			template: "${NAME}&& malicious",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		{
			name:     "reject output redirection >",
			template: "${NAME} > /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		{
			name:     "reject input redirection <",
			template: "${NAME} < /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		{
			name:     "reject append redirection >>",
			template: "${NAME} >> /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous character",
		},
		// Edge cases
		{
			name:     "empty template",
			template: "",
			wantErr:  false,
		},
		{
			name:     "template with only text",
			template: "static-filename.tar.gz",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssetTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssetTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateAssetTemplate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateAssetTemplate_ChecksumTemplate(t *testing.T) {
	// Test validation for checksum templates which use the same pattern
	tests := []struct {
		name     string
		template string
		wantErr  bool
	}{
		{
			name:     "valid checksum template",
			template: "${NAME}_${VERSION}_checksums.txt",
			wantErr:  false,
		},
		{
			name:     "reject dangerous checksum template",
			template: "checksums$(rm -rf /).txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssetTemplate(tt.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAssetTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
