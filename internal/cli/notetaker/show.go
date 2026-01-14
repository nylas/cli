package notetaker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var outputJSON bool

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
			client, err := getClient()
			if err != nil {
				return err
			}

			notetakerID := args[0]
			grantID, err := getGrantID(args[1:])
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			notetaker, err := client.GetNotetaker(ctx, grantID, notetakerID)
			if err != nil {
				return common.WrapGetError("notetaker", err)
			}

			if outputJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(notetaker)
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

			return nil
		},
	}

	cmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	return cmd
}
