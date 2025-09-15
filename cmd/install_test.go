package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"

	"github.com/binary-install/binstaller/pkg/spec"
)

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name            string
		repo            string
		inputVersion    string
		serverResponse  interface{}
		serverStatus    int
		expectedVersion string
		expectedError   bool
		setupEnv        func()
		cleanupEnv      func()
	}{
		{
			name:            "explicit version returns as-is",
			repo:            "owner/repo",
			inputVersion:    "v1.2.3",
			expectedVersion: "v1.2.3",
			expectedError:   false,
		},
		{
			name:            "explicit version without v prefix",
			repo:            "owner/repo",
			inputVersion:    "1.2.3",
			expectedVersion: "1.2.3",
			expectedError:   false,
		},
		{
			name:         "latest resolves to actual tag",
			repo:         "owner/repo",
			inputVersion: "latest",
			serverResponse: GitHubRelease{
				TagName: "v2.0.0",
				Name:    "Release v2.0.0",
			},
			serverStatus:    http.StatusOK,
			expectedVersion: "v2.0.0",
			expectedError:   false,
		},
		{
			name:         "empty version resolves to latest",
			repo:         "owner/repo",
			inputVersion: "",
			serverResponse: GitHubRelease{
				TagName: "v3.0.0",
				Name:    "Release v3.0.0",
			},
			serverStatus:    http.StatusOK,
			expectedVersion: "v3.0.0",
			expectedError:   false,
		},
		{
			name:         "handles GitHub API error",
			repo:         "owner/repo",
			inputVersion: "latest",
			serverResponse: map[string]string{
				"message": "Not Found",
			},
			serverStatus:  http.StatusNotFound,
			expectedError: true,
		},
		{
			name:         "handles empty tag_name",
			repo:         "owner/repo",
			inputVersion: "latest",
			serverResponse: GitHubRelease{
				TagName: "",
				Name:    "Release without tag",
			},
			serverStatus:  http.StatusOK,
			expectedError: true,
		},
		{
			name:         "respects GITHUB_TOKEN",
			repo:         "owner/repo",
			inputVersion: "latest",
			serverResponse: GitHubRelease{
				TagName: "v4.0.0",
				Name:    "Release v4.0.0",
			},
			serverStatus:    http.StatusOK,
			expectedVersion: "v4.0.0",
			expectedError:   false,
			setupEnv: func() {
				os.Setenv("GITHUB_TOKEN", "test-token")
			},
			cleanupEnv: func() {
				os.Unsetenv("GITHUB_TOKEN")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			// Create test server if we need to test API calls
			if tt.inputVersion == "" || tt.inputVersion == "latest" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Verify request path
					expectedPath := "/repos/" + tt.repo + "/releases/latest"
					if r.URL.Path != expectedPath {
						t.Errorf("unexpected path: got %s, want %s", r.URL.Path, expectedPath)
					}

					// Verify GitHub token handling
					// Note: httpclient only adds token for github.com URLs
					// Since this is a test server, we can't verify the token here

					// Send response
					w.WriteHeader(tt.serverStatus)
					if tt.serverResponse != nil {
						json.NewEncoder(w).Encode(tt.serverResponse)
					}
				}))
				defer server.Close()

				// Override GitHub API URL for testing
				oldURL := gitHubAPIBaseURL
				gitHubAPIBaseURL = server.URL
				defer func() { gitHubAPIBaseURL = oldURL }()
			}

			ctx := context.Background()
			version, err := resolveVersion(ctx, tt.repo, tt.inputVersion)

			if tt.expectedError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if version != tt.expectedVersion {
					t.Errorf("unexpected version: got %s, want %s", version, tt.expectedVersion)
				}
			}
		})
	}
}

func TestInstallCommandFlags(t *testing.T) {
	// Reset command for testing
	cmd := InstallCommand

	// Test that flags are properly defined
	binDirFlag := cmd.Flags().Lookup("bin-dir")
	if binDirFlag == nil {
		t.Fatal("bin-dir flag not found")
	}
	if binDirFlag.Shorthand != "b" {
		t.Errorf("bin-dir shorthand: got %s, want b", binDirFlag.Shorthand)
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("dry-run flag not found")
	}
	if dryRunFlag.Shorthand != "n" {
		t.Errorf("dry-run shorthand: got %s, want n", dryRunFlag.Shorthand)
	}
}

func TestInstallCommandArgs(t *testing.T) {
	cmd := InstallCommand

	// Test that command accepts 0 or 1 argument
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("command should accept 0 args: %v", err)
	}

	if err := cmd.Args(cmd, []string{"v1.0.0"}); err != nil {
		t.Errorf("command should accept 1 arg: %v", err)
	}

	if err := cmd.Args(cmd, []string{"v1.0.0", "extra"}); err == nil {
		t.Error("command should reject 2 args")
	}
}

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

func TestDownload(t *testing.T) {
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
			err := download(context.Background(), tt.destPath, tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("download() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to map Go arch to shell script conventions
func mapGoArchToShellArch(goArch string) string {
	switch goArch {
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
