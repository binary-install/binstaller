package shell

import (
	"strings"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestGenerateWithVersion(t *testing.T) {
	tests := []struct {
		name           string
		installSpec    *spec.InstallSpec
		targetVersion  string
		wantSubstrings []string
		wantNotContain []string
	}{
		{
			name: "normal generation without target version",
			installSpec: &spec.InstallSpec{
				Name: "test-tool",
				Repo: "owner/test-tool",
				Asset: spec.AssetConfig{
					Template:         "${NAME}-${VERSION}-${OS}_${ARCH}${EXT}",
					DefaultExtension: ".tar.gz",
				},
			},
			targetVersion: "",
			wantSubstrings: []string{
				`TAG="${1:-latest}"`,
				`if [ "$TAG" = "latest" ]; then`,
				`log_info "checking GitHub for latest tag"`,
			},
			wantNotContain: []string{
				`TAG="v1.2.3"`,
				`Installing ${NAME} version ${VERSION}`,
			},
		},
		{
			name: "target version generation",
			installSpec: &spec.InstallSpec{
				Name: "test-tool",
				Repo: "owner/test-tool",
				Asset: spec.AssetConfig{
					Template:         "${NAME}-${VERSION}-${OS}_${ARCH}${EXT}",
					DefaultExtension: ".tar.gz",
				},
			},
			targetVersion: "v1.2.3",
			wantSubstrings: []string{
				`TAG="v1.2.3"`,
				`REALTAG="v1.2.3"`,
				`Installing ${NAME} version ${VERSION}`,
				`This installer is configured for v1.2.3 only.`,
			},
			wantNotContain: []string{
				`TAG="${1:-latest}"`,
				`if [ "$TAG" = "latest" ]; then`,
				`log_info "checking GitHub for latest tag"`,
				`[tag] is a tag from`,
			},
		},
		{
			name: "target version with embedded checksums filtering",
			installSpec: &spec.InstallSpec{
				Name: "test-tool",
				Repo: "owner/test-tool",
				Asset: spec.AssetConfig{
					Template:         "${NAME}-${VERSION}-${OS}_${ARCH}${EXT}",
					DefaultExtension: ".tar.gz",
				},
				Checksums: &spec.ChecksumConfig{
					Algorithm: "sha256",
					Template:  "${NAME}_${VERSION}_checksums.txt",
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: "test-tool-1.2.3-linux_amd64.tar.gz", Hash: "abc123"},
							{Filename: "test-tool-1.2.3-darwin_amd64.tar.gz", Hash: "def456"},
						},
						"v1.2.4": {
							{Filename: "test-tool-1.2.4-linux_amd64.tar.gz", Hash: "ghi789"},
						},
					},
				},
			},
			targetVersion: "v1.2.3",
			wantSubstrings: []string{
				`1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123`,
				`1.2.3:test-tool-1.2.3-darwin_amd64.tar.gz:def456`,
			},
			wantNotContain: []string{
				`1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:ghi789`,
			},
		},
		{
			name: "normal generation includes all embedded checksums",
			installSpec: &spec.InstallSpec{
				Name: "test-tool",
				Repo: "owner/test-tool",
				Asset: spec.AssetConfig{
					Template:         "${NAME}-${VERSION}-${OS}_${ARCH}${EXT}",
					DefaultExtension: ".tar.gz",
				},
				Checksums: &spec.ChecksumConfig{
					Algorithm: "sha256",
					Template:  "${NAME}_${VERSION}_checksums.txt",
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: "test-tool-1.2.3-linux_amd64.tar.gz", Hash: "abc123"},
						},
						"v1.2.4": {
							{Filename: "test-tool-1.2.4-linux_amd64.tar.gz", Hash: "ghi789"},
						},
					},
				},
			},
			targetVersion: "",
			wantSubstrings: []string{
				`1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123`,
				`1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:ghi789`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateWithVersion(tt.installSpec, tt.targetVersion)
			if err != nil {
				t.Fatalf("GenerateWithVersion() error = %v", err)
			}

			gotStr := string(got)

			// Check for expected substrings
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(gotStr, want) {
					t.Errorf("GenerateWithVersion() missing expected substring: %q", want)
				}
			}

			// Check for unexpected substrings
			for _, unwanted := range tt.wantNotContain {
				if strings.Contains(gotStr, unwanted) {
					t.Errorf("GenerateWithVersion() contains unexpected substring: %q", unwanted)
				}
			}
		})
	}
}

