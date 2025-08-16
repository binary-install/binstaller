package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Format
	}{
		{
			name:     "tar.gz file",
			filename: "binary-linux-amd64.tar.gz",
			want:     FormatTarGz,
		},
		{
			name:     "tgz file",
			filename: "binary-linux-amd64.tgz",
			want:     FormatTarGz,
		},
		{
			name:     "zip file",
			filename: "binary-windows-amd64.zip",
			want:     FormatZip,
		},
		{
			name:     "plain tar file",
			filename: "binary-linux-amd64.tar",
			want:     FormatTar,
		},
		{
			name:     "exe file",
			filename: "binary-windows-amd64.exe",
			want:     FormatRaw,
		},
		{
			name:     "no extension",
			filename: "jq-linux64",
			want:     FormatRaw,
		},
		{
			name:     "unknown extension",
			filename: "binary.unknown",
			want:     FormatRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFormat(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name       string
		setupFile  func(t *testing.T) string
		targetFile string
		wantErr    bool
		validate   func(t *testing.T, destDir string)
	}{
		{
			name: "extract from tar.gz",
			setupFile: func(t *testing.T) string {
				return createTarGzFile(t, map[string][]byte{
					"bin/tool":  []byte("binary content"),
					"README.md": []byte("readme content"),
					"LICENSE":   []byte("license content"),
				})
			},
			targetFile: "bin/tool",
			validate: func(t *testing.T, destDir string) {
				assert.FileExists(t, filepath.Join(destDir, "bin/tool"))
				assert.FileExists(t, filepath.Join(destDir, "README.md"))
				assert.FileExists(t, filepath.Join(destDir, "LICENSE"))

				content, err := os.ReadFile(filepath.Join(destDir, "bin/tool"))
				require.NoError(t, err)
				assert.Equal(t, "binary content", string(content))
			},
		},
		{
			name: "extract from zip",
			setupFile: func(t *testing.T) string {
				return createZipFile(t, map[string][]byte{
					"tool.exe":   []byte("exe content"),
					"README.txt": []byte("readme"),
				})
			},
			targetFile: "tool.exe",
			validate: func(t *testing.T, destDir string) {
				assert.FileExists(t, filepath.Join(destDir, "tool.exe"))
				assert.FileExists(t, filepath.Join(destDir, "README.txt"))

				content, err := os.ReadFile(filepath.Join(destDir, "tool.exe"))
				require.NoError(t, err)
				assert.Equal(t, "exe content", string(content))
			},
		},
		{
			name: "raw file (no extraction)",
			setupFile: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "binary")
				require.NoError(t, os.WriteFile(tmpFile, []byte("raw binary"), 0644))
				return tmpFile
			},
			targetFile: "",
			validate: func(t *testing.T, destDir string) {
				// For raw files, nothing should be extracted
				entries, err := os.ReadDir(destDir)
				require.NoError(t, err)
				assert.Empty(t, entries)
			},
		},
		{
			name: "extract with strip components",
			setupFile: func(t *testing.T) string {
				return createTarGzFile(t, map[string][]byte{
					"prefix/bin/tool":  []byte("binary content"),
					"prefix/README.md": []byte("readme content"),
					"other/file.txt":   []byte("other file"),
				})
			},
			targetFile: "bin/tool",
			validate: func(t *testing.T, destDir string) {
				// With strip components = 1, "prefix/" should be removed
				assert.FileExists(t, filepath.Join(destDir, "bin/tool"))
				assert.FileExists(t, filepath.Join(destDir, "README.md"))
				assert.NoFileExists(t, filepath.Join(destDir, "prefix/bin/tool"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath := tt.setupFile(t)
			destDir := t.TempDir()

			stripComponents := 0
			if tt.name == "extract with strip components" {
				stripComponents = 1
			}

			err := Extract(archivePath, destDir, stripComponents)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.validate != nil {
				tt.validate(t, destDir)
			}
		})
	}
}

func TestFindBinary(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(t *testing.T, dir string)
		targetPath string
		isRaw      bool
		want       string
		wantErr    bool
	}{
		{
			name: "find binary in subdirectory",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.MkdirAll(filepath.Join(dir, "bin"), 0755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "bin", "tool"), []byte("binary"), 0755))
			},
			targetPath: "bin/tool",
			want:       "bin/tool",
		},
		{
			name: "find binary at root",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), []byte("binary"), 0755))
			},
			targetPath: "tool",
			want:       "tool",
		},
		{
			name: "raw binary uses asset filename",
			setupFiles: func(t *testing.T, dir string) {
				// Raw binary scenario - no files in destDir
			},
			targetPath: "${ASSET_FILENAME}",
			isRaw:      true,
			want:       "original.tar.gz", // Will be replaced by asset filename
		},
		{
			name: "binary not found",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "other"), []byte("other"), 0755))
			},
			targetPath: "bin/tool",
			wantErr:    true,
		},
		{
			name: "find with different case on case-insensitive systems",
			setupFiles: func(t *testing.T, dir string) {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "TOOL.EXE"), []byte("binary"), 0755))
			},
			targetPath: "tool.exe",
			want:       "TOOL.EXE",
			// Note: This behavior depends on the filesystem
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.setupFiles != nil {
				tt.setupFiles(t, dir)
			}

			assetFilename := "original.tar.gz"
			got, err := FindBinary(dir, tt.targetPath, assetFilename, tt.isRaw)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if tt.isRaw && tt.targetPath == "${ASSET_FILENAME}" {
				assert.Equal(t, assetFilename, got)
			} else {
				assert.Equal(t, filepath.Join(dir, tt.want), got)
			}
		})
	}
}

// Helper functions to create test archives

func createTarGzFile(t *testing.T, files map[string][]byte) string {
	tmpFile := filepath.Join(t.TempDir(), "archive.tar.gz")

	file, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for path, content := range files {
		header := &tar.Header{
			Name: path,
			Mode: 0644,
			Size: int64(len(content)),
		}

		err := tarWriter.WriteHeader(header)
		require.NoError(t, err)

		_, err = tarWriter.Write(content)
		require.NoError(t, err)
	}

	return tmpFile
}

func createZipFile(t *testing.T, files map[string][]byte) string {
	tmpFile := filepath.Join(t.TempDir(), "archive.zip")

	file, err := os.Create(tmpFile)
	require.NoError(t, err)
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for path, content := range files {
		writer, err := zipWriter.Create(path)
		require.NoError(t, err)

		_, err = writer.Write(content)
		require.NoError(t, err)
	}

	return tmpFile
}

func TestStripComponents(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		stripComponents int
		want            string
		wantSkip        bool
	}{
		{
			name:            "no strip",
			path:            "bin/tool",
			stripComponents: 0,
			want:            "bin/tool",
		},
		{
			name:            "strip one component",
			path:            "prefix/bin/tool",
			stripComponents: 1,
			want:            "bin/tool",
		},
		{
			name:            "strip two components",
			path:            "a/b/c/tool",
			stripComponents: 2,
			want:            "c/tool",
		},
		{
			name:            "strip too many components",
			path:            "bin/tool",
			stripComponents: 3,
			wantSkip:        true,
		},
		{
			name:            "strip exact number of components",
			path:            "a/b/tool",
			stripComponents: 2,
			want:            "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, skip := stripComponents(tt.path, tt.stripComponents)
			assert.Equal(t, tt.wantSkip, skip)
			if !tt.wantSkip {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
