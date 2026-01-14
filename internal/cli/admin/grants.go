package admin

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newGrantsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "grants",
		Aliases: []string{"grant"},
		Short:   "Manage grants",
		Long:    "View and manage grants across all applications.",
	}

	cmd.AddCommand(newGrantListCmd())
	cmd.AddCommand(newGrantStatsCmd())

	return cmd
}

func newGrantListCmd() *cobra.Command {
	var (
		limit       int
		offset      int
		connectorID string
		status      string
		jsonOutput  bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List grants",
		Long:    "List all grants with optional filters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			params := &domain.GrantsQueryParams{
				Limit:       limit,
				Offset:      offset,
				ConnectorID: connectorID,
				Status:      status,
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grants, err := client.ListAllGrants(ctx, params)
			if err != nil {
				return common.WrapListError("grants", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(grants)
			}

			if len(grants) == 0 {
				common.PrintEmptyState("grants")
				return nil
			}

			fmt.Printf("Found %d grant(s):\n\n", len(grants))

			table := common.NewTable("EMAIL", "ID", "PROVIDER", "STATUS")
			for _, grant := range grants {
				email := grant.Email
				if email == "" {
					email = "-"
				}

				status := grant.GrantStatus
				switch grant.GrantStatus {
				case "valid":
					status = common.Green.Sprint(status)
				case "invalid":
					status = common.Red.Sprint(status)
				default:
					status = common.Yellow.Sprint(status)
				}

				table.AddRow(common.Cyan.Sprint(email), grant.ID, string(grant.Provider), status)
			}
			table.Render()

			return nil
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of grants to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	cmd.Flags().StringVar(&connectorID, "connector-id", "", "Filter by connector ID")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (valid, invalid)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newGrantStatsCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show grant statistics",
		Long:  "Show statistics about all grants in the organization.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			stats, err := client.GetGrantStats(ctx)
			if err != nil {
				return common.WrapGetError("grant stats", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(stats)
			}

			_, _ = common.Bold.Println("Grant Statistics")
			fmt.Printf("  Total Grants: %s\n", common.Cyan.Sprintf("%d", stats.Total))
			fmt.Printf("  Valid: %s\n", common.Green.Sprintf("%d", stats.Valid))
			fmt.Printf("  Invalid: %s\n", common.Red.Sprintf("%d", stats.Invalid))

			if len(stats.ByProvider) > 0 {
				fmt.Printf("\nBy Provider:\n")
				table := common.NewTable("PROVIDER", "COUNT")
				for provider, count := range stats.ByProvider {
					table.AddRow(common.Green.Sprint(provider), fmt.Sprintf("%d", count))
				}
				table.Render()
			}

			if len(stats.ByStatus) > 0 {
				fmt.Printf("\nBy Status:\n")
				table := common.NewTable("STATUS", "COUNT")
				for status, count := range stats.ByStatus {
					statusColor := common.Yellow
					switch status {
					case "valid":
						statusColor = common.Green
					case "invalid":
						statusColor = common.Red
					}
					table.AddRow(statusColor.Sprint(status), fmt.Sprintf("%d", count))
				}
				table.Render()
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}
