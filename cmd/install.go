package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/archive"
	"github.com/binary-install/binstaller/pkg/config"
	"github.com/binary-install/binstaller/pkg/fetch"
	"github.com/binary-install/binstaller/pkg/install"
	"github.com/binary-install/binstaller/pkg/resolve"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/binary-install/binstaller/pkg/verify"
	"github.com/spf13/cobra"
)

var (
	installBinDir string
	installDryRun bool
	installDebug  bool
)

// InstallCommand is the command for installing a binary
var InstallCommand = &cobra.Command{
	Use:   "install [VERSION]",
	Short: "Install a binary using the binstaller config",
	Long: `Install a binary using the binstaller config.

This command downloads, verifies, extracts, and installs a binary based on the
configuration in your binstaller.yml file. It provides the same functionality as
the generated installer script but runs natively without shell execution.

Examples:
  # Install latest version
  binst install

  # Install specific version
  binst install v2.40.0

  # Install to custom directory
  binst install -b ~/bin

  # Dry run to see what would be installed
  binst install -n`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInstall,
}

func init() {
	InstallCommand.Flags().StringVarP(&installBinDir, "bindir", "b", "", "Installation directory (default: $BINSTALLER_BIN or $HOME/.local/bin)")
	InstallCommand.Flags().BoolVarP(&installDryRun, "dry-run", "n", false, "Show what would be installed without actually installing")
	InstallCommand.Flags().BoolVarP(&installDebug, "debug", "d", false, "Enable debug logging")
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Set debug logging if requested
	if installDebug {
		log.SetLevel(log.DebugLevel)
	}

	// Determine version to install
	version := "latest"
	if len(args) > 0 {
		version = args[0]
	}

	// Load config
	cfg, configPath, err := config.LoadOrDiscover(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	log.Debugf("Loaded config from %s", configPath)

	// Detect OS and architecture
	osName := os.Getenv("BINSTALLER_OS")
	if osName == "" {
		osName = runtime.GOOS
	}
	arch := os.Getenv("BINSTALLER_ARCH")
	if arch == "" {
		arch = runtime.GOARCH
		// Handle common architecture mappings
		switch arch {
		case "x86_64":
			arch = "amd64"
		case "aarch64":
			arch = "arm64"
		}
	}

	log.Infof("Detected platform: %s/%s", osName, arch)

	// Check for Rosetta 2 emulation on Apple Silicon
	if cfg.Asset != nil && cfg.Asset.ArchEmulation != nil &&
		cfg.Asset.ArchEmulation.Rosetta2 != nil && *cfg.Asset.ArchEmulation.Rosetta2 &&
		osName == "darwin" && arch == "arm64" {
		// Check if we can run x86_64 binaries
		if canRunRosetta2() {
			log.Infof("Apple Silicon with Rosetta 2 found: using amd64 as ARCH")
			arch = "amd64"
		}
	}

	// Resolve version
	resolvedVersion, err := resolve.ResolveVersion(cfg, version)
	if err != nil {
		return fmt.Errorf("failed to resolve version: %w", err)
	}
	log.Infof("Resolved version: %s", resolvedVersion)

	// Generate asset filename
	assetFilename := resolve.AssetFilename(cfg, resolvedVersion, osName, arch)
	log.Infof("Asset filename: %s", assetFilename)

	// Get binary info
	binaries := resolve.GetBinaryInfo(cfg, osName, arch)
	if len(binaries) == 0 {
		return fmt.Errorf("no binaries configured for installation")
	}

	// Resolve install directory
	installDir, err := install.ResolveInstallDir(installBinDir)
	if err != nil {
		return fmt.Errorf("failed to resolve install directory: %w", err)
	}

	// Display dry run info
	if installDryRun {
		fmt.Printf("DRY RUN - Would perform the following:\n")
		fmt.Printf("  Platform: %s/%s\n", osName, arch)
		fmt.Printf("  Version: %s\n", resolvedVersion)
		fmt.Printf("  Asset: %s\n", assetFilename)
		fmt.Printf("  Install to: %s\n", installDir)
		for _, binary := range binaries {
			fmt.Printf("  Binary: %s\n", filepath.Join(installDir, binary.Name))
		}
		return nil
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "binst-install-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download asset
	assetPath := filepath.Join(tmpDir, assetFilename)
	repo := spec.StringValue(cfg.Repo)
	log.Infof("Downloading %s", assetFilename)

	if err := fetch.DownloadAsset(repo, resolvedVersion, assetFilename, assetPath); err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}

	// Verify checksum
	if err := verifyAsset(cfg, assetPath, resolvedVersion, assetFilename, tmpDir, repo); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}
	log.Infof("Checksum verification successful")

	// Extract if needed
	format := archive.DetectFormat(assetFilename)
	extractDir := tmpDir
	isRaw := format == archive.FormatRaw

	if !isRaw {
		log.Infof("Extracting %s", assetFilename)
		stripComponents := 0
		if cfg.Unpack != nil && cfg.Unpack.StripComponents != nil {
			stripComponents = int(*cfg.Unpack.StripComponents)
		}

		if err := archive.Extract(assetPath, extractDir, stripComponents); err != nil {
			return fmt.Errorf("failed to extract archive: %w", err)
		}
	}

	// Install binaries
	for _, binary := range binaries {
		var sourcePath string

		if isRaw {
			// For raw binaries, the downloaded file is the binary
			sourcePath = assetPath
		} else {
			// Find the binary in the extracted directory
			sourcePath, err = archive.FindBinary(extractDir, binary.Path, assetFilename, isRaw)
			if err != nil {
				// List directory contents for debugging
				log.Errorf("Binary not found: %s", binary.Path)
				log.Errorf("Listing contents of %s:", extractDir)
				filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
					if err == nil {
						relPath, _ := filepath.Rel(extractDir, path)
						log.Errorf("  %s", relPath)
					}
					return nil
				})
				return fmt.Errorf("failed to find binary %s: %w", binary.Path, err)
			}
		}

		// Install the binary
		targetPath, err := install.InstallBinary(sourcePath, installDir, binary.Name)
		if err != nil {
			return fmt.Errorf("failed to install %s: %w", binary.Name, err)
		}

		log.Infof("%s installation complete!", binary.Name)
		fmt.Printf("Installed %s to %s\n", binary.Name, targetPath)
	}

	return nil
}

