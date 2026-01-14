// Package inbound provides CLI commands for Nylas Inbound email functionality.
// Nylas Inbound allows applications to receive emails at managed addresses
// without building OAuth flows or connecting to third-party mailboxes.
package inbound

import (
	"github.com/spf13/cobra"
)

// NewInboundCmd creates the inbound command group.
func NewInboundCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "inbound",
		Aliases: []string{"inbox"},
		Short:   "Manage inbound email inboxes",
		Long: `Manage Nylas Inbound email inboxes.

Nylas Inbound enables your application to receive emails at dedicated managed
addresses (e.g., support@yourapp.nylas.email) and process them via webhooks.

Use cases:
  - Capturing messages sent to specific addresses (intake@, leads@, tickets@)
  - Triggering automated workflows from incoming mail
  - Real-time message delivery to workers, LLMs, or downstream systems

Examples:
  # List all inbound inboxes
  nylas inbound list

  # Create a new inbound inbox
  nylas inbound create support

  # View messages for an inbound inbox
  nylas inbound messages <inbox-id>

  # Monitor for new inbound messages in real-time
  nylas inbound monitor <inbox-id>`,
	}

	// Add subcommands
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newMessagesCmd())
	cmd.AddCommand(newMonitorCmd())

	return cmd
}
