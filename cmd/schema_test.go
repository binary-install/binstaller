package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TDD RED Phase: Write failing test for basic schema command
func TestRunSchema_BasicYAMLOutput(t *testing.T) {
	var output bytes.Buffer
	
	// Test the basic schema command with default YAML output
	err := RunSchema("yaml", "", false, &output)
	
	if err != nil {
		t.Errorf("RunSchema() returned error: %v", err)
	}
	
	result := output.String()
	if result == "" {
		t.Error("RunSchema() returned empty output")
	}
	
	// Check that output contains YAML schema content
	if !strings.Contains(result, "InstallSpec") {
		t.Error("Expected YAML output to contain 'InstallSpec' schema")
	}
	
	// Check that it's in YAML format (should not contain JSON brackets)
	if strings.Contains(result, `"$schema"`) {
		t.Error("Expected YAML format, but found JSON syntax")
	}
}

// TDD RED Phase: Write failing test for JSON format option
func TestRunSchema_JSONFormatOutput(t *testing.T) {
	var output bytes.Buffer
	
	// Test the schema command with JSON format
	err := RunSchema("json", "", false, &output)
	
	if err != nil {
		t.Errorf("RunSchema() returned error: %v", err)
	}
	
	result := output.String()
	if result == "" {
		t.Error("RunSchema() returned empty output")
	}
	
	// Check that output contains JSON schema content
	if !strings.Contains(result, "InstallSpec") {
		t.Error("Expected JSON output to contain 'InstallSpec' schema")
	}
	
	// Check that it's in JSON format (should contain JSON syntax)
	if !strings.Contains(result, `"$schema"`) {
		t.Error("Expected JSON format, but found non-JSON syntax")
	}
	
	// Check that it's valid JSON by attempting to parse
	if !strings.HasPrefix(result, "{") || !strings.HasSuffix(strings.TrimSpace(result), "}") {
		t.Error("Expected JSON output to be a valid JSON object")
	}
}

// TDD RED Phase: Write failing test for TypeSpec format option
func TestRunSchema_TypeSpecFormatOutput(t *testing.T) {
	var output bytes.Buffer
	
	// Test the schema command with TypeSpec format
	err := RunSchema("typespec", "", false, &output)
	
	if err != nil {
		t.Errorf("RunSchema() returned error: %v", err)
	}
	
	result := output.String()
	if result == "" {
		t.Error("RunSchema() returned empty output")
	}
	
	// Check that output contains TypeSpec content
	if !strings.Contains(result, "@doc") {
		t.Error("Expected TypeSpec output to contain '@doc' annotation")
	}
	
	// Check that it contains TypeSpec import
	if !strings.Contains(result, "import \"@typespec/json-schema\"") {
		t.Error("Expected TypeSpec output to contain import statement")
	}
	
	// Check that it's TypeSpec syntax, not JSON
	if strings.Contains(result, `"$schema"`) {
		t.Error("Expected TypeSpec format, but found JSON syntax")
	}
}

// TDD RED Phase: Write failing test for --type option
func TestRunSchema_TypeFilterOutput(t *testing.T) {
	var output bytes.Buffer
	
	// Test the schema command with specific type filtering
	err := RunSchema("yaml", "AssetConfig", false, &output)
	
	if err != nil {
		t.Errorf("RunSchema() returned error: %v", err)
	}
	
	result := output.String()
	if result == "" {
		t.Error("RunSchema() returned empty output")
	}
	
	// Check that output contains the specific type description
	if !strings.Contains(result, "Configuration for constructing download URLs") {
		t.Errorf("Expected output to contain AssetConfig description, got: %s", result)
	}
	
	// Check that it doesn't contain the full schema root
	if strings.Contains(result, "$schema") {
		t.Error("Expected filtered output, but found full schema")
	}
	
	// Check that it contains template field specific to AssetConfig
	if !strings.Contains(result, "template") {
		t.Error("Expected AssetConfig to contain 'template' field")
	}
}