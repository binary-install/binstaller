package datasource_test

import (
	"context"
	"os"
	"testing"

	"github.com/binary-install/binstaller/pkg/datasource"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/google/go-cmp/cmp"
)

// setupGoReleaserTest is a helper function to create a temporary goreleaser.yml
// file, create the adapter, and call Detect.
func setupGoReleaserTest(t *testing.T, goreleaserConfigContent string) (*spec.InstallSpec, error) {
	t.Helper()

	// Create a temporary dummy goreleaser.yml file
	tmpFile, err := createTempFile("goreleaser.yml", goreleaserConfigContent)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer cleanupTempFile(tmpFile)

	adapter := datasource.NewGoReleaserAdapter(
		"",             // repo
		tmpFile.Name(), // filePath
		"",             // commit
		"",             // nameOverride
	)

	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	return installSpec, err
}

func TestGoReleaserAdapter_Detect_File_DefaultExtension(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	if installSpec.Asset.DefaultExtension != ".tar.gz" {
		t.Errorf("DefaultExtension: want .tar.gz, got %q", installSpec.Asset.DefaultExtension)
	}
}

func TestGoReleaserAdapter_Detect_File_AssetRules(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	want := []spec.AssetRule{
		{
			When: spec.PlatformCondition{OS: "windows"},
			Ext:  ".zip",
		},
	}
	if diff := cmp.Diff(want, installSpec.Asset.Rules); diff != "" {
		t.Errorf("Asset.Rules mismatch (-want +got):\n%s", diff)
	}
}

func TestGoReleaserAdapter_Detect_File_ChecksumsTemplate(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	want := "checksums.txt"
	if installSpec.Checksums == nil || installSpec.Checksums.Template != want {
		t.Errorf("Checksums.Template: want %q, got %v", want, installSpec.Checksums)
	}
}

func TestGoReleaserAdapter_Detect_File_NameAndRepo(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	if installSpec.Name != "mycli" {
		t.Errorf("Name: want mycli, got %q", installSpec.Name)
	}
	if installSpec.Repo != "myowner/myrepo" {
		t.Errorf("Repo: want myowner/myrepo, got %q", installSpec.Repo)
	}
}

func TestGoReleaserAdapter_Detect_BinaryFormat(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    formats:
      - binary
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	if installSpec.Asset.DefaultExtension != "" {
		t.Errorf("DefaultExtension for binary format should be empty, got: %q", installSpec.Asset.DefaultExtension)
	}
}

func TestGoReleaserAdapter_Detect_DeprecatedFormat(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format: zip # format is Depreated
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	if installSpec.Asset.DefaultExtension != ".zip" {
		t.Errorf("DefaultExtension for binary format should be .zip, got: %q", installSpec.Asset.DefaultExtension)
	}
}

func TestGoReleaserAdapter_Detect_InferNameFromRepo(t *testing.T) {
	goreleaserConfigContent := `
version: 2
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedName := "myrepo"
	if installSpec.Name != expectedName {
		t.Errorf("expected Name to be %q, got %q", expectedName, installSpec.Name)
	}
}

func TestGoReleaserAdapter_Detect_InferNamingConventionOS(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{- title .Os }}_{{ .Arch }}"
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	if installSpec.Asset.NamingConvention == nil {
		t.Fatalf("NamingConvention is nil")
	}
	if installSpec.Asset.NamingConvention.OS != "titlecase" {
		t.Errorf("NamingConvention.OS: want titlecase, got %q", installSpec.Asset.NamingConvention.OS)
	}
}

func TestGoReleaserAdapter_Detect_UnpackWrapInDirectory(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    wrap_in_directory: true
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}
	if installSpec.Unpack == nil {
		t.Fatalf("Unpack should not be nil")
	}
	if installSpec.Unpack.StripComponents == nil || *installSpec.Unpack.StripComponents != 1 {
		t.Errorf("Unpack.StripComponents: want 1, got %v", installSpec.Unpack.StripComponents)
	}
}

func TestGoReleaserAdapter_Detect_SupportedPlatforms(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
builds:
  - goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
  - id: mycli2
    goos:
      - windows
    goarch:
      - amd64
      - arm64
    ignore:
      - goos: windows
        goarch: amd64
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedPlatforms := []spec.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "arm64"},
		// windows/amd64 is ignored in the goreleaser config
	}

	// Sort for deterministic comparison
	sortPlatforms(expectedPlatforms)
	sortPlatforms(installSpec.SupportedPlatforms)

	if diff := cmp.Diff(expectedPlatforms, installSpec.SupportedPlatforms); diff != "" {
		t.Errorf("SupportedPlatforms mismatch (-want +got):\n%s", diff)
	}
}

func TestGoReleaserAdapter_Detect_SupportedPlatformsWithArm(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
builds:
  - goos:
      - darwin
    goarch:
      - arm64
    ignore:
      - goos: darwin
        goarch: arm64
checksum:
  name_template: "checksums.txt"
`
	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedPlatforms := []spec.Platform{
		// darwin/arm64 is ignored, so should be empty
	}

	// Sort for deterministic comparison
	sortPlatforms(expectedPlatforms)
	sortPlatforms(installSpec.SupportedPlatforms)

	if diff := cmp.Diff(expectedPlatforms, installSpec.SupportedPlatforms); diff != "" {
		t.Errorf("SupportedPlatforms mismatch (-want +got):\n%s", diff)
	}
}

func TestGoReleaserAdapter_Detect_ConditinalNameTemplate(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: sigspy
release:
  github:
    owner: actionutils
    name: sigspy
archives:
  - name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	// The expected template string after evaluation with text/template and our varmap.
	// The conditional logic is evaluated based on the placeholder strings.
	// {{ .ProjectName }} -> ${NAME}
	// {{- title .Os }} -> ${OS} (title is no-op)
	// {{- if eq .Arch "amd64" }}x86_64 {{- else if eq .Arch "386" }}i386 {{- else }}{{ .Arch }}{{ end }}
	//   - eq "${ARCH}" "amd64" is false
	//   - eq "${ARCH}" "386" is false
	//   - else block {{ .Arch }} is executed -> ${ARCH}
	// {{- if .Arm }}v{{ .Arm }}{{ end }}
	//   - if "${ARM}" is true (non-empty string)
	//   - v{{ .Arm }} is executed -> v${ARM}
	expectedTemplate := "${NAME}_${OS}_${ARCH}${EXT}"
	if installSpec.Asset.Template != expectedTemplate {
		t.Errorf("Asset.Template: want %q, got %q", expectedTemplate, installSpec.Asset.Template)
	}
}

// Helper function to create a temporary file
func createTempFile(name, content string) (*os.File, error) {
	file, err := os.CreateTemp("", name)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

// Helper function to clean up a temporary file
func cleanupTempFile(file *os.File) {
	file.Close()
	os.Remove(file.Name())
}

// sortPlatforms sorts a slice of spec.Platform for deterministic comparison.
func sortPlatforms(platforms []spec.Platform) {
	// Simple sort by OS then Arch
	for i := 0; i < len(platforms); i++ {
		for j := i + 1; j < len(platforms); j++ {
			if platforms[i].OS > platforms[j].OS || (platforms[i].OS == platforms[j].OS && platforms[i].Arch > platforms[j].Arch) {
				platforms[i], platforms[j] = platforms[j], platforms[i]
			}
		}
	}
}
