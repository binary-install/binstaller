package cmd

import (
	"context"
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

func TestDisplayUnmatchedAssets(t *testing.T) {
	// This test is more about ensuring the function doesn't panic
	// and handles edge cases properly
	
	releaseAssets := []string{
		"app_1.0.0_linux_amd64.tar.gz",
		"app_1.0.0_darwin_arm64.tar.gz",
		"checksums.txt",
		"README.md",
	}
	
	assetFilenames := map[string]string{
		"linux/amd64":  "app_1.0.0_linux_amd64.tar.gz",
		"darwin/arm64": "app_1.0.0_darwin_arm64.tar.gz",
	}
	
	// This should not panic and should identify checksums.txt and README.md as unmatched
	displayUnmatchedAssets(releaseAssets, assetFilenames)
}

func TestDisplayUnmatchedAssetsEmpty(t *testing.T) {
	// Test with empty inputs
	displayUnmatchedAssets([]string{}, map[string]string{})
	
	// Test with no unmatched assets
	releaseAssets := []string{"app_1.0.0_linux_amd64.tar.gz"}
	assetFilenames := map[string]string{"linux/amd64": "app_1.0.0_linux_amd64.tar.gz"}
	displayUnmatchedAssets(releaseAssets, assetFilenames)
}

// Mock test for fetchReleaseAssets would require HTTP mocking
// This is a placeholder showing how such a test could be structured
func TestFetchReleaseAssets(t *testing.T) {
	t.Skip("Skipping integration test - requires HTTP mocking")
	
	ctx := context.Background()
	
	// This would need to be mocked to test properly
	assets, err := fetchReleaseAssets(ctx, "owner/repo", "v1.0.0")
	
	// With proper mocking, we would test:
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(assets) == 0 {
		t.Errorf("expected non-empty assets")
	}
}

func TestResolveLatestVersion(t *testing.T) {
	t.Skip("Skipping integration test - requires HTTP mocking")
	
	ctx := context.Background()
	
	// This would need to be mocked to test properly
	version, err := resolveLatestVersion(ctx, "owner/repo")
	
	// With proper mocking, we would test:
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if version == "" {
		t.Errorf("expected non-empty version")
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