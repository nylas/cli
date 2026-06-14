package email

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newMoveCmd() *cobra.Command {
	var (
		folder  string
		archive bool
	)

	cmd := &cobra.Command{
		Use:   "move <message-id> [grant-id]",
		Short: "Move a message to a folder (or archive it)",
		Long: `Move a message to a different folder, or archive it.

Pass --folder with a folder ID to move the message into that folder. Use the
"nylas email folders list" command to find folder IDs.

Pass --archive to archive the message instead. On Gmail/label-based accounts
this removes all labels (including INBOX); on folder-based (IMAP/Microsoft)
accounts the provider moves the message to its Archive folder.

API reference: https://developer.nylas.com/docs/v3/email/`,
		Example: `  # Move a message to a folder
  nylas email move <message-id> --folder <folder-id>

  # Archive a message
  nylas email move <message-id> --archive`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if folder != "" && archive {
				return common.NewMutuallyExclusiveError("folder", "archive")
			}
			if folder == "" && !archive {
				return common.NewUserError(
					"no destination specified",
					"Pass --folder <folder-id> to move the message, or --archive to archive it.",
				)
			}

			messageID := args[0]
			// nil leaves folders untouched; a non-nil slice (even empty) sets
			// them. --archive sends an empty slice to clear all folders/labels.
			folders := []string{}
			if folder != "" {
				folders = []string{folder}
			}

			_, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.UpdateMessageRequest{Folders: folders}
				if _, err := client.UpdateMessage(ctx, grantID, messageID, req); err != nil {
					return struct{}{}, common.WrapUpdateError("message", err)
				}
				if archive {
					common.PrintSuccess("Message archived")
				} else {
					common.PrintSuccess("Message moved")
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&folder, "folder", "", "Destination folder ID")
	cmd.Flags().BoolVar(&archive, "archive", false, "Archive the message (clear all folders/labels)")

	return cmd
}
