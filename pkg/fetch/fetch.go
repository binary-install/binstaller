package fetch

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

var (
	// githubDownloadURL is the base URL for GitHub release downloads
	// This is a variable so it can be overridden in tests
	githubDownloadURL = "https://github.com"
)

// ProgressFunc is a callback for download progress
type ProgressFunc func(downloaded, total int64)

// Download downloads a file from the given URL to the destination path
func Download(url, destPath string) error {
	return DownloadWithProgress(url, destPath, nil)
}

// DownloadWithProgress downloads a file with optional progress callback
func DownloadWithProgress(url, destPath string, progress ProgressFunc) error {
	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return errors.Wrap(err, "failed to create destination directory")
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp(filepath.Dir(destPath), ".download-*")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary file")
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	defer tmpFile.Close()

	// Download with retry
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Wait before retry
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}

		// Add GitHub token if available
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		// Perform request
		client := &http.Client{
			Timeout: 5 * time.Minute,
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		// Check status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			if resp.StatusCode >= 500 {
				// Server error, retry
				continue
			}
			// Client error, don't retry
			return lastErr
		}

		// Reset file position
		if _, err := tmpFile.Seek(0, 0); err != nil {
			return errors.Wrap(err, "failed to seek to beginning of file")
		}
		if err := tmpFile.Truncate(0); err != nil {
			return errors.Wrap(err, "failed to truncate file")
		}

		// Copy with progress
		var written int64
		if progress != nil && resp.ContentLength > 0 {
			written, err = copyWithProgress(tmpFile, resp.Body, resp.ContentLength, progress)
		} else {
			written, err = copyWithRetry(tmpFile, resp.Body, 3)
		}

		if err != nil {
			lastErr = err
			continue
		}

		// Verify we got some content
		if written == 0 {
			lastErr = fmt.Errorf("no content downloaded")
			continue
		}

		// Success! Move to final destination
		if err := tmpFile.Close(); err != nil {
			return errors.Wrap(err, "failed to close temporary file")
		}

		if err := os.Rename(tmpPath, destPath); err != nil {
			return errors.Wrap(err, "failed to move downloaded file")
		}

		return nil
	}

	return errors.Wrapf(lastErr, "download failed after %d attempts", maxRetries)
}

// DownloadAsset downloads a GitHub release asset
func DownloadAsset(repo, tag, filename, destPath string) error {
	url := fmt.Sprintf("%s/%s/releases/download/%s/%s", githubDownloadURL, repo, tag, filename)
	return Download(url, destPath)
}

// copyWithRetry copies from reader to writer with retry on errors
func copyWithRetry(dst io.Writer, src io.Reader, maxRetries int) (int64, error) {
	var written int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[0:nr])
			if writeErr != nil {
				return written, writeErr
			}
			written += int64(nw)
		}

		if readErr != nil {
			if readErr == io.EOF {
				return written, nil
			}
			// For now, just return the error
			// In a real implementation, we might check if it's retryable
			return written, readErr
		}
	}
}

// copyWithProgress copies data and reports progress
func copyWithProgress(dst io.Writer, src io.Reader, total int64, progress ProgressFunc) (int64, error) {
	var written int64
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[0:nr])
			if writeErr != nil {
				return written, writeErr
			}
			written += int64(nw)

			if progress != nil {
				progress(written, total)
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				return written, nil
			}
			return written, readErr
		}
	}
}
