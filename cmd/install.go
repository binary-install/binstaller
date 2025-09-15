package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/spf13/cobra"
)

var (
	// Flags for install command
	installBinDir string
	installDryRun bool
)

// InstallCommand represents the install command
var InstallCommand = &cobra.Command{
	Use:   "install [VERSION]",
	Short: "Install a binary directly from GitHub releases",
	Long: `Install a binary directly from GitHub releases, achieving script-parity with the generated shell installers.

This command provides a native Go implementation of the installation process, supporting version resolution, checksum verification, and cross-platform binary installation.`,
	Example: `  # Install latest version
  binst install

  # Install specific version
  binst install v1.2.3

  # Install to custom directory
  binst install --bin-dir=/usr/local/bin

  # Dry run mode (verify URLs/versions without installing)
  binst install --dry-run`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInstall,
}

func init() {
	InstallCommand.Flags().StringVarP(&installBinDir, "bin-dir", "b", "", "Installation directory")
	InstallCommand.Flags().BoolVarP(&installDryRun, "dry-run", "n", false, "Dry run mode")
}

// GitHubRelease represents the GitHub API response for a release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// gitHubAPIBaseURL is the base URL for GitHub API calls (overridable for testing)
var gitHubAPIBaseURL = "https://api.github.com"

// resolveVersion resolves a version string to an actual GitHub release tag
func resolveVersion(ctx context.Context, repo, version string) (string, error) {
	if version != "" && version != "latest" {
		// User provided explicit version, use as-is
		return version, nil
	}

	// Resolve "latest" to actual tag using GitHub API
	log.Info("checking GitHub for latest tag")

	url := fmt.Sprintf("%s/repos/%s/releases/latest", gitHubAPIBaseURL, repo)

	client := httpclient.NewGitHubClient()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if release.TagName == "" {
		return "", fmt.Errorf("no tag_name found in GitHub response")
	}

	return release.TagName, nil
}

func runInstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. Resolve config file path
	cfgPath, err := resolveConfigFile(configFile)
	if err != nil {
		return err
	}

	// 2. Load config
	spec, err := loadInstallSpec(cfgPath)
	if err != nil {
		return err
	}

	// Get repo from spec
	if spec.Repo == nil || *spec.Repo == "" {
		return fmt.Errorf("GitHub repo not specified in config")
	}
	repo := *spec.Repo

	// 3. Get version from args (positional VERSION argument)
	version := ""
	if len(args) > 0 {
		version = args[0]
	}

	// 4. Resolve version (latest if not specified)
	resolvedVersion, err := resolveVersion(ctx, repo, version)
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}

	// Strip leading 'v' if present for the version number
	versionNumber := strings.TrimPrefix(resolvedVersion, "v")

	log.Infof("Resolved version: %s (tag: %s)", versionNumber, resolvedVersion)

	if installDryRun {
		log.Info("Dry run mode - skipping actual installation")
		// TODO: In future phases, validate asset URLs exist
		return nil
	}

	// TODO: Phase 2+ implementation
	// - Use pkg/asset.FilenameGenerator for asset resolution
	// - Use pkg/httpclient for downloading files
	// - Use pkg/checksums for verification
	// - Implement extraction and installation

	return fmt.Errorf("installation not yet implemented (Phase 1 complete)")
}
