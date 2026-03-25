package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current dashboard authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, secrets, err := createAuthService()
			if err != nil {
				return wrapDashboardError(err)
			}

			status := authSvc.GetStatus()

			if !status.LoggedIn {
				_, _ = common.Yellow.Println("Not logged in")
				fmt.Println("  nylas dashboard login")
				return nil
			}

			_, _ = common.Green.Println("✓ Logged in")
			if status.UserID != "" {
				fmt.Printf("  User:         %s\n", status.UserID)
			}
			if status.OrgID != "" {
				fmt.Printf("  Organization: %s\n", status.OrgID)
			}
			fmt.Printf("  Org token:    %s\n", presentAbsent(status.HasOrgToken))

			// Active app
			appID, _ := secrets.Get(ports.KeyDashboardAppID)
			appRegion, _ := secrets.Get(ports.KeyDashboardAppRegion)
			if appID != "" {
				fmt.Printf("  Active app:   %s (%s)\n", appID, appRegion)
			}

			dpopSvc, _, dpopErr := createDPoPService()
			if dpopErr == nil {
				fmt.Printf("  DPoP key:     %s\n", dpopSvc.Thumbprint())
			}

			return nil
		},
	}
}

func presentAbsent(present bool) string {
	if present {
		return "present"
	}
	return "absent"
}
