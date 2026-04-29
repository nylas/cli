package setup

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// wizardOpts holds the options parsed from CLI flags.
type wizardOpts struct {
	apiKey    string
	clientID  string
	region    string
	google    bool
	microsoft bool
	github    bool
}

// pathChoice represents the user's initial choice in the wizard.
type pathChoice int

const (
	pathRegister pathChoice = iota + 1
	pathLogin
	pathAPIKey
)

const (
	stepTotal = 4
	divider   = "──────────────────────────────────────────"
)

var (
	verifyAPIKeyFn             = verifyAPIKey
	resolveAPIKeyApplicationFn = ResolveAPIKeyApplication
	ensureSetupCallbackURIFn   = ensureSetupCallbackURI
	activateAPIKeyFn           = dashboard.ActivateAPIKey
	getSetupStatusFn           = GetSetupStatus
	stepGrantSyncFn            = stepGrantSync
	printCompleteFn            = printComplete
)

func runWizard(opts wizardOpts) error {
	fmt.Println()
	_, _ = common.Bold.Println("  Welcome to Nylas! Let's get you set up.")
	fmt.Println()

	status := GetSetupStatus()

	// Non-interactive: --api-key was provided.
	if opts.apiKey != "" {
		return runNonInteractive(opts, status)
	}

	// Interactive: must be a TTY.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		common.PrintError("--api-key is required in non-interactive mode")
		fmt.Println()
		fmt.Println("  Usage: nylas init --api-key <key> [--region us|eu]")
		return fmt.Errorf("non-interactive mode requires --api-key")
	}

	// Step 1: Account
	if err := stepAccount(opts, &status); err != nil {
		return err
	}

	// Step 2: Application (skipped for API key path — handled in stepAccount)
	if err := stepApplication(&status); err != nil {
		printStepRecovery("application", []string{
			"nylas dashboard apps list",
			"nylas dashboard apps create --name 'My App' --region us",
		})
		return fmt.Errorf("application setup failed: %w", err)
	}

	// Step 3: API Key
	if err := stepAPIKey(&status); err != nil {
		printStepRecovery("API key", []string{
			"nylas dashboard apps apikeys create",
		})
		return fmt.Errorf("API key setup failed: %w", err)
	}

	// Step 4: Grants
	stepGrantSync(&status)

	// Done!
	printComplete()
	return nil
}

// runNonInteractive handles the --api-key flag path with no prompts.
func runNonInteractive(opts wizardOpts, status SetupStatus) error {
	if status.HasAPIKey {
		_, _ = common.Yellow.Println("  Updating existing API key configuration")
	}

	region := opts.region
	if region == "" {
		region = "us"
	}

	apiKey := sanitizeAPIKey(opts.apiKey)

	fmt.Println()
	var verifyErr error
	_ = common.RunWithSpinner("Verifying API key...", func() error {
		verifyErr = verifyAPIKeyFn(apiKey, region)
		return verifyErr
	})
	if verifyErr != nil {
		common.PrintError("Invalid API key: %v", verifyErr)
		return verifyErr
	}
	_, _ = common.Green.Println("  ✓ API key is valid")

	selection, err := resolveAPIKeyApplicationFn(apiKey, region, opts.clientID, false)
	if err != nil {
		return err
	}

	if err := ensureSetupCallbackURIFn(apiKey, selection.ClientID, region); err != nil {
		return err
	}

	if err := activateAPIKeyFn(apiKey, selection.ClientID, region, selection.OrgID); err != nil {
		common.PrintError("Could not activate API key: %v", err)
		return err
	}
	_, _ = common.Green.Println("  ✓ Configuration saved")

	// Refresh status after activation.
	status = getSetupStatusFn()
	stepGrantSyncFn(&status)
	printCompleteFn()
	return nil
}

// stepAccount handles Step 1: account registration, login, or API key entry.
func stepAccount(opts wizardOpts, status *SetupStatus) error {
	_, _ = common.Dim.Printf("  %s\n", divider)
	fmt.Println()
	_, _ = common.Bold.Printf("  Step 1 of %d: Account\n", stepTotal)
	fmt.Println()

	if status.HasDashboardAuth {
		_, _ = common.Green.Println("  ✓ Already logged in to Nylas Dashboard")
		return nil
	}
	if status.HasAPIKey {
		_, _ = common.Green.Println("  ✓ API key already configured")
		return nil
	}

	// Determine the path.
	path, err := chooseAccountPath(opts)
	if err != nil {
		return err
	}

	switch path {
	case pathRegister:
		return accountSSO(opts, "register")
	case pathLogin:
		return accountSSO(opts, "login")
	case pathAPIKey:
		return accountAPIKey(opts, status)
	}
	return nil
}

