package webhook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var (
		url          string
		description  string
		triggers     []string
		notifyEmails []string
		format       string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		Long: `Create a new webhook to receive event notifications.

You must specify a webhook URL and at least one trigger type.
Use 'nylas webhook triggers' to see available trigger types.`,
		Example: `  # Create a webhook for new messages
  nylas webhook create --url https://example.com/webhook --triggers message.created

  # Create a webhook for multiple events
  nylas webhook create --url https://example.com/webhook \
    --triggers message.created,event.created,contact.created

  # Create a webhook with description and notification email
  nylas webhook create --url https://example.com/webhook \
    --triggers message.created \
    --description "My message webhook" \
    --notify admin@example.com

  # Create a webhook for all message events
  nylas webhook create --url https://example.com/webhook \
    --triggers message.created --triggers message.updated`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if url == "" {
				return common.NewUserError("Webhook URL is required",
					"Use --url to specify the webhook endpoint")
			}

			if len(triggers) == 0 {
				return common.NewUserError("At least one trigger type is required",
					"Use --triggers to specify trigger types. Run 'nylas webhook triggers' to see available types")
			}

			// Parse comma-separated triggers
			var allTriggers []string
			for _, t := range triggers {
				parts := strings.Split(t, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if p != "" {
						allTriggers = append(allTriggers, p)
					}
				}
			}

			// Validate trigger types
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

			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.CreateWebhookRequest{
				WebhookURL:                 url,
				TriggerTypes:               allTriggers,
				Description:                description,
				NotificationEmailAddresses: notifyEmails,
			}

			webhook, err := common.RunWithSpinnerResult("Creating webhook...", func() (*domain.Webhook, error) {
				return c.CreateWebhook(ctx, req)
			})
			if err != nil {
				return common.NewUserError("Failed to create webhook: "+err.Error(),
					"Check that the webhook URL is accessible and you have permission to create webhooks")
			}

			if format == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(webhook)
			}

			fmt.Printf("%s Webhook created successfully!\n", common.Green.Sprint("✓"))
			fmt.Println()
			fmt.Printf("  ID:     %s\n", webhook.ID)
			fmt.Printf("  URL:    %s\n", webhook.WebhookURL)
			fmt.Printf("  Status: %s\n", webhook.Status)
			fmt.Println()
			fmt.Println("Triggers:")
			for _, t := range webhook.TriggerTypes {
				fmt.Printf("  • %s\n", t)
			}

			if webhook.WebhookSecret != "" {
				fmt.Println()
				fmt.Printf("%s Save your webhook secret - it won't be shown again:\n", common.Yellow.Sprint("Important:"))
				fmt.Printf("  Secret: %s\n", webhook.WebhookSecret)
				fmt.Println()
				fmt.Println("Use this secret to verify webhook signatures.")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&url, "url", "u", "", "Webhook URL (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Webhook description")
	cmd.Flags().StringSliceVarP(&triggers, "triggers", "t", nil, "Trigger types (required, comma-separated or multiple flags)")
	cmd.Flags().StringSliceVarP(&notifyEmails, "notify", "n", nil, "Notification email addresses")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("triggers")

	return cmd
}
