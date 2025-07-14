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
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
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
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
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
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
				Checksums: &spec.ChecksumConfig{
					Algorithm: spec.AlgorithmPtr("sha256"),
					Template:  spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: spec.StringPtr("test-tool-1.2.3-linux_amd64.tar.gz"), Hash: spec.StringPtr("abc123")},
							{Filename: spec.StringPtr("test-tool-1.2.3-darwin_amd64.tar.gz"), Hash: spec.StringPtr("def456")},
						},
						"v1.2.4": {
							{Filename: spec.StringPtr("test-tool-1.2.4-linux_amd64.tar.gz"), Hash: spec.StringPtr("ghi789")},
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
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
				Checksums: &spec.ChecksumConfig{
					Algorithm: spec.AlgorithmPtr("sha256"),
					Template:  spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
					EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
						"v1.2.3": {
							{Filename: spec.StringPtr("test-tool-1.2.3-linux_amd64.tar.gz"), Hash: spec.StringPtr("abc123")},
						},
						"v1.2.4": {
							{Filename: spec.StringPtr("test-tool-1.2.4-linux_amd64.tar.gz"), Hash: spec.StringPtr("ghi789")},
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
							{Filename: spec.StringPtr("file1.tar.gz"), Hash: spec.StringPtr("abc123")},
						},
						"v1.2.4": {
							{Filename: spec.StringPtr("file2.tar.gz"), Hash: spec.StringPtr("def456")},
						},
						"v1.2.5": {
							{Filename: spec.StringPtr("file3.tar.gz"), Hash: spec.StringPtr("ghi789")},
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
							{Filename: spec.StringPtr("file1.tar.gz"), Hash: spec.StringPtr("abc123")},
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
		Name: spec.StringPtr("test-tool"),
		Repo: spec.StringPtr("owner/test-tool"),
		Asset: &spec.AssetConfig{
			Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
			DefaultExtension: spec.StringPtr(".tar.gz"),
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

func TestDryRunFlagParsing(t *testing.T) {
	tests := []struct {
		name           string
		installSpec    *spec.InstallSpec
		wantSubstrings []string
		wantNotContain []string
	}{
		{
			name: "dry run flag support in usage",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`Usage: $this [-b bindir] [-d] [-n]`,
				`-n turns on dry run mode`,
			},
		},
		{
			name: "dry run flag parsing in getopts",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`while getopts "b:dqh?xn" arg`,
				`n) DRY_RUN=1 ;;`,
			},
		},
		{
			name: "dry run variable initialization",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`DRY_RUN=0`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.installSpec)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			gotStr := string(got)

			// Check for expected substrings
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(gotStr, want) {
					t.Errorf("Generate() missing expected substring: %q", want)
				}
			}

			// Check for unexpected substrings
			for _, unwanted := range tt.wantNotContain {
				if strings.Contains(gotStr, unwanted) {
					t.Errorf("Generate() contains unexpected substring: %q", unwanted)
				}
			}
		})
	}
}

func TestDryRunOutputFormat(t *testing.T) {
	tests := []struct {
		name           string
		installSpec    *spec.InstallSpec
		wantSubstrings []string
	}{
		{
			name: "dry run output format for installation path",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`log_info "[DRY RUN] ${BINARY_NAME} dry-run installation succeeded! (Would install to: ${INSTALL_PATH})"`,
			},
		},
		{
			name: "dry run actual download behavior",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`log_info "Downloading ${ASSET_URL}"`,
				`if [ "$DRY_RUN" = "1" ]; then`,
			},
		},
		{
			name: "dry run actual checksum verification",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
				Checksums: &spec.ChecksumConfig{
					Algorithm: spec.AlgorithmPtr("sha256"),
					Template:  spec.StringPtr("${NAME}_${VERSION}_checksums.txt"),
				},
			},
			wantSubstrings: []string{
				`log_info "Downloading checksums from ${CHECKSUM_URL}"`,
				`log_info "Verifying checksum ..."`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.installSpec)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			gotStr := string(got)

			// Check for expected substrings
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(gotStr, want) {
					t.Errorf("Generate() missing expected substring: %q", want)
				}
			}
		})
	}
}

func TestDryRunBehavior(t *testing.T) {
	tests := []struct {
		name           string
		installSpec    *spec.InstallSpec
		wantSubstrings []string
		wantNotContain []string
	}{
		{
			name: "dry run skips installation only",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`if [ "$DRY_RUN" = "1" ]; then`,
				`log_info "[DRY RUN] ${BINARY_NAME} dry-run installation succeeded! (Would install to: ${INSTALL_PATH})"`,
			},
			wantNotContain: []string{
				`log_info "[DRY RUN] Installation would complete successfully"`,
				`log_info "[DRY RUN] Would download: ${ASSET_URL}"`,
				`log_info "[DRY RUN] Would verify checksum from: ${CHECKSUM_URL}"`,
			},
		},
		{
			name: "dry run performs downloads unconditionally",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			wantSubstrings: []string{
				`github_http_download "${TMPDIR}/${ASSET_FILENAME}" "${ASSET_URL}"`,
			},
			wantNotContain: []string{
				`if [ "$DRY_RUN" != "1" ]; then`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.installSpec)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}

			gotStr := string(got)

			// Check for expected substrings
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(gotStr, want) {
					t.Errorf("Generate() missing expected substring: %q", want)
				}
			}

			// Check for unexpected substrings
			for _, unwanted := range tt.wantNotContain {
				if strings.Contains(gotStr, unwanted) {
					t.Errorf("Generate() contains unexpected substring: %q", unwanted)
				}
			}
		})
	}
}

