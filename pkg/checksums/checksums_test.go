package checksums

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestParseChecksumFileInternal(t *testing.T) {
	// Create a temporary file with test checksums
	tempDir, err := os.MkdirTemp("", "checksums-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test checksum file
	checksumFile := filepath.Join(tempDir, "checksums.txt")
	content := `
# Test checksums
abc123 test-1.0.0-linux-amd64.tar.gz
def456  test-1.0.0-darwin-amd64.tar.gz
ghi789 *test-1.0.0-windows-amd64.zip
`
	if err := os.WriteFile(checksumFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test checksum file: %v", err)
	}

	// Parse the checksum file
	checksums, err := parseChecksumFileInternal(checksumFile)
	if err != nil {
		t.Fatalf("parseChecksumFileInternal failed: %v", err)
	}

	// Verify the parsed checksums
	expected := map[string]string{
		"test-1.0.0-linux-amd64.tar.gz":  "abc123",
		"test-1.0.0-darwin-amd64.tar.gz": "def456",
		"test-1.0.0-windows-amd64.zip":   "ghi789",
	}

	if len(checksums) != len(expected) {
		t.Errorf("Expected %d checksums, got %d", len(expected), len(checksums))
	}

	for filename, expectedHash := range expected {
		actualHash, ok := checksums[filename]
		if !ok {
			t.Errorf("Missing checksum for %s", filename)
			continue
		}
		if actualHash != expectedHash {
			t.Errorf("Checksum mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
		}
	}
}

func TestGenerateAssetFilename(t *testing.T) {
	// Create a test spec
	testSpec := &spec.InstallSpec{
		Name: "test-tool",
		Repo: "test-owner/test-repo",
		Asset: spec.AssetConfig{
			Template:         "${NAME}-${VERSION}-${OS}-${ARCH}${EXT}",
			DefaultExtension: ".tar.gz",
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
	}

	// Create an embedder with the test spec
	embedder := &Embedder{
		Spec:    testSpec,
		Version: "1.0.0",
	}

	// Test basic filename generation
	filename, err := embedder.generateAssetFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected := "test-tool-1.0.0-linux-amd64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test with titlecase OS
	testSpec.Asset.NamingConvention.OS = "titlecase"
	filename, err = embedder.generateAssetFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected = "test-tool-1.0.0-Linux-amd64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test with rules
	testSpec.Asset.Rules = []spec.AssetRule{
		{
			When: spec.PlatformCondition{
				OS: "windows",
			},
			Ext: ".zip",
		},
	}
	filename, err = embedder.generateAssetFilename("windows", "amd64")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected = "test-tool-1.0.0-Windows-amd64.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}
}

func TestGenerateAssetFilenameMultipleRules(t *testing.T) {
	// Test the bug fix where multiple rules should apply cumulatively
	testSpec := &spec.InstallSpec{
		Name: "binst",
		Repo: "binary-install/binstaller",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz",
			NamingConvention: &spec.NamingConvention{
				OS:   "titlecase",
				Arch: "lowercase",
			},
			Rules: []spec.AssetRule{
				// First rule: transform amd64 to x86_64
				{
					When: spec.PlatformCondition{
						Arch: "amd64",
					},
					Arch: "x86_64",
				},
				// Second rule: Windows uses .zip extension
				{
					When: spec.PlatformCondition{
						OS: "windows",
					},
					Ext: ".zip",
				},
			},
		},
	}

	embedder := &Embedder{
		Spec:    testSpec,
		Version: "v0.1.0",
	}

	// Test Windows amd64 - should apply BOTH rules
	filename, err := embedder.generateAssetFilename("windows", "amd64")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected := "binst_Windows_x86_64.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test Linux amd64 - should only apply the arch transformation rule
	filename, err = embedder.generateAssetFilename("linux", "amd64")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected = "binst_Linux_x86_64.tar.gz"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}

	// Test Windows 386 - should only apply the extension rule
	filename, err = embedder.generateAssetFilename("windows", "386")
	if err != nil {
		t.Fatalf("generateAssetFilename failed: %v", err)
	}
	expected = "binst_Windows_386.zip"
	if filename != expected {
		t.Errorf("Expected filename %s, got %s", expected, filename)
	}
}

func TestComputeHash(t *testing.T) {
	// Create a temporary file with known content
	tempDir, err := os.MkdirTemp("", "checksums-hash-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := "hello world"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Known hashes for "hello world"
	expectedHashes := map[string]string{
		"sha256": "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		"sha1":   "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
	}

	// Test computing different hashes
	for algo, expected := range expectedHashes {
		hash, err := ComputeHash(testFile, algo)
		if err != nil {
			t.Fatalf("ComputeHash failed for %s: %v", algo, err)
		}
		if hash != expected {
			t.Errorf("Hash mismatch for %s: expected %s, got %s", algo, expected, hash)
		}
	}

	// Test with unsupported algorithm
	_, err = ComputeHash(testFile, "unsupported")
	if err == nil {
		t.Error("Expected error for unsupported algorithm, got nil")
	}
}
