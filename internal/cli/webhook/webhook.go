// Package webhook provides webhook management CLI commands.
package webhook

import (
	"github.com/spf13/cobra"
)

// NewWebhookCmd creates the webhook command group.
func NewWebhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"webhooks", "wh"},
		Short:   "Manage notification destinations",
		Long: `Manage Nylas notification destinations, including webhooks and Pub/Sub channels.

Use webhooks for direct HTTPS push delivery, or Pub/Sub channels for
high-volume queue-based notification delivery.

Note: Notification destination management requires an API key (admin-level access).`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newRotateSecretCmd())
	cmd.AddCommand(newVerifyCmd())
	cmd.AddCommand(newPubSubCmd())
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newTriggersCmd())
	cmd.AddCommand(newServerCmd())

	return cmd
}
