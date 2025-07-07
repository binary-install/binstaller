package httpclient

import (
	"net/http"
	"os"
	"strings"
)

// NewGitHubClient creates an HTTP client configured for GitHub API requests.
// It automatically adds the GitHub token from GITHUB_TOKEN environment variable if available.
func NewGitHubClient() *http.Client {
	return &http.Client{
		Transport: &gitHubTransport{
			Base: http.DefaultTransport,
		},
	}
}

// gitHubTransport is a custom RoundTripper that adds GitHub authentication
type gitHubTransport struct {
	Base http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface
func (t *gitHubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())

	// Add GitHub token if available and the request is to GitHub
	if isGitHubURL(req2.URL.String()) {
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req2.Header.Set("Authorization", "Bearer "+token)
		}
	}

	return t.Base.RoundTrip(req2)
}

// NewRequestWithGitHubAuth creates a new HTTP request and adds GitHub authentication if available.
// This is useful for one-off requests where you don't want to create a custom client.
func NewRequestWithGitHubAuth(method, url string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Add GitHub token if available and the URL is GitHub
	if isGitHubURL(url) {
		if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	return req, nil
}

// isGitHubURL checks if a URL is a GitHub URL
func isGitHubURL(url string) bool {
	return strings.Contains(url, "github.com") || strings.Contains(url, "api.github.com") || strings.Contains(url, "githubusercontent.com")
}
