package dashboard

import (
	"errors"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/adapters/dpop"
	"github.com/nylas/cli/internal/adapters/keyring"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// createDPoPService creates a DPoP service backed by the keyring.
func createDPoPService() (ports.DPoP, ports.SecretStore, error) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, nil, err
	}

	dpopSvc, err := dpop.New(secretStore)
	if err != nil {
		return nil, nil, err
	}

	return dpopSvc, secretStore, nil
}

// createAuthService creates the full dashboard auth service chain.
func createAuthService() (*dashboardapp.AuthService, ports.SecretStore, error) {
	dpopSvc, secretStore, err := createDPoPService()
	if err != nil {
		return nil, nil, err
	}

	baseURL := getDashboardAccountBaseURL(secretStore)
	accountClient := dashboard.NewAccountClient(baseURL, dpopSvc)

	return dashboardapp.NewAuthService(accountClient, secretStore), secretStore, nil
}

// createAppService creates the dashboard app management service.
func createAppService() (*dashboardapp.AppService, error) {
	dpopSvc, secretStore, err := createDPoPService()
	if err != nil {
		return nil, err
	}

	gatewayClient := dashboard.NewGatewayClient(dpopSvc)
	return dashboardapp.NewAppService(gatewayClient, secretStore), nil
}

// getDashboardAccountBaseURL returns the dashboard-account base URL.
// Priority: NYLAS_DASHBOARD_ACCOUNT_URL env var > config file > default.
func getDashboardAccountBaseURL(secrets ports.SecretStore) string {
	if envURL := os.Getenv("NYLAS_DASHBOARD_ACCOUNT_URL"); envURL != "" {
		return envURL
	}
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err == nil && cfg.Dashboard != nil && cfg.Dashboard.AccountBaseURL != "" {
		return cfg.Dashboard.AccountBaseURL
	}
	return domain.DefaultDashboardAccountBaseURL
}

// wrapDashboardError wraps a dashboard error as a CLIError, preserving
// the actual error message.
func wrapDashboardError(err error) error {
	if err == nil {
		return nil
	}
	var cliErr *common.CLIError
	if errors.As(err, &cliErr) {
		return cliErr
	}
	return &common.CLIError{
		Err:     err,
		Message: err.Error(),
	}
}

// dashboardError creates a user-friendly error with the hint included in the
// message itself, so it's always visible regardless of how the error is displayed.
func dashboardError(message, hint string) error {
	if hint != "" {
		message = message + "\n  Hint: " + hint
	}
	return &common.CLIError{Message: message}
}

// Auth method resolution: flags take priority, then interactive menu.
const (
	methodGoogle        = "google"
	methodMicrosoft     = "microsoft"
	methodGitHub        = "github"
	methodEmailPassword = "email"
)

// resolveAuthMethod determines the auth method from flags or prompts interactively.
func resolveAuthMethod(google, microsoft, github, email bool, action string) (string, error) {
	// Count how many flags were set
	set := 0
	if google {
		set++
	}
	if microsoft {
		set++
	}
	if github {
		set++
	}
	if email {
		set++
	}
	if set > 1 {
		return "", dashboardError("only one auth method flag allowed", "Use --google, --microsoft, --github, or --email")
	}

	switch {
	case google:
		return methodGoogle, nil
	case microsoft:
		return methodMicrosoft, nil
	case github:
		return methodGitHub, nil
	case email:
		if action == "register" {
			return "", dashboardError("email/password registration is temporarily disabled", "Use SSO instead: --google, --microsoft, or --github")
		}
		return methodEmailPassword, nil
	default:
		return chooseAuthMethod(action)
	}
}

// chooseAuthMethod presents an interactive menu. SSO first.
// Email/password registration is temporarily disabled.
func chooseAuthMethod(action string) (string, error) {
	opts := []common.SelectOption[string]{
		{Label: "Google (recommended)", Value: methodGoogle},
		{Label: "Microsoft", Value: methodMicrosoft},
		{Label: "GitHub", Value: methodGitHub},
	}
	if action != "register" {
		opts = append(opts, common.SelectOption[string]{Label: "Email and password", Value: methodEmailPassword})
	}

	return common.Select(fmt.Sprintf("How would you like to %s?", action), opts)
}

// selectOrg prompts the user to select an organization if multiple are available.
func selectOrg(orgs []domain.DashboardOrganization) string {
	if len(orgs) <= 1 {
		if len(orgs) == 1 {
			return orgs[0].PublicID
		}
		return ""
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
		return orgs[0].PublicID
	}
	return selected
}

func persistActiveOrg(authSvc *dashboardapp.AuthService, auth *domain.DashboardAuthResponse, orgPublicID string) error {
	selectedOrgID := orgPublicID
	if selectedOrgID == "" && len(auth.Organizations) > 1 {
		selectedOrgID = selectOrg(auth.Organizations)
	}
	if selectedOrgID == "" {
		return nil
	}

	// Apply the selection to the server-side dashboard session so follow-up
	// commands use the same org the user chose during login.
	switchCtx, switchCancel := common.CreateContext()
	defer switchCancel()

	if _, err := authSvc.SwitchOrg(switchCtx, selectedOrgID); err != nil {
		return fmt.Errorf("failed to switch organization: %w", err)
	}
	return nil
}

// printAuthSuccess prints the standard post-login success message.
// It reads the stored active org from the keyring (set by SyncSessionOrg)
// so it reflects the server's actual current org.
func printAuthSuccess(auth *domain.DashboardAuthResponse) {
	_, _ = common.Green.Printf("✓ Authenticated as %s\n", auth.User.PublicID)

	// Show the active org from keyring (most accurate after SyncSessionOrg)
	orgID := ""
	if _, secrets, err := createDPoPService(); err == nil {
		orgID, _ = secrets.Get(ports.KeyDashboardOrgPublicID)
	}
	if orgID == "" && len(auth.Organizations) == 1 {
		orgID = auth.Organizations[0].PublicID
	}

	if orgID != "" {
		// Find the org name if available
		orgLabel := orgID
		for _, org := range auth.Organizations {
			if org.PublicID == orgID && org.Name != "" {
				orgLabel = fmt.Sprintf("%s (%s)", org.Name, orgID)
				break
			}
		}
		fmt.Printf("  Organization: %s\n", orgLabel)
	}

	if len(auth.Organizations) > 1 {
		fmt.Printf("  Available orgs: %d (switch with: nylas dashboard orgs switch)\n", len(auth.Organizations))
	}
}

func syncSessionOrgWithWarning(authSvc *dashboardapp.AuthService) {
	syncCtx, syncCancel := common.CreateContext()
	defer syncCancel()

	if err := authSvc.SyncSessionOrg(syncCtx); err != nil {
		common.PrintWarning("authenticated, but failed to sync the active dashboard organization: %v", err)
	}
}

// acceptPrivacyPolicy prompts for or validates privacy policy acceptance.
func acceptPrivacyPolicy() error {
	accepted, err := common.ConfirmPrompt("Accept Nylas Privacy Policy?", true)
	if err != nil {
		return err
	}
	if !accepted {
		return dashboardError("privacy policy must be accepted to continue", "")
	}
	return nil
}
