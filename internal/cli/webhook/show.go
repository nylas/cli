package webhook

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newShowCmd() *cobra.Command {
	var format string

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

			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			webhook, err := common.RunWithSpinnerResult("Fetching webhook...", func() (*domain.Webhook, error) {
				return c.GetWebhook(ctx, webhookID)
			})
			if err != nil {
				return common.NewUserError("Failed to get webhook: "+err.Error(),
					"Check that the webhook ID is correct. Run 'nylas webhook list' to see available webhooks")
			}

			switch format {
			case "json":
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(webhook)
			case "yaml":
				return yaml.NewEncoder(cmd.OutOrStdout()).Encode(webhook)
			default:
				return displayWebhookDetails(webhook)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")

	return cmd
}

func displayWebhookDetails(webhook any) error {
	data, _ := json.Marshal(webhook)
	var w map[string]any
	_ = json.Unmarshal(data, &w)

	id := getString(w, "id")
	desc := getString(w, "description")
	url := getString(w, "webhook_url")
	secret := getString(w, "webhook_secret")
	status := getString(w, "status")

	statusIcon := getStatusIcon(status)

	fmt.Printf("Webhook: %s\n", id)
	fmt.Println(strings.Repeat("─", 60))

	if desc != "" {
		fmt.Printf("Description:  %s\n", desc)
	}
	fmt.Printf("URL:          %s\n", url)
	fmt.Printf("Status:       %s %s\n", statusIcon, status)

	if secret != "" {
		fmt.Printf("Secret:       %s\n", maskSecret(secret))
	}

	// Trigger types
	fmt.Println("\nTrigger Types:")
	if triggers, ok := w["trigger_types"].([]any); ok {
		for _, t := range triggers {
			fmt.Printf("  • %s\n", t)
		}
	}

	// Notification emails
	if emails, ok := w["notification_email_addresses"].([]any); ok && len(emails) > 0 {
		fmt.Println("\nNotification Emails:")
		for _, e := range emails {
			fmt.Printf("  • %s\n", e)
		}
	}

	// Timestamps
	fmt.Println("\nTimestamps:")
	if created := getString(w, "created_at"); created != "" {
		fmt.Printf("  Created:        %s\n", created)
	}
	if updated := getString(w, "updated_at"); updated != "" {
		fmt.Printf("  Updated:        %s\n", updated)
	}
	if statusUpdated := getString(w, "status_updated_at"); statusUpdated != "" {
		fmt.Printf("  Status Updated: %s\n", statusUpdated)
	}

	return nil
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
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
