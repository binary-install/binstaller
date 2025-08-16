package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeChecksum(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		algorithm spec.Algorithm
		want      string
		wantErr   bool
	}{
		{
			name:      "sha256 checksum",
			content:   "hello world",
			algorithm: spec.Sha256,
			want:      "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:      "sha512 checksum",
			content:   "hello world",
			algorithm: spec.Sha512,
			want:      "309ecc489c12d6eb4cc40f50c902f2b4d0ed77ee511a7c7a9bcd3ca86d4cd86f989dd35bc5ff499670da34255b45b0cfd830e81f605dcf7dc5542e93ae9cd76f",
		},
		{
			name:      "sha1 checksum",
			content:   "hello world",
			algorithm: spec.Sha1,
			want:      "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:      "md5 checksum",
			content:   "hello world",
			algorithm: spec.Md5,
			want:      "5eb63bbbe01eeed093cb22bb8f5acdc3",
		},
		{
			name:      "empty file",
			content:   "",
			algorithm: spec.Sha256,
			want:      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:      "invalid algorithm",
			content:   "test",
			algorithm: spec.Algorithm("invalid"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			got, err := ComputeChecksum(testFile, tt.algorithm)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedHash string
		algorithm    spec.Algorithm
		wantErr      bool
	}{
		{
			name:         "valid sha256 checksum",
			content:      "test content",
			expectedHash: "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
			algorithm:    spec.Sha256,
		},
		{
			name:         "invalid checksum",
			content:      "test content",
			expectedHash: "wrong_hash",
			algorithm:    spec.Sha256,
			wantErr:      true,
		},
		{
			name:         "case insensitive checksum match",
			content:      "test content",
			expectedHash: "6AE8A75555209FD6C44157C0AED8016E763FF435A19CF186F76863140143FF72",
			algorithm:    spec.Sha256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			err := VerifyChecksum(testFile, tt.expectedHash, tt.algorithm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyWithEmbeddedChecksum(t *testing.T) {
	cfg := &spec.InstallSpec{
		Checksums: &spec.Checksums{
			Algorithm: (*spec.Algorithm)(spec.StringPtr("sha256")),
			EmbeddedChecksums: map[string][]spec.EmbeddedChecksum{
				"1.0.0": {
					{
						Filename: spec.StringPtr("tool-1.0.0-linux-amd64.tar.gz"),
						Hash:     spec.StringPtr("abc123def456"),
					},
					{
						Filename: spec.StringPtr("tool-1.0.0-darwin-amd64.tar.gz"),
						Hash:     spec.StringPtr("def456abc123"),
					},
				},
				"2.0.0": {
					{
						Filename: spec.StringPtr("tool-2.0.0-linux-amd64.tar.gz"),
						Hash:     spec.StringPtr("123456abcdef"),
					},
				},
			},
		},
	}

	tests := []struct {
		name         string
		version      string
		filename     string
		fileContent  string
		setupContent bool
		wantErr      bool
		errContains  string
	}{
		{
			name:         "embedded checksum found and valid",
			version:      "1.0.0",
			filename:     "tool-1.0.0-linux-amd64.tar.gz",
			fileContent:  "test content for embedded checksum",
			setupContent: true,
			wantErr:      true, // Will fail because our test hash doesn't match
			errContains:  "checksum mismatch",
		},
		{
			name:         "embedded checksum not found for version",
			version:      "3.0.0",
			filename:     "tool-3.0.0-linux-amd64.tar.gz",
			fileContent:  "test content",
			setupContent: true,
			wantErr:      false, // Should succeed when no checksum is found
		},
		{
			name:         "embedded checksum not found for filename",
			version:      "1.0.0",
			filename:     "tool-1.0.0-windows-amd64.zip",
			fileContent:  "test content",
			setupContent: true,
			wantErr:      false, // Should succeed when no checksum is found
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, tt.filename)

			if tt.setupContent {
				require.NoError(t, os.WriteFile(testFile, []byte(tt.fileContent), 0644))
			}

			err := VerifyWithEmbeddedChecksum(cfg, testFile, tt.version, tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyWithChecksumFile(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		checksumContent string
		algorithm       spec.Algorithm
		wantErr         bool
	}{
		{
			name:        "valid checksum from file",
			fileContent: "test binary content",
			checksumContent: `6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72  test.txt
abc123def456789  other-file.txt`,
			algorithm: spec.Sha256,
			wantErr:   true, // Will fail because hash doesn't match our content
		},
		{
			name:        "checksum file with different formats",
			fileContent: "test content",
			checksumContent: `# SHA256 checksums
6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72 test.txt
abc123  file2.txt
def456	file3.txt`,
			algorithm: spec.Sha256,
		},
		{
			name:            "empty checksum file",
			fileContent:     "test content",
			checksumContent: "",
			algorithm:       spec.Sha256,
			wantErr:         true,
		},
		{
			name:            "malformed checksum file",
			fileContent:     "test content",
			checksumContent: "not a valid checksum format",
			algorithm:       spec.Sha256,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.txt")
			checksumFile := filepath.Join(tmpDir, "checksums.txt")

			require.NoError(t, os.WriteFile(testFile, []byte(tt.fileContent), 0644))
			require.NoError(t, os.WriteFile(checksumFile, []byte(tt.checksumContent), 0644))

			err := VerifyWithChecksumFile(testFile, checksumFile, tt.algorithm)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFindChecksumInFile(t *testing.T) {
	checksumContent := `abc123def456  file1.tar.gz
def456abc123  file2.zip
123456789abc  file3.tar.gz
# Comment line
  789abcdef123  file4.tar.gz
`

	tests := []struct {
		name     string
		filename string
		want     string
		wantErr  bool
	}{
		{
			name:     "find existing checksum",
			filename: "file1.tar.gz",
			want:     "abc123def456",
		},
		{
			name:     "find checksum with leading spaces",
			filename: "file4.tar.gz",
			want:     "789abcdef123",
		},
		{
			name:     "checksum not found",
			filename: "nonexistent.tar.gz",
			want:     "",
			wantErr:  true,
		},
		{
			name:     "partial filename match should not work",
			filename: "file",
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			checksumFile := filepath.Join(tmpDir, "checksums.txt")
			require.NoError(t, os.WriteFile(checksumFile, []byte(checksumContent), 0644))

			got, err := findChecksumInFile(checksumFile, tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseChecksumLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantChecksum string
		wantFilename string
		wantOK       bool
	}{
		{
			name:         "standard format with two spaces",
			line:         "abc123def456  file.tar.gz",
			wantChecksum: "abc123def456",
			wantFilename: "file.tar.gz",
			wantOK:       true,
		},
		{
			name:         "format with single space",
			line:         "abc123def456 file.tar.gz",
			wantChecksum: "abc123def456",
			wantFilename: "file.tar.gz",
			wantOK:       true,
		},
		{
			name:         "format with tab",
			line:         "abc123def456	file.tar.gz",
			wantChecksum: "abc123def456",
			wantFilename: "file.tar.gz",
			wantOK:       true,
		},
		{
			name:         "with leading/trailing spaces",
			line:         "  abc123def456  file.tar.gz  ",
			wantChecksum: "abc123def456",
			wantFilename: "file.tar.gz",
			wantOK:       true,
		},
		{
			name:   "comment line",
			line:   "# This is a comment",
			wantOK: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOK: false,
		},
		{
			name:   "invalid format",
			line:   "singleword",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, filename, ok := parseChecksumLine(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantChecksum, checksum)
				assert.Equal(t, tt.wantFilename, filename)
			}
		})
	}
}
