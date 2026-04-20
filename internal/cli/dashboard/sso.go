package dashboard

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/nylas/cli/internal/adapters/browser"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newSSOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sso",
		Short: "Authenticate via SSO",
		Long:  `Authenticate with the Nylas Dashboard using SSO (Google, Microsoft, or GitHub).`,
	}

	cmd.AddCommand(newSSOLoginCmd())
	cmd.AddCommand(newSSORegisterCmd())

	return cmd
}

func newSSOLoginCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in via SSO",
		Example: `  nylas dashboard sso login --provider google
  nylas dashboard sso login --provider microsoft
  nylas dashboard sso login --provider github`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSSO(provider, "login", false)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "google", "SSO provider (google, microsoft, github)")

	return cmd
}

func newSSORegisterCmd() *cobra.Command {
	var (
		provider                string
		acceptPrivacyPolicyFlag bool
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register via SSO",
		Example: `  nylas dashboard sso register --provider google
  nylas dashboard sso register --provider google --accept-privacy-policy`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := acceptPrivacyPolicy(acceptPrivacyPolicyFlag); err != nil {
				return err
			}
			return runSSO(provider, "register", true)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "google", "SSO provider (google, microsoft, github)")
	cmd.Flags().BoolVar(&acceptPrivacyPolicyFlag, "accept-privacy-policy", false, "Confirm that you accept the Nylas Privacy Policy")

	return cmd
}

func runSSO(provider, mode string, privacyPolicyAccepted bool, orgPublicIDs ...string) error {
	orgPublicID := ""
	if len(orgPublicIDs) > 0 {
		orgPublicID = orgPublicIDs[0]
	}

	loginType, err := mapProvider(provider)
	if err != nil {
		return err
	}

	authSvc, _, err := createAuthService()
	if err != nil {
		return wrapDashboardError(err)
	}

	ctx, cancel := common.CreateLongContext()
	defer cancel()

	var resp *domain.DashboardSSOStartResponse
	err = common.RunWithSpinner("Starting SSO...", func() error {
		resp, err = authSvc.SSOStart(ctx, loginType, mode, privacyPolicyAccepted)
		return err
	})
	if err != nil {
		return wrapDashboardError(err)
	}

	// Show the URL and code
	url := resp.VerificationURIComplete
	if url == "" {
		url = resp.VerificationURI
	}

	fmt.Println()
	_, _ = common.BoldCyan.Printf("  Open: %s\n", url)
	if resp.UserCode != "" && resp.VerificationURIComplete == "" {
		_, _ = common.Bold.Printf("  Code: %s\n", resp.UserCode)
	}
	fmt.Println()

	// Try to open browser
	b := browser.NewDefaultBrowser()
	if openErr := b.Open(url); openErr == nil {
		_, _ = common.Dim.Println("  Browser opened. Complete sign-in there.")
		fmt.Println()
	}

	// Poll with spinner
	interval := time.Duration(resp.Interval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}

	auth, err := pollSSO(ctx, authSvc, resp.FlowID, orgPublicID, interval)
	if err != nil {
		return wrapDashboardError(err)
	}

	if err := persistActiveOrg(authSvc, auth, orgPublicID); err != nil {
		rollbackPostAuthFailure(authSvc)
		return wrapDashboardError(err)
	}

	syncSessionOrgWithWarning(authSvc)

	printAuthSuccess(auth)
	return nil
}

func pollSSO(ctx context.Context, authSvc *dashboardapp.AuthService, flowID, orgPublicID string, interval time.Duration) (*domain.DashboardAuthResponse, error) {
	spinner := common.NewSpinner("Waiting for browser authentication...")
	spinner.Start()
	defer spinner.Stop()

	for {
		select {
		case <-ctx.Done():
			spinner.StopWithError("Timed out")
			return nil, fmt.Errorf("authentication timed out")
		case <-time.After(interval):
		}

		resp, err := authSvc.SSOPoll(ctx, flowID, orgPublicID)
		if err != nil {
			spinner.StopWithError("Failed")
			return nil, err
		}

		switch resp.Status {
		case domain.SSOStatusComplete:
			spinner.StopWithSuccess("Authenticated!")
			if resp.Auth != nil {
				return resp.Auth, nil
			}
			return nil, fmt.Errorf("unexpected empty auth response")

		case domain.SSOStatusMFARequired:
			spinner.Stop()
			if resp.MFA == nil {
				return nil, fmt.Errorf("unexpected empty MFA response")
			}

			mfaOrg, resolveErr := resolveSSOMFAOrg(orgPublicID, resp.MFA.Organizations)
			if resolveErr != nil {
				return nil, resolveErr
			}
			code, readErr := common.PasswordPrompt("MFA code")
			if readErr != nil {
				return nil, readErr
			}

			ctx2, cancel := common.CreateContext()
			var auth *domain.DashboardAuthResponse
			mfaErr := common.RunWithSpinner("Verifying MFA...", func() error {
				auth, err = authSvc.CompleteMFA(ctx2, resp.MFA.User.PublicID, code, mfaOrg)
				return err
			})
			cancel()
			if mfaErr != nil {
				return nil, mfaErr
			}
			return auth, nil

		case domain.SSOStatusAccessDenied:
			spinner.StopWithError("Access denied")
			return nil, fmt.Errorf("%w: access denied by provider", domain.ErrDashboardSSOFailed)

		case domain.SSOStatusExpired:
			spinner.StopWithError("Device code expired")
			return nil, fmt.Errorf("%w: device code expired, please try again", domain.ErrDashboardSSOFailed)

		case domain.SSOStatusPending:
			if resp.RetryAfter > 0 {
				interval = time.Duration(resp.RetryAfter) * time.Second
			}
		}
	}
}

func resolveSSOMFAOrg(orgPublicID string, orgs []domain.DashboardOrganization) (string, error) {
	if orgPublicID != "" || len(orgs) == 0 {
		return orgPublicID, nil
	}
	if len(orgs) == 1 {
		return orgs[0].PublicID, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", dashboardError(
			"multiple organizations available for MFA",
			"Pass --org to choose the organization",
		)
	}

	opts := make([]common.SelectOption[string], len(orgs))
	for i, org := range orgs {
		label := org.Name
		if label == "" {
			label = org.PublicID
		}
		opts[i] = common.SelectOption[string]{Label: label, Value: org.PublicID}
	}

	selected, err := common.Select("Select organization", opts)
	if err != nil {
		return "", err
	}
	return selected, nil
}

// mapProvider maps a user-friendly provider name to the server login type.
func mapProvider(provider string) (string, error) {
	switch strings.ToLower(provider) {
	case "google":
		return domain.SSOLoginTypeGoogle, nil
	case "microsoft":
		return domain.SSOLoginTypeMicrosoft, nil
	case "github":
		return domain.SSOLoginTypeGitHub, nil
	default:
		return "", dashboardError(
			fmt.Sprintf("unsupported SSO provider: %s", provider),
			"Use one of: google, microsoft, github",
		)
	}
}
