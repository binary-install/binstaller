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

func TestIsIgnoredAsset(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		patterns []string
		want     bool
	}{
		// Documentation and metadata
		{"default patterns: README.md", "README.md", nil, true},
		{"default patterns: README", "README", nil, true},
		{"default patterns: LICENSE", "LICENSE", nil, true},
		{"default patterns: LICENSE.txt", "LICENSE.txt", nil, true},
		{"default patterns: LICENSE.md", "LICENSE.md", nil, true},
		{"default patterns: CHANGELOG.md", "CHANGELOG.md", nil, true},
		{"default patterns: NOTICE", "NOTICE", nil, true},

		// Signatures and checksums
		{"default patterns: checksums.txt", "checksums.txt", nil, true},
		{"default patterns: SHA256SUMS", "app_1.0.0_SHA256SUMS", nil, true},
		{"default patterns: .sha256", "app.sha256", nil, true},
		{"default patterns: .sha512", "app.sha512", nil, true},
		{"default patterns: .md5", "app.md5", nil, true},
		{"default patterns: .sig", "app.sig", nil, true},
		{"default patterns: .asc", "app.asc", nil, true},
		{"default patterns: .pem", "app.pem", nil, true},

		// SBOM and metadata
		{"default patterns: .sbom.json", "app.sbom.json", nil, true},
		{"default patterns: .yml", "config.yml", nil, true},
		{"default patterns: .yaml", "config.yaml", nil, true},

		// Scripts
		{"default patterns: .sh", "install.sh", nil, true},
		{"default patterns: .ps1", "install.ps1", nil, true},
		{"default patterns: .bat", "setup.bat", nil, true},

		// Package formats
		{"default patterns: .deb", "app_amd64.deb", nil, true},
		{"default patterns: .rpm", "app-1.0.0.rpm", nil, true},
		{"default patterns: .pkg", "app.pkg", nil, true},
		{"default patterns: .dmg", "app.dmg", nil, true},
		{"default patterns: .msi", "app-installer.msi", nil, true},
		{"default patterns: .apk", "app.apk", nil, true},
		{"default patterns: .snap", "app.snap", nil, true},
		{"default patterns: .flatpak", "app.flatpak", nil, true},

		// Development files
		{"default patterns: .pdb", "app.pdb", nil, true},
		{"default patterns: .debug", "app.debug", nil, true},

		// Source archives
		{"default patterns: source archive", "binst-0.2.5.tar.gz", nil, true},
		{"default patterns: source archive with v", "binst-v0.2.5.zip", nil, true},

		// Binary files
		{"default patterns: linux binary", "app_linux_amd64.tar.gz", nil, false},
		{"default patterns: darwin binary", "app_darwin_arm64.tar.gz", nil, false},
		{"default patterns: windows binary", "app_windows_amd64.zip", nil, false},
		{"default patterns: binary without ext", "app-linux-amd64", nil, false},
		{"default patterns: exe", "app.exe", nil, false},

		// Custom patterns
		{"custom pattern: AppImage", "app.AppImage", []string{`\.AppImage$`}, true},
		{"custom pattern: musl variants", "bat-musl_0.25.0_arm64.deb", []string{`.*-musl.*`}, true},
		{"custom pattern: test prefix", "test-app-linux.tar.gz", []string{`^test-`}, true},
		{"custom pattern: multiple patterns", "debug-app.tar.gz", []string{`^debug-`, `\.AppImage$`}, true},
		{"custom pattern: no match", "app_linux_amd64.tar.gz", []string{`\.AppImage$`}, false},

		// Invalid regex (should be ignored and return false)
		{"invalid regex", "app.tar.gz", []string{`[`}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIgnoredAsset(tt.filename, tt.patterns); got != tt.want {
				t.Errorf("isIgnoredAsset(%q, %v) = %v, want %v", tt.filename, tt.patterns, got, tt.want)
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
