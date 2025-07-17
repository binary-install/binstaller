package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/internal/shell" // Placeholder for script generator
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// validateScriptType validates the script type flag value
func validateScriptType(scriptType string) error {
	if scriptType == "" || scriptType == "installer" {
		return nil
	}
	if scriptType == "runner" {
		return nil
	}
	return fmt.Errorf("invalid script type %q: must be 'installer' or 'runner'", scriptType)
}

var (
	// Flags for gen command
	genOutputFile    string
	genTargetVersion string
	genScriptType    string
	genBinaryName    string
	// Input config file is handled by the global --config flag
)

const unnamedBinaryPlaceholder = "<unnamed>"

// getAvailableBinaryNames extracts non-nil, non-empty binary names from the binaries list
func getAvailableBinaryNames(binaries []spec.BinaryElement) []string {
	var names []string
	for _, bin := range binaries {
		if bin.Name != nil && *bin.Name != "" {
			names = append(names, *bin.Name)
		}
	}
	return names
}

// findBinaryByName searches for a binary with the given name in the binaries list
func findBinaryByName(binaries []spec.BinaryElement, name string) (*spec.BinaryElement, bool) {
	for _, bin := range binaries {
		if bin.Name != nil && *bin.Name == name {
			binCopy := bin // Return a copy to avoid modification issues
			return &binCopy, true
		}
	}
	return nil, false
}

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

  # Generate runner script (runs binary without installing)
  binst gen --type=runner -o run.sh

  # Generate runner for specific binary (when multiple binaries exist)
  binst gen --type=runner --binary=mytool-helper -o run-helper.sh

  # Run binary directly using runner script
  ./run.sh -- --help

  # Generate installer from specific config file
  binst gen --config myapp.binstaller.yml -o myapp-install.sh

  # Generate installer from stdin
  cat myapp.binstaller.yml | binst gen --config - -o install.sh

  # Generate installer for a specific version only
  binst gen --target-version v1.2.3 -o install-v1.2.3.sh

  # Generate runner for specific version
  binst gen --type=runner --target-version v1.2.3 -o run-v1.2.3.sh

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

		// Validate script type
		if err := validateScriptType(genScriptType); err != nil {
			log.WithError(err).Error("Invalid script type")
			return err
		}

		// Default to installer if not specified
		if genScriptType == "" {
			genScriptType = "installer"
		}

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

		// Handle binary selection for runner scripts
		if genScriptType == "runner" && installSpec.Asset != nil && len(installSpec.Asset.Binaries) > 1 {
			if genBinaryName == "" {
				// Warning: multiple binaries found, using the first one
				firstBinary := installSpec.Asset.Binaries[0]
				binaryName := unnamedBinaryPlaceholder
				if firstBinary.Name != nil && *firstBinary.Name != "" {
					binaryName = *firstBinary.Name
				}
				log.Warnf("Multiple binaries found. Generating runner for the first binary '%s'.", binaryName)
				log.Warnf("Use --binary flag to specify a different binary.")

				// Available binaries for reference
				availableBinaries := getAvailableBinaryNames(installSpec.Asset.Binaries)
				if len(availableBinaries) > 0 {
					log.Infof("Available binaries: %s", strings.Join(availableBinaries, ", "))
				}
			} else {
				// Validate the specified binary exists
				selectedBinary, found := findBinaryByName(installSpec.Asset.Binaries, genBinaryName)

				if !found {
					log.Errorf("Binary '%s' not found in configuration", genBinaryName)
					availableBinaries := getAvailableBinaryNames(installSpec.Asset.Binaries)
					if len(availableBinaries) > 0 {
						log.Errorf("Available binaries: %s", strings.Join(availableBinaries, ", "))
					}
					return fmt.Errorf("binary '%s' not found", genBinaryName)
				}

				// Filter installSpec to only include the selected binary
				installSpec.Asset.Binaries = []spec.BinaryElement{*selectedBinary}
				log.Infof("Generating runner for binary '%s'", genBinaryName)
			}
		} else if genScriptType == "runner" && genBinaryName != "" {
			// Warning: --binary flag used but not needed
			if installSpec.Asset == nil || len(installSpec.Asset.Binaries) <= 1 {
				log.Warnf("--binary flag specified but configuration has only one binary. Ignoring flag.")
			}
		}

		// Generate the script using the internal shell generator
		log.Infof("Generating %s script...", genScriptType)
		scriptBytes, err := shell.GenerateWithScriptType(&installSpec, genTargetVersion, genScriptType)
		if err != nil {
			log.WithError(err).Errorf("Failed to generate %s script", genScriptType)
			return fmt.Errorf("failed to generate %s script: %w", genScriptType, err)
		}
		log.Debugf("%s script generated successfully", genScriptType)

		// Write the output script
		if genOutputFile == "" || genOutputFile == "-" {
			// Write to stdout
			log.Debugf("Writing %s script to stdout", genScriptType)
			fmt.Print(string(scriptBytes))
			log.Infof("%s script written to stdout", genScriptType)
		} else {
			// Write to file
			log.Infof("Writing %s script to file: %s", genScriptType, genOutputFile)
			// Ensure the output directory exists
			outputDir := filepath.Dir(genOutputFile)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}

			err = os.WriteFile(genOutputFile, scriptBytes, 0755) // Make script executable
			if err != nil {
				log.WithError(err).Errorf("Failed to write %s script to file: %s", genScriptType, genOutputFile)
				return fmt.Errorf("failed to write %s script to file %s: %w", genScriptType, genOutputFile, err)
			}
			log.Infof("%s script successfully written to %s", genScriptType, genOutputFile)
		}

		return nil
	},
}

func init() {
	// Flags specific to gen command
	// Input config file is handled by the global --config flag
	GenCommand.Flags().StringVarP(&genOutputFile, "output", "o", "-", "Output path for the generated script (use '-' for stdout)")
	GenCommand.Flags().StringVar(&genTargetVersion, "target-version", "", "Generate script for specific version only (disables runtime version selection)")
	GenCommand.Flags().StringVar(&genScriptType, "type", "installer", "Type of script to generate (installer, runner)")
	GenCommand.Flags().StringVar(&genBinaryName, "binary", "", "For runner scripts with multiple binaries: specify which binary to run")
}
