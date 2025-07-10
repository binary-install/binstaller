package httpclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestNewRequestWithGitHubAuth(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		token     string
		wantAuth  bool
		wantToken string
	}{
		{
			name:      "GitHub URL with token",
			url:       "https://github.com/owner/repo/releases/download/v1.0.0/file.tar.gz",
			token:     "ghp_testtoken123",
			wantAuth:  true,
			wantToken: "Bearer ghp_testtoken123",
		},
		{
			name:      "GitHub API URL with token",
			url:       "https://api.github.com/repos/owner/repo/releases/latest",
			token:     "ghp_testtoken456",
			wantAuth:  true,
			wantToken: "Bearer ghp_testtoken456",
		},
		{
			name:     "GitHub URL without token",
			url:      "https://github.com/owner/repo/releases/download/v1.0.0/file.tar.gz",
			token:    "",
			wantAuth: false,
		},
		{
			name:     "Non-GitHub URL with token",
			url:      "https://example.com/file.tar.gz",
			token:    "ghp_testtoken789",
			wantAuth: false,
		},
		{
			name:      "Raw GitHub URL with token",
			url:       "https://raw.githubusercontent.com/owner/repo/main/file.txt",
			token:     "ghp_testtoken999",
			wantAuth:  true,
			wantToken: "Bearer ghp_testtoken999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset the environment variable
			if tt.token != "" {
				os.Setenv("GITHUB_TOKEN", tt.token)
				defer os.Unsetenv("GITHUB_TOKEN")
			} else {
				os.Unsetenv("GITHUB_TOKEN")
			}

			req, err := NewRequestWithGitHubAuth("GET", tt.url)
			if err != nil {
				t.Fatalf("NewRequestWithGitHubAuth() error = %v", err)
			}

			authHeader := req.Header.Get("Authorization")
			if tt.wantAuth {
				if authHeader != tt.wantToken {
					t.Errorf("Authorization header = %v, want %v", authHeader, tt.wantToken)
				}
			} else {
				if authHeader != "" {
					t.Errorf("Authorization header = %v, want empty", authHeader)
				}
			}
		})
	}
}

func TestGitHubTransport(t *testing.T) {
	// Create a test server that echoes back the Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			w.Write([]byte(auth))
		} else {
			w.Write([]byte("no auth"))
		}
	}))
	defer server.Close()

	tests := []struct {
		name         string
		url          string
		token        string
		wantResponse string
	}{
		{
			name:         "Request with token",
			url:          server.URL,
			token:        "ghp_testtoken",
			wantResponse: "no auth", // Test server is not github.com
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.token != "" {
				os.Setenv("GITHUB_TOKEN", tt.token)
				defer os.Unsetenv("GITHUB_TOKEN")
			} else {
				os.Unsetenv("GITHUB_TOKEN")
			}

			client := NewGitHubClient()
			resp, err := client.Get(tt.url)
			if err != nil {
				t.Fatalf("client.Get() error = %v", err)
			}
			defer resp.Body.Close()

			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			response := string(body[:n])

			if response != tt.wantResponse {
				t.Errorf("Response = %v, want %v", response, tt.wantResponse)
			}
		})
	}
}

func TestGitHubTransportPreservesExistingAuth(t *testing.T) {
	// Create a test server that echoes back the Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "" {
			w.Write([]byte(auth))
		} else {
			w.Write([]byte("no auth"))
		}
	}))
	defer server.Close()

	// Mock GitHub URL by replacing server URL with github.com for transport to recognize
	githubURL := "https://github.com/test/test"
	
	// Set up environment token
	os.Setenv("GITHUB_TOKEN", "env_token")
	defer os.Unsetenv("GITHUB_TOKEN")

	// Create a request with pre-existing Authorization header
	req, err := http.NewRequest("GET", githubURL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer existing_token")

	// Mock the transport to use our test server
	transport := &gitHubTransport{
		Base: &mockTransport{server.URL},
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip error: %v", err)
	}
	defer resp.Body.Close()

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	response := string(body[:n])

	// Should preserve the existing Authorization header, not use the environment token
	expected := "Bearer existing_token"
	if response != expected {
		t.Errorf("Response = %v, want %v", response, expected)
	}
}

// mockTransport is a helper for testing that redirects requests to a test server
type mockTransport struct {
	testServerURL string
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to our test server
	newReq := req.Clone(req.Context())
	newReq.URL.Host = strings.TrimPrefix(t.testServerURL, "http://")
	newReq.URL.Scheme = "http"
	return http.DefaultTransport.RoundTrip(newReq)
}

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "github.com URL",
			url:  "https://github.com/owner/repo",
			want: true,
		},
		{
			name: "api.github.com URL",
			url:  "https://api.github.com/repos/owner/repo",
			want: true,
		},
		{
			name: "raw.githubusercontent.com URL",
			url:  "https://raw.githubusercontent.com/owner/repo/main/file",
			want: true,
		},
		{
			name: "non-GitHub URL",
			url:  "https://example.com/file",
			want: false,
		},
		{
			name: "http github.com URL",
			url:  "http://github.com/owner/repo",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGitHubURL(tt.url); got != tt.want {
				t.Errorf("isGitHubURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient()
	if client == nil {
		t.Fatal("NewGitHubClient() returned nil")
	}

	// Check that the transport is set correctly
	transport, ok := client.Transport.(*gitHubTransport)
	if !ok {
		t.Error("NewGitHubClient() did not set gitHubTransport")
	}

	if transport.Base != http.DefaultTransport {
		t.Error("gitHubTransport.Base is not http.DefaultTransport")
	}
}
