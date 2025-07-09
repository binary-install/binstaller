package checksums

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
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

func TestFilterChecksums(t *testing.T) {
	// Create a test spec with supported platforms
	osLowercase := spec.OSLowercase
	archLowercase := spec.ArchLowercase
	linux := spec.Linux
	darwin := spec.Darwin
	windows := spec.Windows
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
			Rules: []spec.AssetRule{
				{
					When: &spec.PlatformCondition{
						OS: spec.StringPtr("windows"),
					},
					EXT: spec.StringPtr(".zip"),
				},
			},
		},
		SupportedPlatforms: []spec.Platform{
			{OS: &linux, Arch: &amd64},
			{OS: &darwin, Arch: &amd64},
			{OS: &darwin, Arch: &arm64},
			{OS: &windows, Arch: &amd64},
		},
	}

	embedder := &Embedder{
		Spec:    testSpec,
		Version: "1.0.0",
	}

	// Create checksums map with valid and invalid entries
	checksums := map[string]string{
		"test-tool-1.0.0-linux-amd64.tar.gz":  "abc123", // Valid
		"test-tool-1.0.0-darwin-amd64.tar.gz": "def456", // Valid
		"test-tool-1.0.0-darwin-arm64.tar.gz": "ghi789", // Valid
		"test-tool-1.0.0-windows-amd64.zip":   "jkl012", // Valid (rule applied)
		"test-tool-1.0.0-linux-386.tar.gz":    "mno345", // Invalid (unsupported platform)
		"README.md":                           "pqr678", // Invalid (not an asset)
		"checksums.txt":                       "stu901", // Invalid (not an asset)
		"test-tool-1.0.0.deb":                 "vwx234", // Invalid (different format)
	}

	// Filter checksums
	filtered := embedder.filterChecksums(checksums)

	// Verify only valid entries remain
	expected := map[string]string{
		"test-tool-1.0.0-linux-amd64.tar.gz":  "abc123",
		"test-tool-1.0.0-darwin-amd64.tar.gz": "def456",
		"test-tool-1.0.0-darwin-arm64.tar.gz": "ghi789",
		"test-tool-1.0.0-windows-amd64.zip":   "jkl012",
	}

	if len(filtered) != len(expected) {
		t.Errorf("Expected %d filtered checksums, got %d", len(expected), len(filtered))
	}

	for filename, expectedHash := range expected {
		actualHash, ok := filtered[filename]
		if !ok {
			t.Errorf("Missing checksum for valid file %s", filename)
			continue
		}
		if actualHash != expectedHash {
			t.Errorf("Checksum mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
		}
	}

	// Verify invalid entries were filtered out
	for filename := range checksums {
		if _, shouldExist := expected[filename]; !shouldExist {
			if _, exists := filtered[filename]; exists {
				t.Errorf("Invalid file %s should have been filtered out", filename)
			}
		}
	}
}

func TestFilterChecksumsNoAssetTemplate(t *testing.T) {
	// Test filtering when no asset template is defined
	testSpec := &spec.InstallSpec{
		Name: spec.StringPtr("test-tool"),
		Repo: spec.StringPtr("test-owner/test-repo"),
		// No Asset config
	}

	embedder := &Embedder{
		Spec:    testSpec,
		Version: "1.0.0",
	}

	// Create checksums map
	checksums := map[string]string{
		"test-tool-1.0.0-linux-amd64.tar.gz": "abc123",
		"README.md":                          "def456",
	}

	// Filter checksums - should return all since no template
	filtered := embedder.filterChecksums(checksums)

	if len(filtered) != len(checksums) {
		t.Errorf("Expected all checksums to be returned when no asset template, got %d of %d",
			len(filtered), len(checksums))
	}
}


func TestEmbedder_EmbedWithMissingChecksumsField(t *testing.T) {
	// Test that the embed command works when checksums field is missing (GitHub issue #84)
	tempDir, err := os.MkdirTemp("", "embed-checksums-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config file WITHOUT a checksums field (mimics `binst init --source=github`)
	configFile := filepath.Join(tempDir, "config.yml")
	configContent := `name: test-tool
repo: test-owner/test-repo
asset:
  template: ${NAME}-${VERSION}-${OS}-${ARCH}${EXT}
  default_extension: .tar.gz
  naming_convention:
    os: lowercase
    arch: lowercase
  rules:
    - when:
        os: windows
      ext: .zip
supported_platforms:
  - os: linux
    arch: amd64
  - os: darwin
    arch: amd64
  - os: windows
    arch: amd64
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Create a temporary checksum file
	checksumFile := filepath.Join(tempDir, "checksums.txt")
	checksumContent := `abc123  test-tool-1.0.0-linux-amd64.tar.gz
def456  test-tool-1.0.0-darwin-amd64.tar.gz
ghi789  test-tool-1.0.0-windows-amd64.zip
`
	if err := os.WriteFile(checksumFile, []byte(checksumContent), 0644); err != nil {
		t.Fatalf("Failed to write checksum file: %v", err)
	}

	// Parse the config file
	ast, err := parser.ParseFile(configFile, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	yamlData, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var installSpec spec.InstallSpec
	if err := yaml.UnmarshalWithOptions(yamlData, &installSpec, yaml.UseOrderedMap()); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// Verify that checksums field doesn't exist initially
	if installSpec.Checksums != nil {
		t.Error("Expected checksums field to be nil initially")
	}

	// Create embedder with checksum-file mode
	embedder := &Embedder{
		Mode:         EmbedModeChecksumFile,
		Version:      "1.0.0",
		Spec:         &installSpec,
		SpecAST:      ast,
		ChecksumFile: checksumFile,
	}

	// This should not fail even though checksums field is missing
	// Previously this would fail with "failed to find path ( $.checksums ): node not found"
	if err := embedder.Embed(); err != nil {
		t.Fatalf("Embed() failed: %v", err)
	}

	// Verify that checksums field was created
	if installSpec.Checksums == nil {
		t.Error("Expected checksums field to be created")
	}

	// Verify that checksums were embedded
	if installSpec.Checksums.EmbeddedChecksums == nil {
		t.Error("Expected embedded checksums to be created")
	}

	embeddedForVersion := installSpec.Checksums.EmbeddedChecksums["1.0.0"]
	if len(embeddedForVersion) == 0 {
		t.Error("Expected embedded checksums for version 1.0.0")
	}

	// Verify specific checksums
	expectedChecksums := map[string]string{
		"test-tool-1.0.0-linux-amd64.tar.gz":  "abc123",
		"test-tool-1.0.0-darwin-amd64.tar.gz": "def456",
		"test-tool-1.0.0-windows-amd64.zip":   "ghi789",
	}

	actualChecksums := make(map[string]string)
	for _, checksum := range embeddedForVersion {
		actualChecksums[spec.StringValue(checksum.Filename)] = spec.StringValue(checksum.Hash)
	}

	for expectedFilename, expectedHash := range expectedChecksums {
		actualHash, exists := actualChecksums[expectedFilename]
		if !exists {
			t.Errorf("Expected checksum for %s not found", expectedFilename)
		} else if actualHash != expectedHash {
			t.Errorf("Checksum mismatch for %s: expected %s, got %s", expectedFilename, expectedHash, actualHash)
		}
	}
}

