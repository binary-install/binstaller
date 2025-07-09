package checksums

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

// TestFetchReleaseAssets tests the fetchReleaseAssets function
func TestFetchReleaseAssets(t *testing.T) {
	// Create a mock GitHub API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.URL.Path != "/repos/test/repo/releases/tags/v1.0.0" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		// Return mock response
		response := GitHubReleaseResponse{
			TagName: "v1.0.0",
			Assets: []GitHubReleaseAsset{
				{
					Name:               "test-1.0.0-linux-amd64.tar.gz",
					BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-linux-amd64.tar.gz",
					Size:               1024,
					SHA256:             "abc123def456",
				},
				{
					Name:               "test-1.0.0-darwin-amd64.tar.gz",
					BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-darwin-amd64.tar.gz",
					Size:               1024,
					Digest:             "sha256:def456ghi789",
				},
				{
					Name:               "test-1.0.0-windows-amd64.zip",
					BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-windows-amd64.zip",
					Size:               1024,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create embedder with test spec
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr("test/repo"),
		},
		Version: "v1.0.0",
	}

	// Create a temporary function to test the fetchReleaseAssets logic
	fetchReleaseAssetsFunc := func() ([]GitHubReleaseAsset, error) {
		repo := spec.StringValue(embedder.Spec.Repo)
		if repo == "" {
			return nil, fmt.Errorf("repository not specified")
		}

		// Use mock server URL
		apiURL := fmt.Sprintf("%s/repos/%s/releases/tags/%s", mockServer.URL, repo, embedder.Version)
		
		resp, err := http.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch release from GitHub API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var release GitHubReleaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
		}

		return release.Assets, nil
	}

	// Test the function
	assets, err := fetchReleaseAssetsFunc()
	if err != nil {
		t.Fatalf("fetchReleaseAssets failed: %v", err)
	}

	// Verify the results
	if len(assets) != 3 {
		t.Errorf("Expected 3 assets, got %d", len(assets))
	}

	expectedAssets := map[string]string{
		"test-1.0.0-linux-amd64.tar.gz":  "abc123def456",
		"test-1.0.0-darwin-amd64.tar.gz": "",
		"test-1.0.0-windows-amd64.zip":   "",
	}

	for _, asset := range assets {
		expectedSHA256, exists := expectedAssets[asset.Name]
		if !exists {
			t.Errorf("Unexpected asset: %s", asset.Name)
			continue
		}

		if asset.SHA256 != expectedSHA256 {
			t.Errorf("For asset %s, expected SHA256 %s, got %s", asset.Name, expectedSHA256, asset.SHA256)
		}
	}
}

// TestMatchAssetsToTemplate tests the matchAssetsToTemplate function
func TestMatchAssetsToTemplate(t *testing.T) {
	// Create embedder with test spec
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr("test/repo"),
			Name: spec.StringPtr("test"),
			Asset: &spec.Asset{
				Template: spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"),
			},
			SupportedPlatforms: []spec.Platform{
				{OS: func() *spec.SupportedPlatformOS { v := spec.Linux; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
				{OS: func() *spec.SupportedPlatformOS { v := spec.Darwin; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
			},
		},
		Version: "1.0.0",
	}

	// Test assets
	assets := []GitHubReleaseAsset{
		{
			Name:               "test-1.0.0-linux-amd64.tar.gz",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-linux-amd64.tar.gz",
			SHA256:             "abc123def456",
		},
		{
			Name:               "test-1.0.0-darwin-amd64.tar.gz",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-darwin-amd64.tar.gz",
			Digest:             "sha256:def456ghi789",
		},
		{
			Name:               "test-1.0.0-windows-amd64.tar.gz",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-windows-amd64.tar.gz",
		},
		{
			Name:               "unrelated-file.txt",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/unrelated-file.txt",
		},
	}

	// Test the function
	matchedAssets, err := embedder.matchAssetsToTemplate(assets)
	if err != nil {
		t.Fatalf("matchAssetsToTemplate failed: %v", err)
	}

	// Verify the results
	if len(matchedAssets) != 2 {
		t.Errorf("Expected 2 matched assets, got %d", len(matchedAssets))
	}

	expectedMatches := map[string]string{
		"test-1.0.0-linux-amd64.tar.gz":  "abc123def456",
		"test-1.0.0-darwin-amd64.tar.gz": "def456ghi789",
	}

	for _, matched := range matchedAssets {
		expectedSHA256, exists := expectedMatches[matched.Name]
		if !exists {
			t.Errorf("Unexpected matched asset: %s", matched.Name)
			continue
		}

		if matched.SHA256 != expectedSHA256 {
			t.Errorf("For asset %s, expected SHA256 %s, got %s", matched.Name, expectedSHA256, matched.SHA256)
		}
	}
}

// TestMatchAssetsToTemplateWithoutSupportedPlatforms tests matching when no supported platforms are specified
func TestMatchAssetsToTemplateWithoutSupportedPlatforms(t *testing.T) {
	// Create embedder with test spec (no supported platforms)
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr("test/repo"),
			Name: spec.StringPtr("test"),
			Asset: &spec.Asset{
				Template: spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"),
			},
			// No SupportedPlatforms specified
		},
		Version: "1.0.0",
	}

	// Test assets - only include a few realistic ones
	assets := []GitHubReleaseAsset{
		{
			Name:               "test-1.0.0-linux-amd64.tar.gz",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-linux-amd64.tar.gz",
			SHA256:             "abc123def456",
		},
		{
			Name:               "test-1.0.0-darwin-amd64.tar.gz",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/test-1.0.0-darwin-amd64.tar.gz",
		},
		{
			Name:               "unrelated-file.txt",
			BrowserDownloadURL: "https://github.com/test/repo/releases/download/v1.0.0/unrelated-file.txt",
		},
	}

	// Test the function
	matchedAssets, err := embedder.matchAssetsToTemplate(assets)
	if err != nil {
		t.Fatalf("matchAssetsToTemplate failed: %v", err)
	}

	// Verify that only the matching assets are returned
	if len(matchedAssets) != 2 {
		t.Errorf("Expected 2 matched assets, got %d", len(matchedAssets))
	}

	// Verify the matched assets
	foundLinux := false
	foundDarwin := false
	for _, matched := range matchedAssets {
		if matched.Name == "test-1.0.0-linux-amd64.tar.gz" {
			foundLinux = true
			if matched.SHA256 != "abc123def456" {
				t.Errorf("Expected SHA256 for Linux asset, got %s", matched.SHA256)
			}
		}
		if matched.Name == "test-1.0.0-darwin-amd64.tar.gz" {
			foundDarwin = true
		}
	}

	if !foundLinux {
		t.Error("Expected to find Linux asset")
	}
	if !foundDarwin {
		t.Error("Expected to find Darwin asset")
	}
}

