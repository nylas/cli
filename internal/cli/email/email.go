// Package email provides CLI commands for email operations.
package email

import (
	"github.com/spf13/cobra"
)

// NewEmailCmd creates the email command with all subcommands.
func NewEmailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "email",
		Short: "Manage emails",
		Long:  "Commands for managing emails: list, read, send, search, and more.",
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newSendCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newMarkCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newFoldersCmd())
	cmd.AddCommand(newThreadsCmd())
	cmd.AddCommand(newDraftsCmd())
	cmd.AddCommand(newAttachmentsCmd())
	cmd.AddCommand(newScheduledCmd())
	cmd.AddCommand(newSmartComposeCmd())
	cmd.AddCommand(newTrackingInfoCmd())
	cmd.AddCommand(newMetadataCmd())
	cmd.AddCommand(newAICmd())
	cmd.AddCommand(newTemplatesCmd())
	cmd.AddCommand(newSignaturesCmd())

	return cmd
}
