package webhook

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

// NewPubSubCmd creates the Pub/Sub notification command group.
func newPubSubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pubsub",
		Short: "Manage Pub/Sub notification channels",
		Long: `Manage Nylas Pub/Sub notification channels for queue-based event delivery.

Pub/Sub channels deliver notifications to Google Cloud Pub/Sub topics and are
useful for higher-volume or latency-sensitive event processing.`,
	}

	common.AddOutputFlags(cmd)
	cmd.AddCommand(newPubSubListCmd())
	cmd.AddCommand(newPubSubShowCmd())
	cmd.AddCommand(newPubSubCreateCmd())
	cmd.AddCommand(newPubSubUpdateCmd())
	cmd.AddCommand(newPubSubDeleteCmd())

	return cmd
}
