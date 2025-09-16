package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/ulikunitz/xz"
)

func TestExtractTarGz(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test tar.gz file
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	if err := createTestTarGz(tarGzPath); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(tarGzPath, destDir); err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Verify extracted files
	expectedFiles := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"file3.txt",
	}

	for _, expectedFile := range expectedFiles {
		path := filepath.Join(destDir, expectedFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}
}

func TestExtractTarGzWithStripComponents(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test tar.gz file with nested structure
	tarGzPath := filepath.Join(tmpDir, "test.tar.gz")
	if err := createTestTarGzNested(tarGzPath); err != nil {
		t.Fatalf("Failed to create test tar.gz: %v", err)
	}

	// Create extractor with strip_components=1
	extractor := NewExtractor(1)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(tarGzPath, destDir); err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Verify that the root directory was stripped
	// Instead of root/dir1/file1.txt, we should have dir1/file1.txt
	expectedFiles := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"file3.txt",
	}

	for _, expectedFile := range expectedFiles {
		path := filepath.Join(destDir, expectedFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}

	// Verify that root directory was stripped
	rootPath := filepath.Join(destDir, "root")
	if _, err := os.Stat(rootPath); !os.IsNotExist(err) {
		t.Error("Root directory should have been stripped")
	}
}

func TestExtractZip(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	if err := createTestZip(zipPath); err != nil {
		t.Fatalf("Failed to create test zip: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(zipPath, destDir); err != nil {
		t.Fatalf("Failed to extract zip: %v", err)
	}

	// Verify extracted files
	expectedFiles := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"file3.txt",
	}

	for _, expectedFile := range expectedFiles {
		path := filepath.Join(destDir, expectedFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}
}

func TestExtractPlainGz(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test plain gz file (not tar.gz)
	gzPath := filepath.Join(tmpDir, "binary.gz")
	if err := createTestPlainGz(gzPath, "binary content"); err != nil {
		t.Fatalf("Failed to create test gz: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(gzPath, destDir); err != nil {
		t.Fatalf("Failed to extract gz: %v", err)
	}

	// Verify extracted file
	extractedPath := filepath.Join(destDir, "binary")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != "binary content" {
		t.Errorf("Expected content 'binary content', got '%s'", string(content))
	}
}

func TestExtractNonArchive(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test binary file (not an archive)
	binaryPath := filepath.Join(tmpDir, "binary")
	if err := os.WriteFile(binaryPath, []byte("binary content"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(binaryPath, destDir); err != nil {
		t.Fatalf("Failed to copy binary: %v", err)
	}

	// Verify copied file
	copiedPath := filepath.Join(destDir, "binary")
	content, err := os.ReadFile(copiedPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(content) != "binary content" {
		t.Errorf("Expected content 'binary content', got '%s'", string(content))
	}
}

func TestStripPath(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		stripComponents int
		expected        string
	}{
		{
			name:            "no strip",
			path:            "dir1/dir2/file.txt",
			stripComponents: 0,
			expected:        "dir1/dir2/file.txt",
		},
		{
			name:            "strip 1",
			path:            "dir1/dir2/file.txt",
			stripComponents: 1,
			expected:        "dir2/file.txt",
		},
		{
			name:            "strip 2",
			path:            "dir1/dir2/file.txt",
			stripComponents: 2,
			expected:        "file.txt",
		},
		{
			name:            "strip all",
			path:            "dir1/dir2/file.txt",
			stripComponents: 3,
			expected:        "",
		},
		{
			name:            "strip more than available",
			path:            "dir1/file.txt",
			stripComponents: 5,
			expected:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := &Extractor{stripComponents: tt.stripComponents}
			result := extractor.stripPath(tt.path)
			if result != tt.expected {
				t.Errorf("stripPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// Helper functions to create test archives

func createTestTarGz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add some test files
	files := map[string]string{
		"dir1/file1.txt": "content1",
		"dir1/file2.txt": "content2",
		"file3.txt":      "content3",
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func createTestTarGzNested(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add some test files with a root directory
	files := map[string]string{
		"root/dir1/file1.txt": "content1",
		"root/dir1/file2.txt": "content2",
		"root/file3.txt":      "content3",
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func createTestZip(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add some test files
	files := map[string]string{
		"dir1/file1.txt": "content1",
		"dir1/file2.txt": "content2",
		"file3.txt":      "content3",
	}

	for name, content := range files {
		w, err := zipWriter.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func createTestPlainGz(path string, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	_, err = gzWriter.Write([]byte(content))
	return err
}

func TestExtractTarXz(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test tar.xz file
	tarXzPath := filepath.Join(tmpDir, "test.tar.xz")
	if err := createTestTarXz(tarXzPath); err != nil {
		t.Fatalf("Failed to create test tar.xz: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(tarXzPath, destDir); err != nil {
		t.Fatalf("Failed to extract tar.xz: %v", err)
	}

	// Verify extracted files
	expectedFiles := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"file3.txt",
	}

	for _, expectedFile := range expectedFiles {
		path := filepath.Join(destDir, expectedFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}
}

func TestExtractTarXzWithStripComponents(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test tar.xz file with nested structure
	tarXzPath := filepath.Join(tmpDir, "test.tar.xz")
	if err := createTestTarXzNested(tarXzPath); err != nil {
		t.Fatalf("Failed to create test tar.xz: %v", err)
	}

	// Create extractor with strip_components=1
	extractor := NewExtractor(1)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(tarXzPath, destDir); err != nil {
		t.Fatalf("Failed to extract tar.xz: %v", err)
	}

	// Verify that the root directory was stripped
	// Instead of root/dir1/file1.txt, we should have dir1/file1.txt
	expectedFiles := []string{
		"dir1/file1.txt",
		"dir1/file2.txt",
		"file3.txt",
	}

	for _, expectedFile := range expectedFiles {
		path := filepath.Join(destDir, expectedFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", expectedFile)
		}
	}

	// Verify that root directory was stripped
	rootPath := filepath.Join(destDir, "root")
	if _, err := os.Stat(rootPath); !os.IsNotExist(err) {
		t.Error("Root directory should have been stripped")
	}
}

func TestExtractPlainXz(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a test plain xz file (not tar.xz)
	xzPath := filepath.Join(tmpDir, "binary.xz")
	if err := createTestPlainXz(xzPath, "binary content"); err != nil {
		t.Fatalf("Failed to create test xz: %v", err)
	}

	// Create extractor and extract
	extractor := NewExtractor(0)
	destDir := filepath.Join(tmpDir, "extracted")
	if err := extractor.Extract(xzPath, destDir); err != nil {
		t.Fatalf("Failed to extract xz: %v", err)
	}

	// Verify extracted file
	extractedPath := filepath.Join(destDir, "binary")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != "binary content" {
		t.Errorf("Expected content 'binary content', got '%s'", string(content))
	}
}

func createTestTarXz(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	xzWriter, err := xz.NewWriter(file)
	if err != nil {
		return err
	}
	defer xzWriter.Close()

	tarWriter := tar.NewWriter(xzWriter)
	defer tarWriter.Close()

	// Add some test files
	files := map[string]string{
		"dir1/file1.txt": "content1",
		"dir1/file2.txt": "content2",
		"file3.txt":      "content3",
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func createTestTarXzNested(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	xzWriter, err := xz.NewWriter(file)
	if err != nil {
		return err
	}
	defer xzWriter.Close()

	tarWriter := tar.NewWriter(xzWriter)
	defer tarWriter.Close()

	// Add some test files with a root directory
	files := map[string]string{
		"root/dir1/file1.txt": "content1",
		"root/dir1/file2.txt": "content2",
		"root/file3.txt":      "content3",
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

func createTestPlainXz(path string, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	xzWriter, err := xz.NewWriter(file)
	if err != nil {
		return err
	}
	defer xzWriter.Close()

	_, err = xzWriter.Write([]byte(content))
	return err
}
