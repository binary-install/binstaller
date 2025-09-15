package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/archive"
	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/checksums"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/buildkite/interpolate"
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

	// Apply defaults (including setting Name from Repo if not specified)
	spec.SetDefaults()

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

	// Phase 2: Asset Resolution and Download
	// 5. Detect OS/Arch
	osName, arch := detectPlatform(spec)
	log.Infof("Detected Platform: %s/%s", osName, arch)

	// 6. Generate asset filename
	generator := asset.NewFilenameGenerator(spec, versionNumber)
	assetFilename, err := generator.GenerateFilename(osName, arch)
	if err != nil {
		return fmt.Errorf("failed to generate asset filename: %w", err)
	}
	log.Infof("Resolved asset filename: %s", assetFilename)

	// 7. Construct download URL
	assetURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, resolvedVersion, assetFilename)
	log.Infof("Asset URL: %s", assetURL)

	if installDryRun {
		// In dry-run mode, just print what would be done
		log.Info("Dry run mode - would download from: " + assetURL)
		return nil
	}

	// 8. Download asset to temporary file
	tmpDir, err := os.MkdirTemp("", "binst-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	assetPath := filepath.Join(tmpDir, assetFilename)
	log.Infof("Downloading %s", assetURL)
	if err := download(ctx, assetPath, assetURL); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Phase 3: Checksum Verification
	log.Infof("Verifying checksum for %s", assetFilename)
	verifier := checksums.NewVerifier(spec, resolvedVersion)
	if err := verifier.VerifyFile(ctx, assetPath, assetFilename); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Phase 3: Archive Extraction
	stripComponents := 0
	if spec.Unpack != nil && spec.Unpack.StripComponents != nil {
		stripComponents = int(*spec.Unpack.StripComponents)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	extractor := archive.NewExtractor(stripComponents)
	log.Infof("Extracting %s", assetFilename)
	if err := extractor.Extract(assetPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Phase 3: Binary Selection
	binaries, err := selectBinaries(spec, osName, arch, extractDir, assetFilename)
	if err != nil {
		return fmt.Errorf("failed to select binaries: %w", err)
	}
	for _, binary := range binaries {
		log.Infof("Selected binary: %s (from %s)", binary.Name, binary.Path)
	}

	// Phase 4: Installation
	// Determine installation directory
	binDir := installBinDir
	if binDir == "" {
		// Check $BINSTALLER_BIN environment variable first
		binDir = os.Getenv("BINSTALLER_BIN")
		if binDir == "" {
			// Default to ~/.local/bin
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			binDir = filepath.Join(homeDir, ".local", "bin")
		}
	}

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Install all binaries
	for _, binary := range binaries {
		destPath := filepath.Join(binDir, binary.Name)
		srcPath := filepath.Join(extractDir, binary.Path)

		log.Infof("Installing %s to %s", binary.Name, destPath)
		if err := installBinary(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to install binary %s: %w", binary.Name, err)
		}
	}

	log.Infof("Successfully installed %s %s to %s", *spec.Name, versionNumber, binDir)
	return nil
}

// detectPlatform detects the current OS and architecture, matching shell script logic
func detectPlatform(spec *spec.InstallSpec) (string, string) {
	osName := detectOS()
	arch := detectArch()

	// Handle Rosetta 2 on Apple Silicon
	if spec.Asset != nil && spec.Asset.ArchEmulation != nil &&
		spec.Asset.ArchEmulation.Rosetta2 != nil && *spec.Asset.ArchEmulation.Rosetta2 {
		if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" && isRosetta2Available() {
			log.Info("Apple Silicon with Rosetta 2 found: using amd64 as ARCH")
			arch = "amd64"
		}
	}

	return osName, arch
}

// detectOS detects the operating system, matching shell script logic
func detectOS() string {
	return runtime.GOOS
}

// detectArch detects the architecture, matching shell script logic
func detectArch() string {
	arch := runtime.GOARCH

	// Map Go arch names to shell script conventions
	switch arch {
	case "arm":
		// TODO: Handle ARM version detection properly
		// For now, use uname to detect ARM version
		return "armv7"
	default:
		return arch
	}
}

// isRosetta2Available checks if Rosetta 2 is available on macOS
func isRosetta2Available() bool {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return false
	}

	// Try to run a simple x86_64 command
	cmd := exec.Command("arch", "-arch", "x86_64", "true")
	err := cmd.Run()
	return err == nil
}

