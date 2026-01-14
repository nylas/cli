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
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(args, yes || force)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation (alias for --yes)")

	return cmd
}

func runDelete(args []string, skipConfirm bool) error {
	inboxID, err := getInboxID(args)
	if err != nil {
		printError("%v", err)
		return err
	}

	client, err := getClient()
	if err != nil {
		printError("%v", err)
		return err
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	// Get inbox details first for confirmation
	inbox, err := client.GetInboundInbox(ctx, inboxID)
	if err != nil {
		printError("Failed to find inbox: %v", err)
		return err
	}

	// Confirm deletion unless --yes flag is set
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
	if err := client.DeleteInboundInbox(ctx, inboxID); err != nil {
		printError("Failed to delete inbox: %v", err)
		return err
	}

	printSuccess("Inbox %s deleted successfully!", inbox.Email)

	return nil
}