func TestGenerateRunner(t *testing.T) {
	tests := []struct {
		name           string
		installSpec    *spec.InstallSpec
		targetVersion  string
		wantSubstrings []string
		wantNotContain []string
	}{
		{
			name: "runner script generation without target version",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			targetVersion: "",
			wantSubstrings: []string{
				`# This script runs test-tool directly without installing`,
				`exec "${BINARY_PATH}" $TOOL_ARGS`,
				`cleanup() {`,
				`trap cleanup EXIT HUP INT TERM`,
				`chmod +x "${BINARY_PATH}"`,
			},
			wantNotContain: []string{
				`install "${BINARY_PATH}" "${INSTALL_PATH}"`,
				`Installation complete!`,
				`Installing binary to`,
			},
		},
		{
			name: "runner script with target version",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			targetVersion: "v1.2.3",
			wantSubstrings: []string{
				`# This script runs test-tool directly without installing`,
				`TAG="v1.2.3"`,
				`exec "${BINARY_PATH}" $TOOL_ARGS`,
				`chmod +x "${BINARY_PATH}"`,
			},
			wantNotContain: []string{
				`TAG="${1:-latest}"`,
				`install "${BINARY_PATH}" "${INSTALL_PATH}"`,
			},
		},
		{
			name: "runner script usage shows correct parameters",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			targetVersion: "",
			wantSubstrings: []string{
				`Usage: $this [-d]`,
				`This script downloads and runs test-tool directly`,
				`Pass arguments after --:`,
				`$this -- --help`,
			},
			wantNotContain: []string{
				`[-b bindir]`,
				`sets bindir or installation directory`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRunner(tt.installSpec, tt.targetVersion)
			if err != nil {
				t.Fatalf("GenerateRunner() error = %v", err)
			}

			gotStr := string(got)

			// Check for expected substrings
			for _, want := range tt.wantSubstrings {
				if !strings.Contains(gotStr, want) {
					t.Errorf("GenerateRunner() missing expected substring: %q", want)
				}
			}

			// Check for unexpected substrings
			for _, unwanted := range tt.wantNotContain {
				if strings.Contains(gotStr, unwanted) {
					t.Errorf("GenerateRunner() contains unexpected substring: %q", unwanted)
				}
			}
		})
	}
}

func TestGenerateWithScriptType(t *testing.T) {
	tests := []struct {
		name        string
		installSpec *spec.InstallSpec
		scriptType  string
		wantError   bool
		checkFunc   func(string) bool
	}{
		{
			name: "installer type generates installer script",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			scriptType: "installer",
			wantError:  false,
			checkFunc: func(script string) bool {
				return strings.Contains(script, `install "${BINARY_PATH}" "${INSTALL_PATH}"`) &&
					!strings.Contains(script, `chmod +x "${BINARY_PATH}"`)
			},
		},
		{
			name: "runner type generates runner script",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			scriptType: "runner",
			wantError:  false,
			checkFunc: func(script string) bool {
				return strings.Contains(script, `exec "${BINARY_PATH}" $TOOL_ARGS`) &&
					strings.Contains(script, `chmod +x "${BINARY_PATH}"`)
			},
		},
		{
			name: "empty type defaults to installer",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			scriptType: "",
			wantError:  false,
			checkFunc: func(script string) bool {
				return strings.Contains(script, `install "${BINARY_PATH}" "${INSTALL_PATH}"`) &&
					!strings.Contains(script, `chmod +x "${BINARY_PATH}"`)
			},
		},
		{
			name: "invalid type returns error",
			installSpec: &spec.InstallSpec{
				Name: spec.StringPtr("test-tool"),
				Repo: spec.StringPtr("owner/test-tool"),
				Asset: &spec.AssetConfig{
					Template:         spec.StringPtr("${NAME}-${VERSION}-${OS}_${ARCH}${EXT}"),
					DefaultExtension: spec.StringPtr(".tar.gz"),
				},
			},
			scriptType: "invalid",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateWithScriptType(tt.installSpec, "", tt.scriptType)
			if tt.wantError {
				if err == nil {
					t.Errorf("GenerateWithScriptType() expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateWithScriptType() unexpected error = %v", err)
			}

			if tt.checkFunc != nil && !tt.checkFunc(string(got)) {
				t.Errorf("GenerateWithScriptType() script check failed for type %q", tt.scriptType)
			}
		})
	}
}
