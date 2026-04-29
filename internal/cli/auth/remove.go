package auth

import (
	adapterconfig "github.com/nylas/cli/internal/adapters/config"
	authapp "github.com/nylas/cli/internal/app/auth"
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

			configStore := adapterconfig.NewDefaultFileStore()
			grantStore, err := common.NewDefaultGrantStore()
			if err != nil {
				return err
			}

			authSvc := authapp.NewService(nil, grantStore, configStore, nil, nil)

			// Check if grant exists locally
			if _, err := grantStore.GetGrant(grantID); err != nil {
				return err
			}

			// Remove from local store only
			if err := authSvc.RemoveLocalGrant(grantID); err != nil {
				return err
			}

			_, _ = common.Green.Printf("✓ Grant %s removed from local cache\n", grantID)
			_, _ = common.Yellow.Println("  Note: Grant is still valid on Nylas server and will still appear in 'nylas auth list'")
			_, _ = common.Yellow.Println("  Use 'nylas auth switch <grant-id>' to set it as default again, or 'nylas auth revoke' to permanently revoke")

			return nil
		},
	}
}
