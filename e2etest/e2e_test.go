package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

var binstallerPath string

// TestMain builds the binstaller binary once before running all tests
func TestMain(m *testing.M) {
	// Create a temporary directory for the binstaller binary
	tempDir, err := os.MkdirTemp("", "binstaller-test")
	if err != nil {
		panic("Failed to create temp directory: " + err.Error())
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			panic("Failed to remove temp directory: " + err.Error())
		}
	}()

	// Build the binstaller tool to a temporary location
	execName := "binst"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	binstallerPath = filepath.Join(tempDir, execName)
	cmd := exec.Command("go", "build", "-o", binstallerPath, "./cmd/binst")
	cmd.Dir = ".." // Go up one level to reach the root directory
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("Failed to build binstaller: " + err.Error())
	}

	// Run the tests
	os.Exit(m.Run())
}

// testInstallScript tests that the binstaller tool can generate a working
// installation script for the specified repository and that the script
// can successfully install the binary.
func testInstallScript(t *testing.T, repo, binaryName, versionFlag, sha string) {
	// Create a temporary directory for all test artifacts
	tempDir := t.TempDir()

	// Init binstaller config
	configPath := filepath.Join(tempDir, binaryName+".binstaller.yml")
	initCmd := exec.Command(binstallerPath, "init", "--verbose", "--source=goreleaser", "--repo", repo, "-o", configPath, "--sha", sha)
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		t.Fatalf("Failed to init binstaller config: %v", err)
	}

	// Check that the config file content
	configContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}
	// Log the config file content
	t.Logf("Config file content:\n%s", configContent)

	// Generate installer script
	installerPath := filepath.Join(tempDir, binaryName+".binstaller.sh")
	genCmd := exec.Command(binstallerPath, "gen", "--config", configPath, "-o", installerPath)
	genCmd.Stderr = os.Stderr
	if err := genCmd.Run(); err != nil {
		t.Fatalf("Failed to generate installation script: %v", err)
	}

	// Create a temporary bin directory
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Run the installation script
	var stderr bytes.Buffer
	var installStdout bytes.Buffer
	installCmd := exec.Command("sh", installerPath, "-b", binDir, "-d")
	installCmd.Stderr = &stderr
	installCmd.Stdout = &installStdout
	if err := installCmd.Run(); err != nil {
		t.Fatalf("Failed to run installation script: %v\nStdout: %s\nStderr: %s", err, installStdout.String(), stderr.String())
	}

	// Check that the binary was installed
	binName := binaryName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binaryPath := filepath.Join(binDir, binName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Fatalf("%s binary was not installed at %s", binName, binaryPath)
	}

	// Check that the binary works
	var stdout bytes.Buffer
	stderr.Reset()
	versionCmd := exec.Command(binaryPath, versionFlag)
	versionCmd.Stdout = &stdout
	versionCmd.Stderr = &stderr
	if err := versionCmd.Run(); err != nil {
		t.Fatalf("Failed to run %s %s: %v", binaryName, versionFlag, err)
	}

	output := stdout.String()
	stderrOutput := stderr.String()
	if output == "" && stderrOutput == "" {
		t.Fatalf("%s %s returned empty output", binaryName, versionFlag)
	}

	t.Logf("Successfully installed and ran %s with %s flag", binaryName, versionFlag)
}

func TestReviewdogE2E(t *testing.T) {
	testInstallScript(t, "reviewdog/reviewdog", "reviewdog", "-version", "7e05fa3e78ba7f2be4999ca2d35b00a3fd92a783")
}

func TestGoreleaserE2E(t *testing.T) {
	testInstallScript(t, "goreleaser/goreleaser", "goreleaser", "--version", "79c76c229d50ca45ef77afa1745df0a0e438d237")
}

func TestGhSetupE2E(t *testing.T) {
	testInstallScript(t, "k1LoW/gh-setup", "gh-setup", "--help", "a2359e4bcda8af5d7e16e1b3fb0eeec1be267e63")
}

func TestSigspyE2E(t *testing.T) {
	testInstallScript(t, "actionutils/sigspy", "sigspy", "--help", "3e1c6f32072cd4b8309d00bd31f498903f71c422")
}

func TestGolangciLintE2E(t *testing.T) {
	testInstallScript(t, "golangci/golangci-lint", "golangci-lint", "--version", "6d2a94be6b20f1c06e95d79479c6fdc34a69c45f")
}

