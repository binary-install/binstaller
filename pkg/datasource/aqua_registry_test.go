package datasource

import (
	"context"
	"strings"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

const sampleAquaYAML = `
packages:
  - name: gh
    type: github_release
    repo_owner: cli
    repo_name: cli
    version_constraint: "true"
    asset: "gh_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
    files:
      - name: gh
        src: "{{.AssetWithoutExt}}_bin"
    supported_envs:
      - linux/amd64
    checksum:
      type: github_release
      asset: "{{.AssetWithoutExt}}.sha256"
      algorithm: sha256
    format: tar.gz
`

const sampleAquaYAMLChecksumTemplate = `
packages:
  - name: gh
    type: github_release
    repo_owner: cli
    repo_name: cli
    version_constraint: "true"
    asset: "gh_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
    files:
      - name: gh
        src: gh
    supported_envs:
      - linux/amd64
    checksum:
      type: github_release
      asset: "{{.Asset}}.sha256"
      algorithm: sha256
    format: tar.gz
`

const sampleAquaYAMLOv = `
packages:
  - name: gh
    type: github_release
    repo_owner: cli
    repo_name: cli
    version_constraint: "true"
    asset: "gh_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
    files:
      - name: gh
        src: gh
    supported_envs:
      - linux/amd64
      - darwin/arm64
    overrides:
      - goos: darwin
        goarch: arm64
        asset: "gh_{{.Version}}_darwin_arm64.zip"
        format: zip
    format: tar.gz
`

func newTestInstallSpec(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAML))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func newTestInstallSpecChecksumTemplate(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAMLChecksumTemplate))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func newTestInstallSpecWithOverrides(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAMLOv))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func TestAquaRegistryAdapter_Name(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "gh"
	if spec.StringValue(installSpec.Name) != want {
		t.Errorf("Name: got %q, want %q", spec.StringValue(installSpec.Name), want)
	}
}

func TestAquaRegistryAdapter_Repo(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "cli/cli"
	if spec.StringValue(installSpec.Repo) != want {
		t.Errorf("Repo: got %q, want %q", spec.StringValue(installSpec.Repo), want)
	}
}

func TestAquaRegistryAdapter_AssetTemplate(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "gh_${TAG}_${OS}_${ARCH}.tar.gz"
	if spec.StringValue(installSpec.Asset.Template) != want {
		t.Errorf("Asset.Template: got %q, want %q", spec.StringValue(installSpec.Asset.Template), want)
	}
}

func TestAquaRegistryAdapter_DefaultExtension(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := ".tar.gz"
	if spec.StringValue(installSpec.Asset.DefaultExtension) != want {
		t.Errorf("Asset.DefaultExtension: got %q, want %q", spec.StringValue(installSpec.Asset.DefaultExtension), want)
	}
}

func TestAquaRegistryAdapter_SupportedPlatforms(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	linuxOS := spec.Linux
	amd64Arch := spec.Amd64
	want := []spec.Platform{
		{OS: &linuxOS, Arch: &amd64Arch},
	}
	if len(installSpec.SupportedPlatforms) != len(want) {
		t.Fatalf("SupportedPlatforms: got %d, want %d", len(installSpec.SupportedPlatforms), len(want))
	}
	for i, p := range want {
		if spec.PlatformOSString(installSpec.SupportedPlatforms[i].OS) != spec.PlatformOSString(p.OS) ||
			spec.PlatformArchString(installSpec.SupportedPlatforms[i].Arch) != spec.PlatformArchString(p.Arch) {
			t.Errorf("SupportedPlatforms[%d]: got %+v, want %+v", i, installSpec.SupportedPlatforms[i], p)
		}
	}
}

