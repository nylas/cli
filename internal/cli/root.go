// Package cli provides the command-line interface.
package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "nylas",
	Short:   "Nylas CLI - Email, Calendar, and API management",
	Version: Version,
	Long: `nylas is a command-line tool for managing emails, calendars, contacts,
and Nylas API authentication.

AUTHENTICATION:
  nylas auth login     Authenticate with an email provider
  nylas auth logout    Logout from current account
  nylas auth status    Check authentication status
  nylas auth list      List all authenticated accounts
  nylas auth switch    Switch between accounts
  nylas auth add       Manually add an existing grant
  nylas auth whoami    Show current user info

EMAIL MANAGEMENT:
  nylas email list           List recent emails
  nylas email read <id>      Read a specific email
  nylas email send           Send an email
  nylas email search <query> Search emails
  nylas email folders list   List folders
  nylas email threads list   List email threads
  nylas email drafts list    List drafts

CALENDAR MANAGEMENT:
  nylas calendar list              List calendars
  nylas calendar events list       List upcoming events
  nylas calendar events show       Show event details
  nylas calendar events create     Create a new event
  nylas calendar events delete     Delete an event
  nylas calendar availability check    Check free/busy status
  nylas calendar availability find     Find available meeting times

CONTACTS MANAGEMENT:
  nylas contacts list            List contacts
  nylas contacts show <id>       Show contact details
  nylas contacts create          Create a new contact
  nylas contacts delete <id>     Delete a contact
  nylas contacts groups          List contact groups

WEBHOOK MANAGEMENT:
  nylas webhook list             List all webhooks
  nylas webhook show <id>        Show webhook details
  nylas webhook create           Create a new webhook
  nylas webhook update <id>      Update a webhook
  nylas webhook delete <id>      Delete a webhook
  nylas webhook triggers         List available trigger types
  nylas webhook test send        Send a test event
  nylas webhook test payload     Get mock payload for trigger

`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Custom config file path")

	rootCmd.AddCommand(newVersionCmd())
}

// GetRootCmd returns the root command for adding subcommands.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}
