package checksums

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/asset"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
)

// GitHubReleaseAsset represents a GitHub release asset from the API
type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Digest             string `json:"digest,omitempty"`
}

// GitHubReleaseResponse represents the GitHub API response for a release
type GitHubReleaseResponse struct {
	Assets []GitHubReleaseAsset `json:"assets"`
}

// fetchReleaseAssets fetches the actual release assets from GitHub API
func (e *Embedder) fetchReleaseAssets() ([]GitHubReleaseAsset, error) {
	repo := spec.StringValue(e.Spec.Repo)
	if repo == "" {
		return nil, fmt.Errorf("repository not specified")
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, e.Version)
	log.Infof("Fetching release assets from: %s", url)

	req, err := httpclient.NewRequestWithGitHubAuth("GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch release data: %s", resp.Status)
	}

	var release GitHubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release data: %w", err)
	}

	return release.Assets, nil
}

// matchAssetsToTemplate matches actual release assets against the configured template
func (e *Embedder) matchAssetsToTemplate(assets []GitHubReleaseAsset) ([]GitHubReleaseAsset, error) {
	var matchedAssets []GitHubReleaseAsset
	var platforms []spec.Platform

	// Determine which platforms to consider
	if len(e.Spec.SupportedPlatforms) > 0 {
		platforms = e.Spec.SupportedPlatforms
	} else {
		generator := asset.NewFilenameGenerator(e.Spec, e.Version)
		platforms = generator.GetAllPossiblePlatforms()
	}

	// Create filename generator
	generator := asset.NewFilenameGenerator(e.Spec, e.Version)

	// Build a map of expected filenames for each platform
	expectedFilenames := make(map[string]spec.Platform)
	for _, platform := range platforms {
		filename, err := generator.GenerateFilename(spec.PlatformOSString(platform.OS), spec.PlatformArchString(platform.Arch))
		if err != nil {
			log.Warnf("Failed to generate filename for %s/%s: %v", spec.PlatformOSString(platform.OS), spec.PlatformArchString(platform.Arch), err)
			continue
		}

		if filename != "" {
			expectedFilenames[filename] = platform
		}
	}

	// Match actual assets against expected filenames
	for _, asset := range assets {
		if _, exists := expectedFilenames[asset.Name]; exists {
			matchedAssets = append(matchedAssets, asset)
		}
	}

	if len(matchedAssets) == 0 {
		return nil, fmt.Errorf("no assets found matching the configured template")
	}

	log.Infof("Found %d matching assets out of %d total assets", len(matchedAssets), len(assets))
	return matchedAssets, nil
}

// calculateChecksums downloads assets and calculates checksums using GitHub API
func (e *Embedder) calculateChecksums() (map[string]string, error) {
	checksums := make(map[string]string)

	// First, fetch the actual release assets from GitHub API
	log.Infof("Fetching release assets for version %s...", e.Version)
	assets, err := e.fetchReleaseAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release assets: %w", err)
	}

	// Match assets against the configured template
	matchedAssets, err := e.matchAssetsToTemplate(assets)
	if err != nil {
		return nil, fmt.Errorf("failed to match assets to template: %w", err)
	}

	// Count assets with and without digests
	var assetsWithDigest, assetsWithoutDigest []GitHubReleaseAsset
	for _, asset := range matchedAssets {
		if asset.Digest != "" && strings.HasPrefix(asset.Digest, "sha256:") {
			assetsWithDigest = append(assetsWithDigest, asset)
		} else {
			assetsWithoutDigest = append(assetsWithoutDigest, asset)
		}
	}

	log.Infof("Found %d matching assets:", len(matchedAssets))
	for _, asset := range matchedAssets {
		if asset.Digest != "" && strings.HasPrefix(asset.Digest, "sha256:") {
			log.Infof("- %s (digest available)", asset.Name)
		} else {
			log.Infof("- %s (no digest, will download)", asset.Name)
		}
	}

	// Use API digests directly for assets that have them
	for _, asset := range assetsWithDigest {
		if strings.HasPrefix(asset.Digest, "sha256:") {
			// Extract the hex hash from the digest
			hash := strings.TrimPrefix(asset.Digest, "sha256:")
			checksums[asset.Name] = hash
			log.Infof("Using API digest for %s", asset.Name)
		}
	}

	// Download and calculate checksums for assets without digests
	if len(assetsWithoutDigest) > 0 {
		log.Infof("Downloading %d assets without digests...", len(assetsWithoutDigest))

		// Create a temporary directory for downloads
		tempDir, err := os.MkdirTemp("", "binstaller-checksums")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp directory: %w", err)
		}
		defer os.RemoveAll(tempDir)

		// Use a wait group to process assets concurrently
		var wg sync.WaitGroup
		resultCh := make(chan *checksumResult, len(assetsWithoutDigest))
		errorCh := make(chan error, len(assetsWithoutDigest))

		// Process each asset that needs to be downloaded
		for _, asset := range assetsWithoutDigest {
			wg.Add(1)
			go func(a GitHubReleaseAsset) {
				defer wg.Done()

				// Download the asset
				assetPath := filepath.Join(tempDir, a.Name)
				log.Infof("Downloading %s", a.BrowserDownloadURL)
				if err := downloadFile(a.BrowserDownloadURL, assetPath); err != nil {
					errorCh <- fmt.Errorf("failed to download asset %s: %w", a.Name, err)
					return
				}

				// Calculate the checksum
				hash, err := ComputeHash(assetPath, spec.AlgorithmString(e.Spec.Checksums.Algorithm))
				if err != nil {
					errorCh <- fmt.Errorf("failed to compute hash for %s: %w", a.Name, err)
					return
				}

				resultCh <- &checksumResult{
					Filename: a.Name,
					Hash:     hash,
				}
			}(asset)
		}

		// Wait for all downloads and hash calculations to finish
		wg.Wait()
		close(resultCh)
		close(errorCh)

		// Check for errors
		for err := range errorCh {
			log.Warnf("Error calculating checksum: %v", err)
		}

		// Collect all results
		for result := range resultCh {
			checksums[result.Filename] = result.Hash
		}
	}

	if len(checksums) == 0 {
		return nil, fmt.Errorf("failed to calculate any checksums")
	}

	log.Infof("Successfully calculated checksums for %d assets", len(checksums))
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
