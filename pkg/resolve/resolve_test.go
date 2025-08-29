package resolve

import (
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetFilename(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *spec.InstallSpec
		version string
		os      string
		arch    string
		want    string
	}{
		{
			name: "basic template substitution",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("gh"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			version: "v2.40.0",
			os:      "linux",
			arch:    "amd64",
			want:    "gh_2.40.0_linux_amd64.tar.gz",
		},
		{
			name: "version without v prefix",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".zip"),
				},
			},
			version: "1.0.0",
			os:      "darwin",
			arch:    "arm64",
			want:    "tool-1.0.0-darwin-arm64.zip",
		},
		{
			name: "apply os/arch rules",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("gh"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					Rules: []spec.RuleElement{
						{
							When: &spec.When{
								OS: spec.StringPtr("darwin"),
							},
							OS:  spec.StringPtr("macOS"),
							EXT: spec.StringPtr(".zip"),
						},
					},
				},
			},
			version: "v2.40.0",
			os:      "darwin",
			arch:    "amd64",
			want:    "gh_2.40.0_macOS_amd64.zip",
		},
		{
			name: "apply arch rule",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					Rules: []spec.RuleElement{
						{
							When: &spec.When{
								Arch: spec.StringPtr("amd64"),
							},
							Arch: spec.StringPtr("x86_64"),
						},
					},
				},
			},
			version: "v1.0.0",
			os:      "linux",
			arch:    "amd64",
			want:    "tool_1.0.0_linux_x86_64.tar.gz",
		},
		{
			name: "titlecase OS naming convention",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					NamingConvention: &spec.NamingConvention{
						OS: (*spec.NamingConventionOS)(spec.StringPtr("titlecase")),
					},
				},
			},
			version: "v1.0.0",
			os:      "linux",
			arch:    "amd64",
			want:    "tool_1.0.0_Linux_amd64.tar.gz",
		},
		{
			name: "custom template from rule",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Asset: &spec.Asset{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					Rules: []spec.RuleElement{
						{
							When: &spec.When{
								OS:   spec.StringPtr("windows"),
								Arch: spec.StringPtr("amd64"),
							},
							Template: spec.StringPtr("${NAME}-${VERSION}-win64${EXT}"),
							EXT:      spec.StringPtr(".zip"),
						},
					},
				},
			},
			version: "v1.0.0",
			os:      "windows",
			arch:    "amd64",
			want:    "tool-1.0.0-win64.zip",
		},
		{
			name: "no extension",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("jq"),
				Asset: &spec.Asset{
					Template: spec.StringPtr("jq-${OS}${ARCH}"),
					Rules: []spec.RuleElement{
						{
							When: &spec.When{
								Arch: spec.StringPtr("amd64"),
							},
							Arch: spec.StringPtr("64"),
						},
					},
				},
			},
			version: "1.6",
			os:      "linux",
			arch:    "amd64",
			want:    "jq-linux64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply defaults
			tt.cfg.SetDefaults()

			got := AssetFilename(tt.cfg, tt.version, tt.os, tt.arch)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveVersion(t *testing.T) {
	// Note: This test would typically require mocking GitHub API calls
	// For now, we'll test the basic logic without actual API calls
	tests := []struct {
		name    string
		cfg     *spec.InstallSpec
		version string
		want    string
		wantErr bool
	}{
		{
			name: "explicit version",
			cfg: &spec.InstallSpec{
				Repo: spec.StringPtr("cli/cli"),
			},
			version: "v2.40.0",
			want:    "v2.40.0",
		},
		{
			name: "version without v prefix",
			cfg: &spec.InstallSpec{
				Repo: spec.StringPtr("cli/cli"),
			},
			version: "2.40.0",
			want:    "2.40.0",
		},
		{
			name: "latest version",
			cfg: &spec.InstallSpec{
				Repo: spec.StringPtr("cli/cli"),
			},
			version: "latest",
			want:    "",    // Would be resolved from GitHub API
			wantErr: false, // Should succeed if GitHub API is available
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveVersion(tt.cfg, tt.version)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.want != "" {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestGetBinaryInfo(t *testing.T) {
	tests := []struct {
		name       string
		cfg        *spec.InstallSpec
		os         string
		arch       string
		wantBinary string
		wantPath   string
	}{
		{
			name: "single binary with name and path",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("gh"),
				Asset: &spec.Asset{
					Binaries: []spec.BinaryElement{
						{
							Name: spec.StringPtr("gh"),
							Path: spec.StringPtr("bin/gh"),
						},
					},
				},
			},
			os:         "linux",
			arch:       "amd64",
			wantBinary: "gh",
			wantPath:   "bin/gh",
		},
		{
			name: "binary with rules override",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Asset: &spec.Asset{
					Binaries: []spec.BinaryElement{
						{
							Name: spec.StringPtr("tool"),
							Path: spec.StringPtr("tool"),
						},
					},
					Rules: []spec.RuleElement{
						{
							When: &spec.When{
								OS: spec.StringPtr("windows"),
							},
							Binaries: []spec.BinaryElement{
								{
									Name: spec.StringPtr("tool.exe"),
									Path: spec.StringPtr("tool.exe"),
								},
							},
						},
					},
				},
			},
			os:         "windows",
			arch:       "amd64",
			wantBinary: "tool.exe",
			wantPath:   "tool.exe",
		},
		{
			name: "raw binary (no extension)",
			cfg: &spec.InstallSpec{
				Name: spec.StringPtr("jq"),
				Asset: &spec.Asset{
					// No default extension means raw binary
					Binaries: []spec.BinaryElement{
						{
							Name: spec.StringPtr("jq"),
							Path: spec.StringPtr("${ASSET_FILENAME}"),
						},
					},
				},
			},
			os:         "linux",
			arch:       "amd64",
			wantBinary: "jq",
			wantPath:   "${ASSET_FILENAME}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply defaults
			tt.cfg.SetDefaults()

			binaries := GetBinaryInfo(tt.cfg, tt.os, tt.arch)
			require.Len(t, binaries, 1)

			assert.Equal(t, tt.wantBinary, binaries[0].Name)
			assert.Equal(t, tt.wantPath, binaries[0].Path)
		})
	}
}