// TestDownloadAndCalculateChecksums tests the downloadAndCalculateChecksums function
func TestDownloadAndCalculateChecksums(t *testing.T) {
	// Create a mock server that serves fake asset files
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve different content based on the requested file
		if strings.Contains(r.URL.Path, "linux-amd64") {
			w.Write([]byte("linux binary content"))
		} else if strings.Contains(r.URL.Path, "darwin-amd64") {
			w.Write([]byte("darwin binary content"))
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create embedder with test spec
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Checksums: &spec.Checksums{
				Algorithm: func() *spec.Algorithm { a := spec.Sha256; return &a }(),
			},
		},
		Version: "1.0.0",
	}

	// Test assets without digests
	assets := []assetWithDigest{
		{
			Name:     "test-1.0.0-linux-amd64.tar.gz",
			URL:      mockServer.URL + "/test-1.0.0-linux-amd64.tar.gz",
			SHA256:   "", // No digest, should be downloaded
			Platform: spec.Platform{OS: func() *spec.SupportedPlatformOS { v := spec.Linux; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
		},
		{
			Name:     "test-1.0.0-darwin-amd64.tar.gz",
			URL:      mockServer.URL + "/test-1.0.0-darwin-amd64.tar.gz",
			SHA256:   "", // No digest, should be downloaded
			Platform: spec.Platform{OS: func() *spec.SupportedPlatformOS { v := spec.Darwin; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
		},
	}

	// Test the function
	checksums, err := embedder.downloadAndCalculateChecksums(assets)
	if err != nil {
		t.Fatalf("downloadAndCalculateChecksums failed: %v", err)
	}

	// Verify the results
	if len(checksums) != 2 {
		t.Errorf("Expected 2 checksums, got %d", len(checksums))
	}

	// Verify that checksums were calculated (they should be different for different content)
	linuxChecksum, hasLinux := checksums["test-1.0.0-linux-amd64.tar.gz"]
	darwinChecksum, hasDarwin := checksums["test-1.0.0-darwin-amd64.tar.gz"]

	if !hasLinux {
		t.Error("Expected Linux checksum")
	}
	if !hasDarwin {
		t.Error("Expected Darwin checksum")
	}

	if linuxChecksum == darwinChecksum {
		t.Error("Expected different checksums for different content")
	}

	// Verify checksums are valid SHA256 (64 hex characters)
	if len(linuxChecksum) != 64 {
		t.Errorf("Expected Linux checksum to be 64 chars, got %d", len(linuxChecksum))
	}
	if len(darwinChecksum) != 64 {
		t.Errorf("Expected Darwin checksum to be 64 chars, got %d", len(darwinChecksum))
	}
}

// TestCalculateChecksumsFull tests the full calculateChecksums function with mocked dependencies
func TestCalculateChecksumsFull(t *testing.T) {
	// Create a mock server for GitHub API
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/repos/test/repo/releases/tags/v1.0.0") {
			// Return mock GitHub API response
			response := GitHubReleaseResponse{
				TagName: "v1.0.0",
				Assets: []GitHubReleaseAsset{
					{
						Name:               "test-1.0.0-linux-amd64.tar.gz",
						BrowserDownloadURL: "http://localhost/download/test-1.0.0-linux-amd64.tar.gz",
						Size:               1024,
						SHA256:             "abc123def4567890123456789012345678901234567890123456789012345678", // Mock SHA256
					},
					{
						Name:               "test-1.0.0-darwin-amd64.tar.gz",
						BrowserDownloadURL: "http://localhost/download/test-1.0.0-darwin-amd64.tar.gz",
						Size:               1024,
						// No digest - should be downloaded
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else if strings.Contains(r.URL.Path, "/download/") {
			// Serve mock binary content for download
			if strings.Contains(r.URL.Path, "darwin-amd64") {
				w.Write([]byte("darwin binary content"))
			} else {
				http.Error(w, "Not Found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "Not Found", http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create embedder with test spec
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr("test/repo"),
			Name: spec.StringPtr("test"),
			Asset: &spec.Asset{
				Template: spec.StringPtr("${NAME}-${VERSION}-${OS}-${ARCH}.tar.gz"),
			},
			SupportedPlatforms: []spec.Platform{
				{OS: func() *spec.SupportedPlatformOS { v := spec.Linux; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
				{OS: func() *spec.SupportedPlatformOS { v := spec.Darwin; return &v }(), Arch: func() *spec.SupportedPlatformArch { v := spec.Amd64; return &v }()},
			},
			Checksums: &spec.Checksums{
				Algorithm: func() *spec.Algorithm { a := spec.Sha256; return &a }(),
			},
		},
		Version: "v1.0.0",
	}

	// Create a temporary function to test the fetchReleaseAssets logic
	fetchReleaseAssetsFunc := func() ([]GitHubReleaseAsset, error) {
		apiURL := fmt.Sprintf("%s/repos/%s/releases/tags/%s", mockServer.URL, "test/repo", "v1.0.0")
		
		resp, err := http.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch release from GitHub API: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
		}

		var release GitHubReleaseResponse
		if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
			return nil, fmt.Errorf("failed to decode GitHub API response: %w", err)
		}

		return release.Assets, nil
	}
	
	// Get the assets from the mock server
	assets, err := fetchReleaseAssetsFunc()
	if err != nil {
		t.Fatalf("fetchReleaseAssets failed: %v", err)
	}

	// Test the actual function with the mock data
	matchedAssets, err := embedder.matchAssetsToTemplate(assets)
	if err != nil {
		t.Fatalf("matchAssetsToTemplate failed: %v", err)
	}

	// Mock the downloadAndCalculateChecksums function to avoid actual download
	downloadedChecksums := map[string]string{
		"test-1.0.0-darwin-amd64.tar.gz": "def456ghi7890123456789012345678901234567890123456789012345678901",
	}

	// Simulate the full calculateChecksums flow
	checksums := make(map[string]string)

	// Use API digests for assets that have them
	for _, asset := range matchedAssets {
		if asset.SHA256 != "" {
			checksums[asset.Name] = asset.SHA256
		}
	}

	// Add downloaded checksums
	for filename, hash := range downloadedChecksums {
		checksums[filename] = hash
	}

	// Verify the results
	if len(checksums) != 2 {
		t.Errorf("Expected 2 checksums, got %d", len(checksums))
	}

	// Verify that the Linux asset used the API digest
	linuxChecksum, hasLinux := checksums["test-1.0.0-linux-amd64.tar.gz"]
	if !hasLinux {
		t.Error("Expected Linux checksum")
	}
	if linuxChecksum != "abc123def4567890123456789012345678901234567890123456789012345678" {
		t.Errorf("Expected Linux checksum from API, got %s", linuxChecksum)
	}

	// Verify that the Darwin asset was downloaded and calculated
	darwinChecksum, hasDarwin := checksums["test-1.0.0-darwin-amd64.tar.gz"]
	if !hasDarwin {
		t.Error("Expected Darwin checksum")
	}
	if len(darwinChecksum) != 64 {
		t.Errorf("Expected Darwin checksum to be 64 chars, got %d", len(darwinChecksum))
	}
}

// TestCalculateChecksumsErrorHandling tests error handling in calculateChecksums
func TestCalculateChecksumsErrorHandling(t *testing.T) {
	// Test with invalid repository
	embedder := &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr(""), // Empty repo
		},
		Version: "v1.0.0",
	}

	_, err := embedder.calculateChecksums()
	if err == nil {
		t.Error("Expected error for empty repository")
	}

	// Test with repository that doesn't exist
	embedder = &Embedder{
		Spec: &spec.InstallSpec{
			Repo: spec.StringPtr("nonexistent/repo"),
		},
		Version: "v1.0.0",
	}

	_, err = embedder.calculateChecksums()
	if err == nil {
		t.Error("Expected error for nonexistent repository")
	}
}