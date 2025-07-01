package datasource

import (
	"cmp"
	"context"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/aquaproj/aqua/v2/pkg/config/registry"
	aquaexpr "github.com/aquaproj/aqua/v2/pkg/expr"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// AquaRegistryAdapter implements SourceAdapter for Aqua registry YAML files.
type AquaRegistryAdapter struct {
	reader io.Reader // Used for stdin, file, etc.
	repo   string    // Used for GitHub fetch, e.g. "owner/name"
	ref    string    // GitHub ref (commit SHA or "HEAD"), default "HEAD"
}

// NewAquaRegistryAdapterFromReader creates an adapter from an io.Reader (stdin, file, etc.).
func NewAquaRegistryAdapterFromReader(reader io.Reader) *AquaRegistryAdapter {
	return &AquaRegistryAdapter{reader: reader}
}

// NewAquaRegistryAdapterFromRepo creates an adapter that fetches the registry YAML from GitHub.
// If ref is empty, "HEAD" is used.
func NewAquaRegistryAdapterFromRepo(repo string, ref string) *AquaRegistryAdapter {
	if ref == "" {
		ref = "HEAD"
	}
	return &AquaRegistryAdapter{repo: repo, ref: ref}
}

// isVersionConstraintSatisfiedForLatest uses EvaluateVersionConstraints to check if the version constraints allow "latest" (simulated by v99999999.0.0).
func isVersionConstraintSatisfiedForLatest(constraint string) bool {
	if constraint == "" {
		return true
	}
	ok, err := aquaexpr.EvaluateVersionConstraints(constraint, "v99999999.0.0", "v99999999.0.0")
	return err == nil && ok
}

// mapToInstallSpec maps a registry.PackageInfo to a *spec.InstallSpec.
func mapToInstallSpec(p registry.PackageInfo) (*spec.InstallSpec, error) {
	installSpec := &spec.InstallSpec{}
	if p.Name != "" {
		installSpec.Name = p.Name
	} else if len(p.Files) > 0 && p.Files[0].Name != "" {
		installSpec.Name = p.Files[0].Name
	}
	if p.RepoOwner != "" && p.RepoName != "" {
		installSpec.Repo = p.RepoOwner + "/" + p.RepoName
	}
	if checkTitleCase(&p) {
		installSpec.Asset.NamingConvention = &spec.NamingConvention{
			OS: "titlecase",
		}
	}
	assetTmpl, err := convertAssetTemplate(p.Asset)
	if err != nil {
		return nil, err
	}
	installSpec.Asset.Template = assetTmpl
	tmplVars := map[string]string{"AssetWithoutExt": assetWithoutExtension(assetTmpl)}
	installSpec.Asset.DefaultExtension = formatToExtension(p.Format)
	if installSpec.Asset.DefaultExtension == "" && hasExtensions(assetTmpl) {
		installSpec.Asset.DefaultExtension = extractExtension(assetTmpl)
	}
	installSpec.SupportedPlatforms = convertSupportedEnvs(p.SupportedEnvs)
	if p.Checksum != nil {
		convertedChecksum, err := ConvertAquaTemplateToInstallSpec(p.Checksum.Asset, tmplVars)
		if err != nil {
			return nil, err
		}
		installSpec.Checksums = &spec.ChecksumConfig{
			Template:  convertedChecksum,
			Algorithm: p.Checksum.Algorithm,
		}
	}

	if p.Rosetta2 {
		installSpec.Asset.ArchEmulation = &spec.ArchEmulation{
			Rosetta2: true,
		}
	}

	// Convert FormatOverrides to Asset.Rules
	for _, fo := range p.FormatOverrides {
		if fo == nil {
			continue
		}
		rule := spec.AssetRule{
			When: spec.PlatformCondition{OS: fo.GOOS},
			Ext:  formatToExtension(fo.Format),
		}
		installSpec.Asset.Rules = append(installSpec.Asset.Rules, rule)
	}

	// Convert Replacements to Asset.Rules
	// This should be before processing overrides since replacement often
	// contains OS/ARCH replacements which should be replaced before overriding
	// templates.
	rules := convertReplacementsToRules(p.Replacements)
	if len(rules) > 0 {
		installSpec.Asset.Rules = append(installSpec.Asset.Rules, rules...)
	}

	// Convert Overrides to Asset.Rules
	for _, ov := range p.Overrides {
		if ov == nil {
			continue
		}
		rule := spec.AssetRule{
			When: spec.PlatformCondition{OS: ov.GOOS, Arch: ov.GOArch},
		}

		// copy tmplVar for overrides
		tmplVarsOv := map[string]string{"AssetWithoutExt": tmplVars["AssetWithoutExt"]}

		if len(ov.Replacements) > 0 {
			rules := convertReplacementsToRules(ov.Replacements)
			for _, ruleRep := range rules {
				if ruleRep.When.OS == "" {
					ruleRep.When.OS = ov.GOOS
				}
				if ruleRep.When.Arch == "" {
					ruleRep.When.Arch = ov.GOArch
				}
				installSpec.Asset.Rules = append(installSpec.Asset.Rules, ruleRep)
			}
		}

		rule.Ext = formatToExtension(ov.Format)
		if ov.Asset != "" {
			assetTmpl, err := convertAssetTemplate(ov.Asset)
			if err != nil {
				return nil, err
			}
			rule.Template = assetTmpl
			tmplVarsOv["AssetWithoutExt"] = assetWithoutExtension(assetTmpl)
		}

		binaries, err := convertFilesToBinaries(ov.Files, tmplVarsOv)
		if err != nil {
			return nil, err
		}
		if len(binaries) > 0 {
			rule.Binaries = binaries
		}

		if rule.Arch != "" || len(rule.Binaries) > 0 || rule.Ext != "" || rule.OS != "" || rule.Template != "" {
			installSpec.Asset.Rules = append(installSpec.Asset.Rules, rule)
		}

	}

	binaries, err := convertFilesToBinaries(p.Files, tmplVars)
	if err != nil {
		return nil, err
	}
	if len(binaries) > 0 {
		installSpec.Asset.Binaries = binaries
	}

	return installSpec, nil
}

func convertReplacementsToRules(r registry.Replacements) []spec.AssetRule {
	rules := make([]spec.AssetRule, 0)
	for k, v := range r {
		rule := spec.AssetRule{}
		if isOS(k) {
			rule.When.OS = k
			rule.OS = v
		} else {
			rule.When.Arch = k
			rule.Arch = v
		}
		rules = append(rules, rule)
	}
	slices.SortStableFunc(rules, func(a, b spec.AssetRule) int {
		return cmp.Or(
			cmp.Compare(a.When.OS, b.When.OS),
			cmp.Compare(a.When.Arch, b.When.Arch),
		)
	})
	return rules
}

func convertFilesToBinaries(files []*registry.File, tmplVars map[string]string) ([]spec.Binary, error) {
	binaries := make([]spec.Binary, 0, len(files))
	for _, f := range files {
		if f.Name != "" {
			path := f.Src
			if path == "" {
				path = f.Name
			} else {
				evaluated, err := ConvertAquaTemplateToInstallSpec(path, tmplVars)
				if err != nil {
					return nil, err
				}
				path = evaluated
			}
			binaries = append(binaries, spec.Binary{Name: f.Name, Path: path})
		}
	}
	return binaries, nil
}

// isOS returns true if the string is a known GOOS value (target OS for Go builds).
// List generated from: go tool dist list -json | jq -r '.[].GOOS' | sort -u (as of 2025-04-28)
// aix, android, darwin, dragonfly, freebsd, illumos, ios, js, linux, netbsd, openbsd, plan9, solaris, wasip1, windows
func isOS(s string) bool {
	switch s {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "js", "linux", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows":
		return true
	}
	return false
}

// GenerateInstallSpec parses the Aqua registry config and returns the first valid InstallSpec for a supported package.
// Currently, only packages of type "github_release" are supported.
// If version overrides are present, the first valid override is returned.
// Returns an error if no valid package is found or if template conversion fails.
func (a *AquaRegistryAdapter) GenerateInstallSpec(ctx context.Context) (*spec.InstallSpec, error) {
	var r io.Reader
	if a.reader != nil {
		r = a.reader
	} else if a.repo != "" {
		// Fetch from GitHub
		ref := a.ref
		if ref == "" {
			ref = "HEAD"
		}
		url := "https://raw.githubusercontent.com/aquaproj/aqua-registry/" + ref + "/pkgs/" + a.repo + "/registry.yaml"
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to fetch registry.yaml from GitHub: " + resp.Status)
		}
		r = resp.Body
	} else {
		return nil, errors.New("no input source provided")
	}
	return genSpecFromRegistryYAML(ctx, r)
}

