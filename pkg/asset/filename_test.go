package asset

import (
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestGenerateFilename(t *testing.T) {
	// Create a test spec
	osLowercase := spec.OSLowercase
	archLowercase := spec.ArchLowercase
	testSpec := &spec.InstallSpec{
		Name: spec.StringPtr("test-tool"),
		Repo: spec.StringPtr("test-owner/test-repo"),
		Asset: &spec.AssetConfig{
			Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}${EXT}"),
			DefaultExtension: spec.StringPtr(".tar.gz"),
			NamingConvention: &spec.NamingConvention{
				OS:   &osLowercase,
				Arch: &archLowercase,
			},
		},
	}

	generator := NewFilenameGenerator(testSpec, "1.0.0")

	// Test basic filename generation
	filename, err := generator.GenerateFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected := "test-tool-1.0.0-linux-amd64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test with titlecase OS
	titlecase := spec.Titlecase
	testSpec.Asset.NamingConvention.OS = &titlecase
	filename, err = generator.GenerateFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected = "test-tool-1.0.0-Linux-amd64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test with rules
	testSpec.Asset.Rules = []spec.AssetRule{
		{
			When: &spec.PlatformCondition{
				OS: spec.StringPtr("windows"),
			},
			EXT: spec.StringPtr(".zip"),
		},
	}
	filename, err = generator.GenerateFilename("windows", "amd64")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected = "test-tool-1.0.0-Windows-amd64.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}
}

func TestGenerateFilenameMultipleRules(t *testing.T) {
	// Test the bug fix where multiple rules should apply cumulatively
	titlecase := spec.Titlecase
	archLowercase := spec.ArchLowercase
	testSpec := &spec.InstallSpec{
		Name: spec.StringPtr("binst"),
		Repo: spec.StringPtr("binary-install/binstaller"),
		Asset: &spec.AssetConfig{
			Template:         spec.StringPtr("${NAME}_${OS}_${ARCH}${EXT}"),
			DefaultExtension: spec.StringPtr(".tar.gz"),
			NamingConvention: &spec.NamingConvention{
				OS:   &titlecase,
				Arch: &archLowercase,
			},
			Rules: []spec.AssetRule{
				// First rule: transform amd64 to x86_64
				{
					When: &spec.PlatformCondition{
						Arch: spec.StringPtr("amd64"),
					},
					Arch: spec.StringPtr("x86_64"),
				},
				// Second rule: Windows uses .zip extension
				{
					When: &spec.PlatformCondition{
						OS: spec.StringPtr("windows"),
					},
					EXT: spec.StringPtr(".zip"),
				},
			},
		},
	}

	generator := NewFilenameGenerator(testSpec, "v0.1.0")

	// Test Windows amd64 - should apply BOTH rules
	filename, err := generator.GenerateFilename("windows", "amd64")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected := "binst_Windows_x86_64.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test Linux amd64 - should only apply the arch transformation rule
	filename, err = generator.GenerateFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected = "binst_Linux_x86_64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test Windows 386 - should only apply the extension rule
	filename, err = generator.GenerateFilename("windows", "386")
	if err != nil {
		t.Fatalf("GenerateFilename failed: %v", err)
	}
	expected = "binst_Windows_386.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}
}

func TestGeneratePossibleFilenames(t *testing.T) {
	// Create a test spec
	osLowercase := spec.OSLowercase
	archLowercase := spec.ArchLowercase
	linux := spec.Linux
	darwin := spec.Darwin
	amd64 := spec.Amd64
	arm64 := spec.Arm64
	
	testSpec := &spec.InstallSpec{
		Name: spec.StringPtr("test-tool"),
		Repo: spec.StringPtr("test-owner/test-repo"),
		Asset: &spec.AssetConfig{
			Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}${EXT}"),
			DefaultExtension: spec.StringPtr(".tar.gz"),
			NamingConvention: &spec.NamingConvention{
				OS:   &osLowercase,
				Arch: &archLowercase,
			},
		},
		SupportedPlatforms: []spec.Platform{
			{OS: &linux, Arch: &amd64},
			{OS: &darwin, Arch: &amd64},
			{OS: &darwin, Arch: &arm64},
		},
	}

	generator := NewFilenameGenerator(testSpec, "1.0.0")

	// Generate possible filenames
	filenames := generator.GeneratePossibleFilenames()

	// Verify expected filenames
	expected := map[string]bool{
		"test-tool-1.0.0-linux-amd64.tar.gz":  true,
		"test-tool-1.0.0-darwin-amd64.tar.gz": true,
		"test-tool-1.0.0-darwin-arm64.tar.gz": true,
	}

	if len(filenames) != len(expected) {
		t.Errorf("Expected %d filenames, got %d", len(expected), len(filenames))
	}

	// Check all expected filenames are present
	for expectedFile := range expected {
		if !filenames[expectedFile] {
			t.Errorf("Expected filename %s not found in generated map", expectedFile)
		}
	}
}

func TestGeneratePossibleFilenamesAllPlatforms(t *testing.T) {
	// Test with no supported platforms - should generate all possible combinations
	osLowercase := spec.OSLowercase
	archLowercase := spec.ArchLowercase
	
	testSpec := &spec.InstallSpec{
		Name: spec.StringPtr("test-tool"),
		Repo: spec.StringPtr("test-owner/test-repo"),
		Asset: &spec.AssetConfig{
			Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}${EXT}"),
			DefaultExtension: spec.StringPtr(".tar.gz"),
			NamingConvention: &spec.NamingConvention{
				OS:   &osLowercase,
				Arch: &archLowercase,
			},
		},
		// No SupportedPlatforms specified
	}

	generator := NewFilenameGenerator(testSpec, "1.0.0")

	// Generate possible filenames
	filenames := generator.GeneratePossibleFilenames()

	// Should have generated filenames for all OS/Arch combinations
	allOSCount := len(GetAllOSValues())
	allArchCount := len(GetAllArchValues())
	expectedCount := allOSCount * allArchCount

	if len(filenames) != expectedCount {
		t.Errorf("Expected %d filenames (all %d OS x %d Arch combinations), got %d", 
			expectedCount, allOSCount, allArchCount, len(filenames))
	}

	// Check a few specific combinations
	expectedSamples := []string{
		"test-tool-1.0.0-linux-amd64.tar.gz",
		"test-tool-1.0.0-darwin-arm64.tar.gz",
		"test-tool-1.0.0-windows-amd64.tar.gz",
		"test-tool-1.0.0-freebsd-386.tar.gz",
	}

	for _, sample := range expectedSamples {
		if !filenames[sample] {
			t.Errorf("Expected filename %s not found in generated map", sample)
		}
	}
}