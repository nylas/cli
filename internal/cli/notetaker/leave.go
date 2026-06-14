package notetaker

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newLeaveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "leave <notetaker-id> [grant-id]",
		Short: "Make an active notetaker leave its meeting",
		Long: `Instruct an active notetaker bot to leave the meeting it is currently in.

This stops the recording and triggers media (recording/transcript) generation,
while keeping the notetaker record and its media. Use this to cleanly end a live
recording.

To cancel a scheduled bot before it joins, or to remove a notetaker and its
media entirely, use "nylas notetaker delete" instead.

API reference: https://developer.nylas.com/docs/v3/notetaker/`,
		Example: `  # Tell a notetaker to leave the meeting now
  nylas notetaker leave <notetaker-id>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := args[0]
			_, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if err := client.LeaveNotetaker(ctx, grantID, notetakerID); err != nil {
					return struct{}{}, err
				}
				common.PrintSuccess("Notetaker instructed to leave the meeting")
				return struct{}{}, nil
			})
			return err
		},
	}
}
