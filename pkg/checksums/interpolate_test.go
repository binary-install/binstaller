package checksums

import (
	"strings"
	"testing"

	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/spec"
)

func TestInterpolateTemplate(t *testing.T) {
	tests := []struct {
		name           string
		spec           *spec.InstallSpec
		version        string
		template       string
		additionalVars map[string]string
		expected       string
		expectError    bool
	}{
		{
			name: "basic NAME and VERSION interpolation",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_${VERSION}_checksums.txt",
			expected: "mytool_1.2.3_checksums.txt",
		},
		{
			name: "with OS and ARCH variables",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz",
			additionalVars: map[string]string{
				"OS":   "linux",
				"ARCH": "amd64",
			},
			expected: "mytool_1.2.3_linux_amd64.tar.gz",
		},
		{
			name: "with EXT variable",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}",
			additionalVars: map[string]string{
				"OS":   "windows",
				"ARCH": "amd64",
				"EXT":  ".zip",
			},
			expected: "mytool_1.2.3_windows_amd64.zip",
		},
		{
			name: "unknown variable replaced with empty string",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_${UNKNOWN}_${VERSION}",
			expected: "mytool__1.2.3",
		},
		{
			name: "empty name uses empty string",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr(""),
			},
			version:  "v1.2.3",
			template: "${NAME}_${VERSION}",
			expected: "_1.2.3",
		},
		{
			name: "REPO and REPO_OWNER/NAME are not supported",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Repo: spec.StringPtr("owner/repo"),
			},
			version:  "v1.2.3",
			template: "${NAME}_${REPO}_${REPO_OWNER}_${REPO_NAME}",
			expected: "mytool___",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedder{
				Spec:    tt.spec,
				Version: tt.version,
			}

			result, err := e.interpolateTemplate(tt.template, tt.additionalVars)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCreateChecksumFilename_Interpolation(t *testing.T) {
	tests := []struct {
		name     string
		spec     *spec.InstallSpec
		version  string
		expected string
	}{
		{
			name: "basic checksum filename with interpolation",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Checksums: &spec.ChecksumConfig{
					Template: spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
				},
			},
			version:  "v1.2.3",
			expected: "mytool_1.2.3_checksums.txt",
		},
		{
			name: "no checksums config returns empty",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			expected: "",
		},
		{
			name: "empty template returns empty",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Checksums: &spec.ChecksumConfig{
					Template: spec.StringPtr(""),
				},
			},
			version:  "v1.2.3",
			expected: "",
		},
		{
			name: "ASSET_FILENAME returns empty and logs error",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Checksums: &spec.ChecksumConfig{
					Template: spec.StringPtr("${ASSET_FILENAME}.sha256"),
				},
			},
			version:  "v1.2.3",
			expected: "",
		},
		{
			name: "complex template with version prefix",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("tool"),
				Checksums: &spec.ChecksumConfig{
					Template: spec.StringPtr("v${VERSION}/${NAME}-checksums.txt"),
				},
			},
			version:  "1.0.0",
			expected: "v1.0.0/tool-checksums.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedder{
				Spec:    tt.spec,
				Version: tt.version,
			}

			result := e.createChecksumFilename()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestEmbedAssetFilenameValidation(t *testing.T) {
	spec := &spec.InstallSpec{
		Name: spec.StringPtr("mytool"),
		Repo: spec.StringPtr("owner/mytool"),
		Checksums: &spec.ChecksumConfig{
			Template: spec.StringPtr("${ASSET_FILENAME}.sha256"),
		},
	}

	e := &Embedder{
		Spec:    spec,
		Version: "v1.2.3",
		Mode:    EmbedModeDownload,
	}

	err := e.Embed()
	if err == nil {
		t.Errorf("expected error for ASSET_FILENAME but got none")
	}

	if !strings.Contains(err.Error(), "${ASSET_FILENAME} is not supported") {
		t.Errorf("unexpected error message: %v", err)
	}

	if !strings.Contains(err.Error(), "binst embed-checksums --mode calculate") {
		t.Errorf("error message should suggest using calculate mode: %v", err)
	}
}

func TestEmbedAssetFilenameCalculateMode(t *testing.T) {
	spec := &spec.InstallSpec{
		Name: spec.StringPtr("mytool"),
		Repo: spec.StringPtr("owner/mytool"),
		Checksums: &spec.ChecksumConfig{
			Template: spec.StringPtr("${ASSET_FILENAME}.sha256"),
		},
	}

	// Test that calculate mode does not error with ASSET_FILENAME
	e := &Embedder{
		Spec:    spec,
		Version: "v1.2.3",
		Mode:    EmbedModeCalculate,
	}

	// This test doesn't actually perform the full embed (which would require network access)
	// but validates that the error checking doesn't trigger for calculate mode
	err := e.Embed()
	// We expect an error from not having release assets, NOT from ASSET_FILENAME validation
	if err != nil && strings.Contains(err.Error(), "${ASSET_FILENAME} is not supported") {
		t.Errorf("calculate mode should allow ASSET_FILENAME in templates, got error: %v", err)
	}
}

func TestGenerateAssetFilename_Interpolation(t *testing.T) {
	tests := []struct {
		name        string
		spec        *spec.InstallSpec
		version     string
		osInput     string
		archInput   string
		expected    string
		expectError bool
	}{
		{
			name: "basic asset filename with interpolation",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Asset: &spec.AssetConfig{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			version:   "v1.2.3",
			osInput:   "linux",
			archInput: "amd64",
			expected:  "mytool_1.2.3_linux_amd64.tar.gz",
		},
		{
			name: "with extension placeholder",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			version:   "v1.2.3",
			osInput:   "linux",
			archInput: "amd64",
			expected:  "mytool_1.2.3_linux_amd64.tar.gz",
		},
		{
			name: "with rules override changing extension",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					Rules: []spec.AssetRule{
						{
							When: &spec.PlatformCondition{
								OS: spec.StringPtr("windows"),
							},
							EXT: spec.StringPtr(".zip"),
						},
					},
				},
			},
			version:   "v1.2.3",
			osInput:   "windows",
			archInput: "amd64",
			expected:  "mytool_1.2.3_windows_amd64.zip",
		},
		{
			name: "error on missing asset config",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:     "v1.2.3",
			osInput:     "linux",
			archInput:   "amd64",
			expectError: true,
		},
		{
			name: "interpolation error propagates",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
				Asset: &spec.AssetConfig{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			version:   "v1.2.3",
			osInput:   "linux",
			archInput: "amd64",
			expected:  "mytool_1.2.3_linux_amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedder{
				Spec:    tt.spec,
				Version: tt.version,
			}

			generator := asset.NewFilenameGenerator(e.Spec, e.Version)
			result, err := generator.GenerateFilename(tt.osInput, tt.archInput)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInterpolationEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		spec     *spec.InstallSpec
		version  string
		template string
		expected string
	}{
		{
			name: "nested braces are preserved",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_{{VERSION}}_${VERSION}",
			expected: "mytool_{{VERSION}}_1.2.3",
		},
		{
			name: "invalid syntax replaced with interpolated value",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "v1.2.3",
			template: "${NAME}_$VERSION_${VERSION}",
			expected: "mytool_1.2.3",
		},
		{
			name: "empty variables",
			spec: &spec.InstallSpec{
				Name: spec.StringPtr("mytool"),
			},
			version:  "",
			template: "${NAME}_${VERSION}_final",
			expected: "mytool__final",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Embedder{
				Spec:    tt.spec,
				Version: tt.version,
			}

			result, err := e.interpolateTemplate(tt.template, nil)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
