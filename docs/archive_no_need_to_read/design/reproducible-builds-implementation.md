---
title: "Reproducible Builds Implementation Plan"
date: "2025-04-18"
author: "goinstaller team"
version: "0.1.1"
status: "draft"
---

# Reproducible Builds Implementation Plan

This document provides a detailed implementation plan for the reproducible builds feature described in [reproducible-builds.md](./reproducible-builds.md).

## Implementation Approach

After careful consideration, we've decided to modify the `Load` function to also return source information. This approach:

1. Minimizes API calls to GitHub
2. Ensures the source info is consistent with what was actually loaded
3. Is more efficient for local files as well

## Code Changes

### 1. Update TemplateContext Struct

Update the `TemplateContext` struct in `main.go` to include a `SourceInfo` field:

```go
type TemplateContext struct {
    *config.Project
    EnableGHAttestation      bool
    RequireAttestation       bool
    GHAttestationVerifyFlags string
    SourceInfo               string // New field for source information
}
```

### 2. Update Load Function

Modify the `Load` function in `main.go` to also return source information:

```go
// Load project configuration from a given repo name or filepath/url.
// Returns the project configuration and source information.
func Load(repo, configPath, file string) (project *config.Project, sourceInfo string, err error) {
    if repo == "" && file == "" {
        return nil, "", fmt.Errorf("repo or file not specified")
    }

    // Get the goinstaller version to include in the source info
    version := getVersion()

    // Load the project configuration
    if file == "" {
        // GitHub repository
        project, sourceInfo, err = loadFromGitHub(repo, configPath, version)
    } else {
        // Local file
        project, sourceInfo, err = loadFromFile(file, version)
    }

    if err != nil {
        return nil, "", err
    }

    // if not specified add in GitHub owner/repo info
    if project.Release.GitHub.Owner == "" {
        // ... existing code ...
    }

    return project, sourceInfo, nil
}

// loadFromGitHub loads a project configuration from a GitHub repository
// and returns the project and source information.
func loadFromGitHub(repo, configPath, version string) (*config.Project, string, error) {
    repo = normalizeRepo(repo)
    log.Infof("reading repo %q on github", repo)
    defaultBranch := getDefaultBranch(repo)

    // Load the project configuration
    project, err := loadURLs(
        fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", repo, defaultBranch),
        configPath,
    )
    if err != nil {
        return nil, "", err
    }

    // Try to get the commit hash, but don't fail if we can't
    commitHash, err := getGitHubCommitHash(repo)
    if err != nil {
        // Fallback to just using the repo and branch
        sourceInfo := fmt.Sprintf("github.com/%s@%s (goinstaller %s)", repo, defaultBranch, version)
        log.Infof("using fallback source info: %s", sourceInfo)
        return project, sourceInfo, nil
    }

    // Use the commit hash in the source info
    sourceInfo := fmt.Sprintf("github.com/%s@%s (goinstaller %s)", repo, commitHash, version)
    log.Infof("using source info with commit hash: %s", sourceInfo)
    return project, sourceInfo, nil
}

// loadFromFile loads a project configuration from a local file
// and returns the project and source information.
func loadFromFile(file, version string) (*config.Project, string, error) {
    log.Infof("reading file %q", file)

    // Load the project configuration
    project, err := loadFile(file)
    if err != nil {
        return nil, "", err
    }

    // Get file information
    fileInfo, err := os.Stat(file)
    if err != nil {
        // Fallback to just using the file path
        sourceInfo := fmt.Sprintf("%s (goinstaller %s)", file, version)
        log.Infof("using fallback source info: %s", sourceInfo)
        return project, sourceInfo, nil
    }

    // Get absolute path for better context
    absPath, _ := filepath.Abs(file)

    // Check if the file is part of a git repository
    gitCommitHash, err := getGitCommitHashForFile(file)
    if err == nil && gitCommitHash != "" {
        // Check if the file has uncommitted changes
        isModified, err := isFileModifiedInGit(file)
        if err == nil && !isModified {
            // File is in a git repository and has no uncommitted changes
            sourceInfo := fmt.Sprintf("%s@%s (git commit, goinstaller %s)", absPath, gitCommitHash, version)
            log.Infof("using source info with git commit hash: %s", sourceInfo)
            return project, sourceInfo, nil
        }
        log.Infof("file has uncommitted changes, not using git commit hash")
    }

    // Calculate the SHA-256 hash of the file content
    hash, err := calculateFileHash(file)
    if err != nil {
        // Fallback to just using the file path, size, and mod time
        fileSize := fileInfo.Size()
        modTime := fileInfo.ModTime().UTC().Format(time.RFC3339)
        sourceInfo := fmt.Sprintf("%s (size: %d bytes, modified: %s, goinstaller %s)",
            absPath, fileSize, modTime, version)
        log.Infof("using fallback source info: %s", sourceInfo)
        return project, sourceInfo, nil
    }

    // Use the content hash in the source info
    fileSize := fileInfo.Size()
    modTime := fileInfo.ModTime().UTC().Format(time.RFC3339)
    sourceInfo := fmt.Sprintf("%s@%s (size: %d bytes, modified: %s, goinstaller %s)",
        absPath, hash, fileSize, modTime, version)
    log.Infof("using source info with content hash: %s", sourceInfo)
    return project, sourceInfo, nil
}

// getVersion returns the version of goinstaller
func getVersion() string {
    // This would be replaced with the actual version from the build
    return "v0.1.0"
}

// Helper functions for calculating source information

// getGitHubCommitHash returns the commit hash of the default branch
func getGitHubCommitHash(repo string) (string, error) {
    url := fmt.Sprintf("https://api.github.com/repos/%s", repo)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    // Use GITHUB_TOKEN if available to avoid rate limiting
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        req.Header.Set("Authorization", "token "+token)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to get commit hash: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
    }

    var repoInfo struct {
        DefaultBranch string `json:"default_branch"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
        return "", err
    }

    // Get the latest commit on the default branch
    return getLatestCommitSHA(repo, repoInfo.DefaultBranch)
}

