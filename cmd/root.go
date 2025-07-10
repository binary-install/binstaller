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
	Short: "Config-driven secure shell-script installer generator",
	Long: `binstaller (binst) is a config-driven secure shell-script installer generator that
creates reproducible installation scripts for static binaries distributed via GitHub releases.

It works with Go binaries, Rust binaries, and any other static binaries - as long as they're
released on GitHub, binstaller can generate installation scripts for them.`,
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
	// Disable automatic command sorting to maintain semantic order
	cobra.EnableCommandSorting = false

	// Add global flags
	RootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Path to InstallSpec config file (default: "+DefaultConfigPathYML+")")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Increase log verbosity")
	RootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress progress output")

	// Mark 'config' flag for auto-detection? Cobra doesn't directly support this.
	// We'll handle default detection logic within commands if the flag is empty.

	// Add command groups
	RootCmd.AddGroup(&cobra.Group{
		ID:    "workflow",
		Title: "Workflow Commands:",
	})
	RootCmd.AddGroup(&cobra.Group{
		ID:    "utility",
		Title: "Utility Commands:",
	})

	// Set group for built-in commands
	RootCmd.SetHelpCommandGroupID("utility")
	RootCmd.SetCompletionCommandGroupID("utility")

	// Add subcommands with groups
	InitCommand.GroupID = "workflow"
	CheckCommand.GroupID = "workflow"
	EmbedChecksumsCommand.GroupID = "workflow"
	GenCommand.GroupID = "workflow"
	HelpfulCommand.GroupID = "utility"
	SchemaCommand.GroupID = "utility"
	
	RootCmd.AddCommand(InitCommand)           // Step 1: Initialize config
	RootCmd.AddCommand(CheckCommand)          // Step 2: Validate config
	RootCmd.AddCommand(EmbedChecksumsCommand) // Step 3: Embed checksums (optional)
	RootCmd.AddCommand(GenCommand)            // Step 4: Generate installer
	RootCmd.AddCommand(HelpfulCommand)        // Utility: Comprehensive help for LLMs
	RootCmd.AddCommand(SchemaCommand)         // Utility: Display configuration schema
}