// chooseAccountPath presents the three-option menu or resolves from flags.
func chooseAccountPath(opts wizardOpts) (pathChoice, error) {
	// If SSO flag was provided, determine register vs login.
	if opts.google || opts.microsoft || opts.github {
		return pathLogin, nil
	}

	return common.Select("Do you have a Nylas account?", []common.SelectOption[pathChoice]{
		{Label: "No, create one (free)", Value: pathRegister},
		{Label: "Yes, log me in", Value: pathLogin},
		{Label: "I already have an API key", Value: pathAPIKey},
	})
}

// accountSSO handles SSO registration or login.
func accountSSO(opts wizardOpts, mode string) error {
	if mode == "register" {
		if err := dashboard.AcceptPrivacyPolicy(); err != nil {
			return err
		}
	}

	provider := resolveProvider(opts)
	if provider == "" {
		var err error
		provider, err = chooseProvider()
		if err != nil {
			return err
		}
	}

	return dashboard.RunSSO(provider, mode, mode == "register")
}

// accountAPIKey handles the "I have an API key" path.
func accountAPIKey(opts wizardOpts, status *SetupStatus) error {
	apiKeyRaw, err := common.PasswordPrompt("API Key")
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey := sanitizeAPIKey(apiKeyRaw)
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	region, err := common.Select("Region", []common.SelectOption[string]{
		{Label: "US", Value: "us"},
		{Label: "EU", Value: "eu"},
	})
	if err != nil {
		return err
	}

	var verifyErr error
	_ = common.RunWithSpinner("Verifying API key...", func() error {
		verifyErr = verifyAPIKey(apiKey, region)
		return verifyErr
	})
	if verifyErr != nil {
		common.PrintError("Invalid API key: %v", verifyErr)
		return verifyErr
	}

	selection, err := ResolveAPIKeyApplication(apiKey, region, opts.clientID, true)
	if err != nil {
		return err
	}
	if err := ensureSetupCallbackURI(apiKey, selection.ClientID, region); err != nil {
		return err
	}
	if err := dashboard.ActivateAPIKey(apiKey, selection.ClientID, region, selection.OrgID); err != nil {
		return fmt.Errorf("could not activate API key: %w", err)
	}

	_, _ = common.Green.Println("  ✓ API key activated")

	// Update status — the API key path skips Steps 2 and 3.
	*status = GetSetupStatus()
	return nil
}

// stepApplication handles Step 2: list or create an application.
func stepApplication(status *SetupStatus) error {
	_, _ = common.Dim.Printf("  %s\n", divider)
	fmt.Println()
	_, _ = common.Bold.Printf("  Step 2 of %d: Application\n", stepTotal)
	fmt.Println()

	// If user entered an API key directly, app is already resolved.
	if status.HasAPIKey && !status.HasDashboardAuth {
		_, _ = common.Green.Println("  ✓ Application configured via API key")
		return nil
	}

	if status.HasActiveApp {
		_, _ = common.Green.Printf("  ✓ Active application: %s (%s)\n", status.ActiveAppID, status.ActiveAppRegion)
		return nil
	}

	appSvc, err := dashboard.CreateAppService()
	if err != nil {
		return err
	}

	orgID, err := dashboard.GetActiveOrgID()
	if err != nil {
		return err
	}

	ctx, cancel := common.CreateContext()
	defer cancel()

	var apps []domain.GatewayApplication
	err = common.RunWithSpinner("Checking for existing applications...", func() error {
		apps, err = appSvc.ListApplications(ctx, orgID, "")
		return err
	})
	if err != nil {
		return err
	}

	var selectedApp domain.GatewayApplication

	switch len(apps) {
	case 0:
		// Create a new application.
		app, createErr := createDefaultApp(appSvc, orgID)
		if createErr != nil {
			return createErr
		}
		selectedApp = domain.GatewayApplication{
			ApplicationID: app.ApplicationID,
			Region:        app.Region,
			Environment:   app.Environment,
			Branding:      app.Branding,
		}
	case 1:
		selectedApp = apps[0]
		name := appDisplayName(selectedApp)
		_, _ = common.Green.Printf("  ✓ Found application: %s\n", name)
	default:
		selected, selectErr := selectApp(apps)
		if selectErr != nil {
			return selectErr
		}
		selectedApp = selected
	}

	// Set as active app.
	if err := setActiveApp(selectedApp.ApplicationID, selectedApp.Region); err != nil {
		return err
	}

	*status = GetSetupStatus()
	return nil
}

