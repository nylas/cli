package agent

import "github.com/spf13/cobra"

// NewAgentCmd creates the agent command group.
func NewAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agent",
		Aliases: []string{"agents"},
		Short:   "Manage Nylas agent resources",
		Long: `Manage Nylas agent resources.

Agent account operations live under the account subcommand. Top-level status
reports the readiness of the nylas connector and the currently configured
managed accounts.

Examples:
  # Create a new agent account
  nylas agent account create me@yourapp.nylas.email

  # List agent accounts
  nylas agent account list

  # List policies
  nylas agent policy list

  # List rules
  nylas agent rule list

  # Check connector and account status
  nylas agent status

  # Show an agent account
  nylas agent account get <agent-id|email>`,
	}

	cmd.AddCommand(newAccountCmd())
	cmd.AddCommand(newPolicyCmd())
	cmd.AddCommand(newRuleCmd())
	cmd.AddCommand(newStatusCmd())

	return cmd
}
