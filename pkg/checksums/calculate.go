package checksums

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/asset"
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
		// If no platforms specified, use all possible combinations
		generator := asset.NewFilenameGenerator(e.Spec, e.Version)
		platforms = generator.GetAllPossiblePlatforms()
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

	// Create filename generator
	generator := asset.NewFilenameGenerator(e.Spec, e.Version)

	// Process each platform
	for _, platform := range platforms {
		wg.Add(1)
		go func(p spec.Platform) {
			defer wg.Done()

			filename, err := generator.GenerateFilename(spec.PlatformOSString(p.OS), spec.PlatformArchString(p.Arch))
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
