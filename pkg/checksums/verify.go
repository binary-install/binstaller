package checksums

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/httpclient"
	"github.com/binary-install/binstaller/pkg/spec"
)

// Verifier handles checksum verification for downloaded assets
type Verifier struct {
	Spec    *spec.InstallSpec
	Version string
}

// NewVerifier creates a new checksum verifier
func NewVerifier(spec *spec.InstallSpec, version string) *Verifier {
	return &Verifier{
		Spec:    spec,
		Version: version,
	}
}

// GetChecksum retrieves the checksum for a given filename
// It first checks embedded checksums, then tries to download checksum file
func (v *Verifier) GetChecksum(ctx context.Context, filename string) (string, error) {
	hash, err := v.getChecksumWithAssetFilename(ctx, filename, filename)
	if err != nil {
		return "", err
	}
	if hash == "" {
		return "", fmt.Errorf("no checksum found for %s", filename)
	}
	return hash, nil
}

// getChecksumWithAssetFilename retrieves the checksum for a given filename
// It accepts both the filename to look up and the asset filename for template interpolation
func (v *Verifier) getChecksumWithAssetFilename(ctx context.Context, filename, assetFilename string) (string, error) {
	if v.Spec.Checksums == nil {
		// Return a special error that VerifyFile can recognize
		return "", nil
	}

	// First, check embedded checksums
	if v.Spec.Checksums.EmbeddedChecksums != nil {
		if checksums, ok := v.Spec.Checksums.EmbeddedChecksums[v.Version]; ok {
			for _, ec := range checksums {
				if spec.StringValue(ec.Filename) == filename {
					return spec.StringValue(ec.Hash), nil
				}
			}
		}
	}

	// If not found in embedded checksums, try to download checksum file
	if spec.StringValue(v.Spec.Checksums.Template) != "" {
		checksumMap, err := v.downloadChecksumFileWithAssetFilename(ctx, assetFilename)
		if err != nil {
			return "", fmt.Errorf("failed to download checksum file: %w", err)
		}

		if hash, ok := checksumMap[filename]; ok {
			return hash, nil
		}

		// Checksum file exists but doesn't contain the file
		return "", fmt.Errorf("no checksum found for %s", filename)
	}

	// No checksum configuration at all - return empty without error
	return "", nil
}

// VerifyFile verifies a file against its expected checksum
func (v *Verifier) VerifyFile(ctx context.Context, filepath, filename string) error {
	expectedHash, err := v.getChecksumWithAssetFilename(ctx, filename, filename)
	if err != nil {
		// Skip verification with warning when checksums are not found
		// This matches the behavior of generated shell scripts
		log.Warnf("No checksum found for %s, skipping verification: %v", filename, err)
		return nil
	}

	// If no checksum was found (nil error but empty hash), skip verification
	if expectedHash == "" {
		log.Warnf("No checksum found for %s, skipping verification", filename)
		return nil
	}

	algorithm := "sha256" // default
	if v.Spec.Checksums != nil && v.Spec.Checksums.Algorithm != nil {
		algorithm = string(*v.Spec.Checksums.Algorithm)
	}

	actualHash, err := ComputeHash(filepath, algorithm)
	if err != nil {
		return fmt.Errorf("failed to compute hash: %w", err)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filename, expectedHash, actualHash)
	}

	log.Infof("Checksum verified for %s", filename)
	return nil
}

// downloadChecksumFileWithAssetFilename downloads and parses the checksum file with asset filename support
func (v *Verifier) downloadChecksumFileWithAssetFilename(ctx context.Context, assetFilename string) (map[string]string, error) {
	// Create embedder to reuse checksum template interpolation
	embedder := &Embedder{
		Spec:    v.Spec,
		Version: v.Version,
	}

	checksumFilename := embedder.createChecksumFilenameWithAsset(assetFilename)
	if checksumFilename == "" {
		return nil, fmt.Errorf("unable to generate checksum filename")
	}

	checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
		spec.StringValue(v.Spec.Repo), v.Version, checksumFilename)

	log.Infof("Downloading checksums from %s", checksumURL)

	// Create request with GitHub auth
	req, err := httpclient.NewRequestWithGitHubAuth("GET", checksumURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req = req.WithContext(ctx)

	client := httpclient.NewGitHubClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download checksum file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksum file, status code: %d", resp.StatusCode)
	}

	// Parse checksum file content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read checksum file: %w", err)
	}

	return parseChecksumContent(string(content)), nil
}

// parseChecksumContent parses checksum file content into a map
func parseChecksumContent(content string) map[string]string {
	checksums := make(map[string]string)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse the line as a checksum entry
		// Format: <hash> [*]<filename>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hash := parts[0]
		filename := parts[1]

		// If the filename starts with *, remove it
		filename = strings.TrimPrefix(filename, "*")

		checksums[filename] = hash
	}

	return checksums
}
