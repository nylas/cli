package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List grants",
		Long:    "List all grants with optional filters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				params := &domain.GrantsQueryParams{
					Limit:       limit,
					Offset:      offset,
					ConnectorID: connectorID,
					Status:      status,
				}

				grants, err := client.ListAllGrants(ctx, params)
				if err != nil {
					return struct{}{}, common.WrapListError("grants", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(grants)
				}

				if len(grants) == 0 {
					common.PrintEmptyState("grants")
					return struct{}{}, nil
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

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 50, "Maximum number of grants to return")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	cmd.Flags().StringVar(&connectorID, "connector-id", "", "Filter by connector ID")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (valid, invalid)")

	return cmd
}

func newGrantStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show grant statistics",
		Long:  "Show statistics about all grants in the organization.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				stats, err := client.GetGrantStats(ctx)
				if err != nil {
					return struct{}{}, common.WrapGetError("grant stats", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(stats)
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

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}
