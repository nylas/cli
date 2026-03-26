package dashboard

import (
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newLoginCmd() *cobra.Command {
	var (
		orgPublicID string
		google      bool
		microsoft   bool
		github      bool
		emailFlag   bool
		userFlag    string
		passFlag    string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in to your Nylas Dashboard account",
		Long: `Authenticate with the Nylas Dashboard.

Choose SSO (recommended) or email/password. Pass a flag to skip the menu.`,
		Example: `  # Interactive — choose auth method
  nylas dashboard login

  # Google SSO (non-interactive)
  nylas dashboard login --google

  # Email/password (non-interactive)
  nylas dashboard login --email --user user@example.com --password secret

  # Login to a specific organization
  nylas dashboard login --google --org org_123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			method, err := resolveAuthMethod(google, microsoft, github, emailFlag, "log in")
			if err != nil {
				return wrapDashboardError(err)
			}

			switch method {
			case methodGoogle, methodMicrosoft, methodGitHub:
				return runSSO(method, "login", false, orgPublicID)
			case methodEmailPassword:
				return runEmailLogin(userFlag, passFlag, orgPublicID)
			default:
				return dashboardError("invalid selection", "Choose a valid option")
			}
		},
	}

	cmd.Flags().BoolVar(&google, "google", false, "Log in with Google SSO")
	cmd.Flags().BoolVar(&microsoft, "microsoft", false, "Log in with Microsoft SSO")
	cmd.Flags().BoolVar(&github, "github", false, "Log in with GitHub SSO")
	cmd.Flags().BoolVar(&emailFlag, "email", false, "Log in with email and password")
	cmd.Flags().StringVar(&orgPublicID, "org", "", "Organization public ID")
	cmd.Flags().StringVar(&userFlag, "user", "", "Email address (non-interactive)")
	cmd.Flags().StringVar(&passFlag, "password", "", "Password (non-interactive, use with care)")

	return cmd
}

func runEmailLogin(userFlag, passFlag, orgPublicID string) error {
	authSvc, _, err := createAuthService()
	if err != nil {
		return wrapDashboardError(err)
	}

	email := userFlag
	if email == "" {
		email, err = common.InputPrompt("Email", "")
		if err != nil {
			return wrapDashboardError(err)
		}
	}
	if email == "" {
		return dashboardError("email is required", "Use --user or enter at prompt")
	}

	password := passFlag
	if password == "" {
		password, err = common.PasswordPrompt("Password")
		if err != nil {
			return wrapDashboardError(err)
		}
	}
	if password == "" {
		return dashboardError("password is required", "Use --password or enter at prompt")
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	var auth *domain.DashboardAuthResponse
	var mfa *domain.DashboardMFARequired

	err = common.RunWithSpinner("Authenticating...", func() error {
		auth, mfa, err = authSvc.Login(ctx, email, password, orgPublicID)
		return err
	})
	if err != nil {
		return wrapDashboardError(err)
	}

	if mfa != nil {
		code, readErr := common.PasswordPrompt("MFA code")
		if readErr != nil {
			return wrapDashboardError(readErr)
		}
		if code == "" {
			return dashboardError("MFA code is required", "Enter the code from your authenticator app")
		}

		mfaOrg := orgPublicID
		if mfaOrg == "" && len(mfa.Organizations) > 0 {
			if len(mfa.Organizations) > 1 {
				mfaOrg = selectOrg(mfa.Organizations)
			} else {
				mfaOrg = mfa.Organizations[0].PublicID
			}
		}

		ctx2, cancel2 := common.CreateContext()
		defer cancel2()

		err = common.RunWithSpinner("Verifying MFA...", func() error {
			auth, err = authSvc.CompleteMFA(ctx2, mfa.User.PublicID, code, mfaOrg)
			return err
		})
		if err != nil {
			return wrapDashboardError(err)
		}
	}

	if orgPublicID == "" && len(auth.Organizations) > 1 {
		orgID := selectOrg(auth.Organizations)
		_ = authSvc.SetActiveOrg(orgID)
	}

	// Sync the actual active org from the server session
	syncCtx, syncCancel := common.CreateContext()
	defer syncCancel()
	_ = authSvc.SyncSessionOrg(syncCtx)

	printAuthSuccess(auth)
	return nil
}
