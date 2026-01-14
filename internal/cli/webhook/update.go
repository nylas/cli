package webhook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		url          string
		description  string
		triggers     []string
		notifyEmails []string
		status       string
		format       string
	)

	cmd := &cobra.Command{
		Use:   "update <webhook-id>",
		Short: "Update a webhook",
		Long: `Update an existing webhook configuration.

You can update the URL, triggers, description, notification emails, or status.`,
		Example: `  # Update webhook URL
  nylas webhook update webhook-abc123 --url https://new.example.com/webhook

  # Update webhook triggers
  nylas webhook update webhook-abc123 --triggers message.created,message.updated

  # Pause a webhook (set status to inactive)
  nylas webhook update webhook-abc123 --status inactive

  # Reactivate a webhook
  nylas webhook update webhook-abc123 --status active

  # Update multiple properties
  nylas webhook update webhook-abc123 \
    --description "Updated webhook" \
    --triggers event.created,event.updated`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookID := args[0]

			// Check if at least one update field is provided
			if url == "" && description == "" && len(triggers) == 0 && len(notifyEmails) == 0 && status == "" {
				return common.NewUserError("No update fields provided",
					"Specify at least one field to update: --url, --description, --triggers, --notify, or --status")
			}

			// Validate status if provided
			if status != "" && status != "active" && status != "inactive" {
				return common.NewUserError("Invalid status value",
					"Status must be 'active' or 'inactive'")
			}

			// Parse and validate trigger types if provided
			var allTriggers []string
			if len(triggers) > 0 {
				for _, t := range triggers {
					parts := strings.Split(t, ",")
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if p != "" {
							allTriggers = append(allTriggers, p)
						}
					}
				}

				validTriggers := domain.AllTriggerTypes()
				for _, t := range allTriggers {
					valid := false
					for _, vt := range validTriggers {
						if t == vt {
							valid = true
							break
						}
					}
					if !valid {
						return common.NewUserError(fmt.Sprintf("Invalid trigger type: %s", t),
							"Run 'nylas webhook triggers' to see available trigger types")
					}
				}
			}

			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Only set fields that were explicitly provided to avoid clearing existing values
			req := &domain.UpdateWebhookRequest{}
			if url != "" {
				req.WebhookURL = url
			}
			if description != "" {
				req.Description = description
			}
			if len(allTriggers) > 0 {
				req.TriggerTypes = allTriggers
			}
			if len(notifyEmails) > 0 {
				req.NotificationEmailAddresses = notifyEmails
			}
			if status != "" {
				req.Status = status
			}

			webhook, err := common.RunWithSpinnerResult("Updating webhook...", func() (*domain.Webhook, error) {
				return c.UpdateWebhook(ctx, webhookID, req)
			})
			if err != nil {
				return common.NewUserError("Failed to update webhook: "+err.Error(),
					"Check that the webhook ID is correct. Run 'nylas webhook list' to see available webhooks")
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(webhook)
			}

			fmt.Printf("%s Webhook updated successfully!\n", common.Green.Sprint("✓"))
			fmt.Println()
			fmt.Printf("  ID:     %s\n", webhook.ID)
			fmt.Printf("  URL:    %s\n", webhook.WebhookURL)
			fmt.Printf("  Status: %s %s\n", getStatusIcon(webhook.Status), webhook.Status)

			if len(webhook.TriggerTypes) > 0 {
				fmt.Println("\nTriggers:")
				for _, t := range webhook.TriggerTypes {
					fmt.Printf("  • %s\n", t)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "", "New webhook URL")
	cmd.Flags().StringVarP(&description, "description", "d", "", "New description")
	cmd.Flags().StringSliceVarP(&triggers, "triggers", "t", nil, "New trigger types (comma-separated or multiple flags)")
	cmd.Flags().StringSliceVarP(&notifyEmails, "notify", "n", nil, "New notification email addresses")
	cmd.Flags().StringVarP(&status, "status", "s", "", "New status (active or inactive)")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	return cmd
}