func TestFilterChecksumsForVersion(t *testing.T) {
	tests := []struct {
		name          string
		installSpec   *spec.InstallSpec
		targetVersion string
		wantVersions  []string
		wantNotExist  []string
	}{
		{
			name: "filters to specific version",
			installSpec: &spec.InstallSpec{
				Checksums: &spec.ChecksumConfig{
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: "file1.tar.gz", Hash: "abc123"},
						},
						"v1.2.4": {
							{Filename: "file2.tar.gz", Hash: "def456"},
						},
						"v1.2.5": {
							{Filename: "file3.tar.gz", Hash: "ghi789"},
						},
					},
				},
			},
			targetVersion: "v1.2.4",
			wantVersions:  []string{"v1.2.4"},
			wantNotExist:  []string{"v1.2.3", "v1.2.5"},
		},
		{
			name: "no checksums returns unchanged",
			installSpec: &spec.InstallSpec{
				Checksums: nil,
			},
			targetVersion: "v1.2.3",
			wantVersions:  []string{},
			wantNotExist:  []string{},
		},
		{
			name: "empty embedded checksums returns unchanged",
			installSpec: &spec.InstallSpec{
				Checksums: &spec.ChecksumConfig{
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{},
				},
			},
			targetVersion: "v1.2.3",
			wantVersions:  []string{},
			wantNotExist:  []string{},
		},
		{
			name: "version not found returns empty checksums",
			installSpec: &spec.InstallSpec{
				Checksums: &spec.ChecksumConfig{
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: "file1.tar.gz", Hash: "abc123"},
						},
					},
				},
			},
			targetVersion: "v1.2.4",
			wantVersions:  []string{},
			wantNotExist:  []string{"v1.2.3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterChecksumsForVersion(tt.installSpec, tt.targetVersion)

			if got.Checksums == nil {
				if len(tt.wantVersions) > 0 {
					t.Errorf("filterChecksumsForVersion() got nil checksums, want versions: %v", tt.wantVersions)
				}
				return
			}

			// Check wanted versions exist
			for _, wantVersion := range tt.wantVersions {
				if _, exists := got.Checksums.EmbeddedChecksums[wantVersion]; !exists {
					t.Errorf("filterChecksumsForVersion() missing expected version: %q", wantVersion)
				}
			}

			// Check unwanted versions don't exist
			for _, unwantedVersion := range tt.wantNotExist {
				if _, exists := got.Checksums.EmbeddedChecksums[unwantedVersion]; exists {
					t.Errorf("filterChecksumsForVersion() contains unexpected version: %q", unwantedVersion)
				}
			}

		})
	}
}

func TestGenerate(t *testing.T) {
	// Test that Generate() calls GenerateWithVersion() with empty target version
	installSpec := &spec.InstallSpec{
		Name: "test-tool",
		Repo: "owner/test-tool",
		Asset: spec.AssetConfig{
			Template:         "${NAME}-${VERSION}-${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz",
		},
	}

	got, err := Generate(installSpec)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	gotStr := string(got)

	// Should behave like normal dynamic version generation
	if !strings.Contains(gotStr, `TAG="${1:-latest}"`) {
		t.Error("Generate() should generate dynamic version script")
	}

	if strings.Contains(gotStr, "Fixed version mode") {
		t.Error("Generate() should not generate fixed version script")
	}
}