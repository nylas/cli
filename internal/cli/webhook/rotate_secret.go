package webhook

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newRotateSecretCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "rotate-secret <webhook-id>",
		Short: "Rotate a webhook secret",
		Long: `Rotate a webhook secret and print the new value.

Rotating a secret changes the signing key used for future webhook deliveries, so
you should update your receiving service before reactivating traffic.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError(
					"secret rotation requires confirmation",
					"Re-run with --yes to rotate the webhook secret",
				)
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				rotated, err := client.RotateWebhookSecret(ctx, args[0])
				if err != nil {
					return struct{}{}, common.WrapUpdateError("webhook secret", err)
				}

				fmt.Printf("%s Webhook secret rotated successfully.\n", common.Green.Sprint("✓"))
				fmt.Println()
				fmt.Printf("  Webhook ID: %s\n", rotated.ID)
				fmt.Printf("  Secret:     %s\n", rotated.WebhookSecret)
				fmt.Println()
				fmt.Println("Update your webhook receiver to use this new secret before processing future events.")
				return struct{}{}, nil
			})
			return err
		},
	}

	common.AddYesFlag(cmd, &yes)
	return cmd
}
