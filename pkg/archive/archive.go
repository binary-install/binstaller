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
)

// Extractor handles extraction of various archive formats
type Extractor struct {
	stripComponents int
}

// NewExtractor creates a new archive extractor
func NewExtractor(stripComponents int) *Extractor {
	return &Extractor{
		stripComponents: stripComponents,
	}
}

// Extract extracts an archive to the specified destination directory
func (e *Extractor) Extract(archivePath, destDir string) error {
	ext := strings.ToLower(filepath.Ext(archivePath))

	switch ext {
	case ".gz":
		// Check if it's a tar.gz
		if strings.HasSuffix(strings.ToLower(archivePath), ".tar.gz") || strings.HasSuffix(strings.ToLower(archivePath), ".tgz") {
			return e.extractTarGz(archivePath, destDir)
		}
		// Plain gzip file (not a tar archive)
		return e.extractGz(archivePath, destDir)
	case ".tgz":
		return e.extractTarGz(archivePath, destDir)
	case ".tar":
		return e.extractTar(archivePath, destDir)
	case ".zip":
		return e.extractZip(archivePath, destDir)
	default:
		// Not an archive, likely a standalone binary
		// Copy the file to destDir
		return e.copyFile(archivePath, destDir)
	}
}

// extractTarGz extracts a tar.gz archive
func (e *Extractor) extractTarGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	return e.extractTarReader(gzReader, destDir)
}

// extractTar extracts a tar archive
func (e *Extractor) extractTar(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	return e.extractTarReader(file, destDir)
}

// extractTarReader extracts from a tar reader
func (e *Extractor) extractTarReader(r io.Reader, destDir string) error {
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Apply strip components
		path := e.stripPath(header.Name)
		if path == "" {
			continue
		}

		// Validate and secure the target path
		targetPath, err := securePath(path, destDir)
		if err != nil {
			return fmt.Errorf("tar entry %q: %w", header.Name, err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			if err := e.extractTarFile(tarReader, targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Validate the symlink before creating it
			if err := validateSymlink(targetPath, header.Linkname, destDir); err != nil {
				return fmt.Errorf("tar entry %q: %w", header.Name, err)
			}

			// Create parent directory if needed
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory for symlink: %w", err)
			}

			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				return fmt.Errorf("failed to create symlink: %w", err)
			}
		}
	}

	return nil
}

// extractTarFile extracts a single file from tar
func (e *Extractor) extractTarFile(tarReader *tar.Reader, destPath string, mode os.FileMode) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, tarReader); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractZip extracts a zip archive
func (e *Extractor) extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		// Apply strip components
		path := e.stripPath(file.Name)
		if path == "" {
			continue
		}

		// Validate and secure the target path
		targetPath, err := securePath(path, destDir)
		if err != nil {
			return fmt.Errorf("zip entry %q: %w", file.Name, err)
		}

		// Check file mode to detect symlinks
		mode := file.FileInfo().Mode()

		if mode.IsDir() {
			if err := os.MkdirAll(targetPath, file.Mode()); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			continue
		}

		if mode&os.ModeSymlink != 0 {
			// Handle symlink
			if err := e.extractZipSymlink(file, targetPath, destDir); err != nil {
				return err
			}
			continue
		}

		if err := e.extractZipFile(file, targetPath); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from zip
func (e *Extractor) extractZipFile(file *zip.File, destPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractGz extracts a plain gzip file (not tar.gz)
func (e *Extractor) extractGz(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Extract to a file with .gz extension removed
	baseName := filepath.Base(archivePath)
	baseName = strings.TrimSuffix(baseName, ".gz")

	destPath := filepath.Join(destDir, baseName)
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, gzReader); err != nil {
		return fmt.Errorf("failed to decompress file: %w", err)
	}

	return nil
}

// copyFile copies a file to the destination directory
func (e *Extractor) copyFile(srcPath, destDir string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destPath := filepath.Join(destDir, filepath.Base(srcPath))
	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// stripPath applies strip components to a path
func (e *Extractor) stripPath(path string) string {
	if e.stripComponents <= 0 {
		return path
	}

	// Clean the path and split into components
	path = filepath.Clean(path)
	parts := strings.Split(path, string(filepath.Separator))

	// Remove leading empty parts (from absolute paths)
	for len(parts) > 0 && parts[0] == "" {
		parts = parts[1:]
	}

	// Apply strip components
	if e.stripComponents >= len(parts) {
		return ""
	}

	return filepath.Join(parts[e.stripComponents:]...)
}

// securePath validates that a path is within the destination directory
// and returns the cleaned, absolute path. It prevents directory traversal attacks.
func securePath(path, destDir string) (string, error) {
	// Clean both paths
	cleanPath := filepath.Clean(path)
	cleanDestDir := filepath.Clean(destDir)

	// Get absolute paths
	absDestDir, err := filepath.Abs(cleanDestDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for destination: %w", err)
	}

	// If the path is not absolute, join it with destDir
	var targetPath string
	if filepath.IsAbs(cleanPath) {
		targetPath = cleanPath
	} else {
		targetPath = filepath.Join(absDestDir, cleanPath)
	}

	// Clean the target path again
	targetPath = filepath.Clean(targetPath)

	// Ensure the path is within destDir
	relPath, err := filepath.Rel(absDestDir, targetPath)
	if err != nil || strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return "", fmt.Errorf("path %q would be outside destination directory", path)
	}

	return targetPath, nil
}

// validateSymlink ensures that a symlink and its target are safe
func validateSymlink(symlinkPath, linkTarget, destDir string) error {
	// First validate the symlink path itself
	_, err := securePath(symlinkPath, destDir)
	if err != nil {
		return fmt.Errorf("invalid symlink path: %w", err)
	}

	// Absolute symlinks are not allowed
	if filepath.IsAbs(linkTarget) {
		return fmt.Errorf("absolute symlink target %q not allowed", linkTarget)
	}

	// Get the directory where the symlink will be created
	symlinkDir := filepath.Dir(symlinkPath)

	// Resolve the target path relative to the symlink directory
	targetPath := filepath.Join(symlinkDir, linkTarget)
	targetPath = filepath.Clean(targetPath)

	// Ensure the resolved target is within destDir
	_, err = securePath(targetPath, destDir)
	if err != nil {
		return fmt.Errorf("symlink target %q would point outside destination directory: %w", linkTarget, err)
	}

	return nil
}

// extractZipSymlink extracts a symlink from a zip file
func (e *Extractor) extractZipSymlink(file *zip.File, targetPath, destDir string) error {
	// Read the link target from the file content
	srcFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open symlink in zip: %w", err)
	}
	defer srcFile.Close()

	linkTargetBytes, err := io.ReadAll(srcFile)
	if err != nil {
		return fmt.Errorf("failed to read symlink target: %w", err)
	}
	linkTarget := string(linkTargetBytes)

	// Validate the symlink before creating it
	if err := validateSymlink(targetPath, linkTarget, destDir); err != nil {
		return fmt.Errorf("zip entry %q: %w", file.Name, err)
	}

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for symlink: %w", err)
	}

	if err := os.Symlink(linkTarget, targetPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// ListFiles returns all regular files in a directory
func ListFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}
