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

// GitHubReleaseResponse represents a GitHub release API response
type GitHubReleaseResponse struct {
	TagName string             `json:"tag_name"`
	Assets  []GitHubReleaseAsset `json:"assets"`
}

// GitHubReleaseAsset represents a GitHub release asset
type GitHubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	SHA256             string `json:"sha256,omitempty"`
	// GitHub API sometimes includes digest information
	Digest string `json:"digest,omitempty"`
}

// assetWithDigest represents an asset with its digest information
type assetWithDigest struct {
	Name     string
	URL      string
	SHA256   string
	Platform spec.Platform
}

// calculateChecksums downloads assets and calculates checksums
func (e *Embedder) calculateChecksums() (map[string]string, error) {
	checksums := make(map[string]string)
	
	// First, fetch actual release assets from GitHub API
	log.Infof("Fetching release assets for version %s...", e.Version)
	releaseAssets, err := e.fetchReleaseAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release assets: %w", err)
	}

	// Match assets to template and get platforms
	matchedAssets, err := e.matchAssetsToTemplate(releaseAssets)
	if err != nil {
		return nil, fmt.Errorf("failed to match assets to template: %w", err)
	}

	if len(matchedAssets) == 0 {
		return nil, fmt.Errorf("no assets found matching the template pattern")
	}

	log.Infof("Found %d matching assets out of %d total assets", len(matchedAssets), len(releaseAssets))

	// Separate assets with and without digests
	var assetsWithDigests []assetWithDigest
	var assetsToDownload []assetWithDigest
	
	for _, asset := range matchedAssets {
		log.Infof("- %s (%s)", asset.Name, func() string {
			if asset.SHA256 != "" {
				return "digest available"
			}
			return "no digest, will download"
		}())
		
		if asset.SHA256 != "" {
			assetsWithDigests = append(assetsWithDigests, asset)
		} else {
			assetsToDownload = append(assetsToDownload, asset)
		}
	}

	// Use API digests for assets that have them
	for _, asset := range assetsWithDigests {
		log.Infof("Using API digest for %s", asset.Name)
		checksums[asset.Name] = asset.SHA256
	}

	// Download and calculate checksums for assets without digests
	if len(assetsToDownload) > 0 {
		log.Infof("Downloading %d assets without digests...", len(assetsToDownload))
		downloadedChecksums, err := e.downloadAndCalculateChecksums(assetsToDownload)
		if err != nil {
			return nil, fmt.Errorf("failed to download and calculate checksums: %w", err)
		}

		// Merge downloaded checksums
		for filename, hash := range downloadedChecksums {
			checksums[filename] = hash
		}
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

// fetchReleaseAssets fetches the list of assets from GitHub API for the specified version
func (e *Embedder) fetchReleaseAssets() ([]GitHubReleaseAsset, error) {
	repo := spec.StringValue(e.Spec.Repo)
	if repo == "" {
		return nil, fmt.Errorf("repository not specified")
	}

	// Construct GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, e.Version)
	
	// Create authenticated request
	req, err := httpclient.NewRequestWithGitHubAuth("GET", apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	// Make the request
	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release from GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse the response
	var release GitHubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	return release.Assets, nil
}

// matchAssetsToTemplate matches GitHub assets to the configured template and extracts platform information
func (e *Embedder) matchAssetsToTemplate(assets []GitHubReleaseAsset) ([]assetWithDigest, error) {
	generator := asset.NewFilenameGenerator(e.Spec, e.Version)
	var platforms []spec.Platform

	// Determine which platforms to check
	if len(e.Spec.SupportedPlatforms) > 0 {
		platforms = e.Spec.SupportedPlatforms
	} else {
		// Get all possible platforms to check against
		platforms = generator.GetAllPossiblePlatforms()
	}

	var matchedAssets []assetWithDigest
	
	// For each platform, check if there's a matching asset
	for _, platform := range platforms {
		filename, err := generator.GenerateFilename(spec.PlatformOSString(platform.OS), spec.PlatformArchString(platform.Arch))
		if err != nil {
			log.Warnf("Failed to generate filename for %s/%s: %v", spec.PlatformOSString(platform.OS), spec.PlatformArchString(platform.Arch), err)
			continue
		}

		// Skip empty filenames
		if filename == "" {
			continue
		}

		// Look for matching asset
		for _, asset := range assets {
			if asset.Name == filename {
				// Extract SHA256 from various possible fields
				sha256 := ""
				if asset.SHA256 != "" {
					sha256 = asset.SHA256
				} else if asset.Digest != "" {
					// Parse digest format like "sha256:abcd1234..."
					if strings.HasPrefix(asset.Digest, "sha256:") {
						sha256 = strings.TrimPrefix(asset.Digest, "sha256:")
					}
				}

				matchedAssets = append(matchedAssets, assetWithDigest{
					Name:     asset.Name,
					URL:      asset.BrowserDownloadURL,
					SHA256:   sha256,
					Platform: platform,
				})
				break
			}
		}
	}

	return matchedAssets, nil
}

// downloadAndCalculateChecksums downloads assets and calculates their checksums
func (e *Embedder) downloadAndCalculateChecksums(assets []assetWithDigest) (map[string]string, error) {
	checksums := make(map[string]string)
	
	// Create a temporary directory for downloads
	tempDir, err := os.MkdirTemp("", "binstaller-checksums")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Use a wait group to process assets concurrently
	var wg sync.WaitGroup
	resultCh := make(chan *checksumResult, len(assets))
	errorCh := make(chan error, len(assets))

	// Process each asset
	for _, asset := range assets {
		wg.Add(1)
		go func(a assetWithDigest) {
			defer wg.Done()

			// Download the asset
			assetPath := filepath.Join(tempDir, a.Name)
			
			log.Infof("Downloading %s", a.URL)
			if err := downloadFile(a.URL, assetPath); err != nil {
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

	return checksums, nil
}
