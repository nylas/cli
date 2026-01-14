package email

import (
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newReadCmd() *cobra.Command {
	var markAsRead bool
	var rawOutput bool

	cmd := &cobra.Command{
		Use:     "read <message-id> [grant-id]",
		Aliases: []string{"show"},
		Short:   "Read a specific email",
		Long:    "Read and display the full content of a specific email message.",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			msg, err := client.GetMessage(ctx, grantID, messageID)
			if err != nil {
				return common.WrapGetError("message", err)
			}

			// Handle JSON output
			jsonOutput, _ := cmd.Flags().GetBool("json")
			if jsonOutput {
				data, err := json.MarshalIndent(msg, "", "  ")
				if err != nil {
					return common.WrapMarshalError("JSON", err)
				}
				fmt.Println(string(data))
				return nil
			}

			// Handle raw output
			if rawOutput {
				printMessageRaw(*msg)
			} else {
				printMessage(*msg, true)
			}

			// Mark as read if requested
			if markAsRead && msg.Unread {
				unread := false
				_, err := client.UpdateMessage(ctx, grantID, messageID, &domain.UpdateMessageRequest{
					Unread: &unread,
				})
				if err != nil {
					_, _ = common.Dim.Printf("(Failed to mark as read: %v)\n", err)
				} else {
					_, _ = common.Dim.Println("(Marked as read)")
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&markAsRead, "mark-read", "r", false, "Mark the message as read after viewing")
	cmd.Flags().BoolVar(&rawOutput, "raw", false, "Show raw email body without HTML processing")

	return cmd
}
