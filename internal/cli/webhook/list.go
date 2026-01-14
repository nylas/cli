package webhook

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			webhooks, err := common.RunWithSpinnerResult("Fetching webhooks...", func() ([]domain.Webhook, error) {
				return c.ListWebhooks(ctx)
			})
			if err != nil {
				return common.NewUserError("Failed to list webhooks: "+err.Error(),
					"Check your API key has webhook management permissions")
			}

			if len(webhooks) == 0 {
				common.PrintEmptyStateWithHint("webhooks", "Create one with: nylas webhook create --url <URL> --triggers <triggers>")
				return nil
			}

			switch format {
			case "json":
				return outputJSON(webhooks)
			case "yaml":
				return outputYAML(webhooks)
			case "csv":
				return outputCSV(webhooks)
			default:
				return outputTable(webhooks, fullIDs)
			}
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "table", "Output format (table, json, yaml, csv)")
	cmd.Flags().BoolVar(&fullIDs, "full-ids", false, "Show full webhook IDs (useful for copy/paste)")

	return cmd
}

func outputJSON(webhooks any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(webhooks)
}

func outputYAML(webhooks any) error {
	return yaml.NewEncoder(os.Stdout).Encode(webhooks)
}

func outputCSV(webhooks any) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	_ = w.Write([]string{"ID", "Description", "URL", "Status", "Triggers"})

	// Get webhooks as slice
	data, _ := json.Marshal(webhooks)
	var items []map[string]any
	_ = json.Unmarshal(data, &items)

	for _, item := range items {
		id, _ := item["id"].(string)
		desc, _ := item["description"].(string)
		url, _ := item["webhook_url"].(string)
		status, _ := item["status"].(string)

		var triggers []string
		if triggerList, ok := item["trigger_types"].([]any); ok {
			for _, t := range triggerList {
				triggers = append(triggers, fmt.Sprintf("%v", t))
			}
		}

		_ = w.Write([]string{id, desc, url, status, strings.Join(triggers, ";")})
	}

	return nil
}

func outputTable(webhooks any, fullIDs bool) error {
	data, _ := json.Marshal(webhooks)
	var items []map[string]any
	_ = json.Unmarshal(data, &items)

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

	for _, item := range items {
		id := fmt.Sprintf("%v", item["id"])
		if !fullIDs {
			id = common.Truncate(id, 20)
		}
		r := row{
			id:     id,
			desc:   common.Truncate(fmt.Sprintf("%v", item["description"]), 25),
			url:    common.Truncate(fmt.Sprintf("%v", item["webhook_url"]), 35),
			status: fmt.Sprintf("%v", item["status"]),
		}

		var triggers []string
		if triggerList, ok := item["trigger_types"].([]any); ok {
			for _, t := range triggerList {
				triggers = append(triggers, fmt.Sprintf("%v", t))
			}
		}
		r.triggers = common.Truncate(strings.Join(triggers, ", "), 30)

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
		statusIcon := getStatusIcon(r.status)
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

func getStatusIcon(status string) string {
	switch status {
	case "active":
		return common.Green.Sprint("●")
	case "inactive":
		return common.Yellow.Sprint("●")
	case "failing":
		return common.Red.Sprint("●")
	default:
		return "○"
	}
}
