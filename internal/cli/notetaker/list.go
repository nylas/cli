package notetaker

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func newListCmd() *cobra.Command {
	var (
		limit      int
		state      string
		outputJSON bool
	)

	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List notetakers",
		Long:    `List all notetakers for a grant. Filter by state using --state flag.`,
		Example: `  # List all notetakers
  nylas notetaker list

  # List only scheduled notetakers
  nylas notetaker list --state scheduled

  # List completed notetakers
  nylas notetaker list --state complete

  # Output as JSON
  nylas notetaker list --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := &domain.NotetakerQueryParams{
				Limit: limit,
				State: state,
			}

			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				notetakers, err := client.ListNotetakers(ctx, grantID, params)
				if err != nil {
					return struct{}{}, common.WrapListError("notetakers", err)
				}

				if outputJSON {
					return struct{}{}, common.PrintJSON(notetakers)
				}

				if len(notetakers) == 0 {
					common.PrintEmptyState("notetakers")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d notetaker(s):\n\n", len(notetakers))

				for _, n := range notetakers {
					_, _ = common.Cyan.Printf("ID: %s\n", n.ID)
					fmt.Printf("  State:   %s\n", formatState(n.State))
					if n.MeetingTitle != "" {
						fmt.Printf("  Title:   %s\n", n.MeetingTitle)
					}
					if n.MeetingLink != "" {
						fmt.Printf("  Link:    %s\n", common.Truncate(n.MeetingLink, 60))
					}
					if n.MeetingInfo != nil && n.MeetingInfo.Provider != "" {
						caser := cases.Title(language.English)
						_, _ = common.Green.Printf("  Provider: %s\n", caser.String(n.MeetingInfo.Provider))
					}
					if !n.JoinTime.IsZero() {
						_, _ = common.Yellow.Printf("  Join:    %s\n", n.JoinTime.Local().Format(common.DisplayWeekdayFull))
					}
					if !n.CreatedAt.IsZero() {
						_, _ = common.Dim.Printf("  Created: %s\n", common.FormatTimeAgo(n.CreatedAt))
					}
					fmt.Println()
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of notetakers to return")
	cmd.Flags().StringVar(&state, "state", "", "Filter by state (scheduled, connecting, attending, complete, cancelled, failed)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

func formatState(state string) string {
	// Map API state names to display-friendly labels
	displayName := state
	switch state {
	case domain.NotetakerStateWaitingForEntry:
		displayName = "waiting"
	case domain.NotetakerStateMediaProcessing:
		displayName = "processing"
	}
	return common.StatusColor(state).Sprint(displayName)
}
