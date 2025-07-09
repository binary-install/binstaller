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
	// Flags for check command
	checkVersion     string
	checkCheckAssets bool
)

// CheckCommand represents the check command
var CheckCommand = &cobra.Command{
	Use:   "check",
	Short: "Check and validate an InstallSpec config file",
	Long: `Checks an InstallSpec configuration file by:
- Validating the configuration format
- Generating all possible asset filenames for supported platforms
- Checking if assets exist in the GitHub release (default: enabled)

This makes it easy to validate your configuration without generating
and running the actual installer script.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running check command...")

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

		version := checkVersion
		if version == "" {
			version = spec.StringValue(installSpec.DefaultVersion)
		}
		
		// If checking assets and version is not specified or is "latest",
		// resolve the actual latest version from GitHub
		if checkCheckAssets && (version == "" || version == "latest") {
			ctx := context.Background()
			repo := spec.StringValue(installSpec.Repo)
			if repo != "" {
				resolvedVersion, err := resolveLatestVersion(ctx, repo)
				if err != nil {
					log.WithError(err).Warn("Failed to resolve latest version, using default")
					version = "1.0.0" // Fallback to example version
				} else {
					log.Infof("Resolved latest version: %s", resolvedVersion)
					version = resolvedVersion
				}
			}
		} else if version == "" || version == "latest" {
			version = "1.0.0" // Use example version for testing when not checking assets
		}

		assetFilenames, err := generateAllAssetFilenames(&installSpec, version)
		if err != nil {
			log.WithError(err).Error("Failed to generate asset filenames")
			return fmt.Errorf("failed to generate asset filenames: %w", err)
		}

		// Check if assets exist in GitHub release if requested
		if checkCheckAssets {
			log.Info("Checking if assets exist in GitHub release...")
			ctx := context.Background()
			// When check-assets is on and no platforms specified, use asset-based detection
			if len(installSpec.SupportedPlatforms) == 0 {
				err := checkAssetsExistWithDetection(ctx, &installSpec, version)
				if err != nil {
					log.WithError(err).Error("Asset availability check failed")
					return fmt.Errorf("asset availability check failed: %w", err)
				}
			} else {
				err := checkAssetsExist(ctx, &installSpec, version, assetFilenames)
				if err != nil {
					log.WithError(err).Error("Asset availability check failed")
					return fmt.Errorf("asset availability check failed: %w", err)
				}
			}
		} else {
			// Only display the generated filenames if not checking assets
			// (checkAssetsExist displays its own table with status)
			displayAssetFilenames(assetFilenames)
		}

		log.Info("✓ Check completed successfully")
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

	// Version should already be resolved at this point
	log.Infof("Checking assets for version: %s", version)

	// Fetch all release assets once
	releaseAssets, err := fetchReleaseAssets(ctx, repo, version)
	if err != nil {
		return fmt.Errorf("failed to fetch release assets: %w", err)
	}

	// Create a map of existing assets for quick lookup
	existingAssets := make(map[string]bool)
	for _, asset := range releaseAssets {
		existingAssets[asset] = true
	}

	// Sort platforms for consistent output
	platforms := make([]string, 0, len(assetFilenames))
	for platform := range assetFilenames {
		platforms = append(platforms, platform)
	}
	sort.Strings(platforms)

	// Check each generated asset
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLATFORM\tASSET FILENAME\tSTATUS")
	fmt.Fprintln(w, "--------\t--------------\t------")

	for _, platform := range platforms {
		filename := assetFilenames[platform]
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
		Timeout:   30 * time.Second,
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
		Timeout:   30 * time.Second,
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

// checkAssetsExistWithDetection checks assets by trying all possible platform combinations
func checkAssetsExistWithDetection(ctx context.Context, installSpec *spec.InstallSpec, version string) error {
	repo := spec.StringValue(installSpec.Repo)
	if repo == "" {
		return fmt.Errorf("repository not specified")
	}

	log.Infof("Checking assets for version: %s", version)

	// Fetch all release assets
	releaseAssets, err := fetchReleaseAssets(ctx, repo, version)
	if err != nil {
		return fmt.Errorf("failed to fetch release assets: %w", err)
	}

	// Create filename generator
	generator := asset.NewFilenameGenerator(installSpec, version)
	
	// Get all possible platforms using the same approach as embed-checksums
	platforms := generator.GetAllPossiblePlatforms()
	
	// Generate all possible asset filenames
	assetFilenames := make(map[string]string) // filename -> platform
	for _, platform := range platforms {
		os := spec.PlatformOSString(platform.OS)
		arch := spec.PlatformArchString(platform.Arch)
		
		if os == "" || arch == "" {
			continue
		}
		
		filename, err := generator.GenerateFilename(os, arch)
		if err != nil {
			continue
		}
		
		if filename != "" {
			platformKey := fmt.Sprintf("%s/%s", os, arch)
			// Store the first matching platform for each filename
			if _, exists := assetFilenames[filename]; !exists {
				assetFilenames[filename] = platformKey
			}
		}
	}
	
	// Create a map of release assets for quick lookup
	releaseAssetMap := make(map[string]bool)
	for _, asset := range releaseAssets {
		releaseAssetMap[asset] = true
	}
	
	// Sort filenames for consistent output
	filenames := make([]string, 0, len(assetFilenames))
	for filename := range assetFilenames {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	
	// Display results based on release assets (not all possible combinations)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ASSET FILENAME\tDETECTED PLATFORM\tSTATUS")
	fmt.Fprintln(w, "--------------\t-----------------\t------")
	
	// Sort release assets for consistent output
	sort.Strings(releaseAssets)
	
	// Track matched and unmatched assets
	var unmatchedAssets []string
	
	for _, assetName := range releaseAssets {
		// Skip non-binary assets (will be shown separately)
		if isNonBinaryAsset(assetName) {
			unmatchedAssets = append(unmatchedAssets, assetName)
			continue
		}
		
		// Check if this asset matches any generated filename
		platform := ""
		for filename, plat := range assetFilenames {
			if filename == assetName {
				platform = plat
				break
			}
		}
		
		if platform != "" {
			fmt.Fprintf(w, "%s\t%s\t✓ MATCHED\n", assetName, platform)
		} else {
			fmt.Fprintf(w, "%s\t-\t✗ NO MATCH\n", assetName)
		}
	}
	
	w.Flush()
	
	// Display non-binary assets separately
	if len(unmatchedAssets) > 0 {
		fmt.Println("\nOther assets in release:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ASSET FILENAME")
		fmt.Fprintln(w, "--------------")
		
		for _, asset := range unmatchedAssets {
			fmt.Fprintf(w, "%s\n", asset)
		}
		w.Flush()
	}
	
	return nil
}

// isNonBinaryAsset checks if an asset is likely not a binary (e.g., checksums, signatures)
func isNonBinaryAsset(filename string) bool {
	nonBinaryPatterns := []string{
		".txt", ".sha256", ".sha512", ".md5", ".sig", ".asc", ".pem",
		".sbom", ".json", ".yml", ".yaml", ".sh", ".ps1", ".md",
		"checksums", "SHASUMS", "README",
	}
	
	for _, pattern := range nonBinaryPatterns {
		if strings.Contains(filename, pattern) {
			return true
		}
	}
	
	// Check if it's a source archive (e.g., "binst-0.2.5.tar.gz")
	if strings.Contains(filename, "-") && (strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".zip")) {
		// Simple heuristic: if filename contains a dash followed by version-like pattern
		parts := strings.Split(filename, "-")
		if len(parts) >= 2 {
			// Check if the part after dash looks like a version
			versionPart := strings.TrimSuffix(strings.TrimSuffix(parts[len(parts)-1], ".tar.gz"), ".zip")
			if strings.Contains(versionPart, ".") || strings.HasPrefix(versionPart, "v") {
				return true
			}
		}
	}
	
	return false
}

// displayUnmatchedAssets displays release assets that don't have matching entries
func displayUnmatchedAssets(releaseAssets []string, assetFilenames map[string]string) {
	// Create a map of expected filenames for quick lookup
	expectedAssets := make(map[string]bool)
	for filename := range assetFilenames {
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
	// Flags specific to check command
	CheckCommand.Flags().StringVar(&checkVersion, "version", "", "Check with specific version (default: uses default_version from spec)")
	CheckCommand.Flags().BoolVar(&checkCheckAssets, "check-assets", true, "Check if generated assets exist in GitHub release")
}
