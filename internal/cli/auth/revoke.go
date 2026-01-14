package auth

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newRevokeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <grant-id>",
		Short: "Revoke a specific grant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := args[0]

			authSvc, _, err := createAuthService()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := authSvc.LogoutGrant(ctx, grantID); err != nil {
				return err
			}

			_, _ = common.Green.Printf("✓ Grant %s revoked\n", grantID)

			return nil
		},
	}
}
