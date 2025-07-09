package cmd

import (
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestValidateSpec(t *testing.T) {
	tests := []struct {
		name        string
		installSpec *spec.InstallSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid spec",
			installSpec: &spec.InstallSpec{
				Repo: spec.StringPtr("owner/repo"),
				Asset: &spec.Asset{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			expectError: false,
		},
		{
			name: "missing repo",
			installSpec: &spec.InstallSpec{
				Asset: &spec.Asset{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			expectError: true,
			errorMsg:    "repo field is required",
		},
		{
			name: "empty repo",
			installSpec: &spec.InstallSpec{
				Repo: spec.StringPtr(""),
				Asset: &spec.Asset{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			expectError: true,
			errorMsg:    "repo field is required",
		},
		{
			name: "invalid repo format",
			installSpec: &spec.InstallSpec{
				Repo: spec.StringPtr("invalid-repo"),
				Asset: &spec.Asset{
					Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
				},
			},
			expectError: true,
			errorMsg:    "repo must be in format 'owner/repo'",
		},
		{
			name: "missing asset config",
			installSpec: &spec.InstallSpec{
				Repo: spec.StringPtr("owner/repo"),
			},
			expectError: true,
			errorMsg:    "asset configuration is required",
		},
		{
			name: "missing asset template",
			installSpec: &spec.InstallSpec{
				Repo: spec.StringPtr("owner/repo"),
				Asset: &spec.Asset{
					Template: spec.StringPtr(""),
				},
			},
			expectError: true,
			errorMsg:    "asset template is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpec(tt.installSpec)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if err != nil && tt.errorMsg != "" {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateAllAssetFilenames(t *testing.T) {
	installSpec := &spec.InstallSpec{
		Repo: spec.StringPtr("owner/repo"),
		Asset: &spec.Asset{
			Template: spec.StringPtr("${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"),
		},
		Name: spec.StringPtr("testapp"),
		SupportedPlatforms: []spec.SupportedPlatformElement{
			{OS: spec.SupportedPlatformOSPtr("linux"), Arch: spec.SupportedPlatformArchPtr("amd64")},
			{OS: spec.SupportedPlatformOSPtr("darwin"), Arch: spec.SupportedPlatformArchPtr("arm64")},
		},
	}

	assetFilenames, err := generateAllAssetFilenames(installSpec, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(assetFilenames) != 2 {
		t.Errorf("expected 2 asset filenames, got %d", len(assetFilenames))
	}
	if _, ok := assetFilenames["linux/amd64"]; !ok {
		t.Errorf("expected linux/amd64 platform")
	}
	if _, ok := assetFilenames["darwin/arm64"]; !ok {
		t.Errorf("expected darwin/arm64 platform")
	}
	if !contains(assetFilenames["linux/amd64"], "testapp_1.0.0_linux_amd64.tar.gz") {
		t.Errorf("expected linux/amd64 filename to contain testapp_1.0.0_linux_amd64.tar.gz")
	}
	if !contains(assetFilenames["darwin/arm64"], "testapp_1.0.0_darwin_arm64.tar.gz") {
		t.Errorf("expected darwin/arm64 filename to contain testapp_1.0.0_darwin_arm64.tar.gz")
	}
}

func TestGetSupportedPlatforms(t *testing.T) {
	t.Run("with custom platforms", func(t *testing.T) {
		installSpec := &spec.InstallSpec{
			SupportedPlatforms: []spec.SupportedPlatformElement{
				{OS: spec.SupportedPlatformOSPtr("linux"), Arch: spec.SupportedPlatformArchPtr("amd64")},
			},
		}

		platforms := getSupportedPlatforms(installSpec)
		if len(platforms) != 1 {
			t.Errorf("expected 1 platform, got %d", len(platforms))
		}
		if spec.PlatformOSString(platforms[0].OS) != "linux" {
			t.Errorf("expected linux OS, got %s", spec.PlatformOSString(platforms[0].OS))
		}
		if spec.PlatformArchString(platforms[0].Arch) != "amd64" {
			t.Errorf("expected amd64 arch, got %s", spec.PlatformArchString(platforms[0].Arch))
		}
	})

	t.Run("with default platforms", func(t *testing.T) {
		installSpec := &spec.InstallSpec{}

		platforms := getSupportedPlatforms(installSpec)
		if len(platforms) != 6 {
			t.Errorf("expected 6 platforms, got %d", len(platforms))
		}

		// Check that we have the expected default platforms
		platformStrs := make([]string, len(platforms))
		for i, p := range platforms {
			platformStrs[i] = spec.PlatformOSString(p.OS) + "/" + spec.PlatformArchString(p.Arch)
		}

		expectedPlatforms := []string{
			"linux/amd64", "linux/arm64",
			"darwin/amd64", "darwin/arm64",
			"windows/amd64", "windows/arm64",
		}

		for _, expected := range expectedPlatforms {
			if !containsString(platformStrs, expected) {
				t.Errorf("expected platform %s not found in %v", expected, platformStrs)
			}
		}
	})
}

func TestGenerateChecksumFilename(t *testing.T) {
	tests := []struct {
		name        string
		installSpec *spec.InstallSpec
		version     string
		wantFile    string
		wantErr     bool
		errorMsg    string
	}{
		{
			name: "valid checksums template",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				Checksums: &spec.Checksums{
					Template: spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
				},
			},
			version:  "1.0.0",
			wantFile: "myapp_1.0.0_checksums.txt",
			wantErr:  false,
		},
		{
			name: "checksums template with TAG variable",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				Checksums: &spec.Checksums{
					Template: spec.StringPtr("${NAME}_${TAG}_SHA256SUMS"),
				},
			},
			version:  "v1.0.0",
			wantFile: "myapp_v1.0.0_SHA256SUMS",
			wantErr:  false,
		},
		{
			name: "checksums template strips v prefix for VERSION",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				Checksums: &spec.Checksums{
					Template: spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
				},
			},
			version:  "v1.0.0",
			wantFile: "myapp_1.0.0_checksums.txt",
			wantErr:  false,
		},
		{
			name: "per-asset checksums pattern",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				Checksums: &spec.Checksums{
					Template: spec.StringPtr("${ASSET_FILENAME}.sha256"),
				},
			},
			version:  "1.0.0",
			wantFile: "",
			wantErr:  true,
			errorMsg: "per-asset checksums",
		},
		{
			name: "no checksums configured",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				// Checksums is nil
			},
			version:  "1.0.0",
			wantFile: "",
			wantErr:  true,
			errorMsg: "checksums template not specified",
		},
		{
			name: "empty checksums template",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("myapp"),
				Checksums: &spec.Checksums{
					Template: spec.StringPtr(""),
				},
			},
			version:  "1.0.0",
			wantFile: "",
			wantErr:  true,
			errorMsg: "checksums template not specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFile, err := generateChecksumFilename(tt.installSpec, tt.version)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if err != nil && tt.errorMsg != "" {
					if !contains(err.Error(), tt.errorMsg) {
						t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if gotFile != tt.wantFile {
					t.Errorf("generateChecksumFilename() = %v, want %v", gotFile, tt.wantFile)
				}
			}
		})
	}
}

func TestIsNonBinaryAsset(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		// Non-binary files
		{"checksums.txt", true},
		{"app_1.0.0_SHA256SUMS", true},
		{"app.sha256", true},
		{"app.sha512", true},
		{"app.md5", true},
		{"app.sig", true},
		{"app.asc", true},
		{"app.pem", true},
		{"app.sbom.json", true},
		{"config.yml", true},
		{"config.yaml", true},
		{"install.sh", true},
		{"install.ps1", true},
		{"README.md", true},
		{"binst-0.2.5.tar.gz", true}, // source archive
		{"binst-v0.2.5.zip", true},   // source archive
		
		// Binary files
		{"app_linux_amd64.tar.gz", false},
		{"app_darwin_arm64.tar.gz", false},
		{"app_windows_amd64.zip", false},
		{"app-linux-amd64", false},
		{"app.exe", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			if got := isNonBinaryAsset(tt.filename); got != tt.want {
				t.Errorf("isNonBinaryAsset(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// Integration test for the check command
func TestCheckCommand(t *testing.T) {
	// Skip integration tests as they require complex setup with cobra
	t.Skip("Integration tests require proper cobra command setup")
}

// Test helper function for removeFromSlice
func TestRemoveFromSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected []string
	}{
		{
			name:     "remove existing item",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: []string{"a", "c"},
		},
		{
			name:     "remove non-existing item",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "remove from empty slice",
			slice:    []string{},
			item:     "a",
			expected: []string{},
		},
		{
			name:     "remove duplicate items",
			slice:    []string{"a", "b", "b", "c"},
			item:     "b",
			expected: []string{"a", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeFromSlice(tt.slice, tt.item)
			if len(result) != len(tt.expected) {
				t.Errorf("removeFromSlice() returned %d items, want %d", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("removeFromSlice()[%d] = %v, want %v", i, v, tt.expected[i])
				}
			}
		})
	}
}

// Helper functions for testing
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
