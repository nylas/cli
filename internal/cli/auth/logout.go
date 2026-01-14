package auth

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Revoke current authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := authSvc.Logout(ctx); err != nil {
				return err
			}

			_, _ = common.Green.Println("✓ Successfully logged out")

			return nil
		},
	}
}
