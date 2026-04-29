package email

import (
	"context"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newMarkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mark",
		Short: "Mark messages as read/unread/starred",
		Long:  "Update message flags like read status and star.",
	}

	cmd.AddCommand(newMarkReadCmd())
	cmd.AddCommand(newMarkUnreadCmd())
	cmd.AddCommand(newMarkStarredCmd())
	cmd.AddCommand(newMarkUnstarredCmd())

	return cmd
}

func newMarkReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <message-id> [grant-id]",
		Short: "Mark a message as read",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return markMessage(args, false, nil)
		},
	}
}

func newMarkUnreadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unread <message-id> [grant-id]",
		Short: "Mark a message as unread",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return markMessage(args, true, nil)
		},
	}
}

func newMarkStarredCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "starred <message-id> [grant-id]",
		Short: "Star a message",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			starred := true
			return markMessage(args, false, &starred)
		},
	}
}

func newMarkUnstarredCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unstarred <message-id> [grant-id]",
		Short: "Remove star from a message",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			starred := false
			return markMessage(args, false, &starred)
		},
	}
}

func markMessage(args []string, unread bool, starred *bool) error {
	messageID := args[0]
	remainingArgs := args[1:]

	_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
		req := &domain.UpdateMessageRequest{}

		// Set the appropriate flags based on which mark command was called
		if starred != nil {
			req.Starred = starred
		} else {
			req.Unread = &unread
		}

		_, err := client.UpdateMessage(ctx, grantID, messageID, req)
		if err != nil {
			return struct{}{}, common.WrapUpdateError("message", err)
		}

		if starred != nil {
			if *starred {
				common.PrintSuccess("Message starred")
			} else {
				common.PrintSuccess("Star removed from message")
			}
		} else {
			if unread {
				common.PrintSuccess("Message marked as unread")
			} else {
				common.PrintSuccess("Message marked as read")
			}
		}

		return struct{}{}, nil
	})
	return err
}
