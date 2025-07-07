package checksums

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
)

// calculateChecksums downloads assets and calculates checksums
func (e *Embedder) calculateChecksums() (map[string]string, error) {
	checksums := make(map[string]string)
	var platforms []spec.Platform

	// Determine which platforms to calculate checksums for
	if len(e.Spec.SupportedPlatforms) > 0 {
		// Use the supported platforms from the spec
		platforms = e.Spec.SupportedPlatforms
	} else {
		// If no platforms specified, use common ones
		platforms = getCommonPlatforms()
	}

	// Create a temporary directory for downloads
	tempDir, err := os.MkdirTemp("", "binstaller-checksums")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a wait group to process platforms concurrently
	var wg sync.WaitGroup
	resultCh := make(chan *checksumResult, len(platforms))
	errorCh := make(chan error, len(platforms))

	// Process each platform
	for _, platform := range platforms {
		wg.Add(1)
		go func(p spec.Platform) {
			defer wg.Done()

			filename, err := e.generateAssetFilename(spec.PlatformOSString(p.OS), spec.PlatformArchString(p.Arch))
			if err != nil {
				errorCh <- fmt.Errorf("failed to generate asset filename for %s/%s: %w", spec.PlatformOSString(p.OS), spec.PlatformArchString(p.Arch), err)
				return
			}

			// Skip empty filenames
			if filename == "" {
				log.Warnf("Skipping empty filename for %s/%s", spec.PlatformOSString(p.OS), spec.PlatformArchString(p.Arch))
				return
			}

			// Download the asset
			assetPath := filepath.Join(tempDir, filename)
			assetURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
				spec.StringValue(e.Spec.Repo), e.Version, filename)

			log.Infof("Downloading %s", assetURL)
			if err := downloadFile(assetURL, assetPath); err != nil {
				// Just log the error but don't fail the entire process
				log.Warnf("Failed to download asset %s: %v", assetURL, err)
				return
			}

			// Calculate the checksum
			hash, err := ComputeHash(assetPath, spec.AlgorithmString(e.Spec.Checksums.Algorithm))
			if err != nil {
				errorCh <- fmt.Errorf("failed to compute hash for %s: %w", filename, err)
				return
			}

			resultCh <- &checksumResult{
				Filename: filename,
				Hash:     hash,
			}
		}(platform)
	}

	// Wait for all downloads and hash calculations to finish
	wg.Wait()
	close(resultCh)
	close(errorCh)

	// Check for errors
	for err := range errorCh {
		// Log the error but continue processing
		log.Warnf("Error calculating checksum: %v", err)
	}

	// Collect all results
	for result := range resultCh {
		checksums[result.Filename] = result.Hash
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("failed to calculate any checksums")
	}

	return checksums, nil
}

// checksumResult represents a checksum calculation result
type checksumResult struct {
	Filename string
	Hash     string
}

// generateAssetFilename creates an asset filename for a specific OS and Arch
func (e *Embedder) generateAssetFilename(osInput, archInput string) (string, error) {
	if e.Spec == nil || e.Spec.Asset == nil || spec.StringValue(e.Spec.Asset.Template) == "" {
		return "", fmt.Errorf("asset template not defined in spec")
	}

	// Keep original values for rule matching
	osMatch := strings.ToLower(osInput)
	archMatch := strings.ToLower(archInput)

	// Create formatted values for template substitution
	osValue := osMatch
	archValue := archMatch

	// Apply OS/Arch naming conventions for template values
	if e.Spec.Asset.NamingConvention != nil {
		if spec.NamingConventionOSString(e.Spec.Asset.NamingConvention.OS) == "titlecase" {
			osValue = titleCase(osValue)
		}
	}

	// Apply rules to get the right extension and override OS/Arch if needed
	ext := spec.StringValue(e.Spec.Asset.DefaultExtension)
	template := spec.StringValue(e.Spec.Asset.Template)

	// Check if any rule applies - use osMatch/archMatch for condition checking
	for _, rule := range e.Spec.Asset.Rules {
		if rule.When != nil &&
			(spec.StringValue(rule.When.OS) == "" || spec.StringValue(rule.When.OS) == osMatch) &&
			(spec.StringValue(rule.When.Arch) == "" || spec.StringValue(rule.When.Arch) == archMatch) {
			if spec.StringValue(rule.OS) != "" {
				osValue = spec.StringValue(rule.OS)
			}
			if spec.StringValue(rule.Arch) != "" {
				archValue = spec.StringValue(rule.Arch)
			}
			if spec.StringValue(rule.EXT) != "" {
				ext = spec.StringValue(rule.EXT)
			}
			if spec.StringValue(rule.Template) != "" {
				template = spec.StringValue(rule.Template)
			}
		}
	}

	// Asset templates support OS, ARCH, and EXT in addition to NAME and VERSION
	additionalVars := map[string]string{
		"OS":   osValue,
		"ARCH": archValue,
		"EXT":  ext,
	}

	// Perform variable substitution in the template
	filename, err := e.interpolateTemplate(template, additionalVars)
	if err != nil {
		return "", fmt.Errorf("failed to interpolate asset template: %w", err)
	}

	return filename, nil
}

// titleCase converts a string to title case (first letter uppercase, rest lowercase)
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

// downloadFile downloads a file from a URL to a local path
func downloadFile(url, filepath string) error {
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Create request with GitHub auth if needed
	req, err := httpclient.NewRequestWithGitHubAuth("GET", url)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Get the data
	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

// getCommonPlatforms returns a list of common platforms
func getCommonPlatforms() []spec.Platform {
	linuxOS := spec.Linux
	darwinOS := spec.Darwin
	windowsOS := spec.Windows
	amd64Arch := spec.Amd64
	arm64Arch := spec.Arm64
	x86Arch := spec.The386

	return []spec.Platform{
		{OS: &linuxOS, Arch: &amd64Arch},
		{OS: &linuxOS, Arch: &arm64Arch},
		{OS: &darwinOS, Arch: &amd64Arch},
		{OS: &darwinOS, Arch: &arm64Arch},
		{OS: &windowsOS, Arch: &amd64Arch},
		{OS: &windowsOS, Arch: &x86Arch},
	}
}