// stepAPIKey handles Step 3: create and activate an API key.
func stepAPIKey(status *SetupStatus) error {
	_, _ = common.Dim.Printf("  %s\n", divider)
	fmt.Println()
	_, _ = common.Bold.Printf("  Step 3 of %d: API Key\n", stepTotal)
	fmt.Println()

	if status.HasAPIKey {
		_, _ = common.Green.Println("  ✓ API key already configured")
		return nil
	}

	if !status.HasActiveApp {
		return fmt.Errorf("no active application — cannot create API key")
	}

	appSvc, err := dashboard.CreateAppService()
	if err != nil {
		return err
	}

	keyName := "CLI-" + time.Now().Format("20060102-150405")

	ctx, cancel := common.CreateContext()
	defer cancel()

	var key *domain.GatewayCreatedAPIKey
	err = common.RunWithSpinner("Creating API key...", func() error {
		key, err = appSvc.CreateAPIKey(ctx, status.ActiveAppID, status.ActiveAppRegion, keyName, 0)
		return err
	})
	if err != nil {
		return err
	}

	_, _ = common.Green.Println("  ✓ API key created")

	// Activate the key directly (no 3-option menu in the wizard).
	err = common.RunWithSpinner("Activating API key...", func() error {
		if err := ensureSetupCallbackURI(key.APIKey, status.ActiveAppID, status.ActiveAppRegion); err != nil {
			return err
		}
		return dashboard.ActivateAPIKey(key.APIKey, status.ActiveAppID, status.ActiveAppRegion, "")
	})
	if err != nil {
		return err
	}

	_, _ = common.Green.Println("  ✓ API key activated")
	*status = GetSetupStatus()
	return nil
}

// stepGrantSync handles Step 4: sync grants from the Nylas API.
func stepGrantSync(status *SetupStatus) {
	_, _ = common.Dim.Printf("  %s\n", divider)
	fmt.Println()
	_, _ = common.Bold.Printf("  Step 4 of %d: Email Accounts\n", stepTotal)
	fmt.Println()

	if status.HasGrants {
		_, _ = common.Green.Println("  ✓ Email accounts already synced")
		return
	}

	if !status.HasAPIKey {
		_, _ = common.Yellow.Println("  Skipped — no API key configured")
		return
	}

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		_, _ = common.Yellow.Printf("  Could not access keyring: %v\n", err)
		return
	}

	apiKey, _ := secretStore.Get(ports.KeyAPIKey)
	clientID, _ := secretStore.Get(ports.KeyClientID)
	configStore := config.NewDefaultFileStore()
	cfg, _ := configStore.Load()
	region := cfg.Region

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		_, _ = common.Yellow.Printf("  Could not access grant store: %v\n", err)
		return
	}

	var result *SyncResult
	err = common.RunWithSpinner("Checking for existing email accounts...", func() error {
		result, err = SyncGrants(grantStore, apiKey, clientID, region)
		return err
	})
	if err != nil {
		_, _ = common.Yellow.Printf("  Could not sync grants: %v\n", err)
		fmt.Println()
		fmt.Println("  To authenticate later:")
		fmt.Println("    nylas auth login")
		return
	}

	if len(result.ValidGrants) == 0 {
		_, _ = common.Dim.Println("  No existing email accounts found")
		fmt.Println()
		fmt.Println("  To authenticate with your email provider:")
		fmt.Println("    nylas auth login")
		return
	}

	// Handle default grant selection.
	if result.DefaultGrantID != "" {
		// Single grant, auto-set.
		fmt.Println()
		_, _ = common.Green.Printf("  ✓ Set %s as default account\n", result.ValidGrants[0].Email)
	} else if len(result.ValidGrants) > 1 {
		// Multiple grants, prompt.
		defaultID, _ := PromptDefaultGrant(grantStore, result.ValidGrants)
		if defaultID != "" {
			result.DefaultGrantID = defaultID
			for _, g := range result.ValidGrants {
				if g.ID == defaultID {
					_, _ = common.Green.Printf("  ✓ Set %s as default account\n", g.Email)
					break
				}
			}
		}
	}

	// Update config file with grants.
	updateConfigGrants(configStore, cfg, result)
}

// updateConfigGrants writes the local default grant preference to the config file.
func updateConfigGrants(configStore *config.FileStore, cfg *domain.Config, result *SyncResult) {
	if cfg == nil || result == nil {
		return
	}
	cfg.DefaultGrant = result.DefaultGrantID
	cfg.Grants = nil
	_ = configStore.Save(cfg)
}
