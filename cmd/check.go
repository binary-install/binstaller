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
	"github.com/buildkite/interpolate"
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
- Validating the configuration format and required fields
- Generating asset filenames for all configured platforms
- Verifying if assets exist in the GitHub release (default: enabled)
- Validating checksums template configuration

This helps validate your configuration before generating installer scripts.

Asset Status Meanings:
  ✓ EXISTS       - Asset generated from config exists in GitHub release
  ✗ MISSING      - Asset generated from config not found in release
  ✗ NO MATCH     - Release asset exists but doesn't match any configured platform
  ⚠ NOT SUPPORTED - Feature not supported (e.g., per-asset checksums)
  -              - Non-binary file (e.g., .txt, .json, .sbom)

The unified table shows:
1. Configured platforms and their generated filenames
2. Checksums file status (if configured)
3. Unmatched release assets that might need configuration

Examples:
  # Check the default config file
  binst check

  # Check a specific config file
  binst check -c myapp.binstaller.yml

  # Check without verifying GitHub assets
  binst check --check-assets=false

  # Check with a specific version
  binst check --version v1.2.3`,
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

	// Check checksums filename if configured
	checksumFilename := ""
	checksumError := ""
	if installSpec.Checksums != nil && installSpec.Checksums.Template != nil {
		cf, err := generateChecksumFilename(installSpec, version)
		if err != nil {
			if strings.Contains(err.Error(), "per-asset checksums") {
				checksumError = "per-asset"
			}
		} else {
			checksumFilename = cf
		}
	}

	// Build a comprehensive list of all assets
	type assetEntry struct {
		platform string
		filename string
		status   string
		priority int // 0=configured, 1=other binary, 2=non-binary
	}
	var allAssets []assetEntry

	// Add configured platform assets
	for platform, filename := range assetFilenames {
		status := "✓ EXISTS"
		if !existingAssets[filename] {
			status = "✗ MISSING"
		}
		allAssets = append(allAssets, assetEntry{
			platform: platform,
			filename: filename,
			status:   status,
			priority: 0,
		})
		// Mark as processed
		delete(existingAssets, filename)
	}

	// Add checksums if configured
	if checksumError == "per-asset" {
		allAssets = append(allAssets, assetEntry{
			platform: "checksums",
			filename: "(per-asset pattern)",
			status:   "⚠ NOT SUPPORTED",
			priority: 0,
		})
	} else if checksumFilename != "" {
		status := "✗ MISSING"
		if existingAssets[checksumFilename] {
			status = "✓ EXISTS"
			delete(existingAssets, checksumFilename)
		}
		allAssets = append(allAssets, assetEntry{
			platform: "checksums",
			filename: checksumFilename,
			status:   status,
			priority: 0,
		})
	}

	// Add remaining assets from release
	for asset := range existingAssets {
		if isNonBinaryAsset(asset) {
			allAssets = append(allAssets, assetEntry{
				platform: "-",
				filename: asset,
				status:   "-",
				priority: 2,
			})
		} else {
			allAssets = append(allAssets, assetEntry{
				platform: "-",
				filename: asset,
				status:   "✗ NO MATCH",
				priority: 1,
			})
		}
	}

	// Sort: configured first, then other binaries, then non-binaries
	// Within each group, sort by status (EXISTS before MISSING) and name
	sort.Slice(allAssets, func(i, j int) bool {
		if allAssets[i].priority != allAssets[j].priority {
			return allAssets[i].priority < allAssets[j].priority
		}
		// Same priority - sort by status
		if allAssets[i].status != allAssets[j].status {
			// Define status order
			statusOrder := map[string]int{
				"✓ EXISTS":        0,
				"✗ MISSING":       1,
				"✗ NO MATCH":      2,
				"⚠ NOT SUPPORTED": 3,
				"-":               4,
			}
			return statusOrder[allAssets[i].status] < statusOrder[allAssets[j].status]
		}
		// Same status - sort by filename
		return allAssets[i].filename < allAssets[j].filename
	})

	// Display unified table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PLATFORM\tASSET FILENAME\tSTATUS")
	fmt.Fprintln(w, "--------\t--------------\t------")

	for _, asset := range allAssets {
		fmt.Fprintf(w, "%s\t%s\t%s\n", asset.platform, asset.filename, asset.status)
	}

	w.Flush()

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

	// Check if checksums file is configured
	checksumFilename := ""
	if installSpec.Checksums != nil && installSpec.Checksums.Template != nil {
		if cf, err := generateChecksumFilename(installSpec, version); err == nil {
			checksumFilename = cf
		}
	}

	// First pass: categorize assets
	type assetInfo struct {
		name     string
		platform string
		status   string
	}
	var assets []assetInfo

	for _, assetName := range releaseAssets {
		// Check if this is the checksums file
		if checksumFilename != "" && assetName == checksumFilename {
			continue // Will be handled separately
		}

		// Determine the type and status of the asset
		var info assetInfo
		info.name = assetName

		if isNonBinaryAsset(assetName) {
			// Non-binary assets (signatures, checksums, etc.)
			info.platform = "-"
			info.status = "-"
		} else {
			// Check if this asset matches any generated filename
			platform := ""
			for filename, plat := range assetFilenames {
				if filename == assetName {
					platform = plat
					break
				}
			}

			if platform != "" {
				info.platform = platform
				info.status = "✓ MATCHED"
			} else {
				info.platform = "-"
				info.status = "✗ NO MATCH"
			}
		}

		assets = append(assets, info)
	}

	// Sort assets: MATCHED first, then NO MATCH, then non-binary
	sort.Slice(assets, func(i, j int) bool {
		// Define sort priority
		getPriority := func(status string) int {
			switch status {
			case "✓ MATCHED":
				return 0
			case "✗ NO MATCH":
				return 1
			default: // "-"
				return 2
			}
		}

		pi := getPriority(assets[i].status)
		pj := getPriority(assets[j].status)

		if pi != pj {
			return pi < pj
		}
		// If same priority, sort by name
		return assets[i].name < assets[j].name
	})

	// Display sorted assets
	for _, asset := range assets {
		fmt.Fprintf(w, "%s\t%s\t%s\n", asset.name, asset.platform, asset.status)
	}

	// Add checksums row if configured
	if installSpec.Checksums != nil && installSpec.Checksums.Template != nil {
		checksumFilename, err := generateChecksumFilename(installSpec, version)
		if err != nil {
			// Show error message for unsupported checksums configuration
			if strings.Contains(err.Error(), "per-asset checksums") {
				fmt.Fprintf(w, "(per-asset pattern)\tchecksums\t⚠ NOT SUPPORTED\n")
			}
		} else {
			if releaseAssetMap[checksumFilename] {
				fmt.Fprintf(w, "%s\tchecksums\t✓ MATCHED\n", checksumFilename)
			} else {
				fmt.Fprintf(w, "%s\tchecksums\t✗ MISSING\n", checksumFilename)
			}
		}
	}

	w.Flush()

	return nil
}

// isNonBinaryAsset checks if an asset is likely not a binary (e.g., checksums, signatures)
func isNonBinaryAsset(filename string) bool {
	nonBinaryPatterns := []string{
		".txt", ".sha256", ".sha512", ".md5", ".sig", ".asc", ".pem",
		".sbom", ".json", ".yml", ".yaml", ".sh", ".ps1", ".md",
		"checksums", "SHASUMS", "SHA256SUMS", "README", "LICENSE",
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

// generateChecksumFilename generates the checksums filename using the template
func generateChecksumFilename(installSpec *spec.InstallSpec, version string) (string, error) {
	if installSpec.Checksums == nil || installSpec.Checksums.Template == nil {
		return "", fmt.Errorf("checksums template not specified")
	}

	checksumTemplate := spec.StringValue(installSpec.Checksums.Template)
	if checksumTemplate == "" {
		return "", fmt.Errorf("checksums template not specified")
	}

	// Check if template uses ASSET_FILENAME
	if strings.Contains(checksumTemplate, "${ASSET_FILENAME}") {
		// This is a per-asset checksum pattern, not supported in check command
		return "", fmt.Errorf("per-asset checksums (${ASSET_FILENAME}) not supported in check command")
	}

	// Create environment map for interpolation
	envMap := map[string]string{
		"NAME":    spec.StringValue(installSpec.Name),
		"TAG":     version,
		"VERSION": strings.TrimPrefix(version, "v"),
	}

	// Perform variable substitution
	env := interpolate.NewMapEnv(envMap)
	checksumFilename, err := interpolate.Interpolate(env, checksumTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to interpolate checksums template: %w", err)
	}

	return checksumFilename, nil
}

// removeFromSlice removes a string from a slice
func removeFromSlice(slice []string, item string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}

func init() {
	// Flags specific to check command
	CheckCommand.Flags().StringVar(&checkVersion, "version", "", "Check with specific version (default: uses default_version from spec)")
	CheckCommand.Flags().BoolVar(&checkCheckAssets, "check-assets", true, "Check if generated assets exist in GitHub release")
}
