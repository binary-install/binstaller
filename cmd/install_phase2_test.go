package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		name         string
		spec         *spec.InstallSpec
		expectedOS   string
		expectedArch string
	}{
		{
			name:         "Basic detection",
			spec:         &spec.InstallSpec{},
			expectedOS:   runtime.GOOS,
			expectedArch: mapGoArchToShellArch(runtime.GOARCH),
		},
		{
			name: "Rosetta2 disabled",
			spec: &spec.InstallSpec{
				Asset: &spec.Asset{
					ArchEmulation: &spec.ArchEmulation{
						Rosetta2: boolPtr(false),
					},
				},
			},
			expectedOS:   runtime.GOOS,
			expectedArch: mapGoArchToShellArch(runtime.GOARCH),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os, arch := detectPlatform(tt.spec)
			if os != tt.expectedOS {
				t.Errorf("detectPlatform() os = %v, want %v", os, tt.expectedOS)
			}
			if arch != tt.expectedArch {
				t.Errorf("detectPlatform() arch = %v, want %v", arch, tt.expectedArch)
			}
		})
	}
}

func TestDetectOS(t *testing.T) {
	osName := detectOS()
	expected := runtime.GOOS

	// Special case mappings
	switch runtime.GOOS {
	case "sunos":
		expected = "solaris"
	}

	if osName != expected {
		t.Errorf("detectOS() = %v, want %v", osName, expected)
	}
}

func TestDetectArch(t *testing.T) {
	arch := detectArch()
	expected := mapGoArchToShellArch(runtime.GOARCH)

	if arch != expected {
		t.Errorf("detectArch() = %v, want %v", arch, expected)
	}
}

func TestValidateURL(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}

		switch r.URL.Path {
		case "/valid":
			w.WriteHeader(http.StatusOK)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "Valid URL",
			url:     server.URL + "/valid",
			wantErr: false,
		},
		{
			name:    "Not found URL",
			url:     server.URL + "/notfound",
			wantErr: true,
		},
		{
			name:    "Invalid URL",
			url:     "http://[::1]:invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(context.Background(), tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDownloadWithProgress(t *testing.T) {
	// Create test server
	testContent := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		switch r.URL.Path {
		case "/download":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testContent)))
			w.WriteHeader(http.StatusOK)
			w.Write(testContent)
		case "/notfound":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("Unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	// Create temp directory for downloads
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		url      string
		destPath string
		wantErr  bool
	}{
		{
			name:     "Successful download",
			url:      server.URL + "/download",
			destPath: tempDir + "/test.txt",
			wantErr:  false,
		},
		{
			name:     "Not found",
			url:      server.URL + "/notfound",
			destPath: tempDir + "/notfound.txt",
			wantErr:  true,
		},
		{
			name:     "Invalid destination",
			url:      server.URL + "/download",
			destPath: "/invalid/path/file.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := downloadWithProgress(context.Background(), tt.destPath, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("downloadWithProgress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to map Go arch to shell script conventions
func mapGoArchToShellArch(goArch string) string {
	switch goArch {
	case "amd64":
		return "amd64"
	case "386":
		return "386"
	case "arm64":
		return "arm64"
	case "arm":
		return "armv7"
	default:
		return goArch
	}
}

// Helper function to create bool pointer
func boolPtr(b bool) *bool {
	return &b
}
