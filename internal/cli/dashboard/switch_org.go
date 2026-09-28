package dashboard

import (
	"context"
	"fmt"
	"io"

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
or pass --org to switch directly.

A session from 'nylas oauth login' belongs to the organization chosen on the
sign-in page, so switching opens the browser to sign in again: choose the
organization there. --org then checks that the one chosen is the one named.`,
		Example: `  # Interactive — choose from your orgs
  nylas dashboard orgs switch

  # Switch directly by org ID
  nylas dashboard orgs switch --org org_abc123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			authSvc, _, err := createAuthService()
			if err != nil {
				return wrapDashboardError(err)
			}
			if authSvc.IsOAuthSession() {
				ctx, cancel := common.CreateLongContext()
				defer cancel()
				return switchOAuthSessionOrg(ctx, cmd.OutOrStdout(), orgFlag, OAuthRelogin)
			}
			if orgFlag == "" && !isInteractive() {
				return dashboardError(
					"organization selection requires an interactive terminal",
					"Pass --org to choose the organization in non-interactive runs",
				)
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

// switchOAuthSessionOrg changes the organization of a session from `nylas
// oauth login`. The server binds that session to the organization picked on
// its consent screen and refuses to switch it, so the switch is a new sign-in.
func switchOAuthSessionOrg(
	ctx context.Context,
	out io.Writer,
	orgFlag string,
	relogin func(context.Context) (*domain.DashboardOAuthExchangeResponse, error),
) error {
	if relogin == nil {
		return dashboardError(
			"this session came from `nylas oauth login` and its organization is chosen at sign-in",
			"Run `nylas oauth login` again and choose the organization there",
		)
	}

	_, _ = fmt.Fprintln(out, "This session came from `nylas oauth login`, which is tied to the organization chosen at sign-in.")
	_, _ = fmt.Fprintln(out, "Opening your browser to sign in again: choose the organization there.")

	resp, err := relogin(ctx)
	if err != nil {
		return wrapDashboardError(err)
	}

	if orgFlag != "" && resp.OrgPublicID != orgFlag {
		return dashboardError(
			fmt.Sprintf("signed in to organization %s, not %s", resp.OrgPublicID, orgFlag),
			"Run `nylas dashboard orgs switch` again and choose "+orgFlag+" on the sign-in page",
		)
	}

	_, _ = common.Green.Fprintf(out, "✓ Switched to organization: %s\n", resp.OrgPublicID)
	return nil
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
