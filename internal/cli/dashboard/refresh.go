package dashboard

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh dashboard session tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			err = common.RunWithSpinner("Refreshing session...", func() error {
				return authSvc.Refresh(ctx)
			})
			if err != nil {
				if errors.Is(err, domain.ErrDashboardSessionExpired) {
					fmt.Println("Session expired. Please log in again:")
					fmt.Println("  nylas dashboard login")
				}
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Println("✓ Session refreshed")
			return nil
		},
	}
}
