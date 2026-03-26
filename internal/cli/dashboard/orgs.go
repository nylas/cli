package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

func newOrgsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "orgs",
		Short: "Manage organizations",
		Long:  `List and manage organizations you belong to.`,
	}

	cmd.AddCommand(newOrgsListCmd())
	cmd.AddCommand(newSwitchOrgCmd())

	return cmd
}

func newOrgsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List organizations you belong to",
		Example: `  nylas dashboard orgs list
  nylas dashboard orgs list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			session, err := authSvc.GetCurrentSession(ctx)
			if err != nil {
				return wrapDashboardError(err)
			}

			if len(session.Relations) == 0 {
				fmt.Println("No organizations found.")
				return nil
			}

			rows := make([]orgRow, len(session.Relations))
			for i, rel := range session.Relations {
				current := ""
				if rel.OrgPublicID == session.CurrentOrg {
					current = "✓"
				}
				rows[i] = orgRow{
					PublicID: rel.OrgPublicID,
					Name:     rel.OrgName,
					Role:     rel.Role,
					Current:  current,
				}
			}

			return common.WriteListWithColumns(cmd, rows, orgColumns)
		},
	}
}
