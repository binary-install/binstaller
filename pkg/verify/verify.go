package verify

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/pkg/errors"
)

// ComputeChecksum computes the checksum of a file using the specified algorithm
func ComputeChecksum(filePath string, algorithm spec.Algorithm) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	var h hash.Hash
	switch algorithm {
	case spec.Sha256:
		h = sha256.New()
	case spec.Sha512:
		h = sha512.New()
	case spec.Sha1:
		h = sha1.New()
	case spec.Md5:
		h = md5.New()
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", errors.Wrap(err, "failed to compute checksum")
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// VerifyChecksum verifies that a file matches the expected checksum
func VerifyChecksum(filePath, expectedHash string, algorithm spec.Algorithm) error {
	computedHash, err := ComputeChecksum(filePath, algorithm)
	if err != nil {
		return err
	}

	// Compare case-insensitively
	if !strings.EqualFold(computedHash, expectedHash) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, computedHash)
	}

	return nil
}

// VerifyWithEmbeddedChecksum verifies a file using embedded checksums from the config
func VerifyWithEmbeddedChecksum(cfg *spec.InstallSpec, filePath, version, filename string) error {
	if cfg.Checksums == nil || cfg.Checksums.EmbeddedChecksums == nil {
		// No embedded checksums, skip verification
		return nil
	}

	// Strip v prefix from version for lookup
	versionKey := strings.TrimPrefix(version, "v")

	checksums, ok := cfg.Checksums.EmbeddedChecksums[versionKey]
	if !ok {
		// No checksums for this version
		return nil
	}

	// Find checksum for the specific filename
	for _, checksum := range checksums {
		if spec.StringValue(checksum.Filename) == filename {
			expectedHash := spec.StringValue(checksum.Hash)
			if expectedHash == "" {
				return fmt.Errorf("empty checksum for %s", filename)
			}

			algorithm := spec.Sha256 // Default
			if cfg.Checksums.Algorithm != nil {
				algorithm = *cfg.Checksums.Algorithm
			}

			return VerifyChecksum(filePath, expectedHash, algorithm)
		}
	}

	// No checksum found for this filename
	return nil
}

// VerifyWithChecksumFile verifies a file using a checksum file
func VerifyWithChecksumFile(filePath, checksumFile string, algorithm spec.Algorithm) error {
	filename := filepath.Base(filePath)

	expectedHash, err := findChecksumInFile(checksumFile, filename)
	if err != nil {
		return err
	}

	return VerifyChecksum(filePath, expectedHash, algorithm)
}

// findChecksumInFile finds the checksum for a specific file in a checksum file
func findChecksumInFile(checksumFile, targetFilename string) (string, error) {
	file, err := os.Open(checksumFile)
	if err != nil {
		return "", errors.Wrap(err, "failed to open checksum file")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		checksum, filename, ok := parseChecksumLine(line)
		if ok && filename == targetFilename {
			return checksum, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to read checksum file")
	}

	return "", fmt.Errorf("checksum not found for %s", targetFilename)
}

// parseChecksumLine parses a line from a checksum file
// Supports formats like:
// - "abc123  filename.tar.gz" (two spaces)
// - "abc123 filename.tar.gz" (one space)
// - "abc123	filename.tar.gz" (tab)
func parseChecksumLine(line string) (checksum, filename string, ok bool) {
	line = strings.TrimSpace(line)

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}

	// Split by whitespace
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}
