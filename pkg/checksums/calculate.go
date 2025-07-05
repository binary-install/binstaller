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

			filename, err := e.generateAssetFilename(p.OS, p.Arch)
			if err != nil {
				errorCh <- fmt.Errorf("failed to generate asset filename for %s/%s: %w", p.OS, p.Arch, err)
				return
			}

			// Skip empty filenames
			if filename == "" {
				log.Warnf("Skipping empty filename for %s/%s", p.OS, p.Arch)
				return
			}

			// Download the asset
			assetPath := filepath.Join(tempDir, filename)
			assetURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
				e.Spec.Repo, e.Version, filename)

			log.Infof("Downloading %s", assetURL)
			if err := downloadFile(assetURL, assetPath); err != nil {
				// Just log the error but don't fail the entire process
				log.Warnf("Failed to download asset %s: %v", assetURL, err)
				return
			}

			// Calculate the checksum
			hash, err := ComputeHash(assetPath, e.Spec.Checksums.Algorithm)
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
	if e.Spec == nil || e.Spec.Asset.Template == "" {
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
		if e.Spec.Asset.NamingConvention.OS == "titlecase" {
			osValue = titleCase(osValue)
		}
	}

	// Apply rules to get the right extension and override OS/Arch if needed
	ext := e.Spec.Asset.DefaultExtension
	template := e.Spec.Asset.Template

	// Check if any rule applies - use osMatch/archMatch for condition checking
	for _, rule := range e.Spec.Asset.Rules {
		if (rule.When.OS == "" || rule.When.OS == osMatch) &&
			(rule.When.Arch == "" || rule.When.Arch == archMatch) {
			if rule.OS != "" {
				osValue = rule.OS
			}
			if rule.Arch != "" {
				archValue = rule.Arch
			}
			if rule.Ext != "" {
				ext = rule.Ext
			}
			if rule.Template != "" {
				template = rule.Template
			}
		}
	}

	// Perform variable substitution in the template
	filename := template
	filename = strings.ReplaceAll(filename, "${NAME}", e.Spec.Name)
	filename = strings.ReplaceAll(filename, "${VERSION}", e.Version)
	filename = strings.ReplaceAll(filename, "${OS}", osValue)
	filename = strings.ReplaceAll(filename, "${ARCH}", archValue)
	filename = strings.ReplaceAll(filename, "${EXT}", ext)

	// For consistency with the shell script, also handle repo owner/name expansion
	if strings.Contains(filename, "${REPO_OWNER}") || strings.Contains(filename, "${REPO_NAME}") {
		parts := strings.SplitN(e.Spec.Repo, "/", 2)
		if len(parts) == 2 {
			filename = strings.ReplaceAll(filename, "${REPO_OWNER}", parts[0])
			filename = strings.ReplaceAll(filename, "${REPO_NAME}", parts[1])
		}
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

	// Get the data
	resp, err := http.Get(url)
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
	return []spec.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "windows", Arch: "386"},
	}
}
