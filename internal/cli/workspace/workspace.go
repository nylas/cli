package workspace

import "github.com/spf13/cobra"

func NewWorkspaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspace",
		Aliases: []string{"workspaces", "ws"},
		Short:   "Manage workspaces",
		Long: `Manage Nylas workspaces.

Workspaces group agent accounts and attach policies and rules.

Examples:
  nylas workspace list
  nylas workspace get <workspace-id>
  nylas workspace create --name "My Workspace"
  nylas workspace update <workspace-id> --policy-id <policy-id>
  nylas workspace delete <workspace-id> --yes`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())

	return cmd
}