func TestAquaRegistryAdapter_Checksums(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	if installSpec.Checksums == nil {
		t.Fatal("Checksums: got nil, want non-nil")
	}
	want := "gh_${TAG}_${OS}_${ARCH}.sha256"
	if spec.StringValue(installSpec.Checksums.Template) != want {
		t.Errorf("Checksums.Template: got %q, want %q", spec.StringValue(installSpec.Checksums.Template), want)
	}
	if spec.AlgorithmString(installSpec.Checksums.Algorithm) != "sha256" {
		t.Errorf("Checksums.Algorithm: got %q, want %q", spec.AlgorithmString(installSpec.Checksums.Algorithm), "sha256")
	}
}

func TestAquaRegistryAdapter_Checksums_TemplateWithAsset(t *testing.T) {
	installSpec := newTestInstallSpecChecksumTemplate(t)
	want := "${ASSET_FILENAME}.sha256"
	if spec.StringValue(installSpec.Checksums.Template) != want {
		t.Errorf("Checksums.Template: got %q, want %q", spec.StringValue(installSpec.Checksums.Template), want)
	}
}

func TestAquaRegistryAdapter_Binaries(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	binaries := installSpec.Asset.Binaries
	wantPath := "gh_${TAG}_${OS}_${ARCH}_bin"
	if len(binaries) != 1 || spec.StringValue(binaries[0].Name) != "gh" || spec.StringValue(binaries[0].Path) != wantPath {
		t.Errorf("Asset.Binaries: got %+v, want [{Name: \"gh\", Path: %q}]", binaries, wantPath)
	}
}

func TestAquaRegistryAdapter_AssetRules_Empty(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	if len(installSpec.Asset.Rules) != 0 {
		t.Errorf("Asset.Rules: got %+v, want empty", installSpec.Asset.Rules)
	}
}

func TestAquaRegistryAdapter_Overrides(t *testing.T) {
	installSpec := newTestInstallSpecWithOverrides(t)
	found := false
	for _, rule := range installSpec.Asset.Rules {
		if spec.StringValue(rule.When.OS) == "darwin" && spec.StringValue(rule.When.Arch) == "arm64" &&
			spec.StringValue(rule.EXT) == ".zip" && spec.StringValue(rule.Template) == "gh_${TAG}_darwin_arm64.zip" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Overrides: expected rule for darwin/arm64 with .zip and correct template, got %+v", installSpec.Asset.Rules)
	}
}

func newTestInstallSpecWithAssetWithoutExt(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAML))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func TestAquaRegistryAdapter_AssetWithoutExt(t *testing.T) {
	installSpec := newTestInstallSpecWithAssetWithoutExt(t)
	wantChecksum := "gh_${TAG}_${OS}_${ARCH}.sha256"
	if spec.StringValue(installSpec.Checksums.Template) != wantChecksum {
		t.Errorf("Checksums.Template: got %q, want %q", spec.StringValue(installSpec.Checksums.Template), wantChecksum)
	}
	binaries := installSpec.Asset.Binaries
	wantBinaryPath := "gh_${TAG}_${OS}_${ARCH}_bin"
	if len(binaries) != 1 || spec.StringValue(binaries[0].Path) != wantBinaryPath {
		t.Errorf("Asset.Binaries[0].Path: got %q, want %q", spec.StringValue(binaries[0].Path), wantBinaryPath)
	}
}

func TestIsVersionConstraintSatisfiedForLatest(t *testing.T) {
	tests := []struct {
		constraint string
		want       bool
	}{
		{"", true},
		{"true", true},
		{"false", false},
		{`semver(">= 0.4.0")`, true},
		{`semver("< 0.4.0")`, false},
		{`semverWithVersion(">= 4.2.0", trimPrefix(Version, "kustomize/"))`, true},
	}

	for _, tt := range tests {
		t.Run(tt.constraint, func(t *testing.T) {
			got := isVersionConstraintSatisfiedForLatest(tt.constraint)
			if got != tt.want {
				t.Errorf("isVersionConstraintSatisfiedForLatest(%q) = %v, want %v", tt.constraint, got, tt.want)
			}
		})
	}
}
