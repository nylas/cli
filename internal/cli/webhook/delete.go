package webhook

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <webhook-id>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a webhook",
		Long: `Delete a webhook by ID.

This permanently removes the webhook and stops all event notifications.`,
		Example: `  # Delete a webhook (with confirmation)
  nylas webhook delete webhook-abc123

  # Delete without confirmation
  nylas webhook delete webhook-abc123 --force`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookID := args[0]

			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			// Get webhook details first for confirmation
			ctx, cancel := common.CreateContext()
			webhook, err := c.GetWebhook(ctx, webhookID)
			cancel()

			if err != nil {
				return common.NewUserError("Failed to find webhook: "+err.Error(),
					"Check that the webhook ID is correct. Run 'nylas webhook list' to see available webhooks")
			}

			// Confirm deletion unless --force is used
			if !force {
				fmt.Printf("Webhook to delete:\n")
				fmt.Printf("  ID:  %s\n", webhook.ID)
				fmt.Printf("  URL: %s\n", webhook.WebhookURL)
				if webhook.Description != "" {
					fmt.Printf("  Description: %s\n", webhook.Description)
				}
				fmt.Printf("  Triggers: %v\n", webhook.TriggerTypes)
				fmt.Println()

				fmt.Print("Are you sure you want to delete this webhook? [y/N] ")
				var confirm string
				_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"

				if confirm != "y" && confirm != "Y" && confirm != "yes" && confirm != "Yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			ctx, cancel = common.CreateContext()
			defer cancel()

			err = common.RunWithSpinner("Deleting webhook...", func() error {
				return c.DeleteWebhook(ctx, webhookID)
			})
			if err != nil {
				return common.NewUserError("Failed to delete webhook: "+err.Error(),
					"Check your permissions. The webhook may have already been deleted")
			}

			fmt.Printf("%s Webhook deleted successfully!\n", common.Green.Sprint("✓"))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Delete without confirmation")

	return cmd
}
