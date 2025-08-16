package fetch

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		wantErr     bool
		validate    func(t *testing.T, path string)
	}{
		{
			name: "successful download",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/octet-stream")
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "test binary content")
				}))
			},
			validate: func(t *testing.T, path string) {
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "test binary content", string(content))
			},
		},
		{
			name: "download with redirect",
			setupServer: func() *httptest.Server {
				redirected := false
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if !redirected {
						redirected = true
						http.Redirect(w, r, "/redirected", http.StatusFound)
						return
					}
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "redirected content")
				}))
			},
			validate: func(t *testing.T, path string) {
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "redirected content", string(content))
			},
		},
		{
			name: "download failure - 404",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			wantErr: true,
		},
		{
			name: "download with retry on temporary error",
			setupServer: func() *httptest.Server {
				attempts := 0
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					attempts++
					if attempts < 2 {
						w.WriteHeader(http.StatusServiceUnavailable)
						return
					}
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "success after retry")
				}))
			},
			validate: func(t *testing.T, path string) {
				content, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "success after retry", string(content))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, "downloaded-file")

			err := Download(server.URL, destPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.FileExists(t, destPath)

			if tt.validate != nil {
				tt.validate(t, destPath)
			}
		})
	}
}

func TestDownloadAsset(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		tag         string
		filename    string
		setupServer func() *httptest.Server
		wantErr     bool
	}{
		{
			name:     "successful asset download",
			repo:     "owner/repo",
			tag:      "v1.0.0",
			filename: "binary-linux-amd64.tar.gz",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					expectedPath := "/owner/repo/releases/download/v1.0.0/binary-linux-amd64.tar.gz"
					assert.Equal(t, expectedPath, r.URL.Path)

					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "asset content")
				}))
			},
		},
		{
			name:     "download with GitHub token",
			repo:     "private/repo",
			tag:      "v2.0.0",
			filename: "tool.zip",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check for authorization header
					authHeader := r.Header.Get("Authorization")
					if authHeader != "Bearer test-token" {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}

					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, "private asset")
				}))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test environment
			if tt.name == "download with GitHub token" {
				os.Setenv("GITHUB_TOKEN", "test-token")
				defer os.Unsetenv("GITHUB_TOKEN")
			}

			server := tt.setupServer()
			defer server.Close()

			tmpDir := t.TempDir()
			destPath := filepath.Join(tmpDir, tt.filename)

			// Override GitHub URL for testing
			originalURL := githubDownloadURL
			githubDownloadURL = server.URL
			defer func() { githubDownloadURL = originalURL }()

			err := DownloadAsset(tt.repo, tt.tag, tt.filename, destPath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.FileExists(t, destPath)
		})
	}
}

func TestDownloadWithProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := make([]byte, 1024*10) // 10KB
		for i := range content {
			content[i] = byte(i % 256)
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)

		// Write in chunks to simulate progress
		chunkSize := 1024
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}
			w.Write(content[i:end])
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "large-file")

	progressCalled := false
	err := DownloadWithProgress(server.URL, destPath, func(downloaded, total int64) {
		progressCalled = true
		assert.True(t, downloaded <= total)
		assert.True(t, total > 0)
	})

	require.NoError(t, err)
	assert.True(t, progressCalled, "progress callback should have been called")

	// Verify file size
	info, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.Equal(t, int64(1024*10), info.Size())
}

func TestCopyWithRetry(t *testing.T) {
	content := []byte("test content for retry")

	t.Run("successful copy", func(t *testing.T) {
		src := &mockReader{data: content}
		dst := &mockWriter{}

		n, err := copyWithRetry(dst, src, 3)
		require.NoError(t, err)
		assert.Equal(t, int64(len(content)), n)
		assert.Equal(t, content, dst.data)
	})

	t.Run("error handling", func(t *testing.T) {
		src := &mockReader{data: content, failCount: 1}
		dst := &mockWriter{}

		_, err := copyWithRetry(dst, src, 3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "temporary error")
	})

	t.Run("fail after max retries", func(t *testing.T) {
		src := &mockReader{data: content, failCount: 5}
		dst := &mockWriter{}

		_, err := copyWithRetry(dst, src, 3)
		assert.Error(t, err)
	})
}

// Mock reader for testing retries
type mockReader struct {
	data      []byte
	pos       int
	attempts  int
	failCount int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	m.attempts++
	if m.attempts <= m.failCount {
		return 0, fmt.Errorf("temporary error")
	}

	if m.pos >= len(m.data) {
		return 0, io.EOF
	}

	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

// Mock writer for testing
type mockWriter struct {
	data []byte
}

func (m *mockWriter) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}
