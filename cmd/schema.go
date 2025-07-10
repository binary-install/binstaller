package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

// SchemaCommand represents the schema command
var SchemaCommand = &cobra.Command{
	Use:   "schema",
	Short: "Display configuration schema",
	Long: `Display binstaller configuration schema directly from the CLI.

This command shows the JSON schema for binstaller configuration files
in various formats (YAML, JSON, TypeSpec) and allows filtering by type.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		typeFilter, _ := cmd.Flags().GetString("type")
		list, _ := cmd.Flags().GetBool("list")
		
		return RunSchema(format, typeFilter, list, os.Stdout)
	},
}

// RunSchema executes the schema command with the given parameters
func RunSchema(format, typeFilter string, list bool, output interface{}) error {
	writer, ok := output.(io.Writer)
	if !ok {
		return fmt.Errorf("output must be an io.Writer")
	}

	// Validate format
	if format != "yaml" && format != "json" {
		return fmt.Errorf("format %s not implemented", format)
	}

	if typeFilter != "" {
		return fmt.Errorf("type filtering not implemented")
	}

	if list {
		return fmt.Errorf("list option not implemented")
	}

	// Load and convert schema
	schema, err := loadInstallSpecSchema()
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	outputBytes, err := convertSchemaToFormat(schema, format)
	if err != nil {
		return fmt.Errorf("failed to convert to %s: %w", format, err)
	}

	// Write output
	_, err = writer.Write(outputBytes)
	return err
}

// loadInstallSpecSchema loads and parses the InstallSpec JSON schema
func loadInstallSpecSchema() (interface{}, error) {
	schemaPath := filepath.Join("..", "schema", "output", "@typespec", "json-schema", "InstallSpec.json")
	installSpecJSON, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var jsonSchema interface{}
	if err := json.Unmarshal(installSpecJSON, &jsonSchema); err != nil {
		return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	return jsonSchema, nil
}

// convertToYAML converts a JSON schema to YAML format
func convertToYAML(schema interface{}) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to YAML: %w", err)
	}
	return yamlBytes, nil
}

// convertToJSON converts a JSON schema to formatted JSON
func convertToJSON(schema interface{}) ([]byte, error) {
	jsonBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to convert to JSON: %w", err)
	}
	return jsonBytes, nil
}

// convertSchemaToFormat converts schema to the specified format
func convertSchemaToFormat(schema interface{}, format string) ([]byte, error) {
	switch format {
	case "yaml":
		return convertToYAML(schema)
	case "json":
		return convertToJSON(schema)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func init() {
	SchemaCommand.Flags().StringP("format", "f", "yaml", "Output format (yaml, json, typespec)")
	SchemaCommand.Flags().StringP("type", "t", "", "Display specific schema type")
	SchemaCommand.Flags().BoolP("list", "l", false, "List available schema types")
}