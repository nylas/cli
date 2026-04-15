package agent

import "github.com/spf13/cobra"

func newAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage agent accounts",
		Long: `Manage Nylas agent accounts.

Agent accounts are managed email identities backed by the Nylas provider.
This command always uses provider=nylas and keeps connector setup out of the
user's path.

Examples:
  # Create a new agent account
  nylas agent account create me@yourapp.nylas.email

  # List agent accounts
  nylas agent account list

  # Show one agent account
  nylas agent account get <agent-id|email>

  # Delete an agent account
  nylas agent account delete <agent-id|email>`,
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}
