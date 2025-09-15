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
	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
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
		// In dry-run mode, verify the URL exists
		log.Info("Validating asset URL...")
		if err := validateURL(ctx, assetURL); err != nil {
			return fmt.Errorf("asset validation failed: %w", err)
		}
		log.Info("Asset URL is valid")
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
	if err := downloadWithProgress(ctx, assetPath, assetURL); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// TODO: Phase 3+ implementation
	// - Use pkg/checksums for verification
	// - Implement extraction
	// - Install binary

	return fmt.Errorf("installation not yet implemented (Phase 2 complete)")
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
	osName := runtime.GOOS

	// Map Go OS names to shell script conventions
	switch osName {
	case "windows":
		// Check for MSYS, MinGW, Cygwin environments
		// In Go, we just use "windows"
		return "windows"
	case "sunos":
		// Try to detect illumos vs solaris
		// For now, just return solaris
		return "solaris"
	default:
		return osName
	}
}

// detectArch detects the architecture, matching shell script logic
func detectArch() string {
	arch := runtime.GOARCH

	// Map Go arch names to shell script conventions
	switch arch {
	case "amd64":
		return "amd64"
	case "386":
		return "386"
	case "arm64":
		return "arm64"
	case "arm":
		// Go doesn't distinguish ARM versions like the shell script
		// Default to armv7 for compatibility
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

// validateURL checks if a URL exists by making a HEAD request
func validateURL(ctx context.Context, url string) error {
	client := httpclient.NewGitHubClient()
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("URL returned status %d", resp.StatusCode)
	}

	return nil
}

// downloadWithProgress downloads a file with progress reporting
func downloadWithProgress(ctx context.Context, destPath, url string) error {
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

	// Create a progress reader
	contentLength := resp.ContentLength
	reader := &progressReader{
		Reader:  resp.Body,
		Total:   contentLength,
		Current: 0,
	}

	// Copy with progress
	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// progressReader wraps an io.Reader to report progress
type progressReader struct {
	Reader  io.Reader
	Total   int64
	Current int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Current += int64(n)
		if pr.Total > 0 {
			percentage := float64(pr.Current) * 100.0 / float64(pr.Total)
			fmt.Printf("\r%.1f%% (%d/%d bytes)", percentage, pr.Current, pr.Total)
		} else {
			fmt.Printf("\r%d bytes downloaded", pr.Current)
		}
	}
	if err == io.EOF {
		fmt.Println() // New line after progress
	}
	return n, err
}
