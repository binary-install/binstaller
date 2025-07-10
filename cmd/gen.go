package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/internal/shell" // Placeholder for script generator
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

var (
	// Flags for gen command
	genOutputFile    string
	genTargetVersion string
	// Input config file is handled by the global --config flag
)

// GenCommand represents the gen command
var GenCommand = &cobra.Command{
	Use:   "gen",
	Short: "Generate an installer script from an InstallSpec config file",
	Long: `Reads an InstallSpec configuration file (e.g., .binstaller.yml) and
generates a POSIX-compatible shell installer script.`,
	Example: `  # Generate installer script using default config
  binst gen

  # Generate installer with custom output file
  binst gen -o install.sh

  # Generate installer from specific config file
  binst gen --config myapp.binstaller.yml -o myapp-install.sh

  # Generate installer from stdin
  cat myapp.binstaller.yml | binst gen --config - -o install.sh

  # Generate installer for a specific version only
  binst gen --target-version v1.2.3 -o install-v1.2.3.sh

  # Typical workflow with init and gen
  binst init --source=github --repo=owner/repo
  binst gen -o install.sh

  # Generate and execute installer script directly
  binst gen | sh

  # View generated script's help
  binst gen | sh -s -- -h

  # Install to custom directory
  binst gen | sh -s -- -b /usr/local/bin

  # Install specific version
  binst gen | sh -s -- v1.2.3

  # Test installer with dry run mode
  binst gen | sh -s -- -n`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running gen command...")

		// Determine config file path using common logic
		cfgFile, err := resolveConfigFile(configFile)
		if err != nil {
			log.WithError(err).Error("Config file detection failed")
			return err
		}
		if configFile == "" {
			log.Infof("Using default config file: %s", cfgFile)
		}
		log.Debugf("Using config file: %s", cfgFile)

		// Read the InstallSpec YAML file
		log.Debugf("Reading InstallSpec from: %s", cfgFile)
		var yamlData []byte
		if cfgFile == "-" {
			log.Debug("Reading install spec from stdin")
			yamlData, err = io.ReadAll(os.Stdin)
			if err != nil {
				log.WithError(err).Error("Failed to read install spec from stdin")
				return fmt.Errorf("failed to read install spec from stdin: %w", err)
			}
		} else {
			yamlData, err = os.ReadFile(cfgFile)
			if err != nil {
				log.WithError(err).Errorf("Failed to read install spec file: %s", cfgFile)
				return fmt.Errorf("failed to read install spec file %s: %w", cfgFile, err)
			}
		}

		// Unmarshal YAML into InstallSpec struct
		log.Debug("Unmarshalling InstallSpec YAML")
		var installSpec spec.InstallSpec
		err = yaml.Unmarshal(yamlData, &installSpec)
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal install spec YAML from: %s", cfgFile)
			return fmt.Errorf("failed to unmarshal install spec YAML from %s: %w", cfgFile, err)
		}

		// Generate the script using the internal shell generator
		log.Info("Generating installer script...")
		scriptBytes, err := shell.GenerateWithVersion(&installSpec, genTargetVersion) // Pass the loaded spec and target version
		if err != nil {
			log.WithError(err).Error("Failed to generate installer script")
			return fmt.Errorf("failed to generate installer script: %w", err)
		}
		log.Debug("Installer script generated successfully")

		// Write the output script
		if genOutputFile == "" || genOutputFile == "-" {
			// Write to stdout
			log.Debug("Writing installer script to stdout")
			fmt.Print(string(scriptBytes))
			log.Info("Installer script written to stdout")
		} else {
			// Write to file
			log.Infof("Writing installer script to file: %s", genOutputFile)
			// Ensure the output directory exists
			outputDir := filepath.Dir(genOutputFile)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}

			err = os.WriteFile(genOutputFile, scriptBytes, 0755) // Make script executable
			if err != nil {
				log.WithError(err).Errorf("Failed to write installer script to file: %s", genOutputFile)
				return fmt.Errorf("failed to write installer script to file %s: %w", genOutputFile, err)
			}
			log.Infof("Installer script successfully written to %s", genOutputFile)
		}

		return nil
	},
}

func init() {
	// Flags specific to gen command
	// Input config file is handled by the global --config flag
	GenCommand.Flags().StringVarP(&genOutputFile, "output", "o", "-", "Output path for the generated script (use '-' for stdout)")
	GenCommand.Flags().StringVar(&genTargetVersion, "target-version", "", "Generate script for specific version only (disables runtime version selection)")
}