// download downloads a file without progress reporting
func download(ctx context.Context, destPath, url string) error {
	client := httpclient.NewGitHubClient()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Create the destination file
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy without progress
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// BinaryInfo holds information about a binary to install
type BinaryInfo struct {
	Name string
	Path string
}

// selectBinaries selects all binaries from the extracted files based on the spec
func selectBinaries(installSpec *spec.InstallSpec, osName, arch string, extractDir string, assetFilename string) ([]BinaryInfo, error) {
	// Get binaries configuration
	binariesConfig := getBinariesForPlatform(installSpec, osName, arch)
	if len(binariesConfig) == 0 {
		return nil, fmt.Errorf("no binaries configured")
	}

	var result []BinaryInfo

	// Process each binary in the configuration
	for _, binary := range binariesConfig {
		binaryName := spec.StringValue(binary.Name)
		if binaryName == "" {
			binaryName = spec.StringValue(installSpec.Name)
		}

		binaryPath := spec.StringValue(binary.Path)
		if binaryPath == "" {
			binaryPath = binaryName
		}

		// Interpolate variables in the path
		binaryPath, err := interpolateBinaryPath(binaryPath, assetFilename, extractDir)
		if err != nil {
			return nil, fmt.Errorf("failed to interpolate binary path: %w", err)
		}

		// Verify the binary exists
		fullPath := filepath.Join(extractDir, binaryPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("binary not found at %s", binaryPath)
		}

		result = append(result, BinaryInfo{
			Name: binaryName,
			Path: binaryPath,
		})
	}

	return result, nil
}

// interpolateBinaryPath handles variable interpolation in binary paths
func interpolateBinaryPath(path string, assetFilename string, extractDir string) (string, error) {
	// Handle ${ASSET_FILENAME} using interpolate package
	if strings.Contains(path, "${ASSET_FILENAME}") {
		// Create environment map
		envMap := map[string]string{
			"ASSET_FILENAME": assetFilename,
		}
		env := interpolate.NewMapEnv(envMap)
		interpolated, err := interpolate.Interpolate(env, path)
		if err != nil {
			return "", fmt.Errorf("failed to interpolate path: %w", err)
		}
		path = interpolated
	}

	// Special case: if path is the asset filename itself, check if it's the only file
	if path == assetFilename {
		files, err := archive.ListFiles(extractDir)
		if err != nil {
			return "", fmt.Errorf("failed to list extracted files: %w", err)
		}

		if len(files) == 1 {
			return files[0], nil
		}
	}

	return path, nil
}

// getBinariesForPlatform returns the binaries configuration for the given platform
func getBinariesForPlatform(spec *spec.InstallSpec, osName, arch string) []spec.BinaryElement {
	if spec.Asset == nil {
		return nil
	}

	// Start with default binaries
	binaries := spec.Asset.Binaries

	// Apply matching rules
	for _, rule := range spec.Asset.Rules {
		if matchesRule(rule.When, osName, arch) && len(rule.Binaries) > 0 {
			binaries = rule.Binaries
		}
	}

	return binaries
}

// matchesRule checks if a platform matches a rule condition
func matchesRule(when *spec.When, osName, arch string) bool {
	if when == nil {
		return true
	}

	// Check OS match
	if when.OS != nil && *when.OS != osName {
		return false
	}

	// Check architecture match
	if when.Arch != nil && *when.Arch != arch {
		return false
	}

	return true
}

// installBinary copies the binary to its destination and makes it executable
func installBinary(src, dest string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy file
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Ensure file is executable
	if err := os.Chmod(dest, 0755); err != nil {
		return fmt.Errorf("failed to make file executable: %w", err)
	}

	return nil
}
