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