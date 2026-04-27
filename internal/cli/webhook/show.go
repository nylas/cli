package webhook

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <webhook-id>",
		Short: "Show webhook details",
		Long: `Show detailed information about a specific webhook.

Displays the webhook configuration including URL, trigger types,
status, and notification settings.`,
		Example: `  # Show webhook details
  nylas webhook show webhook-abc123

  # Show in JSON format
  nylas webhook show webhook-abc123 --format json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookID := args[0]

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				webhook, err := common.RunWithSpinnerResult("Fetching webhook...", func() (*domain.Webhook, error) {
					return client.GetWebhook(ctx, webhookID)
				})
				if err != nil {
					return struct{}{}, common.WrapGetError("webhook", err)
				}

				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(webhook)
				}
				return struct{}{}, displayWebhookDetails(webhook)
			})
			return err
		},
	}

	return cmd
}

func displayWebhookDetails(webhook *domain.Webhook) error {
	statusIcon := common.StatusIcon(webhook.Status)

	fmt.Printf("Webhook: %s\n", webhook.ID)
	fmt.Println(strings.Repeat("─", 60))

	if webhook.Description != "" {
		fmt.Printf("Description:  %s\n", webhook.Description)
	}
	fmt.Printf("URL:          %s\n", webhook.WebhookURL)
	fmt.Printf("Status:       %s %s\n", statusIcon, webhook.Status)

	if webhook.WebhookSecret != "" {
		fmt.Printf("Secret:       %s\n", maskSecret(webhook.WebhookSecret))
	}

	// Trigger types
	fmt.Println("\nTrigger Types:")
	for _, trigger := range webhook.TriggerTypes {
		fmt.Printf("  • %s\n", trigger)
	}

	// Notification emails
	if len(webhook.NotificationEmailAddresses) > 0 {
		fmt.Println("\nNotification Emails:")
		for _, email := range webhook.NotificationEmailAddresses {
			fmt.Printf("  • %s\n", email)
		}
	}

	// Timestamps
	fmt.Println("\nTimestamps:")
	if !webhook.CreatedAt.IsZero() {
		fmt.Printf("  Created:        %s\n", webhook.CreatedAt.Format(time.RFC3339))
	}
	if !webhook.UpdatedAt.IsZero() {
		fmt.Printf("  Updated:        %s\n", webhook.UpdatedAt.Format(time.RFC3339))
	}
	if !webhook.StatusUpdatedAt.IsZero() {
		fmt.Printf("  Status Updated: %s\n", webhook.StatusUpdatedAt.Format(time.RFC3339))
	}

	return nil
}

// maskSecret masks a secret for display, showing first 4 and last 4 characters.
// Handles edge cases for short secrets to prevent panics.
func maskSecret(secret string) string {
	switch {
	case len(secret) <= 8:
		// Too short to show any characters safely
		return strings.Repeat("*", len(secret))
	case len(secret) <= 12:
		// Show first 2 and last 2 only
		return secret[:2] + strings.Repeat("*", len(secret)-4) + secret[len(secret)-2:]
	default:
		// Show first 4 and last 4
		return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
	}
}
