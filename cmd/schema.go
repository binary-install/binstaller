package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/binary-install/binstaller/schema"
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
	if format != "yaml" && format != "json" && format != "typespec" {
		return fmt.Errorf("format %s not implemented", format)
	}

	// typeFilter will be handled in convertSchemaToFormat

	// Handle list option
	if list {
		return listSchemaTypes(writer)
	}

	// Convert to requested format
	outputBytes, err := convertSchemaToFormat(nil, format, typeFilter)
	if err != nil {
		return fmt.Errorf("failed to convert to %s: %w", format, err)
	}

	// Write output
	_, err = writer.Write(outputBytes)
	return err
}

// loadInstallSpecSchema loads and parses the InstallSpec JSON schema
func loadInstallSpecSchema() (interface{}, error) {
	return schema.GetInstallSpecSchema()
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

// convertToTypeSpec reads and returns the TypeSpec source file
func convertToTypeSpec() ([]byte, error) {
	return schema.GetTypeSpecSource(), nil
}

// filterSchemaByType extracts a specific type from the schema's $defs section
func filterSchemaByType(schema interface{}, typeName string) (interface{}, error) {
	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("schema is not a map")
	}

	// Check if the type is the root InstallSpec
	if typeName == "InstallSpec" {
		// Return the root schema without $defs
		rootSchema := make(map[string]interface{})
		for k, v := range schemaMap {
			if k != "$defs" {
				rootSchema[k] = v
			}
		}
		return rootSchema, nil
	}

	// Look for the type in $defs
	defs, ok := schemaMap["$defs"]
	if !ok {
		return nil, fmt.Errorf("schema does not contain $defs section")
	}

	defsMap, ok := defs.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("$defs is not a map")
	}

	typeDef, ok := defsMap[typeName]
	if !ok {
		return nil, fmt.Errorf("type %s not found in $defs", typeName)
	}

	return typeDef, nil
}

// listSchemaTypes lists all available schema types
func listSchemaTypes(writer io.Writer) error {
	schema, err := loadInstallSpecSchema()
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	schemaMap, ok := schema.(map[string]interface{})
	if !ok {
		return fmt.Errorf("schema is not a map")
	}

	// Get types from $defs
	defs, ok := schemaMap["$defs"]
	if !ok {
		return fmt.Errorf("schema does not contain $defs section")
	}

	defsMap, ok := defs.(map[string]interface{})
	if !ok {
		return fmt.Errorf("$defs is not a map")
	}

	// Collect all type names
	typeNames := []string{"InstallSpec"}
	for typeName := range defsMap {
		typeNames = append(typeNames, typeName)
	}

	// Sort alphabetically
	sort.Strings(typeNames)

	// Output sorted list
	for _, typeName := range typeNames {
		_, err = fmt.Fprintln(writer, typeName)
		if err != nil {
			return err
		}
	}

	return nil
}

// convertSchemaToFormat converts schema to the specified format
func convertSchemaToFormat(schema interface{}, format string, typeFilter string) ([]byte, error) {
	// TypeSpec format doesn't support type filtering
	if format == "typespec" {
		if typeFilter != "" {
			return nil, fmt.Errorf("type filtering not supported for TypeSpec format")
		}
		return convertToTypeSpec()
	}

	// Load schema if not provided
	if schema == nil {
		var err error
		schema, err = loadInstallSpecSchema()
		if err != nil {
			return nil, fmt.Errorf("failed to load schema: %w", err)
		}
	}

	// Apply type filtering if specified
	if typeFilter != "" {
		filteredSchema, err := filterSchemaByType(schema, typeFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to filter schema by type: %w", err)
		}
		schema = filteredSchema
	}

	// Convert to requested format
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
