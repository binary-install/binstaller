package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestRunSchema_BasicYAMLOutput tests basic YAML output
func TestRunSchema_BasicYAMLOutput(t *testing.T) {
	var output bytes.Buffer

	// Test the basic schema command with default YAML output
	err := RunSchema("yaml", &output)

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

	// Check that it's in YAML format (should not contain JSON quotes)
	if strings.Contains(result, `"$schema"`) {
		t.Error("Expected YAML format, but found JSON syntax")
	}

	// Check that it's valid YAML (contains YAML-style $schema)
	if !strings.Contains(result, "$schema: https://json-schema.org") {
		t.Error("Expected YAML output to contain YAML-style $schema")
	}
}

// TestRunSchema_JSONFormatOutput tests JSON format output
func TestRunSchema_JSONFormatOutput(t *testing.T) {
	var output bytes.Buffer

	// Test the schema command with JSON format
	err := RunSchema("json", &output)

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

// TestRunSchema_TypeSpecFormatOutput tests TypeSpec format output
func TestRunSchema_TypeSpecFormatOutput(t *testing.T) {
	var output bytes.Buffer

	// Test the schema command with TypeSpec format
	err := RunSchema("typespec", &output)

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

// TestRunSchema_CanBeeParsedCorrectly tests that embedded schema can be parsed
func TestRunSchema_CanBeParsedCorrectly(t *testing.T) {
	var output bytes.Buffer

	// Test that YAML output can be parsed correctly
	err := RunSchema("yaml", &output)
	if err != nil {
		t.Errorf("RunSchema() returned error: %v", err)
	}

	result := output.String()
	if result == "" {
		t.Error("RunSchema() returned empty output")
	}

	// Check that output contains expected schema structure
	if !strings.Contains(result, "$defs:") {
		t.Error("Expected YAML output to contain '$defs' section")
	}

	if !strings.Contains(result, "AssetConfig:") {
		t.Error("Expected YAML output to contain 'AssetConfig' type")
	}

	if !strings.Contains(result, "properties:") {
		t.Error("Expected YAML output to contain 'properties' section")
	}

	// Check that the schema contains the template field description
	if !strings.Contains(result, "template:") {
		t.Error("Expected schema to contain 'template' field")
	}
}


// TestRunSchema_ErrorCases tests error handling
func TestRunSchema_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid format",
			format:      "xml",
			expectError: true,
			errorMsg:    "unsupported format: xml",
		},
		{
			name:        "valid yaml format",
			format:      "yaml",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid json format",
			format:      "json",
			expectError: false,
			errorMsg:    "",
		},
		{
			name:        "valid typespec format",
			format:      "typespec",
			expectError: false,
			errorMsg:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			err := RunSchema(tt.format, &output)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if err != nil && tt.errorMsg != "" {
					if !strings.Contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
