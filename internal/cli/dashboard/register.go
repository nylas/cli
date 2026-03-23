package dashboard

import (
	"github.com/spf13/cobra"
)

func newRegisterCmd() *cobra.Command {
	var (
		google    bool
		microsoft bool
		github    bool
	)

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Create a new Nylas Dashboard account",
		Long: `Register a new Nylas Dashboard account using SSO.

Email/password registration is temporarily disabled. Use SSO instead.`,
		Example: `  # Interactive — choose SSO provider
  nylas dashboard register

  # Google SSO (non-interactive)
  nylas dashboard register --google

  # Microsoft SSO
  nylas dashboard register --microsoft

  # GitHub SSO
  nylas dashboard register --github`,
		RunE: func(cmd *cobra.Command, args []string) error {
			method, err := resolveAuthMethod(google, microsoft, github, false, "register")
			if err != nil {
				return wrapDashboardError(err)
			}

			switch method {
			case methodGoogle, methodMicrosoft, methodGitHub:
				return runSSORegister(method)
			default:
				return dashboardError("invalid selection", "Choose a valid SSO provider")
			}
		},
	}

	cmd.Flags().BoolVar(&google, "google", false, "Register with Google SSO")
	cmd.Flags().BoolVar(&microsoft, "microsoft", false, "Register with Microsoft SSO")
	cmd.Flags().BoolVar(&github, "github", false, "Register with GitHub SSO")

	return cmd
}

func runSSORegister(provider string) error {
	if err := acceptPrivacyPolicy(); err != nil {
		return err
	}
	return runSSO(provider, "register", true)
}

