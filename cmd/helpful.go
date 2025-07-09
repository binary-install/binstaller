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

// HelpfulCommand represents the helpful command
var HelpfulCommand = &cobra.Command{
	Use:    "helpful",
	Short:  "Display comprehensive help for all commands",
	Long:   `Displays help information for all binstaller commands in a single, styled output.`,
	Hidden: true, // Hide from normal help output
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get root command
		root := cmd.Root()

		// Process root command and all subcommands
		processCommand(root, "", os.Stdout)

		return nil
	},
}

// processCommand recursively processes a command and its subcommands
func processCommand(cmd *cobra.Command, prefix string, w io.Writer) {
	// Skip certain commands
	if shouldSkipCommand(cmd) {
		return
	}

	// Build command path
	cmdPath := buildCommandPath(cmd, prefix)

	// Write section header (styled)
	if cmdPath != "binst" { // Skip header for root command
		fmt.Fprintln(w)
		fmt.Fprintln(w, headerStyle.Render(fmt.Sprintf("## %s", cmdPath)))
		fmt.Fprintln(w)
	}

	// Use the command's built-in Help() function
	cmd.SetOut(w)
	cmd.Help()

	// Add separator between commands
	fmt.Fprintln(w)
	fmt.Fprintln(w, separatorStyle.Render(strings.Repeat("─", 80)))
	fmt.Fprintln(w)

	// Process subcommands
	for _, subCmd := range cmd.Commands() {
		if !subCmd.Hidden && subCmd.Name() != "help" {
			processCommand(subCmd, cmdPath, w)
		}
	}
}

// shouldSkipCommand determines if a command should be skipped
func shouldSkipCommand(cmd *cobra.Command) bool {
	skipCommands := []string{"completion", "help", "helpful"}
	for _, skip := range skipCommands {
		if cmd.Name() == skip {
			return true
		}
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