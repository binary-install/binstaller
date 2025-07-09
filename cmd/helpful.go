package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Style definitions
var (
	// Color profile detection
	profile = colorprofile.Detect(os.Stdout, os.Environ())

	// Styles with adaptive colors based on terminal capabilities
	headerStyle = func() lipgloss.Style {
		if profile == colorprofile.TrueColor || profile == colorprofile.ANSI256 {
			return lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("212"))
		}
		return lipgloss.NewStyle().Bold(true)
	}()

	separatorStyle = func() lipgloss.Style {
		if profile == colorprofile.TrueColor || profile == colorprofile.ANSI256 {
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))
		}
		return lipgloss.NewStyle().Faint(true)
	}()
)

// HelpfulConfig configures the helpful command behavior
type HelpfulConfig struct {
	// SkipFunc determines if a command should be skipped
	SkipFunc func(cmd *cobra.Command) bool
	// Output writer
	Output io.Writer
}

// HelpfulCommand represents the helpful command
var HelpfulCommand = &cobra.Command{
	Use:   "helpful",
	Short: "Display comprehensive help for all commands",
	Long: `Displays help information for all binstaller commands in a single, styled output.
This is especially useful for getting a complete overview of the tool's capabilities,
including for LLMs or automated documentation tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &HelpfulConfig{
			SkipFunc: func(c *cobra.Command) bool {
				// Skip the helpful command itself
				if c == cmd {
					return true
				}
				return defaultSkipFunc(c)
			},
			Output: os.Stdout,
		}
		return RunHelpful(cmd, config)
	},
}

// RunHelpful executes the helpful command with the given configuration
func RunHelpful(cmd *cobra.Command, config *HelpfulConfig) error {
	// Get root command
	root := cmd.Root()

	// Process root command and all subcommands
	processCommandWithConfig(root, "", config)

	return nil
}

// processCommandWithConfig recursively processes a command and its subcommands with config
func processCommandWithConfig(cmd *cobra.Command, prefix string, config *HelpfulConfig) {
	// Skip certain commands
	if config.SkipFunc != nil && config.SkipFunc(cmd) {
		return
	}

	// Build command path
	cmdPath := buildCommandPath(cmd, prefix)

	// Write section header (styled)
	if cmdPath != cmd.Root().Name() { // Skip header for root command
		fmt.Fprintln(config.Output)
		fmt.Fprintln(config.Output, headerStyle.Render(fmt.Sprintf("## %s", cmdPath)))
		fmt.Fprintln(config.Output)
	}

	// Use the command's built-in Help() function
	cmd.SetOut(config.Output)
	cmd.Help()

	// Add separator between commands
	fmt.Fprintln(config.Output)
	fmt.Fprintln(config.Output, separatorStyle.Render(strings.Repeat("─", 80)))
	fmt.Fprintln(config.Output)

	// Process subcommands
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden {
			processCommandWithConfig(subCmd, cmdPath, config)
		}
	}
}

// defaultSkipFunc is the default skip function that skips standard utility commands
func defaultSkipFunc(cmd *cobra.Command) bool {
	// Skip commands that are typically not useful in comprehensive help
	switch cmd.Name() {
	case "completion", "help":
		return true
	}
	
	return false
}

// buildCommandPath builds the full command path
func buildCommandPath(cmd *cobra.Command, prefix string) string {
	if prefix == "" {
		return cmd.Name()
	}
	return fmt.Sprintf("%s %s", prefix, cmd.Name())
}

func init() {
	// The helpful command is registered in root.go
}