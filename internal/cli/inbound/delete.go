package inbound

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var (
		force bool
		yes   bool
	)

	cmd := &cobra.Command{
		Use:   "delete <inbox-id>",
		Short: "Delete an inbound inbox",
		Long: `Delete an inbound inbox.

This will permanently delete the inbox and all associated messages.
This action cannot be undone.

Examples:
  # Delete an inbox (with confirmation)
  nylas inbound delete abc123

  # Delete without confirmation
  nylas inbound delete abc123 --yes

  # Use environment variable for inbox ID
  export NYLAS_INBOUND_GRANT_ID=abc123
  nylas inbound delete --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inboxID, err := getInboxID(args)
			if err != nil {
				return err
			}

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			skipConfirm := yes || force

			// Get inbox details first for confirmation
			ctx, cancel := common.CreateContext()
			inbox, err := client.GetInboundInbox(ctx, inboxID)
			cancel()

			if err != nil {
				return common.WrapGetError("inbox", err)
			}

			// Confirm deletion unless --yes flag is set
			// Use stronger confirmation for destructive action
			if !skipConfirm {
				fmt.Printf("You are about to delete the inbound inbox:\n")
				fmt.Printf("  Email: %s\n", common.Cyan.Sprint(inbox.Email))
				fmt.Printf("  ID:    %s\n", inbox.ID)
				fmt.Println()
				_, _ = common.Yellow.Println("This action cannot be undone. All messages in this inbox will be deleted.")
				fmt.Println()

				fmt.Print("Type 'delete' to confirm: ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if input != "delete" {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			// Delete the inbox
			ctx2, cancel2 := common.CreateContext()
			defer cancel2()

			err = common.RunWithSpinner("Deleting inbox...", func() error {
				return client.DeleteInboundInbox(ctx2, inboxID)
			})
			if err != nil {
				return common.WrapDeleteError("inbox", err)
			}

			// Remove from local grant store
			removeGrantLocally(inboxID)

			printSuccess("Inbox %s deleted successfully!", inbox.Email)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation (alias for --yes)")

	return cmd
}
