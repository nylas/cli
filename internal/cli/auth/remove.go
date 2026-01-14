package auth

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <grant-id>",
		Short: "Remove a grant from local config (without revoking on server)",
		Long: `Remove a grant from local configuration only.

This does NOT revoke the grant on the Nylas server - it only removes
the grant from your local CLI configuration. The grant will still be
valid and can be re-added later with 'nylas auth add'.

Use 'nylas auth revoke' if you want to permanently revoke the grant
on the Nylas server.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := args[0]

			grantStore, err := createGrantStore()
			if err != nil {
				return err
			}

			// Check if grant exists locally
			if _, err := grantStore.GetGrant(grantID); err != nil {
				return err
			}

			// Remove from local store only
			if err := grantStore.DeleteGrant(grantID); err != nil {
				return err
			}

			_, _ = common.Green.Printf("✓ Grant %s removed from local config\n", grantID)
			_, _ = common.Yellow.Println("  Note: Grant is still valid on Nylas server")
			_, _ = common.Yellow.Println("  Use 'nylas auth add' to re-add it, or 'nylas auth revoke' to permanently revoke")

			return nil
		},
	}
}
