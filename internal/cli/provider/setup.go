package provider

import "github.com/spf13/cobra"

// newSetupCmd creates the 'setup' subcommand group.
func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up provider integrations",
		Long:  "Automated setup wizards for provider integrations.",
	}

	cmd.AddCommand(newGoogleSetupCmd())

	return cmd
}
