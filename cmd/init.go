package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/binary-install/binstaller/pkg/datasource"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Flags for init command
	initSource       string
	initSourceFile   string
	initRepo         string // Repo for GitHub source OR explicit override
	initName         string // Explicit override for binary name
	initTag          string
	initCommitSHA    string
	initOutputFile   string
)

// InitCommand represents the init command
var InitCommand = &cobra.Command{
	Use:   "init",
	Short: "Generate an InstallSpec config file from various sources",
	Long: `Initializes a binstaller configuration file (.config/binstaller.yml) by detecting
settings from a source like a GoReleaser config file or a GitHub repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infof("Running init command...")

		var adapter datasource.SourceAdapter

		switch initSource {
		case "goreleaser":
			adapter = datasource.NewGoReleaserAdapter(
				initRepo,       // repo
				initSourceFile, // filePath
				initCommitSHA,  // commit
				initName,       // nameOverride
			)
		case "github":
			adapter = datasource.NewGitHubAdapter(initRepo)
		case "aqua":
			// Use --file for registry YAML, or stdin if not specified
			switch initSourceFile {
			case "":
				// No file: use repo (and optionally commit SHA/ref)
				if initRepo == "" {
					return fmt.Errorf("--repo is required for aqua source when --file is not specified")
				}
				adapter = datasource.NewAquaRegistryAdapterFromRepo(initRepo, initCommitSHA)
			case "-":
				// --file=- means stdin
				adapter = datasource.NewAquaRegistryAdapterFromReader(os.Stdin)
			default:
				// --file=path
				f, err := os.Open(initSourceFile)
				if err != nil {
					return fmt.Errorf("failed to open aqua registry file: %w", err)
				}
				defer f.Close()
				adapter = datasource.NewAquaRegistryAdapterFromReader(f)
			}
		default:
			err := fmt.Errorf("unknown source specified: %s. Valid sources are: goreleaser, github, aqua", initSource)
			log.WithError(err).Error("invalid source")
			return err
		}

		ctx := context.Background()

		// Generate the InstallSpec
		log.Infof("Generating InstallSpec using source: %s", initSource)
		installSpec, err := adapter.GenerateInstallSpec(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to detect install spec")
			return fmt.Errorf("failed to detect install spec: %w", err)
		}
		if installSpec.Schema == "" {
			installSpec.Schema = "v1"
		}
		log.Info("Successfully detected InstallSpec")

		// Marshal the spec to YAML
		log.Debug("Marshalling InstallSpec to YAML")
		yamlData, err := yaml.Marshal(installSpec)
		if err != nil {
			log.WithError(err).Error("Failed to marshal InstallSpec to YAML")
			return fmt.Errorf("failed to marshal install spec to YAML: %w", err)
		}

		// Write the output
		if initOutputFile == "" || initOutputFile == "-" {
			// Write to stdout
			log.Debug("Writing InstallSpec YAML to stdout")
			fmt.Println(string(yamlData))
			log.Info("InstallSpec YAML written to stdout")
		} else {
			// Write to file
			log.Infof("Writing InstallSpec YAML to file: %s", initOutputFile)
			
			// Ensure the output directory exists
			outputDir := filepath.Dir(initOutputFile)
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
				return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
			}
			
			err = os.WriteFile(initOutputFile, yamlData, 0644) // Use standard file permissions
			if err != nil {
				log.WithError(err).Errorf("Failed to write InstallSpec to file: %s", initOutputFile)
				return fmt.Errorf("failed to write install spec to file %s: %w", initOutputFile, err)
			}
			log.Infof("InstallSpec successfully written to %s", initOutputFile)
		}

		return nil
	},
}

func init() {
	// Required flags
	InitCommand.Flags().StringVar(&initSource, "source", "", "Source type to detect spec from (required: goreleaser, aqua, github)")
	_ = InitCommand.MarkFlagRequired("source")

	// Optional flags (depending on source)
	InitCommand.Flags().StringVar(&initSourceFile, "file", "", "Path to source file (e.g., .goreleaser.yml)")
	InitCommand.Flags().StringVar(&initRepo, "repo", "", "GitHub repository (owner/repo) for source 'goreleaser'/'github', or explicit override")
	InitCommand.Flags().StringVar(&initName, "name", "", "Explicit binary name override")
	InitCommand.Flags().StringVar(&initTag, "tag", "", "Release tag/ref to inspect (for source 'github')")
	InitCommand.Flags().StringVar(&initCommitSHA, "sha", "", "Commit SHA for source 'goreleaser'")
	InitCommand.Flags().StringVarP(&initOutputFile, "output", "o", DefaultConfigPathYML, "Write spec to file instead of stdout (use '-' for stdout)")

	// TODO: Add dependencies between flags (e.g., --file required if --source goreleaser and no --repo)
}
