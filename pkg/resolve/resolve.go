package resolve

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/google/go-github/v60/github"
	"github.com/pkg/errors"
)

// BinaryInfo contains information about a binary to install
type BinaryInfo struct {
	Name string
	Path string
}

// AssetFilename generates the asset filename based on the template and rules
func AssetFilename(cfg *spec.InstallSpec, version, osName, arch string) string {
	// Strip v prefix from version for template substitution
	versionForTemplate := strings.TrimPrefix(version, "v")

	// Start with defaults
	assetOS := osName
	assetArch := arch
	ext := spec.StringValue(cfg.Asset.DefaultExtension)
	template := spec.StringValue(cfg.Asset.Template)

	// Apply OS naming convention
	if cfg.Asset.NamingConvention != nil && cfg.Asset.NamingConvention.OS != nil {
		if string(*cfg.Asset.NamingConvention.OS) == "titlecase" {
			assetOS = capitalize(assetOS)
		}
	}

	// Apply rules
	for _, rule := range cfg.Asset.Rules {
		if matchesRule(rule, osName, arch) {
			if rule.OS != nil {
				assetOS = *rule.OS
			}
			if rule.Arch != nil {
				assetArch = *rule.Arch
			}
			if rule.EXT != nil {
				ext = *rule.EXT
			}
			if rule.Template != nil {
				template = *rule.Template
			}
		}
	}

	// Perform template substitution
	result := template
	result = strings.ReplaceAll(result, "${NAME}", spec.StringValue(cfg.Name))
	result = strings.ReplaceAll(result, "${VERSION}", versionForTemplate)
	result = strings.ReplaceAll(result, "${OS}", assetOS)
	result = strings.ReplaceAll(result, "${ARCH}", assetArch)
	result = strings.ReplaceAll(result, "${EXT}", ext)

	return result
}

// ResolveVersion resolves the version to use, fetching latest if needed
func ResolveVersion(cfg *spec.InstallSpec, version string) (string, error) {
	if version != "latest" && version != "" {
		return version, nil
	}

	// Fetch latest version from GitHub
	client := github.NewClient(nil)

	// Use GitHub token if available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		client = github.NewClient(nil).WithAuthToken(token)
	}

	repo := spec.StringValue(cfg.Repo)
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repository format: %s", repo)
	}

	ctx := context.Background()
	release, resp, err := client.Repositories.GetLatestRelease(ctx, parts[0], parts[1])
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			// Try to list releases and get the first one
			releases, _, err := client.Repositories.ListReleases(ctx, parts[0], parts[1], &github.ListOptions{
				PerPage: 1,
			})
			if err != nil {
				return "", errors.Wrap(err, "failed to fetch releases")
			}
			if len(releases) == 0 {
				return "", fmt.Errorf("no releases found for %s", repo)
			}
			return *releases[0].TagName, nil
		}
		return "", errors.Wrap(err, "failed to fetch latest release")
	}

	return *release.TagName, nil
}

// GetBinaryInfo returns information about binaries to install
func GetBinaryInfo(cfg *spec.InstallSpec, osName, arch string) []BinaryInfo {
	binaries := cfg.Asset.Binaries

	// Check if any rule provides binary overrides
	for _, rule := range cfg.Asset.Rules {
		if matchesRule(rule, osName, arch) && len(rule.Binaries) > 0 {
			binaries = rule.Binaries
			break
		}
	}

	result := make([]BinaryInfo, len(binaries))
	for i, bin := range binaries {
		result[i] = BinaryInfo{
			Name: spec.StringValue(bin.Name),
			Path: spec.StringValue(bin.Path),
		}
	}

	return result
}

// matchesRule checks if a rule applies to the given OS and architecture
func matchesRule(rule spec.RuleElement, osName, arch string) bool {
	if rule.When.OS != nil && *rule.When.OS != osName {
		return false
	}
	if rule.When.Arch != nil && *rule.When.Arch != arch {
		return false
	}
	return true
}

// capitalize capitalizes the first letter of a string
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}
