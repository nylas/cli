package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func newSwitchOrgCmd() *cobra.Command {
	var orgFlag string

	cmd := &cobra.Command{
		Use:   "switch",
		Short: "Switch the active organization",
		Long: `Switch your dashboard session to a different organization.

Lists all organizations you belong to and lets you select one,
or pass --org to switch directly.`,
		Example: `  # Interactive — choose from your orgs
  nylas dashboard orgs switch

  # Switch directly by org ID
  nylas dashboard orgs switch --org org_abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get current session to list available orgs
			var session *domain.DashboardSessionResponse
			err = common.RunWithSpinner("Loading organizations...", func() error {
				session, err = authSvc.GetCurrentSession(ctx)
				return err
			})
			if err != nil {
				return wrapDashboardError(err)
			}

			if len(session.Relations) == 0 {
				fmt.Println("No organizations found.")
				return nil
			}

			targetOrgID := orgFlag
			if targetOrgID == "" {
				targetOrgID, err = selectOrgFromSession(session)
				if err != nil {
					return wrapDashboardError(err)
				}
			}

			if targetOrgID == session.CurrentOrg {
				_, _ = common.Green.Printf("✓ Already on organization: %s\n", formatSessionOrg(session, session.CurrentOrg))
				return nil
			}

			// Switch org via API
			ctx2, cancel2 := common.CreateContext()
			defer cancel2()

			var resp *domain.DashboardSwitchOrgResponse
			err = common.RunWithSpinner("Switching organization...", func() error {
				resp, err = authSvc.SwitchOrg(ctx2, targetOrgID)
				return err
			})
			if err != nil {
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Printf("✓ Switched to organization: %s\n", formatOrgLabel(resp.Org.PublicID, resp.Org.Name))
			return nil
		},
	}

	cmd.Flags().StringVar(&orgFlag, "org", "", "Organization public ID to switch to")

	return cmd
}

// selectOrgFromSession prompts the user to select an org from the session's relations.
func selectOrgFromSession(session *domain.DashboardSessionResponse) (string, error) {
	opts := make([]common.SelectOption[string], 0, len(session.Relations))
	for _, rel := range session.Relations {
		label := formatOrgLabel(rel.OrgPublicID, rel.OrgName)
		if rel.OrgPublicID == session.CurrentOrg {
			label += " (current)"
		}
		if rel.Role != "" {
			label += " [" + rel.Role + "]"
		}
		opts = append(opts, common.SelectOption[string]{Label: label, Value: rel.OrgPublicID})
	}

	return common.Select("Select organization", opts)
}

// formatSessionOrg returns a display label for an org in a session, looking up the name from relations.
func formatSessionOrg(session *domain.DashboardSessionResponse, orgPublicID string) string {
	for _, rel := range session.Relations {
		if rel.OrgPublicID == orgPublicID && rel.OrgName != "" {
			return formatOrgLabel(orgPublicID, rel.OrgName)
		}
	}
	return orgPublicID
}

// formatOrgLabel returns a display label for an org.
func formatOrgLabel(publicID, name string) string {
	if name != "" {
		return fmt.Sprintf("%s (%s)", name, publicID)
	}
	return publicID
}

// orgRow is a flat struct for table output of organizations.
type orgRow struct {
	PublicID string `json:"public_id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Current  string `json:"current"`
}

var orgColumns = []ports.Column{
	{Header: "PUBLIC ID", Field: "PublicID"},
	{Header: "NAME", Field: "Name"},
	{Header: "ROLE", Field: "Role"},
	{Header: "CURRENT", Field: "Current"},
}
