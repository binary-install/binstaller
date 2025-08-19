package spec

import (
	"strings"
	"testing"
)

func TestValidateShellSafe(t *testing.T) {
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
			errMsg:   "dangerous",
		},
		{
			name:     "reject pipe",
			template: "${NAME}|cat /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous",
		},
		{
			name:     "reject ampersand",
			template: "${NAME}&& malicious",
			wantErr:  true,
			errMsg:   "dangerous",
		},
		{
			name:     "reject output redirection >",
			template: "${NAME} > /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous",
		},
		{
			name:     "reject input redirection <",
			template: "${NAME} < /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous",
		},
		{
			name:     "reject append redirection >>",
			template: "${NAME} >> /etc/passwd",
			wantErr:  true,
			errMsg:   "dangerous",
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
			err := ValidateShellSafe(tt.template, "asset template")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateShellSafe() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateShellSafe() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateShellSafe_ChecksumTemplate(t *testing.T) {
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
			err := ValidateShellSafe(tt.template, "asset template")
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateShellSafe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    *InstallSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid spec with asset template",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}-v${VERSION}-${OS}-${ARCH}${EXT}"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid spec without templates",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
			},
			wantErr: false,
		},
		{
			name: "invalid asset template with command substitution",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}$(rm -rf /)"),
				},
			},
			wantErr: true,
			errMsg:  "asset.template",
		},
		{
			name: "invalid checksum template",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Checksums: &Checksums{
					Template: StringPtr("checksums`evil`.txt"),
				},
			},
			wantErr: true,
			errMsg:  "checksums.template",
		},
		{
			name: "invalid rule template",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}-${VERSION}"),
					Rules: []RuleElement{
						{
							When:     &When{OS: StringPtr("linux")},
							Template: StringPtr("${NAME};malicious"),
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.rules[0].template",
		},
		{
			name: "multiple invalid templates",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}|bad"),
				},
				Checksums: &Checksums{
					Template: StringPtr("checksums$(bad)"),
				},
			},
			wantErr: true,
			errMsg:  "asset.template", // Should fail on first error
		},
		{
			name: "valid spec with multiple rules",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}"),
					Rules: []RuleElement{
						{
							When:     &When{OS: StringPtr("darwin")},
							Template: StringPtr("${NAME}-${VERSION}-apple-darwin"),
						},
						{
							When:     &When{OS: StringPtr("linux")},
							Template: StringPtr("${NAME}-${VERSION}-unknown-linux-gnu"),
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallSpec.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("InstallSpec.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}
