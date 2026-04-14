package agent

import "github.com/spf13/cobra"

// NewAgentCmd creates the agent command group.
func NewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents"},
		Short:   "Manage Nylas agent accounts",
		Long: `Manage Nylas agent accounts.

Agent accounts are managed email identities backed by the Nylas provider.
This command always uses provider=nylas and keeps the connector setup out of
the user's path.

Examples:
  # Create a new agent account
  nylas agent create me@yourapp.nylas.email

  # List agent accounts
  nylas agent list

  # Check connector and account status
  nylas agent status

  # Delete an agent account
  nylas agent delete <agent-id>`,
	}

	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