// TestTargetVersionGeneration tests the --target-version flag functionality
func TestTargetVersionGeneration(t *testing.T) {
	// Create a temporary directory for test artifacts
	tempDir := t.TempDir()

	// Create a test config with embedded checksums
	configContent := `schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: ${NAME}-${VERSION}-${OS}_${ARCH}${EXT}
  default_extension: .tar.gz
checksums:
  algorithm: sha256
  template: ${NAME}_${VERSION}_checksums.txt
  embedded_checksums:
    v1.2.3:
      - filename: test-tool-1.2.3-linux_amd64.tar.gz
        hash: abc123def456abc123def456abc123def456abc123def456abc123def456abc1
    v1.2.4:
      - filename: test-tool-1.2.4-linux_amd64.tar.gz
        hash: def456ghi789def456ghi789def456ghi789def456ghi789def456ghi789def4
supported_platforms:
  - os: linux
    arch: amd64
  - os: darwin
    arch: amd64
`
	configPath := filepath.Join(tempDir, "test.binstaller.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Run("target_version_script_generation", func(t *testing.T) {
		// Generate installer script with target version
		installerPath := filepath.Join(tempDir, "install-v1.2.3.sh")
		genCmd := exec.Command(binstallerPath, "gen", "--config", configPath, "--target-version", "v1.2.3", "-o", installerPath)
		genCmd.Stderr = os.Stderr
		if err := genCmd.Run(); err != nil {
			t.Fatalf("Failed to generate installation script with target version: %v", err)
		}

		// Read the generated script
		scriptContent, err := os.ReadFile(installerPath)
		if err != nil {
			t.Fatalf("Failed to read generated script: %v", err)
		}

		// Verify target version is embedded
		if !bytes.Contains(scriptContent, []byte(`TAG="v1.2.3"`)) {
			t.Error("Generated script should contain fixed TAG=v1.2.3")
		}

		// Verify installing message with version
		if !bytes.Contains(scriptContent, []byte("Installing ${NAME} version ${VERSION}")) {
			t.Error("Generated script should contain 'Installing' message with version")
		}

		// Verify usage message mentions the fixed version
		if !bytes.Contains(scriptContent, []byte("This installer is configured for v1.2.3 only")) {
			t.Error("Generated script should mention fixed version in usage")
		}

		// Verify no dynamic version logic
		if bytes.Contains(scriptContent, []byte(`TAG="${1:-latest}"`)) {
			t.Error("Generated script should not contain dynamic TAG assignment")
		}

		if bytes.Contains(scriptContent, []byte("checking GitHub for latest tag")) {
			t.Error("Generated script should not contain GitHub API calls")
		}

		// Verify only target version checksums are included
		if !bytes.Contains(scriptContent, []byte("1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123")) {
			t.Error("Generated script should contain v1.2.3 checksums")
		}

		if bytes.Contains(scriptContent, []byte("1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:def456")) {
			t.Error("Generated script should not contain v1.2.4 checksums")
		}

		t.Logf("Successfully generated fixed version script for v1.2.3")
	})

	t.Run("normal_generation_includes_all_checksums", func(t *testing.T) {
		// Generate normal installer script without target version
		installerPath := filepath.Join(tempDir, "install-normal.sh")
		genCmd := exec.Command(binstallerPath, "gen", "--config", configPath, "-o", installerPath)
		genCmd.Stderr = os.Stderr
		if err := genCmd.Run(); err != nil {
			t.Fatalf("Failed to generate normal installation script: %v", err)
		}

		// Read the generated script
		scriptContent, err := os.ReadFile(installerPath)
		if err != nil {
			t.Fatalf("Failed to read generated script: %v", err)
		}

		// Verify dynamic version logic is present
		if !bytes.Contains(scriptContent, []byte(`TAG="${1:-latest}"`)) {
			t.Error("Normal script should contain dynamic TAG assignment")
		}

		// Verify all checksums are included
		if !bytes.Contains(scriptContent, []byte("1.2.3:test-tool-1.2.3-linux_amd64.tar.gz:abc123")) {
			t.Error("Normal script should contain v1.2.3 checksums")
		}

		if !bytes.Contains(scriptContent, []byte("1.2.4:test-tool-1.2.4-linux_amd64.tar.gz:def456")) {
			t.Error("Normal script should contain v1.2.4 checksums")
		}

		// Verify no fixed version installing message
		if bytes.Contains(scriptContent, []byte("Installing ${NAME} version ${VERSION}")) {
			t.Error("Normal script should not contain fixed version installing message")
		}

		t.Logf("Successfully generated normal dynamic script")
	})
}

// TestTargetVersionFlag tests the CLI flag validation
func TestTargetVersionFlag(t *testing.T) {
	tempDir := t.TempDir()

	// Create a minimal test config
	configContent := `schema: v1
name: test-tool
repo: owner/test-tool
asset:
  template: ${NAME}-${VERSION}-${OS}_${ARCH}${EXT}
  default_extension: .tar.gz
`
	configPath := filepath.Join(tempDir, "test.binstaller.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	t.Run("help_shows_target_version_flag", func(t *testing.T) {
		var stdout bytes.Buffer
		helpCmd := exec.Command(binstallerPath, "gen", "--help")
		helpCmd.Stdout = &stdout
		if err := helpCmd.Run(); err != nil {
			t.Fatalf("Failed to run gen --help: %v", err)
		}

		helpOutput := stdout.String()
		if !bytes.Contains([]byte(helpOutput), []byte("--target-version")) {
			t.Error("gen --help should show --target-version flag")
		}

		if !bytes.Contains([]byte(helpOutput), []byte("Generate script for specific version only")) {
			t.Error("gen --help should describe --target-version flag")
		}
	})

	t.Run("target_version_flag_works", func(t *testing.T) {
		installerPath := filepath.Join(tempDir, "test-target.sh")
		genCmd := exec.Command(binstallerPath, "gen", "--config", configPath, "--target-version", "v2.0.0", "-o", installerPath)
		genCmd.Stderr = os.Stderr
		if err := genCmd.Run(); err != nil {
			t.Fatalf("Failed to generate script with --target-version: %v", err)
		}

		// Verify the script was generated
		if _, err := os.Stat(installerPath); os.IsNotExist(err) {
			t.Error("Script file was not generated")
		}

		// Verify the script contains the target version
		scriptContent, err := os.ReadFile(installerPath)
		if err != nil {
			t.Fatalf("Failed to read generated script: %v", err)
		}

		if !bytes.Contains(scriptContent, []byte(`TAG="v2.0.0"`)) {
			t.Error("Script should contain the specified target version")
		}
	})
}
