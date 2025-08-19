package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// Load reads and parses a binstaller config file from the given path
func Load(path string) (*spec.InstallSpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read config file: %s", path)
	}

	var cfg spec.InstallSpec
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to parse config file: %s", path)
	}

	// Apply defaults
	cfg.SetDefaults()

	return &cfg, nil
}

// Discover searches for a binstaller config file in the current directory
// and parent directories, following the same logic as other binst commands
func Discover() (string, error) {
	// Start from current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get current directory")
	}

	// Look for .config/binstaller.yml in current and parent directories
	for {
		configPath := filepath.Join(dir, ".config", "binstaller.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}

		// Check if we've reached the root
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no binstaller config found")
}

// LoadOrDiscover loads a config from the given path, or discovers one if path is empty
func LoadOrDiscover(configPath string) (*spec.InstallSpec, string, error) {
	var path string
	var err error

	if configPath != "" {
		path = configPath
	} else {
		path, err = Discover()
		if err != nil {
			return nil, "", err
		}
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, "", err
	}

	return cfg, path, nil
}
