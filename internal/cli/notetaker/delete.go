package notetaker

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var noConfirm bool

	cmd := &cobra.Command{
		Use:     "delete <notetaker-id> [grant-id]",
		Aliases: []string{"rm", "cancel"},
		Short:   "Delete or cancel a notetaker",
		Long: `Delete a notetaker. If the notetaker is scheduled or active, this will cancel it.

This action cannot be undone. Once deleted, any recordings or transcripts
that haven't been saved will be lost.`,
		Example: `  # Delete a notetaker (with confirmation)
  nylas notetaker delete abc123

  # Delete without confirmation
  nylas notetaker delete abc123 --yes`,
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

			// Get notetaker details first for confirmation
			ctx, cancel := common.CreateContext()
			defer cancel()

			notetaker, err := client.GetNotetaker(ctx, grantID, notetakerID)
			if err != nil {
				return common.WrapGetError("notetaker", err)
			}

			// Confirmation
			if !noConfirm {
				fmt.Printf("Delete notetaker %s?\n", notetakerID)
				if notetaker.MeetingTitle != "" {
					fmt.Printf("  Title: %s\n", notetaker.MeetingTitle)
				}
				fmt.Printf("  State: %s\n", formatState(notetaker.State))
				fmt.Print("\nThis action cannot be undone. Continue? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.ToLower(strings.TrimSpace(confirm))
				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			// Delete
			ctx2, cancel2 := common.CreateContext()
			defer cancel2()

			if err := client.DeleteNotetaker(ctx2, grantID, notetakerID); err != nil {
				return common.WrapDeleteError("notetaker", err)
			}

			_, _ = common.BoldGreen.Printf("✓ Notetaker %s deleted successfully\n", notetakerID)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
