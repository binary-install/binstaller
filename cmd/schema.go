package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/binary-install/binstaller/schema"
	"github.com/spf13/cobra"
)

// SchemaCommand represents the schema command
var SchemaCommand = &cobra.Command{
	Use:   "schema",
	Short: "Display configuration schema",
	Long: `Display binstaller configuration schema directly from the CLI.

This command shows the binstaller configuration schema in various formats.
For filtering and processing, use yq or jq tools on the output.`,
	Example: `  # Display schema in YAML format (default)
  binst schema

  # Display schema in JSON format
  binst schema --format json

  # Display original TypeSpec source
  binst schema --format typespec

  # Filter specific types using yq
  binst schema | yq '."$defs".AssetConfig'

  # List all available types using yq
  binst schema | yq '."$defs" | keys'

  # Get only the root schema properties using yq
  binst schema | yq 'del(."$defs")'

  # Filter specific types using jq
  binst schema --format json | jq '."$defs".Platform'`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		return RunSchema(format, os.Stdout)
	},
}

// RunSchema executes the schema command with the given parameters
func RunSchema(format string, output interface{}) error {
	writer, ok := output.(io.Writer)
	if !ok {
		return fmt.Errorf("output must be an io.Writer")
	}

	// Get the schema in the requested format
	var outputBytes []byte
	switch format {
	case "yaml":
		outputBytes = schema.GetInstallSpecSchemaYAML()
	case "json":
		outputBytes = schema.GetInstallSpecSchemaJSON()
	case "typespec":
		outputBytes = schema.GetTypeSpecSource()
	default:
		return fmt.Errorf("unsupported format: %s (supported: yaml, json, typespec)", format)
	}

	// Write output
	_, err := writer.Write(outputBytes)
	return err
}


func init() {
	SchemaCommand.Flags().StringP("format", "f", "yaml", "Output format (yaml, json, typespec)")
}
