package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGenCommandFlags(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantType    string
		wantOutput  string
		wantVersion string
		wantError   bool
	}{
		{
			name:        "default type is installer",
			args:        []string{},
			wantType:    "installer",
			wantOutput:  "-",
			wantVersion: "",
			wantError:   false,
		},
		{
			name:        "explicit installer type",
			args:        []string{"--type", "installer"},
			wantType:    "installer",
			wantOutput:  "-",
			wantVersion: "",
			wantError:   false,
		},
		{
			name:        "runner type",
			args:        []string{"--type", "runner"},
			wantType:    "runner",
			wantOutput:  "-",
			wantVersion: "",
			wantError:   false,
		},
		{
			name:        "runner type with output file",
			args:        []string{"--type", "runner", "-o", "run.sh"},
			wantType:    "runner",
			wantOutput:  "run.sh",
			wantVersion: "",
			wantError:   false,
		},
		{
			name:        "runner type with target version",
			args:        []string{"--type", "runner", "--target-version", "v1.2.3"},
			wantType:    "runner",
			wantOutput:  "-",
			wantVersion: "v1.2.3",
			wantError:   false,
		},
		{
			name:        "invalid type accepted by cobra but would fail validation",
			args:        []string{"--type", "invalid"},
			wantType:    "invalid",
			wantOutput:  "-",
			wantVersion: "",
			wantError:   false, // Cobra accepts any string, validation happens in RunE
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags before each test
			genOutputFile = ""
			genTargetVersion = ""
			genScriptType = ""

			cmd := &cobra.Command{Use: "gen"}
			cmd.Flags().StringVarP(&genOutputFile, "output", "o", "-", "Output path for the generated script")
			cmd.Flags().StringVar(&genTargetVersion, "target-version", "", "Generate script for specific version only")
			cmd.Flags().StringVar(&genScriptType, "type", "installer", "Type of script to generate (installer, runner)")

			err := cmd.ParseFlags(tt.args)
			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for args %v, but got none", tt.args)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error parsing flags %v: %v", tt.args, err)
			}

			if genScriptType != tt.wantType {
				t.Errorf("genScriptType = %v, want %v", genScriptType, tt.wantType)
			}

			if genOutputFile != tt.wantOutput {
				t.Errorf("genOutputFile = %v, want %v", genOutputFile, tt.wantOutput)
			}

			if genTargetVersion != tt.wantVersion {
				t.Errorf("genTargetVersion = %v, want %v", genTargetVersion, tt.wantVersion)
			}
		})
	}
}

func TestGenCommandValidateScriptType(t *testing.T) {
	tests := []struct {
		name       string
		scriptType string
		wantError  bool
	}{
		{
			name:       "installer type is valid",
			scriptType: "installer",
			wantError:  false,
		},
		{
			name:       "runner type is valid",
			scriptType: "runner",
			wantError:  false,
		},
		{
			name:       "empty type defaults to installer",
			scriptType: "",
			wantError:  false,
		},
		{
			name:       "invalid type",
			scriptType: "invalid",
			wantError:  true,
		},
		{
			name:       "case sensitive validation",
			scriptType: "INSTALLER",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateScriptType(tt.scriptType)
			if tt.wantError && err == nil {
				t.Errorf("validateScriptType(%v) expected error, but got none", tt.scriptType)
			}
			if !tt.wantError && err != nil {
				t.Errorf("validateScriptType(%v) unexpected error: %v", tt.scriptType, err)
			}
		})
	}
}

func TestGenCommandUsageExamples(t *testing.T) {
	// Test that command usage includes runner examples
	examples := GenCommand.Example

	// Should include runner examples in command examples
	expectedExamples := []string{
		"--type=runner",
		"run.sh",
	}

	for _, example := range expectedExamples {
		if !strings.Contains(examples, example) {
			t.Errorf("Command examples missing expected content: %q", example)
		}
	}
}
