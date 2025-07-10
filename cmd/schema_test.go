package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestSchemaCommand(t *testing.T) {
	// Test basic command creation
	if SchemaCommand.Use != "schema" {
		t.Errorf("Expected command name 'schema', got %s", SchemaCommand.Use)
	}
}

func TestSchemaCommandListFlag(t *testing.T) {
	// Create a new command instance to avoid flag conflicts
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")

	// Test --list flag
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Reset flags
	schemaListTypes = true
	schemaFormat = "yaml"
	schemaType = ""
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty output for --list flag")
	}
	
	// Check that it contains expected type names
	expectedTypes := []string{"InstallSpec", "AssetConfig", "Platform", "ChecksumConfig"}
	for _, expectedType := range expectedTypes {
		if !bytes.Contains(buf.Bytes(), []byte(expectedType)) {
			t.Errorf("Expected output to contain %s", expectedType)
		}
	}
}

func TestSchemaCommandYamlOutput(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Reset flags
	schemaListTypes = false
	schemaFormat = "yaml"
	schemaType = ""
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty YAML output")
	}
	
	// Verify it's valid YAML
	var result map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Output is not valid YAML: %v", err)
	}
	
	// Check for expected schema properties
	if _, ok := result["$schema"]; !ok {
		t.Error("Expected $schema property in output")
	}
	if _, ok := result["properties"]; !ok {
		t.Error("Expected properties in output")
	}
}

func TestSchemaCommandJsonOutput(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Reset flags
	schemaListTypes = false
	schemaFormat = "json"
	schemaType = ""
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty JSON output")
	}
	
	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
	
	// Check for expected schema properties
	if _, ok := result["$schema"]; !ok {
		t.Error("Expected $schema property in output")
	}
	if _, ok := result["properties"]; !ok {
		t.Error("Expected properties in output")
	}
}

func TestSchemaCommandTypeSpecOutput(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Reset flags
	schemaListTypes = false
	schemaFormat = "typespec"
	schemaType = ""
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty TypeSpec output")
	}
	
	// Check that it contains TypeSpec syntax
	if !bytes.Contains(buf.Bytes(), []byte("import \"@typespec/json-schema\"")) {
		t.Error("Expected TypeSpec import statement")
	}
	if !bytes.Contains(buf.Bytes(), []byte("model InstallSpec")) {
		t.Error("Expected InstallSpec model definition")
	}
}

func TestSchemaCommandSpecificType(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Test specific type output
	schemaListTypes = false
	schemaFormat = "json"
	schemaType = "AssetConfig"
	
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Command execution failed: %v", err)
	}
	
	output := buf.String()
	if len(output) == 0 {
		t.Error("Expected non-empty output for specific type")
	}
	
	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
	
	// Check that it's specifically the AssetConfig type
	if id, ok := result["$id"].(string); !ok || id != "AssetConfig.json" {
		t.Errorf("Expected $id to be 'AssetConfig.json', got %v", result["$id"])
	}
}

func TestSchemaCommandInvalidFormat(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Test invalid format
	schemaListTypes = false
	schemaFormat = "invalid"
	schemaType = ""
	
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid format")
	}
	
	if err.Error() != "invalid format: invalid (supported: yaml, json, typespec)" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestSchemaCommandInvalidType(t *testing.T) {
	// Create a new command instance
	cmd := &cobra.Command{
		Use:   "schema",
		RunE:  runSchema,
	}
	cmd.Flags().StringVar(&schemaFormat, "format", "yaml", "Output format (yaml, json, typespec)")
	cmd.Flags().StringVar(&schemaType, "type", "", "Display specific schema type")
	cmd.Flags().BoolVar(&schemaListTypes, "list", false, "List available schema types")
	
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	
	// Test invalid type
	schemaListTypes = false
	schemaFormat = "json"
	schemaType = "NonExistentType"
	
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error for invalid type")
	}
	
	if !bytes.Contains([]byte(err.Error()), []byte("type 'NonExistentType' not found")) {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestLoadJsonSchema(t *testing.T) {
	// Test loading the schema
	schema, err := loadJsonSchema()
	if err != nil {
		t.Fatalf("Failed to load JSON schema: %v", err)
	}
	
	// Check basic structure
	if _, ok := schema["$schema"]; !ok {
		t.Error("Expected $schema property")
	}
	if _, ok := schema["properties"]; !ok {
		t.Error("Expected properties")
	}
	if _, ok := schema["$defs"]; !ok {
		t.Error("Expected $defs section")
	}
}

func TestExtractSchemaType(t *testing.T) {
	// Load the schema first
	schema, err := loadJsonSchema()
	if err != nil {
		t.Fatalf("Failed to load JSON schema: %v", err)
	}
	
	// Test extracting root type
	rootType, err := extractSchemaType(schema, "InstallSpec")
	if err != nil {
		t.Fatalf("Failed to extract InstallSpec type: %v", err)
	}
	
	if rootType["$id"] != "InstallSpec.json" {
		t.Error("Expected InstallSpec.json as $id")
	}
	
	// Test extracting type from $defs
	assetConfigType, err := extractSchemaType(schema, "AssetConfig")
	if err != nil {
		t.Fatalf("Failed to extract AssetConfig type: %v", err)
	}
	
	if assetConfigType["$id"] != "AssetConfig.json" {
		t.Error("Expected AssetConfig.json as $id")
	}
	
	// Test non-existent type
	_, err = extractSchemaType(schema, "NonExistentType")
	if err == nil {
		t.Error("Expected error for non-existent type")
	}
}

func TestIsValidFormat(t *testing.T) {
	validFormats := []string{"yaml", "json", "typespec"}
	for _, format := range validFormats {
		if !isValidFormat(format) {
			t.Errorf("Expected %s to be valid format", format)
		}
	}
	
	invalidFormats := []string{"xml", "toml", "invalid"}
	for _, format := range invalidFormats {
		if isValidFormat(format) {
			t.Errorf("Expected %s to be invalid format", format)
		}
	}
}

func TestExtractFirstLine(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Single line", "Single line"},
		{"First line\nSecond line", "First line"},
		{"  Trimmed  ", "Trimmed"},
		{"", ""},
	}
	
	for _, test := range tests {
		result := extractFirstLine(test.input)
		if result != test.expected {
			t.Errorf("Expected %q, got %q", test.expected, result)
		}
	}
}

// TestSchemaFilesExist checks that the required schema files exist
func TestSchemaFilesExist(t *testing.T) {
	// Check that InstallSpec.json exists
	if _, err := os.Stat("../schema/output/@typespec/json-schema/InstallSpec.json"); os.IsNotExist(err) {
		t.Error("InstallSpec.json file does not exist")
	}
	
	// Check that main.tsp exists
	if _, err := os.Stat("../schema/main.tsp"); os.IsNotExist(err) {
		t.Error("main.tsp file does not exist")
	}
}