// getLatestCommitSHA gets the SHA of the latest commit on the specified branch
func getLatestCommitSHA(repo, branch string) (string, error) {
    url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, branch)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        req.Header.Set("Authorization", "token "+token)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("failed to get commit SHA: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
    }

    var commit struct {
        SHA string `json:"sha"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
        return "", err
    }

    return commit.SHA, nil
}

// getGitCommitHashForFile returns the commit hash of the last commit that modified the file
func getGitCommitHashForFile(file string) (string, error) {
    // Get the absolute path of the file
    absPath, err := filepath.Abs(file)
    if err != nil {
        return "", err
    }

    // Check if the file is in a git repository
    cmd := exec.Command("git", "rev-parse", "--show-toplevel")
    cmd.Dir = filepath.Dir(absPath)
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("file is not in a git repository: %v", err)
    }

    // Get the commit hash of the last commit that modified the file
    cmd = exec.Command("git", "log", "-n", "1", "--pretty=format:%H", "--", absPath)
    cmd.Dir = filepath.Dir(absPath)
    var out bytes.Buffer
    cmd.Stdout = &out
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("failed to get git commit hash: %v", err)
    }

    return strings.TrimSpace(out.String()), nil
}

// isFileModifiedInGit checks if the file has uncommitted changes in git
func isFileModifiedInGit(file string) (bool, error) {
    // Get the absolute path of the file
    absPath, err := filepath.Abs(file)
    if err != nil {
        return false, err
    }

    // Check if the file has uncommitted changes
    cmd := exec.Command("git", "diff", "--quiet", "--", absPath)
    cmd.Dir = filepath.Dir(absPath)
    err = cmd.Run()

    // If the command exits with a non-zero status, the file has uncommitted changes
    return err != nil, nil
}

// calculateFileHash calculates the SHA-256 hash of a file
func calculateFileHash(file string) (string, error) {
    f, err := os.Open(file)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

### 3. Update Functions that Call Load

Update the `processGodownloader` function in `shell_godownloader.go` to use the updated `Load` function:

```go
func processGodownloader(repo, path, filename string, attestationOpts AttestationOptions) ([]byte, error) {
    // Load the configuration and get source information in one call
    cfg, sourceInfo, err := Load(repo, path, filename)
    if err != nil {
        return nil, fmt.Errorf("unable to parse: %s", err)
    }

    // We only handle the first archive.
    if len(cfg.Archives) == 0 {
        return nil, fmt.Errorf("no archives found in configuration")
    }

    archive := cfg.Archives[0]

    // get archive name template
    archName, err := makeName("", archive.NameTemplate)
    if err != nil {
        return nil, fmt.Errorf("unable generate archive name: %s", err)
    }

    // Store the modified name template back to the archive
    archive.NameTemplate = "NAME=" + archName

    // get checksum name template
    checkName, err := makeName("", cfg.Checksum.NameTemplate)
    if err != nil {
        return nil, fmt.Errorf("unable generate checksum name: %s", err)
    }

    // Store the modified checksum name template
    cfg.Checksum.NameTemplate = "CHECKSUM=" + checkName
    if err != nil {
        return nil, fmt.Errorf("unable generate checksum name: %s", err)
    }

    // Create a template context with the config, attestation options, and source info
    ctx := TemplateContext{
        Project:                  cfg,
        EnableGHAttestation:      attestationOpts.EnableGHAttestation,
        RequireAttestation:       attestationOpts.RequireAttestation,
        GHAttestationVerifyFlags: attestationOpts.GHAttestationVerifyFlags,
        SourceInfo:               sourceInfo,
    }

    return makeShell(shellGodownloader, ctx)
}
```

Note that we're now getting the source information directly from the `Load` function, rather than calculating it separately. This ensures that the source information is consistent with what was actually loaded and reduces the number of API calls.

### 4. Update makeShell Function

Update the `makeShell` function in `main.go` to add the `sourceInfo` function:

```go
func makeShell(tplsrc string, ctx TemplateContext) ([]byte, error) {
    // if we want to add a timestamp in the templates this
    //  function will generate it
    funcMap := template.FuncMap{
        "join":             strings.Join,
        "platformBinaries": makePlatformBinaries,
        // Keep the timestamp function for backward compatibility
        "timestamp": func() string {
            return time.Now().UTC().Format(time.RFC3339)
        },
        // Add the sourceInfo function that returns the source information
        "sourceInfo": func() string {
            return ctx.SourceInfo
        },
        "replace": strings.ReplaceAll,
        "time": func(s string) string {
            return time.Now().UTC().Format(s)
        },
        "tolower": strings.ToLower,
        "toupper": strings.ToUpper,
        "trim":    strings.TrimSpace,
        // ... rest of the function
    }
    // ... rest of the function
}
```

Note that we're keeping the `timestamp` function for backward compatibility, but adding the new `sourceInfo` function that returns the value from the context.

### 5. Update Shell Script Template

Update the shell script template in `shell_godownloader.go` to use the new source information:

```go
const shellGodownloader = `#!/bin/sh
set -e
# Code generated by godownloader from {{ sourceInfo }}. DO NOT EDIT.
#

usage() {
  this=$1
  cat <<EOF
$this: download go binaries for {{ $.Project.Release.GitHub.Owner }}/{{ $.Project.Release.GitHub.Name }}
...
```

The key change here is replacing `{{ timestamp }}` with `{{ sourceInfo }}` in the header comment. This ensures that the generated script includes the deterministic source information instead of a timestamp.

For GitHub repositories, the source information will look like:
```
# Code generated by godownloader from github.com/owner/repo@abc123 (goinstaller v0.1.0). DO NOT EDIT.
```

For local files in a git repository with no uncommitted changes, the source information will look like:
```
# Code generated by godownloader from /path/to/file@abc123 (git commit, goinstaller v0.1.0). DO NOT EDIT.
```

For local files that are not in a git repository or have uncommitted changes, the source information will look like:
```
# Code generated by godownloader from /path/to/file@abc123 (size: 1234 bytes, modified: 2025-04-18T11:30:00Z, goinstaller v0.1.0). DO NOT EDIT.
```

This makes it clear where the script was generated from and provides enough information to reproduce the exact same script if needed.

## Testing

### Unit Tests

Add unit tests for the new functions in a new file `reproducible_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSourceInfo(t *testing.T) {
	// Test with a GitHub repository
	repo := "goreleaser/godownloader"
	sourceInfo, err := getSourceInfo(repo, "")
	if err != nil {
		t.Fatalf("getSourceInfo failed for GitHub repo: %v", err)
	}
	if sourceInfo == "" {
		t.Fatal("sourceInfo is empty for GitHub repo")
	}
	t.Logf("GitHub source info: %s", sourceInfo)

	// Test with a local file
	tempFile, err := os.CreateTemp("", "godownloader-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content to the file
	content := []byte(`
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: "{{ .ProjectName }}_checksums.txt"
`)
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	sourceInfo, err = getSourceInfo("", tempFile.Name())
	if err != nil {
		t.Fatalf("getSourceInfo failed for local file: %v", err)
	}
	if sourceInfo == "" {
		t.Fatal("sourceInfo is empty for local file")
	}
	t.Logf("Local file source info: %s", sourceInfo)
}

func TestReproducibility(t *testing.T) {
	// Create a temporary file with a fixed content
	tempFile, err := os.CreateTemp("", "godownloader-test-*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	// Write some content to the file
	content := []byte(`
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
checksum:
  name_template: "{{ .ProjectName }}_checksums.txt"
`)
	if _, err := tempFile.Write(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Generate the script twice
	opts := AttestationOptions{
		EnableGHAttestation:      false,
		RequireAttestation:       false,
		GHAttestationVerifyFlags: "",
	}

	script1, err := processGodownloader("", "", tempFile.Name(), opts)
	if err != nil {
		t.Fatalf("First processGodownloader failed: %v", err)
	}

	script2, err := processGodownloader("", "", tempFile.Name(), opts)
	if err != nil {
		t.Fatalf("Second processGodownloader failed: %v", err)
	}

	// Compare the scripts
	if string(script1) != string(script2) {
		t.Fatal("Generated scripts are not identical")
	}

	t.Log("Generated scripts are identical, reproducibility confirmed")
}
```

### Integration Tests

Add an integration test in `e2etest/reproducible_test.go`:

```go
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestReproducibleBuilds(t *testing.T) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Generate the installation script for a repository twice
	repo := "goreleaser/goreleaser"
	script1Path := filepath.Join(tempDir, "script1.sh")
	script2Path := filepath.Join(tempDir, "script2.sh")

	// Generate the first script
	var stdout1 bytes.Buffer
	generateCmd1 := exec.Command(goinstallerPath, "--repo="+repo)
	generateCmd1.Stdout = &stdout1
	if err := generateCmd1.Run(); err != nil {
		t.Fatalf("Failed to generate first installation script: %v", err)
	}

	// Write the first script to a file
	if err := os.WriteFile(script1Path, stdout1.Bytes(), 0755); err != nil {
		t.Fatalf("Failed to write first installation script: %v", err)
	}

	// Generate the second script
	var stdout2 bytes.Buffer
	generateCmd2 := exec.Command(goinstallerPath, "--repo="+repo)
	generateCmd2.Stdout = &stdout2
	if err := generateCmd2.Run(); err != nil {
		t.Fatalf("Failed to generate second installation script: %v", err)
	}

	// Write the second script to a file
	if err := os.WriteFile(script2Path, stdout2.Bytes(), 0755); err != nil {
		t.Fatalf("Failed to write second installation script: %v", err)
	}

	// Compare the scripts
	script1, err := os.ReadFile(script1Path)
	if err != nil {
		t.Fatalf("Failed to read first script: %v", err)
	}

	script2, err := os.ReadFile(script2Path)
	if err != nil {
		t.Fatalf("Failed to read second script: %v", err)
	}

	if !bytes.Equal(script1, script2) {
		t.Fatal("Generated scripts are not identical")
	}

	t.Log("Generated scripts are identical, reproducibility confirmed")
}
```

### Manual Testing

1. Test with a GitHub repository:
   - Generate a script for a GitHub repository: `./goinstaller --repo=goreleaser/goreleaser > script1.sh`
   - Generate the script again: `./goinstaller --repo=goreleaser/goreleaser > script2.sh`
   - Compare the scripts: `diff script1.sh script2.sh`
   - Verify that the scripts are identical
   - Verify that the source information includes the repository name and commit hash

2. Test with a local file in a git repository:
   - Clone a repository: `git clone https://github.com/goreleaser/goreleaser.git`
   - Generate a script for a file in the repository: `./goinstaller --file=goreleaser/.goreleaser.yml > script1.sh`
   - Generate the script again: `./goinstaller --file=goreleaser/.goreleaser.yml > script2.sh`
   - Compare the scripts: `diff script1.sh script2.sh`
   - Verify that the scripts are identical
   - Verify that the source information includes the file path and git commit hash

3. Test with a local file with uncommitted changes:
   - Clone a repository: `git clone https://github.com/goreleaser/goreleaser.git`
   - Make a change to a file: `echo "# Test" >> goreleaser/.goreleaser.yml`
   - Generate a script for the modified file: `./goinstaller --file=goreleaser/.goreleaser.yml > script1.sh`
   - Generate the script again: `./goinstaller --file=goreleaser/.goreleaser.yml > script2.sh`
   - Compare the scripts: `diff script1.sh script2.sh`
   - Verify that the scripts are identical
   - Verify that the source information includes the file path and content hash, not the git commit hash

4. Test with a local file not in a git repository:
   - Create a temporary file: `echo "archives:" > /tmp/test.yml`
   - Generate a script for the file: `./goinstaller --file=/tmp/test.yml > script1.sh`
   - Generate the script again: `./goinstaller --file=/tmp/test.yml > script2.sh`
   - Compare the scripts: `diff script1.sh script2.sh`
   - Verify that the scripts are identical
   - Verify that the source information includes the file path and content hash

5. Test with different inputs:
   - Generate scripts with different inputs and verify that they have different source information
   - Modify the input file and verify that the source information changes

## Implementation Notes

- The implementation should handle errors gracefully and provide fallback information if it can't get the commit hash or file hash.
- For GitHub repositories, we need to make an additional API call to get the commit hash, which might be rate-limited.
- For local files in git repositories, we use the git commit hash if the file has no uncommitted changes, which provides better traceability.
- For local files with uncommitted changes or not in git repositories, we calculate a hash of the file content, which will change if the file content changes.
- The implementation needs to check if a file is in a git repository and if it has uncommitted changes, which requires executing git commands.
- The goinstaller version is included in the source information to provide additional context about how the script was generated.
- The version information is obtained from the build process, which sets the version during compilation.

## Future Work

- Add a command-line flag to disable the source information for users who prefer not to include it.
- Explore other ways to make the build process more reproducible.
