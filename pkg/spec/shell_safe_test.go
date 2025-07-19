package spec

import (
	"testing"
)

func TestShellSafeString(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
		errMsg    string
	}{
		// Valid cases
		{
			name:      "empty string",
			value:     "",
			fieldName: "test",
			wantErr:   false,
		},
		{
			name:      "simple alphanumeric",
			value:     "mytool123",
			fieldName: "name",
			wantErr:   false,
		},
		{
			name:      "with dash and underscore",
			value:     "my-tool_v2",
			fieldName: "name",
			wantErr:   false,
		},
		{
			name:      "version string",
			value:     "v1.2.3",
			fieldName: "version",
			wantErr:   false,
		},
		{
			name:      "repo format",
			value:     "owner/repo-name",
			fieldName: "repo",
			wantErr:   false,
		},
		{
			name:      "file extension",
			value:     ".tar.gz",
			fieldName: "extension",
			wantErr:   false,
		},
		// Command substitution
		{
			name:      "command substitution with $()",
			value:     "tool$(malicious)",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "command substitution",
		},
		{
			name:      "command substitution with backticks",
			value:     "tool`evil`",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "backtick",
		},
		// Shell metacharacters
		{
			name:      "semicolon",
			value:     "tool;rm -rf /",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "semicolon",
		},
		{
			name:      "pipe",
			value:     "tool|cat /etc/passwd",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "pipe",
		},
		{
			name:      "ampersand",
			value:     "tool&background",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "ampersand",
		},
		{
			name:      "output redirection",
			value:     "tool > /etc/passwd",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "output redirection",
		},
		{
			name:      "logical AND",
			value:     "tool && malicious",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "logical AND",
		},
		{
			name:      "newline",
			value:     "tool\nmalicious",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "newline",
		},
		// Control characters
		{
			name:      "null byte",
			value:     "tool\x00null",
			fieldName: "name",
			wantErr:   true,
			errMsg:    "control character",
		},
		{
			name:      "tab is allowed",
			value:     "tool\ttab",
			fieldName: "name",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ShellSafeString(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShellSafeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ShellSafeString() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestInstallSpec_ValidateAllFields(t *testing.T) {
	tests := []struct {
		name    string
		spec    *InstallSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid spec",
			spec: &InstallSpec{
				Name: StringPtr("test-tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template:         StringPtr("${NAME}-${VERSION}"),
					DefaultExtension: StringPtr(".tar.gz"),
					Binaries: []BinaryElement{
						{Name: StringPtr("tool"), Path: StringPtr("bin/tool")},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "dangerous name",
			spec: &InstallSpec{
				Name: StringPtr("tool$(rm -rf /)"),
				Repo: StringPtr("owner/repo"),
			},
			wantErr: true,
			errMsg:  "name contains dangerous",
		},
		{
			name: "dangerous repo",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo;evil"),
			},
			wantErr: true,
			errMsg:  "repo contains dangerous",
		},
		{
			name: "dangerous default_bin_dir with command substitution",
			spec: &InstallSpec{
				Name:          StringPtr("tool"),
				Repo:          StringPtr("owner/repo"),
				DefaultBinDir: StringPtr("$(malicious)/bin"),
			},
			wantErr: true,
			errMsg:  "default_bin_dir contains dangerous",
		},
		{
			name: "valid default_bin_dir with shell variables",
			spec: &InstallSpec{
				Name:          StringPtr("tool"),
				Repo:          StringPtr("owner/repo"),
				DefaultBinDir: StringPtr("${HOME}/.local/bin"),
			},
			wantErr: false,
		},
		{
			name: "dangerous default_version",
			spec: &InstallSpec{
				Name:           StringPtr("tool"),
				Repo:           StringPtr("owner/repo"),
				DefaultVersion: StringPtr("v1.0.0|evil"),
			},
			wantErr: true,
			errMsg:  "default_version contains dangerous",
		},
		{
			name: "dangerous extension",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template:         StringPtr("${NAME}"),
					DefaultExtension: StringPtr(".tar.gz;evil"),
				},
			},
			wantErr: true,
			errMsg:  "asset.default_extension contains dangerous",
		},
		{
			name: "dangerous binary name",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}"),
					Binaries: []BinaryElement{
						{Name: StringPtr("tool`evil`"), Path: StringPtr("tool")},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.binaries[0].name contains dangerous",
		},
		{
			name: "dangerous binary path",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}"),
					Binaries: []BinaryElement{
						{Name: StringPtr("tool"), Path: StringPtr("bin/tool$(bad)")},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.binaries[0].path contains dangerous",
		},
		{
			name: "dangerous rule os",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}"),
					Rules: []RuleElement{
						{
							When: &When{OS: StringPtr("linux")},
							OS:   StringPtr("linux;evil"),
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.rules[0].os contains dangerous",
		},
		{
			name: "dangerous rule arch",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}"),
					Rules: []RuleElement{
						{
							When: &When{OS: StringPtr("linux")},
							Arch: StringPtr("amd64|bad"),
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.rules[0].arch contains dangerous",
		},
		{
			name: "dangerous rule ext",
			spec: &InstallSpec{
				Name: StringPtr("tool"),
				Repo: StringPtr("owner/repo"),
				Asset: &Asset{
					Template: StringPtr("${NAME}"),
					Rules: []RuleElement{
						{
							When: &When{OS: StringPtr("windows")},
							EXT:  StringPtr(".exe$(bad)"),
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "asset.rules[0].ext contains dangerous",
		},
		{
			name: "control character in name",
			spec: &InstallSpec{
				Name: StringPtr("tool\x00null"),
				Repo: StringPtr("owner/repo"),
			},
			wantErr: true,
			errMsg:  "control character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.ValidateAllFields()
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallSpec.ValidateAllFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("InstallSpec.ValidateAllFields() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}
