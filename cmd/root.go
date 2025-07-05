package cmd

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/spf13/cobra"
)

const (
	// Default config file paths
	DefaultConfigPathYML  = ".config/binstaller.yml"
	DefaultConfigPathYAML = ".config/binstaller.yaml"
)

// resolveConfigFile determines the config file path to use.
// If configFile is not empty, it returns configFile.
// Otherwise, it tries to find default config files in order.
func resolveConfigFile(configFile string) (string, error) {
	if configFile != "" {
		return configFile, nil
	}

	// Try default paths in order
	candidates := []string{DefaultConfigPathYML, DefaultConfigPathYAML}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("config file not specified via --config and default (%s or %s) not found", DefaultConfigPathYML, DefaultConfigPathYAML)
}

var (
	// Global flags
	configFile string
	verbose    bool
	quiet      bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "binst",
	Short: "binst installs binaries from various sources using a spec file.",
	Long: `binstaller (binst) is a tool to generate installer scripts or directly
install binaries based on an InstallSpec configuration file.

It supports generating the spec from sources like GoReleaser config or GitHub releases.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.SetHandler(cli.Default)
		if verbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("Verbose logging enabled")
		} else if quiet {
			log.SetLevel(log.ErrorLevel) // Or FatalLevel? ErrorLevel allows warnings.
		} else {
			log.SetLevel(log.InfoLevel)
		}
		log.Debugf("Config file: %s", configFile)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		log.WithError(err).Fatal("command execution failed")
		// os.Exit(1) // log.Fatal exits automatically
	}
}

func init() {
	// Add global flags
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to InstallSpec config file (default: "+DefaultConfigPathYML+")")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Increase log verbosity")
	RootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress progress output")

	// Mark 'config' flag for auto-detection? Cobra doesn't directly support this.
	// We'll handle default detection logic within commands if the flag is empty.

	// Add subcommands
	RootCmd.AddCommand(InitCommand)
	RootCmd.AddCommand(GenCommand)
	RootCmd.AddCommand(EmbedChecksumsCommand)
}
