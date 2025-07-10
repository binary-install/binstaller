package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// We'll load files from the filesystem instead of embedding
// since the embed path is complex

// SchemaCommand represents the schema command
var SchemaCommand = &cobra.Command{
	Use:   "schema",
	Short: "Display binstaller configuration schema",
	Long: `Display binstaller configuration schema in various formats.

The schema command allows you to view the complete configuration schema
for binstaller, including all available types and their documentation.
This is useful for understanding the configuration format and for
LLMs to generate accurate configuration files.

By default, the command outputs the full schema in YAML format.`,
	RunE: runSchema,
}

var (
	schemaFormat     string
	schemaType       string
	schemaListTypes  bool
)

func init() {
	SchemaCommand.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	SchemaCommand.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	SchemaCommand.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
}

func runSchema(cmd *cobra.Command, args []string) error {
	// Handle --list flag
	if schemaListTypes {
		return listSchemaTypes(cmd.OutOrStdout())
	}

	// Validate format
	if !isValidFormat(schemaFormat) {
		return fmt.Errorf("invalid format: %s (supported: yaml, json, typespec)", schemaFormat)
	}

	// Handle TypeSpec format
	if schemaFormat == "typespec" {
		return outputTypeSpecSchema(cmd.OutOrStdout())
	}

	// Load and process JSON schema
	schemaData, err := loadJsonSchema()
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Handle specific type
	if schemaType != "" {
		typeSchema, err := extractSchemaType(schemaData, schemaType)
		if err != nil {
			return fmt.Errorf("failed to extract type '%s': %w", schemaType, err)
		}
		schemaData = typeSchema
	}

	// Output in requested format
	switch schemaFormat {
	case "yaml":
		return outputYamlSchema(schemaData, cmd.OutOrStdout())
	case "json":
		return outputJsonSchema(schemaData, cmd.OutOrStdout())
	default:
		return fmt.Errorf("unsupported format: %s", schemaFormat)
	}
}

func isValidFormat(format string) bool {
	switch format {
	case "yaml", "json", "typespec":
		return true
	default:
		return false
	}
}

func loadJsonSchema() (map[string]interface{}, error) {
	// Try to find the schema file
	schemaPath := findSchemaFile("InstallSpec.json")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(content, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	return schema, nil
}

func findSchemaFile(filename string) string {
	// Common possible locations for schema files
	candidates := []string{
		filepath.Join("schema", "output", "@typespec", "json-schema", filename),
		filepath.Join("..", "schema", "output", "@typespec", "json-schema", filename),
		filepath.Join("..", "..", "schema", "output", "@typespec", "json-schema", filename),
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	
	// Default to the first candidate if none found
	return candidates[0]
}

func findTypeSpecFile() string {
	// Common possible locations for TypeSpec files
	candidates := []string{
		filepath.Join("schema", "main.tsp"),
		filepath.Join("..", "schema", "main.tsp"),
		filepath.Join("..", "..", "schema", "main.tsp"),
	}
	
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	
	// Default to the first candidate if none found
	return candidates[0]
}

func extractSchemaType(schema map[string]interface{}, typeName string) (map[string]interface{}, error) {
	// Handle root InstallSpec type
	if typeName == "InstallSpec" {
		return schema, nil
	}

	// Extract from $defs
	defs, ok := schema["$defs"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no $defs section found in schema")
	}

	typeSchema, ok := defs[typeName].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("type '%s' not found in schema", typeName)
	}

	// Create a standalone schema for the type
	result := map[string]interface{}{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$id":     fmt.Sprintf("%s.json", typeName),
	}

	// Copy the type definition to the root
	for k, v := range typeSchema {
		result[k] = v
	}

	return result, nil
}

func outputYamlSchema(schema map[string]interface{}, writer io.Writer) error {
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	defer encoder.Close()

	return encoder.Encode(schema)
}

func outputJsonSchema(schema map[string]interface{}, writer io.Writer) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(schema)
}

func outputTypeSpecSchema(writer io.Writer) error {
	// Find and read the TypeSpec file
	schemaPath := findTypeSpecFile()
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read TypeSpec file: %w", err)
	}

	_, err = fmt.Fprint(writer, string(content))
	return err
}

func listSchemaTypes(writer io.Writer) error {
	schema, err := loadJsonSchema()
	if err != nil {
		return fmt.Errorf("failed to load schema: %w", err)
	}

	// Get types from $defs
	defs, ok := schema["$defs"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("no $defs section found in schema")
	}

	// Collect type names and descriptions
	types := make([]typeInfo, 0, len(defs)+1)
	
	// Add root InstallSpec type
	if desc, ok := schema["description"].(string); ok {
		types = append(types, typeInfo{
			Name:        "InstallSpec",
			Description: extractFirstLine(desc),
		})
	}

	// Add types from $defs
	for name, def := range defs {
		if defMap, ok := def.(map[string]interface{}); ok {
			desc := ""
			if description, ok := defMap["description"].(string); ok {
				desc = extractFirstLine(description)
			}
			types = append(types, typeInfo{
				Name:        name,
				Description: desc,
			})
		}
	}

	// Sort alphabetically
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name < types[j].Name
	})

	// Output the list
	for _, t := range types {
		if t.Description != "" {
			fmt.Fprintf(writer, "%-20s - %s\n", t.Name, t.Description)
		} else {
			fmt.Fprintf(writer, "%-20s\n", t.Name)
		}
	}

	return nil
}

type typeInfo struct {
	Name        string
	Description string
}

func extractFirstLine(text string) string {
	lines := strings.Split(text, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}