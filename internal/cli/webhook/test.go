package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test webhook functionality",
		Long: `Test webhook functionality with test events and mock payloads.

Use 'nylas webhook test send' to send a test event to a URL.
Use 'nylas webhook test payload' to see a mock payload for a trigger type.`,
	}

	cmd.AddCommand(newTestSendCmd())
	cmd.AddCommand(newTestPayloadCmd())

	return cmd
}

func newTestSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <webhook-url>",
		Short: "Send a test event to a webhook URL",
		Long: `Send a test webhook event to verify your endpoint is working.

This sends a test event to the specified URL to help verify
that your webhook endpoint is properly configured and receiving events.`,
		Example: `  # Send a test event to a webhook URL
  nylas webhook test send https://example.com/webhook`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			webhookURL := args[0]

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				err := common.RunWithSpinner("Sending test event...", func() error {
					return client.SendWebhookTestEvent(ctx, webhookURL)
				})
				if err != nil {
					return struct{}{}, common.NewUserError(fmt.Sprintf("Failed to send test event: %v", err),
						"Check that the URL is correct and accessible. Ensure your endpoint is publicly reachable")
				}

				fmt.Printf("%s Test event sent successfully!\n", common.Green.Sprint("✓"))
				fmt.Println()
				fmt.Printf("  URL: %s\n", webhookURL)
				fmt.Println()
				fmt.Println("Check your webhook endpoint logs to verify the event was received.")

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newTestPayloadCmd() *cobra.Command {
	var triggerType string

	cmd := &cobra.Command{
		Use:   "payload [trigger-type]",
		Short: "Get a mock payload for a trigger type",
		Long: `Get a sample webhook payload for a specific trigger type.

This helps you understand the structure of webhook payloads
so you can properly handle them in your application.`,
		Example: `  # Get mock payload for message.created trigger
  nylas webhook test payload message.created

  # Get mock payload in JSON format
  nylas webhook test payload --trigger event.created --format json

  # Interactive: list available triggers
  nylas webhook test payload`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get trigger type from args or flag
			if len(args) > 0 {
				triggerType = args[0]
			}

			if triggerType == "" {
				// Show interactive selection
				fmt.Println("Available trigger types:")
				fmt.Println()

				categories := domain.TriggerTypeCategories()
				categoryOrder := []string{"grant", "message", "thread", "event", "contact", "calendar", "folder"}

				for _, cat := range categoryOrder {
					if triggers, ok := categories[cat]; ok {
						fmt.Printf("  %s:\n", cat)
						for _, t := range triggers {
							fmt.Printf("    • %s\n", t)
						}
						fmt.Println()
					}
				}

				fmt.Println("Use: nylas webhook test payload <trigger-type>")
				return nil
			}

			// Validate trigger type
			validTriggers := domain.AllTriggerTypes()
			valid := false
			for _, vt := range validTriggers {
				if triggerType == vt {
					valid = true
					break
				}
			}
			if !valid {
				return common.NewUserError(fmt.Sprintf("Invalid trigger type: %s", triggerType),
					"Run 'nylas webhook test payload' to see available trigger types")
			}

			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				payload, err := common.RunWithSpinnerResult("Fetching mock payload...", func() (any, error) {
					return client.GetWebhookMockPayload(ctx, triggerType)
				})
				if err != nil {
					return struct{}{}, common.NewUserError(fmt.Sprintf("Failed to get mock payload: %v", err),
						"Check the trigger type is valid")
				}

				fmt.Printf("Mock payload for '%s':\n\n", triggerType)

				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return struct{}{}, enc.Encode(payload)
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&triggerType, "trigger", "t", "", "Trigger type")

	return cmd
}
