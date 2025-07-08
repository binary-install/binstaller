package checksums

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/google/go-cmp/cmp"
)

// Helper functions for creating platform pointers
func platformOSPtr(s string) *spec.SupportedPlatformOS {
	os := spec.SupportedPlatformOS(s)
	return &os
}

func platformArchPtr(s string) *spec.SupportedPlatformArch {
	arch := spec.SupportedPlatformArch(s)
	return &arch
}

func TestEmbedder_filterChecksums_VersionHandling(t *testing.T) {
	tests := []struct {
		name             string
		version          string
		assetTemplate    string
		checksums        map[string]string
		expectedFiltered map[string]string
	}{
		{
			name:          "version with v prefix correctly strips v for VERSION variable",
			version:       "v0.15.0",
			assetTemplate: "gum_${VERSION}_${OS}_${ARCH}${EXT}",
			checksums: map[string]string{
				"gum_0.15.0_Linux_x86_64.tar.gz":   "checksum1",
				"gum_0.15.0_Darwin_x86_64.tar.gz":  "checksum2",
				"gum_0.15.0_Windows_x86_64.zip":    "checksum3",
				"gum_v0.15.0_Linux_x86_64.tar.gz":  "checksum4",
				"gum_v0.15.0_Darwin_x86_64.tar.gz": "checksum5",
				"other_file.txt":                   "checksum6",
			},
			expectedFiltered: map[string]string{
				"gum_0.15.0_Linux_x86_64.tar.gz":  "checksum1",
				"gum_0.15.0_Darwin_x86_64.tar.gz": "checksum2",
				"gum_0.15.0_Windows_x86_64.zip":   "checksum3",
			},
		},
		{
			name:          "version without v prefix matches assets without v",
			version:       "0.15.0",
			assetTemplate: "gum_${VERSION}_${OS}_${ARCH}${EXT}",
			checksums: map[string]string{
				"gum_0.15.0_Linux_x86_64.tar.gz":   "checksum1",
				"gum_0.15.0_Darwin_x86_64.tar.gz":  "checksum2",
				"gum_0.15.0_Windows_x86_64.zip":    "checksum3",
				"gum_v0.15.0_Linux_x86_64.tar.gz":  "checksum4",
				"gum_v0.15.0_Darwin_x86_64.tar.gz": "checksum5",
				"other_file.txt":                   "checksum6",
			},
			expectedFiltered: map[string]string{
				"gum_0.15.0_Linux_x86_64.tar.gz":  "checksum1",
				"gum_0.15.0_Darwin_x86_64.tar.gz": "checksum2",
				"gum_0.15.0_Windows_x86_64.zip":   "checksum3",
			},
		},
		{
			name:          "template using TAG variable preserves v prefix",
			version:       "v0.15.0",
			assetTemplate: "gum_${TAG}_${OS}_${ARCH}${EXT}",
			checksums: map[string]string{
				"gum_0.15.0_Linux_x86_64.tar.gz":   "checksum1",
				"gum_0.15.0_Darwin_x86_64.tar.gz":  "checksum2",
				"gum_0.15.0_Windows_x86_64.zip":    "checksum3",
				"gum_v0.15.0_Linux_x86_64.tar.gz":  "checksum4",
				"gum_v0.15.0_Darwin_x86_64.tar.gz": "checksum5",
				"gum_v0.15.0_Windows_x86_64.zip":   "checksum6",
				"other_file.txt":                   "checksum7",
			},
			expectedFiltered: map[string]string{
				"gum_v0.15.0_Linux_x86_64.tar.gz":  "checksum4",
				"gum_v0.15.0_Darwin_x86_64.tar.gz": "checksum5",
				"gum_v0.15.0_Windows_x86_64.zip":   "checksum6",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal InstallSpec
			installSpec := &spec.InstallSpec{
				Name: spec.StringPtr("gum"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr(tt.assetTemplate),
					DefaultExtension: spec.StringPtr(".tar.gz"),
					Rules: []spec.AssetRule{
						{
							When: &spec.When{
								OS: spec.StringPtr("darwin"),
							},
							OS: spec.StringPtr("Darwin"),
						},
						{
							When: &spec.When{
								OS: spec.StringPtr("linux"),
							},
							OS: spec.StringPtr("Linux"),
						},
						{
							When: &spec.When{
								OS: spec.StringPtr("windows"),
							},
							OS:  spec.StringPtr("Windows"),
							EXT: spec.StringPtr(".zip"),
						},
						{
							When: &spec.When{
								Arch: spec.StringPtr("amd64"),
							},
							Arch: spec.StringPtr("x86_64"),
						},
					},
				},
				SupportedPlatforms: []spec.Platform{
					{OS: platformOSPtr("linux"), Arch: platformArchPtr("amd64")},
					{OS: platformOSPtr("darwin"), Arch: platformArchPtr("amd64")},
					{OS: platformOSPtr("windows"), Arch: platformArchPtr("amd64")},
				},
			}

			embedder := &Embedder{
				Version: tt.version,
				Spec:    installSpec,
			}

			filtered := embedder.filterChecksums(tt.checksums)
			if diff := cmp.Diff(tt.expectedFiltered, filtered); diff != "" {
				t.Errorf("filterChecksums() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEmbedder_parseChecksumFile(t *testing.T) {
	// Create a temporary checksum file
	tempDir := t.TempDir()
	checksumFile := filepath.Join(tempDir, "checksums.txt")

	checksumContent := `# SHA256 checksums
a1b2c3d4  gum_0.15.0_Linux_x86_64.tar.gz
e5f6g7h8  gum_0.15.0_Darwin_x86_64.tar.gz
i9j0k1l2  gum_0.15.0_Windows_x86_64.zip
m3n4o5p6  *gum_0.15.0_Linux_arm64.tar.gz

q7r8s9t0  other_file.txt
`
	err := os.WriteFile(checksumFile, []byte(checksumContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write checksum file: %v", err)
	}

	// Test parsing the checksum file
	checksums, err := parseChecksumFileInternal(checksumFile)
	if err != nil {
		t.Fatalf("parseChecksumFileInternal() error = %v", err)
	}

	expected := map[string]string{
		"gum_0.15.0_Linux_x86_64.tar.gz":  "a1b2c3d4",
		"gum_0.15.0_Darwin_x86_64.tar.gz": "e5f6g7h8",
		"gum_0.15.0_Windows_x86_64.zip":   "i9j0k1l2",
		"gum_0.15.0_Linux_arm64.tar.gz":   "m3n4o5p6",
		"other_file.txt":                  "q7r8s9t0",
	}

	if diff := cmp.Diff(expected, checksums); diff != "" {
		t.Errorf("parseChecksumFileInternal() mismatch (-want +got):\n%s", diff)
	}
}

func TestEmbedder_createChecksumFilename(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		template string
		expected string
	}{
		{
			name:     "simple template",
			version:  "v0.15.0",
			template: "checksums.txt",
			expected: "checksums.txt",
		},
		{
			name:     "template with version strips v prefix",
			version:  "v0.15.0",
			template: "${NAME}_${VERSION}_checksums.txt",
			expected: "gum_0.15.0_checksums.txt",
		},
		{
			name:     "template with version without v",
			version:  "0.15.0",
			template: "${NAME}_${VERSION}_checksums.txt",
			expected: "gum_0.15.0_checksums.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installSpec := &spec.InstallSpec{
				Name: spec.StringPtr("gum"),
				Checksums: &spec.ChecksumConfig{
					Template: spec.StringPtr(tt.template),
				},
			}

			embedder := &Embedder{
				Version: tt.version,
				Spec:    installSpec,
			}

			result := embedder.createChecksumFilename()
			if result != tt.expected {
				t.Errorf("createChecksumFilename() = %v, want %v", result, tt.expected)
			}
		})
	}
}