func genSpecFromRegistryYAML(ctx context.Context, r io.Reader) (*spec.InstallSpec, error) {
	// Parse YAML into Aqua's official struct
	var regConfig registry.Config
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&regConfig); err != nil {
		return nil, err
	}

	// Implement mapping/filtering logic from regConfig.Packages to InstallSpec

	for _, pkg := range regConfig.PackageInfos {
		if pkg.Type != "github_release" {
			continue
		}

		// Main package: only if VersionConstraints is empty or evaluated to "true"
		if isVersionConstraintSatisfiedForLatest(pkg.VersionConstraints) {
			spec, err := mapToInstallSpec(*pkg)
			if err != nil {
				return nil, err
			}
			return spec, nil
		}

		// version_overrides: only those with VersionConstraints "true"
		for _, vo := range pkg.VersionOverrides {
			if isVersionConstraintSatisfiedForLatest(vo.VersionConstraints) && (vo.Type == "" || vo.Type == "github_release") {
				// Map override fields onto a copy of pkg, then map to InstallSpec
				override := mergeVersionOverride(*pkg, *vo)
				spec, err := mapToInstallSpec(override)
				if err != nil {
					return nil, err
				}
				return spec, nil
			}
		}
	}

	return nil, errors.New("no valid github_release package found in registry")
}

// convertSupportedEnvs converts registry.SupportedEnvs to []spec.Platform.
func convertSupportedEnvs(envs registry.SupportedEnvs) []spec.Platform {
	var platforms []spec.Platform
	for _, env := range envs {
		parts := strings.SplitN(env, "/", 2)
		if len(parts) == 2 {
			platforms = append(platforms, spec.Platform{OS: parts[0], Arch: parts[1]})
		}
	}
	return platforms
}

func convertAssetTemplate(tmpl string) (string, error) {
	s, err := ConvertAquaTemplateToInstallSpec(tmpl, nil)
	if err != nil {
		return "", err
	}
	if !strings.HasSuffix(s, "${EXT}") && !hasExtensions(s) {
		s += "${EXT}"
	}
	return s, nil
}

func checkTitleCase(p *registry.PackageInfo) bool {
	return strings.Contains(p.Asset, "title .OS")
}

func assetWithoutExtension(assetTmpl string) string {
	return strings.TrimSuffix(trimExtension(assetTmpl), "${EXT}")
}
