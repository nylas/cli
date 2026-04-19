package agent

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
		Use:   "delete [agent-id|email]",
		Short: "Delete an agent account",
		Long: `Delete a Nylas agent account.

This permanently revokes the provider=nylas grant.

Examples:
  nylas agent account delete 123456
  nylas agent account delete me@yourapp.nylas.email --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := getRequiredAgentIdentifier(args)
			if err != nil {
				return err
			}

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			grantID, err := resolveAgentID(ctx, client, identifier)
			cancel()
			if err != nil {
				return common.WrapGetError("agent account", err)
			}

			ctx2, cancel2 := common.CreateContext()
			account, err := client.GetAgentAccount(ctx2, grantID)
			cancel2()
			if err != nil {
				return common.WrapGetError("agent account", err)
			}

			if !yes && !force {
				fmt.Printf("You are about to delete the agent account:\n")
				fmt.Printf("  Email: %s\n", common.Cyan.Sprint(account.Email))
				fmt.Printf("  ID:    %s\n", account.ID)
				fmt.Println()
				_, _ = common.Yellow.Println("This action cannot be undone.")
				fmt.Println()

				fmt.Print("Are you sure? [y/N]: ")
				reader := bufio.NewReader(os.Stdin)
				input, _ := reader.ReadString('\n')
				if !isDeleteConfirmed(input) {
					fmt.Println("Deletion cancelled.")
					return nil
				}
			}

			ctx3, cancel3 := common.CreateContext()
			defer cancel3()

			err = common.RunWithSpinner("Deleting agent account...", func() error {
				return client.DeleteAgentAccount(ctx3, grantID)
			})
			if err != nil {
				return common.WrapDeleteError("agent account", err)
			}

			removeGrantLocally(grantID)
			printSuccess("Agent account %s deleted successfully!", account.Email)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete without confirmation (alias for --yes)")

	return cmd
}

func isDeleteConfirmed(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes", "delete":
		return true
	default:
		return false
	}
}
