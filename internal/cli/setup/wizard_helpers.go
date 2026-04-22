package setup

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

var setupCallbackProvisioner = EnsureOAuthCallbackURI

// printComplete prints the final success message.
func printComplete() {
	fmt.Println()
	_, _ = common.Bold.Println("  ══════════════════════════════════════════")
	fmt.Println()
	_, _ = common.Green.Println("  ✓ Setup complete! You're ready to go.")
	fmt.Println()
	fmt.Println("  Try these commands:")
	fmt.Println("    nylas email list          List recent emails")
	fmt.Println("    nylas calendar events     Upcoming events")
	fmt.Println("    nylas auth status         Check configuration")
	fmt.Println()
	fmt.Println("  Documentation: https://cli.nylas.com/")
	fmt.Println()
}

// printStepRecovery prints manual recovery instructions when a step fails.
func printStepRecovery(step string, commands []string) {
	fmt.Println()
	_, _ = common.Yellow.Printf("  Could not complete %s setup automatically.\n", step)
	fmt.Println("  To continue manually:")
	for _, cmd := range commands {
		fmt.Printf("    %s\n", cmd)
	}
	fmt.Println()
}

// resolveProvider returns the SSO provider from flags, or empty string if not set.
func resolveProvider(opts wizardOpts) string {
	switch {
	case opts.google:
		return "google"
	case opts.microsoft:
		return "microsoft"
	case opts.github:
		return "github"
	default:
		return ""
	}
}

// chooseProvider presents an SSO provider menu.
func chooseProvider() (string, error) {
	return common.Select("How would you like to authenticate?", []common.SelectOption[string]{
		{Label: "Google (recommended)", Value: "google"},
		{Label: "Microsoft", Value: "microsoft"},
		{Label: "GitHub", Value: "github"},
	})
}

// selectApp prompts the user to select from multiple applications.
func selectApp(apps []domain.GatewayApplication) (domain.GatewayApplication, error) {
	opts := make([]common.SelectOption[int], len(apps))
	for i, app := range apps {
		opts[i] = common.SelectOption[int]{Label: appDisplayName(app), Value: i}
	}

	idx, err := common.Select("Select application", opts)
	if err != nil {
		return apps[0], nil
	}
	return apps[idx], nil
}

// createDefaultApp creates a new application with defaults.
func createDefaultApp(appSvc *dashboardapp.AppService, orgID string) (*domain.GatewayCreatedApplication, error) {
	fmt.Println("  No applications found. Creating one for you...")
	fmt.Println()

	name, err := common.InputPrompt("App name", "My First App")
	if err != nil {
		name = "My First App"
	}

	region, err := common.Select("Region", []common.SelectOption[string]{
		{Label: "US", Value: "us"},
		{Label: "EU", Value: "eu"},
	})
	if err != nil {
		region = "us"
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	var app *domain.GatewayCreatedApplication
	err = common.RunWithSpinner("Creating application...", func() error {
		app, err = appSvc.CreateApplication(ctx, orgID, region, name)
		return err
	})
	if err != nil {
		return nil, err
	}

	_, _ = common.Green.Printf("  ✓ Application created: %s (%s)\n", app.ApplicationID, region)
	return app, nil
}

// setActiveApp stores the active application in the keyring.
func setActiveApp(appID, region string) error {
	_, secrets, err := dashboard.CreateAuthService()
	if err != nil {
		return err
	}

	if err := secrets.Set(ports.KeyDashboardAppID, appID); err != nil {
		return err
	}
	return secrets.Set(ports.KeyDashboardAppRegion, region)
}

// appDisplayName returns a human-readable display name for an application.
func appDisplayName(app domain.GatewayApplication) string {
	name := ""
	if app.Branding != nil {
		name = app.Branding.Name
	}
	env := app.Environment
	if env == "" {
		env = "production"
	}

	displayID := app.ApplicationID
	if len(displayID) > 20 {
		displayID = displayID[:17] + "..."
	}

	if name != "" {
		return fmt.Sprintf("%s — %s (%s, %s)", name, displayID, env, app.Region)
	}
	return fmt.Sprintf("%s (%s, %s)", displayID, env, app.Region)
}

// verifyAPIKey checks that an API key works by listing applications.
func verifyAPIKey(apiKey, region string) error {
	client := nylasadapter.NewHTTPClient()
	client.SetRegion(region)
	client.SetCredentials("", "", apiKey)

	ctx, cancel := common.CreateContext()
	defer cancel()

	_, err := client.ListApplications(ctx)
	return err
}

func ensureSetupCallbackURI(apiKey, clientID, region string) error {
	if strings.TrimSpace(clientID) == "" {
		return fmt.Errorf("client ID is required to configure the OAuth callback URI")
	}

	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil || cfg == nil {
		cfg = domain.DefaultConfig()
	}

	result, err := setupCallbackProvisioner(apiKey, clientID, region, cfg.CallbackPort)
	if err != nil {
		printCallbackURIManualInstructions(result.RequiredURI, err)
		return nil
	}

	switch {
	case result.AlreadyExists:
		_, _ = common.Green.Println("  ✓ Callback URI already configured")
	case result.Created:
		_, _ = common.Green.Printf("  ✓ Added callback URI: %s\n", result.RequiredURI)
	}

	return nil
}

func printCallbackURIManualInstructions(requiredCallbackURI string, err error) {
	fmt.Println()
	fmt.Println("Setting up callback URI for OAuth authentication...")
	_, _ = common.Yellow.Printf("  Could not add callback URI automatically: %v\n", err)
	fmt.Printf("  Please add this callback URI manually in the Nylas dashboard:\n")
	fmt.Printf("    %s\n", requiredCallbackURI)
	fmt.Println()
	fmt.Printf("  Dashboard: https://dashboard.nylas.com/applications\n")
	fmt.Printf("  Navigate to: Your App → Settings → Callback URIs → Add URI\n")
}

// sanitizeAPIKey removes invisible characters from a pasted API key.
func sanitizeAPIKey(key string) string {
	var result strings.Builder
	result.Grow(len(key))
	for _, r := range key {
		if r >= ' ' && r <= '~' {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
