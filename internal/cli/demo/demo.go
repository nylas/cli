// Package demo provides CLI commands for demo mode with sample data.
// Demo mode allows users to explore CLI features without requiring credentials.
package demo

import (
	"github.com/spf13/cobra"
)

// NewDemoCmd creates the demo parent command with all demo subcommands.
func NewDemoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "demo",
		Short: "Explore CLI features with sample data (no credentials required)",
		Args:  cobra.NoArgs,
		Long: `Demo mode lets you try out the Nylas CLI without any account or credentials.

All commands use realistic sample data so you can explore:
  - Email management (list, send simulation)
  - Calendar events (list, create simulation)
  - Contacts management
  - Scheduling capabilities
  - AI notetaker features
  - Interactive TUI

This is perfect for:
  - Evaluating the CLI before signing up
  - Learning how commands work
  - Taking screenshots for documentation
  - Testing integrations with mock data

To connect your real email account, run: nylas auth login`,
		Example: `  # Explore the interactive TUI with sample data
  nylas demo tui

  # List sample emails
  nylas demo email list

  # List sample calendar events
  nylas demo calendar list

  # List sample contacts
  nylas demo contacts list

  # Try the scheduler
  nylas demo scheduler list

  # Try the notetaker
  nylas demo notetaker list`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Add demo subcommands
	cmd.AddCommand(newDemoTUICmd())
	cmd.AddCommand(newDemoEmailCmd())
	cmd.AddCommand(newDemoCalendarCmd())
	cmd.AddCommand(newDemoContactsCmd())
	cmd.AddCommand(newDemoSchedulerCmd())
	cmd.AddCommand(newDemoNotetakerCmd())

	return cmd
}
