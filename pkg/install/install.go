package install

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// ResolveInstallDir resolves the installation directory, handling defaults and expansions
func ResolveInstallDir(binDir string) (string, error) {
	if binDir == "" {
		// Use default from environment or HOME
		if envBin := os.Getenv("BINSTALLER_BIN"); envBin != "" {
			binDir = envBin
		} else if home := os.Getenv("HOME"); home != "" {
			binDir = filepath.Join(home, ".local", "bin")
		} else {
			return "", fmt.Errorf("could not determine install directory: no HOME environment variable")
		}
	}

	// Expand path (handles ~ and environment variables)
	binDir = expandPath(binDir)

	// Make absolute
	absPath, err := filepath.Abs(binDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to resolve install directory")
	}

	return absPath, nil
}

// InstallBinary installs a binary from source to the target directory
func InstallBinary(sourcePath, targetDir, targetName string) (string, error) {
	// Add .exe extension on Windows if not present
	if runtime.GOOS == "windows" && !strings.HasSuffix(targetName, ".exe") {
		targetName += ".exe"
	}

	targetPath := filepath.Join(targetDir, targetName)

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to create install directory")
	}

	// Open source file
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", errors.Wrap(err, "failed to open source file")
	}
	defer source.Close()

	// Create temporary file in target directory for atomic replacement
	tmpFile, err := os.CreateTemp(targetDir, "."+targetName+"-*")
	if err != nil {
		return "", errors.Wrap(err, "failed to create temporary file")
	}
	tmpPath := tmpFile.Name()

	// Clean up on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Copy content
	if _, err := io.Copy(tmpFile, source); err != nil {
		tmpFile.Close()
		return "", errors.Wrap(err, "failed to copy binary")
	}

	// Set executable permissions
	if err := tmpFile.Chmod(0755); err != nil {
		tmpFile.Close()
		return "", errors.Wrap(err, "failed to set permissions")
	}

	if err := tmpFile.Close(); err != nil {
		return "", errors.Wrap(err, "failed to close temporary file")
	}

	// Atomic replacement
	if err := atomicInstall(tmpPath, targetPath); err != nil {
		return "", err
	}

	success = true
	return targetPath, nil
}

// atomicInstall performs an atomic file replacement
func atomicInstall(sourcePath, targetPath string) error {
	// On Unix, rename is atomic
	if err := os.Rename(sourcePath, targetPath); err != nil {
		// On Windows or cross-device, fall back to remove + rename
		if runtime.GOOS == "windows" || os.IsExist(err) {
			if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
				return errors.Wrap(err, "failed to remove existing file")
			}
			if err := os.Rename(sourcePath, targetPath); err != nil {
				return errors.Wrap(err, "failed to install binary")
			}
		} else {
			return errors.Wrap(err, "failed to install binary")
		}
	}
	return nil
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string) string {
	// Expand ~ to HOME
	if strings.HasPrefix(path, "~/") {
		if home := os.Getenv("HOME"); home != "" {
			path = filepath.Join(home, path[2:])
		}
	}

	// Expand environment variables
	path = os.ExpandEnv(path)

	return path
}

// DryRunOutput returns the message to display for a dry run
func DryRunOutput(sourcePath, targetPath string) string {
	return fmt.Sprintf("Would install %s to %s", sourcePath, targetPath)
}
