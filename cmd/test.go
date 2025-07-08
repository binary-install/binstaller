package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var (
	// Flags for test command
	testVersion     string
	testCheckAssets bool
)

// TestCommand represents the test command
var TestCommand = &cobra.Command{
	Use:   "test",
	Short: "Test and validate an InstallSpec config file",
	Long: `Tests an InstallSpec configuration file by:
- Validating the configuration format
- Generating all possible asset filenames for supported platforms
- Optionally checking if assets exist in the GitHub release

This makes it easy to validate your configuration without generating
and running the actual installer script.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running test command...")

		// Determine config file path using common logic
		cfgFile, err := resolveConfigFile(configFile)
		if err != nil {
			log.WithError(err).Error("Config file detection failed")
			return err
		}
		if configFile == "" {
			log.Infof("Using default config file: %s", cfgFile)
		}
		log.Debugf("Using config file: %s", cfgFile)

		// Read the InstallSpec YAML file
		log.Debugf("Reading InstallSpec from: %s", cfgFile)
		yamlData, err := os.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).Errorf("Failed to read install spec file: %s", cfgFile)
			return fmt.Errorf("failed to read install spec file %s: %w", cfgFile, err)
		}

		// Unmarshal YAML into InstallSpec struct
		log.Debug("Unmarshalling InstallSpec YAML")
		var installSpec spec.InstallSpec
		err = yaml.Unmarshal(yamlData, &installSpec)
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal install spec YAML from: %s", cfgFile)
			return fmt.Errorf("failed to unmarshal install spec YAML from %s: %w", cfgFile, err)
		}

		// Apply defaults
		installSpec.SetDefaults()

		// Validate the spec
		if err := validateSpec(&installSpec); err != nil {
			log.WithError(err).Error("InstallSpec validation failed")
			return fmt.Errorf("validation failed: %w", err)
		}

		log.Info("✓ InstallSpec validation passed")

		// Generate asset filenames for all supported platforms
		log.Info("Generating asset filenames for all supported platforms...")
		
		version := testVersion
		if version == "" {
			version = spec.StringValue(installSpec.DefaultVersion)
		}
		if version == "" || version == "latest" {
			version = "1.0.0" // Use example version for testing
		}

		assetFilenames, err := generateAllAssetFilenames(&installSpec, version)
		if err != nil {
			log.WithError(err).Error("Failed to generate asset filenames")
			return fmt.Errorf("failed to generate asset filenames: %w", err)
		}

		// Display the generated filenames
		displayAssetFilenames(assetFilenames)

		// Check if assets exist in GitHub release if requested
		if testCheckAssets {
			log.Info("Checking if assets exist in GitHub release...")
			ctx := context.Background()
			err := checkAssetsExist(ctx, &installSpec, version, assetFilenames)
			if err != nil {
				log.WithError(err).Error("Asset availability check failed")
				return fmt.Errorf("asset availability check failed: %w", err)
			}
		}

		log.Info("✓ Test completed successfully")
		return nil
	},
}

// validateSpec performs basic validation of the InstallSpec
func validateSpec(installSpec *spec.InstallSpec) error {
	if installSpec.Repo == nil || *installSpec.Repo == "" {
		return fmt.Errorf("repo field is required")
	}

	// Validate repository format (owner/repo)
	repoPattern := regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)
	if !repoPattern.MatchString(*installSpec.Repo) {
		return fmt.Errorf("repo must be in format 'owner/repo', got: %s", *installSpec.Repo)
	}

	if installSpec.Asset == nil {
		return fmt.Errorf("asset configuration is required")
	}

	if installSpec.Asset.Template == nil || *installSpec.Asset.Template == "" {
		return fmt.Errorf("asset template is required")
	}

	return nil
}

// generateAllAssetFilenames generates asset filenames for all supported platforms
func generateAllAssetFilenames(installSpec *spec.InstallSpec, version string) (map[string]string, error) {
	assetFilenames := make(map[string]string)

	// Get supported platforms, or use default common platforms
	platforms := getSupportedPlatforms(installSpec)

	// Generate filename for each platform
	for _, platform := range platforms {
		os := spec.PlatformOSString(platform.OS)
		arch := spec.PlatformArchString(platform.Arch)
		
		if os == "" || arch == "" {
			continue
		}

		// Create filename generator
		generator := asset.NewFilenameGenerator(installSpec, version)
		
		// Generate filename for this platform
		filename, err := generator.GenerateFilename(os, arch)
		if err != nil {
			log.WithError(err).Warnf("Failed to generate filename for %s/%s", os, arch)
			continue
		}

		platformKey := fmt.Sprintf("%s/%s", os, arch)
		assetFilenames[platformKey] = filename
	}

	return assetFilenames, nil
}

// getSupportedPlatforms returns the list of supported platforms
func getSupportedPlatforms(installSpec *spec.InstallSpec) []spec.SupportedPlatformElement {
	if len(installSpec.SupportedPlatforms) > 0 {
		return installSpec.SupportedPlatforms
	}

	// Default to common platforms
	return []spec.SupportedPlatformElement{
		{OS: spec.SupportedPlatformOSPtr("linux"), Arch: spec.SupportedPlatformArchPtr("amd64")},
		{OS: spec.SupportedPlatformOSPtr("linux"), Arch: spec.SupportedPlatformArchPtr("arm64")},
		{OS: spec.SupportedPlatformOSPtr("darwin"), Arch: spec.SupportedPlatformArchPtr("amd64")},
		{OS: spec.SupportedPlatformOSPtr("darwin"), Arch: spec.SupportedPlatformArchPtr("arm64")},
		{OS: spec.SupportedPlatformOSPtr("windows"), Arch: spec.SupportedPlatformArchPtr("amd64")},
		{OS: spec.SupportedPlatformOSPtr("windows"), Arch: spec.SupportedPlatformArchPtr("arm64")},
	}
}

// displayAssetFilenames displays the generated asset filenames in a table format
func displayAssetFilenames(assetFilenames map[string]string) {
	if len(assetFilenames) == 0 {
		fmt.Println("No asset filenames generated")
		return
	}

	// Sort platforms for consistent output
	platforms := make([]string, 0, len(assetFilenames))
	for platform := range assetFilenames {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLATFORM\tASSET FILENAME")
	fmt.Fprintln(w, "--------\t--------------")

	for _, platform := range platforms {
		filename := assetFilenames[platform]
		fmt.Fprintf(w, "%s\t%s\n", platform, filename)
	}

	w.Flush()
}

// checkAssetsExist checks if the generated asset filenames exist in the GitHub release
func checkAssetsExist(ctx context.Context, installSpec *spec.InstallSpec, version string, assetFilenames map[string]string) error {
	repo := spec.StringValue(installSpec.Repo)
	if repo == "" {
		return fmt.Errorf("repository not specified")
	}

	// Resolve version to actual tag if needed
	actualVersion := version
	if version == "latest" {
		resolvedVersion, err := resolveLatestVersion(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to resolve latest version: %w", err)
		}
		actualVersion = resolvedVersion
	}

	log.Infof("Checking assets for version: %s", actualVersion)

	// Fetch all release assets once
	releaseAssets, err := fetchReleaseAssets(ctx, repo, actualVersion)
	if err != nil {
		return fmt.Errorf("failed to fetch release assets: %w", err)
	}

	// Create a map of existing assets for quick lookup
	existingAssets := make(map[string]bool)
	for _, asset := range releaseAssets {
		existingAssets[asset] = true
	}

	// Check each generated asset
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLATFORM\tASSET FILENAME\tSTATUS")
	fmt.Fprintln(w, "--------\t--------------\t------")

	for platform, filename := range assetFilenames {
		status := "✓ EXISTS"
		if !existingAssets[filename] {
			status = "✗ MISSING"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", platform, filename, status)
	}

	w.Flush()

	// Display unmatched assets from the release
	displayUnmatchedAssets(releaseAssets, assetFilenames)

	return nil
}

// resolveLatestVersion resolves "latest" to the actual latest release tag
func resolveLatestVersion(ctx context.Context, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	
	req, err := httpclient.NewRequestWithGitHubAuth("GET", url)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: httpclient.NewGitHubClient().Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("failed to parse release response: %w", err)
	}

	return release.TagName, nil
}

// fetchReleaseAssets fetches all assets from a GitHub release
func fetchReleaseAssets(ctx context.Context, repo, version string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, url.PathEscape(version))
	
	req, err := httpclient.NewRequestWithGitHubAuth("GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: httpclient.NewGitHubClient().Transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release struct {
		Assets []struct {
			Name string `json:"name"`
		} `json:"assets"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release response: %w", err)
	}

	assets := make([]string, len(release.Assets))
	for i, asset := range release.Assets {
		assets[i] = asset.Name
	}

	return assets, nil
}

// displayUnmatchedAssets displays release assets that don't have matching entries
func displayUnmatchedAssets(releaseAssets []string, assetFilenames map[string]string) {
	// Create a map of expected filenames for quick lookup
	expectedAssets := make(map[string]bool)
	for _, filename := range assetFilenames {
		expectedAssets[filename] = true
	}

	// Find unmatched assets
	unmatchedAssets := make([]string, 0)
	for _, asset := range releaseAssets {
		if !expectedAssets[asset] {
			unmatchedAssets = append(unmatchedAssets, asset)
		}
	}

	// Display unmatched assets if any
	if len(unmatchedAssets) > 0 {
		fmt.Println("\nUnmatched assets in release (no corresponding platform):")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ASSET FILENAME")
		fmt.Fprintln(w, "--------------")
		
		sort.Strings(unmatchedAssets)
		for _, asset := range unmatchedAssets {
			fmt.Fprintf(w, "%s\n", asset)
		}
		w.Flush()
	}
}


func init() {
	// Flags specific to test command
	TestCommand.Flags().StringVar(&testVersion, "version", "", "Test with specific version (default: uses default_version from spec)")
	TestCommand.Flags().BoolVar(&testCheckAssets, "check-assets", false, "Check if generated assets exist in GitHub release")
}