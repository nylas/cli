package email

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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

	client, err := getClient()
	if err != nil {
		return err
	}

	var grantID string
	if len(args) > 1 {
		grantID = args[1]
	} else {
		grantID, err = common.GetGrantID(nil)
		if err != nil {
			return err
		}
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	req := &domain.UpdateMessageRequest{}

	// Set the appropriate flags based on which mark command was called
	if starred != nil {
		req.Starred = starred
	} else {
		req.Unread = &unread
	}

	_, err = client.UpdateMessage(ctx, grantID, messageID, req)
	if err != nil {
		return common.WrapUpdateError("message", err)
	}

	if starred != nil {
		if *starred {
			printSuccess("Message starred")
		} else {
			printSuccess("Star removed from message")
		}
	} else {
		if unread {
			printSuccess("Message marked as unread")
		} else {
			printSuccess("Message marked as read")
		}
	}

	return nil
}