// verifyAsset verifies the downloaded asset using embedded checksums or checksum file
func verifyAsset(cfg *spec.InstallSpec, assetPath, version, assetFilename, tmpDir, repo string) error {
	// Try embedded checksum first
	if err := verify.VerifyWithEmbeddedChecksum(cfg, assetPath, version, assetFilename); err != nil {
		// If no embedded checksum or verification failed, try checksum file
		if cfg.Checksums != nil && cfg.Checksums.Template != nil {
			// Download checksum file
			checksumFilename := generateInstallChecksumFilename(cfg, version)
			checksumPath := filepath.Join(tmpDir, checksumFilename)

			log.Infof("Downloading checksums from %s", checksumFilename)
			if err := fetch.DownloadAsset(repo, version, checksumFilename, checksumPath); err != nil {
				log.Warnf("Failed to download checksum file: %v", err)
				log.Infof("No checksum found, skipping verification")
				return nil
			}

			// Verify using checksum file
			algorithm := spec.Sha256
			if cfg.Checksums.Algorithm != nil {
				algorithm = *cfg.Checksums.Algorithm
			}

			return verify.VerifyWithChecksumFile(assetPath, checksumPath, algorithm)
		}

		// If embedded checksum verification failed (not just missing), return the error
		if strings.Contains(err.Error(), "checksum mismatch") {
			return err
		}

		// No checksums available
		log.Infof("No checksum found, skipping verification")
	}

	return nil
}

// generateInstallChecksumFilename generates the checksum filename from template
func generateInstallChecksumFilename(cfg *spec.InstallSpec, version string) string {
	template := spec.StringValue(cfg.Checksums.Template)
	versionForTemplate := strings.TrimPrefix(version, "v")

	result := template
	result = strings.ReplaceAll(result, "${NAME}", spec.StringValue(cfg.Name))
	result = strings.ReplaceAll(result, "${VERSION}", versionForTemplate)

	return result
}

// canRunRosetta2 checks if Rosetta 2 is available on Apple Silicon
func canRunRosetta2() bool {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return false
	}

	// Try to run a simple x86_64 command
	cmd := exec.Command("arch", "-arch", "x86_64", "true")
	return cmd.Run() == nil
}
