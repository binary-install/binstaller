package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenCommandMultiBinaryRunner(t *testing.T) {
	// Simple test for multi-binary runner generation
	tests := []struct {
		name           string
		yamlContent    string
		binaryFlag     string
		expectError    bool
		expectBinaries []string
	}{
		{
			name: "runner with multiple binaries - no flag",
			yamlContent: `
schema: v1
name: multi-tool
repo: example/multi-tool
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  binaries:
    - name: tool1
      path: bin/tool1
    - name: tool2
      path: bin/tool2
`,
			binaryFlag:     "",
			expectError:    false,
			expectBinaries: []string{"tool1"}, // Only first binary
		},
		{
			name: "runner with multiple binaries - with flag",
			yamlContent: `
schema: v1
name: multi-tool
repo: example/multi-tool
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  binaries:
    - name: tool1
      path: bin/tool1
    - name: tool2
      path: bin/tool2
`,
			binaryFlag:     "tool2",
			expectError:    false,
			expectBinaries: []string{"tool2"}, // Only selected binary
		},
		{
			name: "runner with invalid binary name",
			yamlContent: `
schema: v1
name: multi-tool
repo: example/multi-tool
asset:
  template: "${NAME}_${VERSION}_${OS}_${ARCH}.tar.gz"
  binaries:
    - name: tool1
      path: bin/tool1
    - name: tool2
      path: bin/tool2
`,
			binaryFlag:  "nonexistent",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir := t.TempDir()
			specFile := filepath.Join(tmpDir, "test.yml")
			outputFile := filepath.Join(tmpDir, "output.sh")

			// Write spec file
			if err := os.WriteFile(specFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("Failed to write spec file: %v", err)
			}

			// Set flags
			configFile = specFile
			genScriptType = "runner"
			genBinaryName = tt.binaryFlag
			genOutputFile = outputFile
			genTargetVersion = ""

			// Run command
			err := GenCommand.RunE(GenCommand, []string{})

			// Check error
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Read generated script
			scriptContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read generated script: %v", err)
			}

			// Check expected binaries
			scriptStr := string(scriptContent)
			for _, binary := range tt.expectBinaries {
				if !strings.Contains(scriptStr, "BINARY_NAME='"+binary+"'") {
					t.Errorf("Expected binary %q not found in script", binary)
				}
			}
		})
	}
}
