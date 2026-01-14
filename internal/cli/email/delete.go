package email

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <message-id> [grant-id]",
		Short: "Delete an email message",
		Long:  "Delete an email message (moves to trash).",
		Args:  cobra.RangeArgs(1, 2),
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

			// Confirmation
			if !force {
				fmt.Printf("Delete message %s? [y/N]: ", messageID)
				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.ToLower(strings.TrimSpace(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			err = client.DeleteMessage(ctx, grantID, messageID)
			if err != nil {
				return common.WrapDeleteError("message", err)
			}

			printSuccess("Message deleted (moved to trash)")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
