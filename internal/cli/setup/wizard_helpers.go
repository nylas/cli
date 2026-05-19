package setup

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/adapters/oauth"
	authapp "github.com/nylas/cli/internal/app/auth"
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

	displayID := common.Truncate(app.ApplicationID, 20)

	if name != "" {
		return fmt.Sprintf("%s — %s (%s, %s)", name, displayID, env, app.Region)
	}
	return fmt.Sprintf("%s (%s, %s)", displayID, env, app.Region)
}

// verifyAPIKey checks that an API key works by listing applications.
func verifyAPIKey(apiKey, region string) error {
	client := nylasadapter.NewHTTPClient()
	configStore := config.NewDefaultFileStore()
	cfg, _ := configStore.Load()
	if cfg != nil && cfg.API != nil && cfg.API.BaseURL != "" {
		client.ApplyConfig(cfg)
	} else {
		client.SetRegion(region)
	}
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
	fmt.Printf("  Dashboard: https://dashboard-v3.nylas.com/applications\n")
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

// promptAuthLogin asks the user to connect an email account.
// Returns nil grant if the user declines.
func promptAuthLogin(configStore ports.ConfigStore, grantStore ports.GrantStore) (*domain.Grant, error) {
	yes, err := common.ConfirmPrompt("Connect an email account now?", true)
	if err != nil || !yes {
		return nil, nil
	}

	provider, err := common.Select("Which provider?", []common.SelectOption[domain.Provider]{
		{Label: "Google (Gmail)", Value: domain.ProviderGoogle},
		{Label: "Microsoft (Outlook)", Value: domain.ProviderMicrosoft},
		{Label: "Exchange on-premises (EWS)", Value: domain.ProviderEWS},
		{Label: "iCloud", Value: domain.ProviderICloud},
		{Label: "Yahoo", Value: domain.ProviderYahoo},
		{Label: "IMAP (other)", Value: domain.ProviderIMAP},
	})
	if err != nil {
		return nil, err
	}

	authSvc, err := buildAuthService(configStore, grantStore)
	if err != nil {
		return nil, err
	}

	switch provider {
	case domain.ProviderGoogle, domain.ProviderMicrosoft, domain.ProviderEWS:
		fmt.Println()
		fmt.Println("  Opening browser for authentication...")
		fmt.Println("  Complete the sign-in process in your browser.")

		ctx, cancel := common.CreateLongContext()
		defer cancel()

		return authSvc.Login(ctx, provider)
	default:
		return initCredentialLogin(authSvc, provider)
	}
}

func initCredentialLogin(authSvc *authapp.Service, provider domain.Provider) (*domain.Grant, error) {
	var settings map[string]any
	var apiProvider string
	var err error

	switch provider {
	case domain.ProviderICloud:
		fmt.Println()
		_, _ = common.Dim.Println("  iCloud requires an app-specific password.")
		_, _ = common.Dim.Println("  Generate one at: https://appleid.apple.com/account/manage")
		fmt.Println()

		username, promptErr := common.InputPrompt("iCloud email", "")
		if promptErr != nil {
			return nil, promptErr
		}
		password, promptErr := common.PasswordPrompt("App-specific password")
		if promptErr != nil {
			return nil, promptErr
		}
		settings = map[string]any{"username": username, "password": password}
		apiProvider = "icloud"

	case domain.ProviderYahoo:
		fmt.Println()
		_, _ = common.Dim.Println("  Yahoo requires an app password.")
		_, _ = common.Dim.Println("  Generate one at: https://login.yahoo.com/account/security/app-passwords")
		fmt.Println()

		email, promptErr := common.InputPrompt("Yahoo email", "")
		if promptErr != nil {
			return nil, promptErr
		}
		password, promptErr := common.PasswordPrompt("App password")
		if promptErr != nil {
			return nil, promptErr
		}
		settings = map[string]any{
			"imap_username": email,
			"imap_password": password,
			"imap_host":     "imap.mail.yahoo.com",
			"imap_port":     993,
			"type":          "yahoo",
		}
		apiProvider = "imap"

	case domain.ProviderIMAP:
		fmt.Println()

		username, promptErr := common.InputPrompt("IMAP username (email)", "")
		if promptErr != nil {
			return nil, promptErr
		}
		password, promptErr := common.PasswordPrompt("IMAP password")
		if promptErr != nil {
			return nil, promptErr
		}
		host, promptErr := common.InputPrompt("IMAP host", "")
		if promptErr != nil {
			return nil, promptErr
		}
		settings = map[string]any{
			"imap_username": username,
			"imap_password": password,
			"imap_host":     host,
			"imap_port":     promptPortDefault("IMAP port", 993),
		}
		apiProvider = "imap"

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	ctx, cancel := common.CreateLongContext()
	defer cancel()

	var grant *domain.Grant
	err = common.RunWithSpinner("Authenticating...", func() error {
		grant, err = authSvc.LoginWithCredentials(ctx, apiProvider, settings)
		return err
	})
	if err != nil {
		return nil, err
	}
	return grant, nil
}

func buildAuthService(configStore ports.ConfigStore, grantStore ports.GrantStore) (*authapp.Service, error) {
	cfg, _ := configStore.Load()
	if cfg == nil {
		cfg = domain.DefaultConfig()
	}

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, fmt.Errorf("could not access keyring: %w", err)
	}

	client := nylasadapter.NewHTTPClient()
	client.ApplyConfig(cfg)
	apiKey, _ := secretStore.Get(ports.KeyAPIKey)
	clientID, _ := secretStore.Get(ports.KeyClientID)
	clientSecret, _ := secretStore.Get(ports.KeyClientSecret)
	client.SetCredentials(clientID, clientSecret, apiKey)

	oauthServer := oauth.NewCallbackServer(cfg.CallbackPort)
	b := browser.NewDefaultBrowser()
	return authapp.NewService(client, grantStore, configStore, oauthServer, b), nil
}

func promptPortDefault(title string, defaultPort int) int {
	raw, err := common.InputPrompt(title, strconv.Itoa(defaultPort))
	if err != nil || strings.TrimSpace(raw) == "" {
		return defaultPort
	}
	port, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || port <= 0 || port > 65535 {
		return defaultPort
	}
	return port
}
