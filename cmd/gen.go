package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/internal/cmdutil"
	"github.com/binary-install/binstaller/internal/shell" // Placeholder for script generator
	"github.com/binary-install/binstaller/pkg/spec"
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

// handleRunnerBinarySelection handles binary selection logic for runner scripts
func handleRunnerBinarySelection(installSpec *spec.InstallSpec, scriptType, binaryName string) error {
	// Only apply to runner scripts
	if scriptType != "runner" {
		return nil
	}

	// Check if we need to handle binary selection
	if installSpec.Asset == nil || len(installSpec.Asset.Binaries) <= 1 {
		// Single binary or no binaries - check if binary flag was unnecessarily used
		if binaryName != "" && len(installSpec.Asset.Binaries) <= 1 {
			log.Warnf("--binary flag specified but configuration has only one binary. Ignoring flag.")
		}
		return nil
	}

	// Multiple binaries case
	if binaryName == "" {
		// No binary specified - use first one with warning
		firstBinary := installSpec.Asset.Binaries[0]
		binaryName := ""
		if firstBinary.Name != nil && *firstBinary.Name != "" {
			binaryName = *firstBinary.Name
		}
		log.Warnf("Multiple binaries found. Generating runner for the first binary '%s'.", binaryName)
		log.Warnf("Use --binary flag to specify a different binary.")

		availableBinaries := getAvailableBinaryNames(installSpec.Asset.Binaries)
		if len(availableBinaries) > 0 {
			log.Infof("Available binaries: %s", strings.Join(availableBinaries, ", "))
		}
		// Keep only the first binary
		installSpec.Asset.Binaries = installSpec.Asset.Binaries[:1]
	} else {
		// Binary specified - validate and select it
		selectedBinary, found := findBinaryByName(installSpec.Asset.Binaries, binaryName)
		if !found {
			log.Errorf("Binary '%s' not found in configuration", binaryName)
			availableBinaries := getAvailableBinaryNames(installSpec.Asset.Binaries)
			if len(availableBinaries) > 0 {
				log.Errorf("Available binaries: %s", strings.Join(availableBinaries, ", "))
			}
			return fmt.Errorf("binary '%s' not found", binaryName)
		}

		// Filter installSpec to only include the selected binary
		installSpec.Asset.Binaries = []spec.BinaryElement{*selectedBinary}
		log.Infof("Generating runner for binary '%s'", binaryName)
	}

	return nil
}

// writeScript writes the generated script to the specified output
func writeScript(scriptBytes []byte, outputFile, scriptType string) error {
	if outputFile == "" || outputFile == "-" {
		// Write to stdout
		log.Debugf("Writing %s script to stdout", scriptType)
		fmt.Print(string(scriptBytes))
		log.Infof("%s script written to stdout", scriptType)
	} else {
		// Write to file
		log.Infof("Writing %s script to file: %s", scriptType, outputFile)

		// Ensure the output directory exists
		outputDir := filepath.Dir(outputFile)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}

		err := os.WriteFile(outputFile, scriptBytes, 0755) // Make script executable
		if err != nil {
			log.WithError(err).Errorf("Failed to write %s script to file: %s", scriptType, outputFile)
			return fmt.Errorf("failed to write %s script to file %s: %w", scriptType, outputFile, err)
		}
		log.Infof("%s script successfully written to %s", scriptType, outputFile)
	}
	return nil
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

		// Validate and normalize script type
		if err := validateScriptType(genScriptType); err != nil {
			log.WithError(err).Error("Invalid script type")
			return err
		}
		if genScriptType == "" {
			genScriptType = "installer"
		}

		// Resolve config file path
		cfgFile, err := resolveConfigFile(configFile)
		if err != nil {
			log.WithError(err).Error("Config file detection failed")
			return err
		}
		if configFile == "" {
			log.Infof("Using default config file: %s", cfgFile)
		}
		log.Debugf("Using config file: %s", cfgFile)

		// Load and parse InstallSpec
		installSpec, err := cmdutil.LoadInstallSpec(cfgFile)
		if err != nil {
			return err
		}

		// Handle binary selection for runner scripts
		if err := handleRunnerBinarySelection(installSpec, genScriptType, genBinaryName); err != nil {
			return err
		}

		// Generate the script
		log.Infof("Generating %s script...", genScriptType)
		scriptBytes, err := shell.GenerateWithScriptType(installSpec, genTargetVersion, genScriptType)
		if err != nil {
			log.WithError(err).Errorf("Failed to generate %s script", genScriptType)
			return fmt.Errorf("failed to generate %s script: %w", genScriptType, err)
		}
		log.Debugf("%s script generated successfully", genScriptType)

		// Write the output
		return writeScript(scriptBytes, genOutputFile, genScriptType)
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
