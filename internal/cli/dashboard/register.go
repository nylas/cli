package dashboard

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newRegisterCmd() *cobra.Command {
	var (
		region    string
		google    bool
		microsoft bool
		github    bool
		emailFlag bool
		userFlag  string
		passFlag  string
		codeFlag  string
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Create a new Nylas Dashboard account",
		Long: `Register a new Nylas Dashboard account.

Choose SSO (recommended) or email/password. Pass a flag to skip the menu.`,
		Example: `  # Interactive — choose method
  nylas dashboard register

  # Google SSO (non-interactive)
  nylas dashboard register --google

  # Email/password fully non-interactive
  nylas dashboard register --email --user me@co.com --password s3cret --code AB12CD34 --region us`,
		RunE: func(cmd *cobra.Command, args []string) error {
			method, err := resolveAuthMethod(google, microsoft, github, emailFlag, "register")
			if err != nil {
				return wrapDashboardError(err)
			}

			switch method {
			case methodGoogle, methodMicrosoft, methodGitHub:
				return runSSORegister(method)
			case methodEmailPassword:
				return runEmailRegister(userFlag, passFlag, codeFlag, region)
			default:
				return dashboardError("invalid selection", "Choose a valid option")
			}
		},
	}

	cmd.Flags().BoolVar(&google, "google", false, "Register with Google SSO")
	cmd.Flags().BoolVar(&microsoft, "microsoft", false, "Register with Microsoft SSO")
	cmd.Flags().BoolVar(&github, "github", false, "Register with GitHub SSO")
	cmd.Flags().BoolVar(&emailFlag, "email", false, "Register with email and password")
	cmd.Flags().StringVarP(&region, "region", "r", "us", "Account region (us or eu)")
	cmd.Flags().StringVar(&userFlag, "user", "", "Email address (non-interactive)")
	cmd.Flags().StringVar(&passFlag, "password", "", "Password (non-interactive, use with care)")
	cmd.Flags().StringVar(&codeFlag, "code", "", "Verification code (non-interactive, skip prompt)")

	return cmd
}

func runSSORegister(provider string) error {
	if err := acceptPrivacyPolicy(); err != nil {
		return err
	}
	return runSSO(provider, "register", true)
}

func runEmailRegister(userFlag, passFlag, codeFlag, region string) error {
	if err := acceptPrivacyPolicy(); err != nil {
		return err
	}

	authSvc, _, err := createAuthService()
	if err != nil {
		return wrapDashboardError(err)
	}

	email := userFlag
	if email == "" {
		email, err = readLine("Email: ")
		if err != nil {
			return wrapDashboardError(err)
		}
	}
	if email == "" {
		return dashboardError("email is required", "Use --user or enter at prompt")
	}

	password := passFlag
	if password == "" {
		password, err = readPassword("Password: ")
		if err != nil {
			return wrapDashboardError(err)
		}
		confirm, cErr := readPassword("Confirm password: ")
		if cErr != nil {
			return wrapDashboardError(cErr)
		}
		if password != confirm {
			return dashboardError("passwords do not match", "Try again")
		}
	}
	if password == "" {
		return dashboardError("password is required", "Use --password or enter at prompt")
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	var resp *domain.DashboardRegisterResponse
	err = common.RunWithSpinner("Creating account...", func() error {
		resp, err = authSvc.Register(ctx, email, password, true)
		return err
	})
	if err != nil {
		return wrapDashboardError(err)
	}

	_, _ = common.Green.Println("✓ Verification code sent to your email")
	_, _ = common.Dim.Printf("  Expires: %s\n", resp.ExpiresAt)

	code := codeFlag
	if code == "" {
		fmt.Println()
		code, err = readLine("Enter verification code: ")
		if err != nil {
			return wrapDashboardError(err)
		}
	}
	if code == "" {
		return dashboardError("verification code is required", "Check your email, or use --code")
	}

	ctx2, cancel2 := common.CreateContext()
	defer cancel2()

	var authResp *domain.DashboardAuthResponse
	err = common.RunWithSpinner("Verifying...", func() error {
		authResp, err = authSvc.VerifyEmailCode(ctx2, email, code, region)
		return err
	})
	if err != nil {
		return wrapDashboardError(err)
	}

	if len(authResp.Organizations) > 1 {
		orgID := selectOrg(authResp.Organizations)
		_ = authSvc.SetActiveOrg(orgID)
	}

	printAuthSuccess(authResp)
	fmt.Println("\nNext steps:")
	fmt.Println("  nylas dashboard apps list      List your applications")
	fmt.Println("  nylas dashboard apps create    Create a new application")

	return nil
}
