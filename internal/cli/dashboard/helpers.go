package dashboard

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

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

// readPassword prompts for a password without terminal echo.
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	pwBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(string(pwBytes)), nil
}

// readLine prompts for a line of text input.
func readLine(prompt string) (string, error) {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(input), nil
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
		return methodEmailPassword, nil
	default:
		return chooseAuthMethod(action)
	}
}

// chooseAuthMethod presents an interactive menu. SSO first.
func chooseAuthMethod(action string) (string, error) {
	fmt.Printf("\nHow would you like to %s?\n\n", action)
	_, _ = common.Cyan.Println("  [1] Google      (recommended)")
	fmt.Println("  [2] Microsoft")
	fmt.Println("  [3] GitHub")
	_, _ = common.Dim.Println("  [4] Email and password")
	fmt.Println()

	choice, err := readLine("Choose [1-4]: ")
	if err != nil {
		return "", err
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return methodGoogle, nil
	case "2":
		return methodMicrosoft, nil
	case "3":
		return methodGitHub, nil
	case "4":
		return methodEmailPassword, nil
	default:
		return "", dashboardError("invalid selection", "Choose 1-4")
	}
}

// selectOrg prompts the user to select an organization if multiple are available.
func selectOrg(orgs []domain.DashboardOrganization) string {
	if len(orgs) <= 1 {
		if len(orgs) == 1 {
			return orgs[0].PublicID
		}
		return ""
	}

	fmt.Println("\nAvailable organizations:")
	for i, org := range orgs {
		name := org.Name
		if name == "" {
			name = org.PublicID
		}
		fmt.Printf("  [%d] %s\n", i+1, name)
	}
	fmt.Println()

	choice, err := readLine(fmt.Sprintf("Select organization [1-%d]: ", len(orgs)))
	if err != nil {
		return orgs[0].PublicID
	}

	var selected int
	if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(orgs) {
		return orgs[0].PublicID
	}
	return orgs[selected-1].PublicID
}

// printAuthSuccess prints the standard post-login success message.
func printAuthSuccess(auth *domain.DashboardAuthResponse) {
	_, _ = common.Green.Printf("✓ Authenticated as %s\n", auth.User.PublicID)
	if len(auth.Organizations) > 0 {
		fmt.Printf("  Organization: %s\n", auth.Organizations[0].PublicID)
	}
}

// acceptPrivacyPolicy prompts for or validates privacy policy acceptance.
func acceptPrivacyPolicy() error {
	if !common.Confirm("Accept Nylas Privacy Policy?", true) {
		return dashboardError("privacy policy must be accepted to continue", "")
	}
	return nil
}
