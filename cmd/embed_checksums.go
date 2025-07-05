package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/checksums"
	"github.com/binary-install/binstaller/pkg/spec"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/spf13/cobra"
)

var (
	// Flags for embed-checksums command
	embedVersion string
	embedOutput  string
	embedMode    string
	embedFile    string
)

// EmbedChecksumsCommand represents the embed-checksums command
var EmbedChecksumsCommand = &cobra.Command{
	Use:   "embed-checksums",
	Short: "Embed checksums for release assets into a binstaller configuration",
	Long: `Reads an InstallSpec configuration file and embeds checksums for the assets.
This command supports three modes of operation:
- download: Fetches the checksum file from GitHub releases
- checksum-file: Uses a local checksum file
- calculate: Downloads the assets and calculates checksums directly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running embed-checksums command...")

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

		ast, err := parser.ParseFile(cfgFile, parser.ParseComments)
		if err != nil {
			return err
		}

		yamlData, err := os.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).Errorf("Failed to read install spec file: %s", cfgFile)
			return fmt.Errorf("failed to read install spec file %s: %w", cfgFile, err)
		}

		// Unmarshal YAML into InstallSpec struct
		log.Debug("Unmarshalling InstallSpec YAML")
		var installSpec spec.InstallSpec
		err = yaml.UnmarshalWithOptions(yamlData, &installSpec, yaml.UseOrderedMap())
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal install spec YAML from: %s", cfgFile)
			return fmt.Errorf("failed to unmarshal install spec YAML from %s: %w", cfgFile, err)
		}

		// Create the embedder
		var mode checksums.EmbedMode
		switch embedMode {
		case "download":
			mode = checksums.EmbedModeDownload
		case "checksum-file":
			mode = checksums.EmbedModeChecksumFile
		case "calculate":
			mode = checksums.EmbedModeCalculate
		default:
			return fmt.Errorf("invalid mode: %s. Must be one of: download, checksum-file, calculate", embedMode)
		}

		// Validate checksum-file mode has a file
		if mode == checksums.EmbedModeChecksumFile && embedFile == "" {
			log.Error("--file flag is required for checksum-file mode")
			return fmt.Errorf("--file flag is required for checksum-file mode")
		}

		embedder := &checksums.Embedder{
			Mode:         mode,
			Version:      embedVersion,
			Spec:         &installSpec,
			SpecAST:      ast,
			ChecksumFile: embedFile,
		}

		// Embed the checksums
		log.Infof("Embedding checksums using %s mode for version: %s", mode, embedVersion)
		if err := embedder.Embed(); err != nil {
			log.WithError(err).Error("Failed to embed checksums")
			return fmt.Errorf("failed to embed checksums: %w", err)
		}

		// Determine output file
		outputFile := embedOutput
		if outputFile == "" {
			outputFile = cfgFile
			log.Infof("No output specified, overwriting input file: %s", outputFile)
		}

		// Write the updated InstallSpec back to the output file
		log.Infof("Writing updated InstallSpec to file: %s", outputFile)

		// Ensure the output directory exists
		outputDir := filepath.Dir(outputFile)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}

		// Write the YAML to the output file
		if err := os.WriteFile(outputFile, []byte(ast.String()), 0644); err != nil {
			log.WithError(err).Errorf("Failed to write InstallSpec to file: %s", outputFile)
			return fmt.Errorf("failed to write InstallSpec to file %s: %w", outputFile, err)
		}
		log.Infof("InstallSpec successfully updated with embedded checksums")

		return nil
	},
}

func init() {
	// Flags specific to embed-checksums command
	EmbedChecksumsCommand.Flags().StringVarP(&embedVersion, "version", "v", "", "Version to embed checksums for (default: latest)")
	EmbedChecksumsCommand.Flags().StringVarP(&embedOutput, "output", "o", "", "Output path for the updated InstallSpec (default: overwrite input file)")
	EmbedChecksumsCommand.Flags().StringVarP(&embedMode, "mode", "m", "download", "Checksums acquisition mode (download, checksum-file, calculate)")
	EmbedChecksumsCommand.Flags().StringVarP(&embedFile, "file", "f", "", "Path to checksum file (required for checksum-file mode)")

	// Mark required flags
	EmbedChecksumsCommand.MarkFlagRequired("mode")
}
