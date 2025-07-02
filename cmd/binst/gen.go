package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/internal/shell" // Placeholder for script generator
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Flags for gen command
	genOutputFile string
	// Input config file is handled by the global --config flag
)

// genCmd represents the gen command
var genCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate an installer script from an InstallSpec config file",
	Long: `Reads an InstallSpec configuration file (e.g., .binstaller.yml) and
generates a POSIX-compatible shell installer script.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running gen command...")

		// Determine config file path
		cfgFile := configFile // Use the global flag value
		if cfgFile == "" {
			// Default detection logic if global flag is not set
			defaultPath := ".config/binstaller.yml"
			if _, err := os.Stat(defaultPath); err == nil {
				cfgFile = defaultPath
				log.Infof("Using default config file: %s", cfgFile)
			} else {
				// Try .config/binstaller.yaml as fallback
				defaultPathYaml := ".config/binstaller.yaml"
				if _, errYaml := os.Stat(defaultPathYaml); errYaml == nil {
					cfgFile = defaultPathYaml
					log.Infof("Using default config file: %s", cfgFile)
				} else {
					err := fmt.Errorf("config file not specified via --config and default (.config/binstaller.yml or .config/binstaller.yaml) not found")
					log.WithError(err).Error("Config file detection failed")
					return err
				}
			}
		}
		log.Debugf("Using config file: %s", cfgFile)

		// Read the InstallSpec YAML file
		log.Debugf("Reading InstallSpec from: %s", cfgFile)
		var yamlData []byte
		var err error
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
		scriptBytes, err := shell.Generate(&installSpec) // Pass the loaded spec
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
	rootCmd.AddCommand(genCmd)

	// Flags specific to gen command
	// Input config file is handled by the global --config flag
	genCmd.Flags().StringVarP(&genOutputFile, "output", "o", "-", "Output path for the generated script (use '-' for stdout)")
}
