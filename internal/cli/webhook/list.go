package webhook

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newListCmd() *cobra.Command {
	var format string
	var fullIDs bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all webhooks",
		Long: `List all webhooks configured for your Nylas application.

Shows webhook ID, description, URL, status, and trigger types.`,
		Example: `  # List all webhooks
  nylas webhook list

  # List with full IDs (useful for copy/paste)
  nylas webhook list --full-ids

  # List in JSON format
  nylas webhook list --format json

  # List in YAML format
  nylas webhook list --format yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if we should use structured output from global flags
			if common.IsJSON(cmd) {
				_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
					webhooks, err := common.RunWithSpinnerResult("Fetching webhooks...", func() ([]domain.Webhook, error) {
						return client.ListWebhooks(ctx)
					})
					if err != nil {
						return struct{}{}, common.WrapListError("webhooks", err)
					}

					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(webhooks)
				})
				return err
			}

			// Use WithClientNoGrant for all output formats
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				webhooks, err := common.RunWithSpinnerResult("Fetching webhooks...", func() ([]domain.Webhook, error) {
					return client.ListWebhooks(ctx)
				})
				if err != nil {
					return struct{}{}, common.WrapListError("webhooks", err)
				}

				if len(webhooks) == 0 {
					common.PrintEmptyStateWithHint("webhooks", "Create one with: nylas webhook create --url <URL> --triggers <triggers>")
					return struct{}{}, nil
				}

				switch format {
				case "json":
					return struct{}{}, outputJSON(webhooks)
				case "yaml":
					return struct{}{}, outputYAML(webhooks)
				case "csv":
					return struct{}{}, outputCSV(webhooks)
				default:
					return struct{}{}, outputTable(webhooks, fullIDs)
				}
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml, csv)")
	cmd.Flags().BoolVar(&fullIDs, "full-ids", false, "Show full webhook IDs (useful for copy/paste)")

	return cmd
}

func outputJSON(webhooks []domain.Webhook) error {
	return common.PrintJSON(webhooks)
}

func outputYAML(webhooks []domain.Webhook) error {
	return yaml.NewEncoder(os.Stdout).Encode(webhooks)
}

func outputCSV(webhooks []domain.Webhook) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	_ = w.Write([]string{"ID", "Description", "URL", "Status", "Triggers"})

	for _, webhook := range webhooks {
		_ = w.Write([]string{
			webhook.ID,
			webhook.Description,
			webhook.WebhookURL,
			webhook.Status,
			strings.Join(webhook.TriggerTypes, ";"),
		})
	}

	return nil
}

func outputTable(webhooks []domain.Webhook, fullIDs bool) error {
	// Calculate column widths
	headers := []string{"ID", "Description", "URL", "Status", "Triggers"}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}

	type row struct {
		id, desc, url, status, triggers string
	}
	var rows []row

	for _, webhook := range webhooks {
		id := webhook.ID
		if !fullIDs {
			id = common.Truncate(id, 20)
		}
		r := row{
			id:     id,
			desc:   common.Truncate(webhook.Description, 25),
			url:    common.Truncate(webhook.WebhookURL, 35),
			status: webhook.Status,
		}

		r.triggers = common.Truncate(strings.Join(webhook.TriggerTypes, ", "), 30)

		rows = append(rows, r)

		if len(r.id) > widths[0] {
			widths[0] = len(r.id)
		}
		if len(r.desc) > widths[1] {
			widths[1] = len(r.desc)
		}
		if len(r.url) > widths[2] {
			widths[2] = len(r.url)
		}
		if len(r.status) > widths[3] {
			widths[3] = len(r.status)
		}
		if len(r.triggers) > widths[4] {
			widths[4] = len(r.triggers)
		}
	}

	// Print header
	fmt.Printf("%-*s  %-*s  %-*s  %-*s  %s\n",
		widths[0], headers[0],
		widths[1], headers[1],
		widths[2], headers[2],
		widths[3], headers[3],
		headers[4])

	// Print separator
	for i, w := range widths {
		if i > 0 {
			fmt.Print("  ")
		}
		fmt.Print(strings.Repeat("-", w))
	}
	fmt.Println()

	// Print rows
	for _, r := range rows {
		statusIcon := common.StatusIcon(r.status)
		fmt.Printf("%-*s  %-*s  %-*s  %s %-*s  %s\n",
			widths[0], r.id,
			widths[1], r.desc,
			widths[2], r.url,
			statusIcon, widths[3]-2, r.status,
			r.triggers)
	}

	fmt.Printf("\nTotal: %d webhooks\n", len(rows))
	return nil
}
