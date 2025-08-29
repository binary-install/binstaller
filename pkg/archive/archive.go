package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Format represents the archive format
type Format string

const (
	FormatTarGz Format = "tar.gz"
	FormatTar   Format = "tar"
	FormatZip   Format = "zip"
	FormatRaw   Format = "raw"
)

// DetectFormat detects the archive format based on the filename
func DetectFormat(filename string) Format {
	lower := strings.ToLower(filename)

	if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
		return FormatTarGz
	}
	if strings.HasSuffix(lower, ".tar") {
		return FormatTar
	}
	if strings.HasSuffix(lower, ".zip") {
		return FormatZip
	}

	// Default to raw for unknown formats or no extension
	return FormatRaw
}

// Extract extracts an archive to the destination directory
func Extract(archivePath, destDir string, stripComponentsCount int) error {
	format := DetectFormat(archivePath)

	switch format {
	case FormatTarGz:
		return extractTarGz(archivePath, destDir, stripComponentsCount)
	case FormatTar:
		return extractTar(archivePath, destDir, stripComponentsCount)
	case FormatZip:
		return extractZip(archivePath, destDir, stripComponentsCount)
	case FormatRaw:
		// Raw files don't need extraction
		return nil
	default:
		return fmt.Errorf("unsupported archive format: %s", format)
	}
}

// FindBinary finds the target binary in the extracted directory
func FindBinary(destDir, targetPath, assetFilename string, isRaw bool) (string, error) {
	// Handle special case for raw binaries
	if isRaw && targetPath == "${ASSET_FILENAME}" {
		return assetFilename, nil
	}

	// Look for the binary at the target path
	fullPath := filepath.Join(destDir, targetPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}

	// Try case-insensitive search
	dir := filepath.Dir(fullPath)
	name := filepath.Base(fullPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("binary not found at %s", targetPath)
	}

	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("binary not found at %s", targetPath)
}

// extractTarGz extracts a tar.gz archive
func extractTarGz(archivePath, destDir string, stripComponentsCount int) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzReader.Close()

	return extractTarReader(gzReader, destDir, stripComponentsCount)
}

// extractTar extracts a plain tar archive
func extractTar(archivePath, destDir string, stripComponentsCount int) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open archive")
	}
	defer file.Close()

	return extractTarReader(file, destDir, stripComponentsCount)
}

// extractTarReader extracts from a tar reader
func extractTarReader(r io.Reader, destDir string, stripComponentsCount int) error {
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "failed to read tar header")
		}

		// Apply strip components
		path, skip := stripComponents(header.Name, stripComponentsCount)
		if skip {
			continue
		}

		target := filepath.Join(destDir, path)

		// Ensure the target path is within destDir
		if !strings.HasPrefix(target, destDir) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "failed to create directory")
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return errors.Wrap(err, "failed to create parent directory")
			}

			file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return errors.Wrap(err, "failed to create file")
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return errors.Wrap(err, "failed to extract file")
			}

			file.Close()
		}
	}

	return nil
}

// extractZip extracts a zip archive
func extractZip(archivePath, destDir string, stripComponentsCount int) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to open zip archive")
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Apply strip components
		path, skip := stripComponents(file.Name, stripComponentsCount)
		if skip {
			continue
		}

		target := filepath.Join(destDir, path)

		// Ensure the target path is within destDir
		if !strings.HasPrefix(target, destDir) {
			return fmt.Errorf("invalid path in archive: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, file.Mode()); err != nil {
				return errors.Wrap(err, "failed to create directory")
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return errors.Wrap(err, "failed to create parent directory")
		}

		fileReader, err := file.Open()
		if err != nil {
			return errors.Wrap(err, "failed to open file in archive")
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, file.Mode())
		if err != nil {
			return errors.Wrap(err, "failed to create file")
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return errors.Wrap(err, "failed to extract file")
		}
	}

	return nil
}

// stripComponents removes the specified number of leading path components
func stripComponents(path string, count int) (string, bool) {
	if count == 0 {
		return path, false
	}

	parts := strings.Split(path, "/")
	if len(parts) <= count {
		// Skip this entry entirely
		return "", true
	}

	return strings.Join(parts[count:], "/"), false
}
