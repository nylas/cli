package notetaker

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <notetaker-id> [grant-id]",
		Short: "Show notetaker details",
		Long:  `Show detailed information about a specific notetaker.`,
		Example: `  # Show notetaker details
  nylas notetaker show abc123

  # Output as JSON
  nylas notetaker show abc123 --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := args[0]

			_, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				notetaker, err := client.GetNotetaker(ctx, grantID, notetakerID)
				if err != nil {
					return struct{}{}, common.WrapGetError("notetaker", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(notetaker)
				}

				_, _ = common.Cyan.Printf("Notetaker: %s\n", notetaker.ID)
				fmt.Printf("State:     %s\n", formatState(notetaker.State))

				if notetaker.MeetingTitle != "" {
					fmt.Printf("Title:     %s\n", notetaker.MeetingTitle)
				}
				if notetaker.MeetingLink != "" {
					fmt.Printf("Link:      %s\n", notetaker.MeetingLink)
				}

				if notetaker.MeetingInfo != nil {
					if notetaker.MeetingInfo.Provider != "" {
						_, _ = common.Green.Printf("Provider:  %s\n", notetaker.MeetingInfo.Provider)
					}
					if notetaker.MeetingInfo.MeetingCode != "" {
						fmt.Printf("Code:      %s\n", notetaker.MeetingInfo.MeetingCode)
					}
				}

				if notetaker.BotConfig != nil {
					if notetaker.BotConfig.Name != "" {
						fmt.Printf("Bot Name:  %s\n", notetaker.BotConfig.Name)
					}
				}

				if !notetaker.JoinTime.IsZero() {
					_, _ = common.Yellow.Printf("Join Time: %s\n", notetaker.JoinTime.Local().Format(common.DisplayWeekdayFullWithTZ))
				}

				// Show media info if available
				if notetaker.MediaData != nil {
					fmt.Println("\nMedia:")
					if notetaker.MediaData.Recording != nil {
						_, _ = common.Green.Printf("  Recording: %s\n", notetaker.MediaData.Recording.URL)
						_, _ = common.Dim.Printf("    Size: %d bytes\n", notetaker.MediaData.Recording.Size)
					}
					if notetaker.MediaData.Transcript != nil {
						_, _ = common.Green.Printf("  Transcript: %s\n", notetaker.MediaData.Transcript.URL)
						_, _ = common.Dim.Printf("    Size: %d bytes\n", notetaker.MediaData.Transcript.Size)
					}
				}

				fmt.Println()
				_, _ = common.Dim.Printf("Created: %s\n", notetaker.CreatedAt.Local().Format(common.DisplayWeekdayFullWithTZ))
				if !notetaker.UpdatedAt.IsZero() {
					_, _ = common.Dim.Printf("Updated: %s\n", notetaker.UpdatedAt.Local().Format(common.DisplayWeekdayFullWithTZ))
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}
