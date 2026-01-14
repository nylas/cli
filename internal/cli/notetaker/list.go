package notetaker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			params := &domain.NotetakerQueryParams{
				Limit: limit,
				State: state,
			}

			notetakers, err := client.ListNotetakers(ctx, grantID, params)
			if err != nil {
				return common.WrapListError("notetakers", err)
			}

			if outputJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(notetakers)
			}

			if len(notetakers) == 0 {
				common.PrintEmptyState("notetakers")
				return nil
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

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 20, "Maximum number of notetakers to return")
	cmd.Flags().StringVar(&state, "state", "", "Filter by state (scheduled, connecting, attending, complete, cancelled, failed)")
	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}

func formatState(state string) string {
	switch state {
	case domain.NotetakerStateScheduled:
		return common.Yellow.Sprint("scheduled")
	case domain.NotetakerStateConnecting:
		return common.Cyan.Sprint("connecting")
	case domain.NotetakerStateWaitingForEntry:
		return common.Cyan.Sprint("waiting")
	case domain.NotetakerStateAttending:
		return common.Green.Sprint("attending")
	case domain.NotetakerStateMediaProcessing:
		return common.Cyan.Sprint("processing")
	case domain.NotetakerStateComplete:
		return common.Green.Sprint("complete")
	case domain.NotetakerStateCancelled:
		return common.Dim.Sprint("cancelled")
	case domain.NotetakerStateFailed:
		return common.Red.Sprint("failed")
	default:
		return state
	}
}
