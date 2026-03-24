package setup

import (
	"fmt"
	"strings"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

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
	fmt.Println()
	fmt.Println("  How would you like to authenticate?")
	fmt.Println()
	_, _ = common.Cyan.Println("    [1] Google      (recommended)")
	fmt.Println("    [2] Microsoft")
	fmt.Println("    [3] GitHub")
	fmt.Println()

	choice, err := readLine("  Choose [1-3]: ")
	if err != nil {
		return "google", nil
	}

	switch strings.TrimSpace(choice) {
	case "1", "":
		return "google", nil
	case "2":
		return "microsoft", nil
	case "3":
		return "github", nil
	default:
		return "google", nil
	}
}

// selectApp prompts the user to select from multiple applications.
func selectApp(apps []domain.GatewayApplication) (domain.GatewayApplication, error) {
	fmt.Printf("  Found %d applications:\n\n", len(apps))
	for i, app := range apps {
		fmt.Printf("    [%d] %s\n", i+1, appDisplayName(app))
	}
	fmt.Println()

	choice, err := readLine(fmt.Sprintf("  Select application [1-%d]: ", len(apps)))
	if err != nil {
		return apps[0], nil
	}

	var selected int
	if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(apps) {
		_, _ = common.Yellow.Println("  Invalid selection, using first application")
		return apps[0], nil
	}
	return apps[selected-1], nil
}

// createDefaultApp creates a new application with defaults.
func createDefaultApp(appSvc *dashboardapp.AppService, orgID string) (*domain.GatewayCreatedApplication, error) {
	fmt.Println("  No applications found. Creating one for you...")
	fmt.Println()

	name, err := readLine("  App name [My First App]: ")
	if err != nil || name == "" {
		name = "My First App"
	}

	region, err := readLine("  Region [us/eu] (default: us): ")
	if err != nil || region == "" {
		region = "us"
	}
	if region != "us" && region != "eu" {
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
