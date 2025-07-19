package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
)

// loadInstallSpec loads and parses the InstallSpec from the config file
func loadInstallSpec(cfgFile string) (*spec.InstallSpec, error) {
	// Read the InstallSpec YAML file
	log.Debugf("Reading InstallSpec from: %s", cfgFile)
	var yamlData []byte
	var err error

	if cfgFile == "-" {
		log.Debug("Reading install spec from stdin")
		yamlData, err = io.ReadAll(os.Stdin)
		if err != nil {
			log.WithError(err).Error("Failed to read install spec from stdin")
			return nil, fmt.Errorf("failed to read install spec from stdin: %w", err)
		}
	} else {
		yamlData, err = os.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).Errorf("Failed to read install spec file: %s", cfgFile)
			return nil, fmt.Errorf("failed to read install spec file %s: %w", cfgFile, err)
		}
	}

	// Unmarshal YAML into InstallSpec struct
	log.Debug("Unmarshalling InstallSpec YAML")
	var installSpec spec.InstallSpec
	err = yaml.Unmarshal(yamlData, &installSpec)
	if err != nil {
		log.WithError(err).Errorf("Failed to unmarshal install spec YAML from: %s", cfgFile)
		return nil, fmt.Errorf("failed to unmarshal install spec YAML from %s: %w", cfgFile, err)
	}

	return &installSpec, nil
}
