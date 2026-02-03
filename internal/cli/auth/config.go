package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newConfigCmd() *cobra.Command {
	var (
		region   string
		clientID string
		apiKey   string
		reset    bool
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure API credentials",
		Long: `Configure Nylas API credentials.

You can provide credentials via flags or interactively.
Get your credentials from https://dashboard-v3.nylas.com

The CLI only requires your API Key - Client ID is auto-detected.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configSvc, configStore, _, err := createConfigService()
			if err != nil {
				return err
			}

			if reset {
				if err := configSvc.ResetConfig(); err != nil {
					return err
				}
				_, _ = common.Green.Println("✓ Configuration reset")
				return nil
			}

			reader := bufio.NewReader(os.Stdin)

			// Interactive mode if API key not provided
			if apiKey == "" {
				fmt.Println("Configure Nylas API Credentials")
				fmt.Println("Get your API key from: https://dashboard-v3.nylas.com")
				fmt.Println()

				fmt.Print("API Key (hidden): ")
				apiKeyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return common.WrapError(err)
				}
				fmt.Println()
				apiKey = sanitizeAPIKey(string(apiKeyBytes))
			}

			if apiKey == "" {
				return common.NewUserError("API key is required", "Enter your Nylas API key when prompted or use --api-key")
			}

			// Get region if not provided
			if region == "" {
				fmt.Print("Region [us/eu] (default: us): ")
				input, _ := reader.ReadString('\n')
				region = strings.TrimSpace(input)
				if region == "" {
					region = "us"
				}
			}

			// Auto-detect Client ID from API key if not provided
			var selectedApp *domain.Application
			var orgID string

			if clientID == "" {
				fmt.Println()
				fmt.Println("Detecting applications...")

				client := nylasadapter.NewHTTPClient()
				client.SetRegion(region)
				client.SetCredentials("", "", apiKey) // Only API key needed for ListApplications

				ctx, cancel := common.CreateContext()
				apps, err := client.ListApplications(ctx)
				cancel()

				if err != nil {
					_, _ = common.Yellow.Printf("  Could not auto-detect Client ID: %v\n", err)
					fmt.Println()
					fmt.Print("Client ID (manual entry): ")
					input, _ := reader.ReadString('\n')
					clientID = strings.TrimSpace(input)
				} else if len(apps) == 0 {
					return fmt.Errorf("no applications found for this API key")
				} else if len(apps) == 1 {
					// Single app - auto-select
					app := apps[0]
					selectedApp = &app
					clientID = getAppClientID(app)
					orgID = app.OrganizationID
					_, _ = common.Green.Printf("  ✓ Found application: %s\n", getAppDisplayName(app))
				} else {
					// Multiple apps - let user choose
					fmt.Printf("  Found %d applications:\n\n", len(apps))
					for i, app := range apps {
						fmt.Printf("  [%d] %s\n", i+1, getAppDisplayName(app))
					}
					fmt.Println()
					fmt.Print("Select application (1-", len(apps), "): ")
					input, _ := reader.ReadString('\n')
					choice := strings.TrimSpace(input)

					var selected int
					if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(apps) {
						return common.NewInputError(fmt.Sprintf("invalid selection: %s", choice))
					}

					app := apps[selected-1]
					selectedApp = &app
					clientID = getAppClientID(app)
					orgID = app.OrganizationID
					_, _ = common.Green.Printf("  ✓ Selected: %s\n", getAppDisplayName(app))
				}
			}

			if clientID == "" {
				return common.NewUserError("client ID is required", "Client ID should be auto-detected or can be entered manually")
			}

			if err := configSvc.SetupConfig(region, clientID, "", apiKey, orgID); err != nil {
				return err
			}

			_, _ = common.Green.Println("✓ Configuration saved")

			// Display organization ID if available
			if orgID != "" {
				fmt.Printf("  Organization ID: %s\n", orgID)
			}

			// Load config to get callback port
			cfg, err := configStore.Load()
			if err != nil {
				_, _ = common.Yellow.Printf("  Warning: Could not load config: %v\n", err)
			}

			// Ensure callback URI exists in the application
			if selectedApp != nil && cfg != nil {
				// Get callback port from config (defaults to 9007)
				callbackPort := cfg.CallbackPort
				if callbackPort == 0 {
					callbackPort = 9007
				}

				requiredCallbackURI := fmt.Sprintf("http://localhost:%d/callback", callbackPort)
				hasCallbackURI := false

				// Check if callback URI already exists
				for _, cb := range selectedApp.CallbackURIs {
					if cb.URL == requiredCallbackURI {
						hasCallbackURI = true
						break
					}
				}

				fmt.Println()
				if !hasCallbackURI {
					fmt.Println("Setting up callback URI for OAuth authentication...")

					client := nylasadapter.NewHTTPClient()
					client.SetRegion(region)
					client.SetCredentials(clientID, "", apiKey)

					// Get the application ID to use for update
					appID := selectedApp.ID
					if appID == "" {
						appID = selectedApp.ApplicationID
					}

					// Build list of all callback URIs (existing + new)
					callbackURIs := make([]string, 0, len(selectedApp.CallbackURIs)+1)
					for _, cb := range selectedApp.CallbackURIs {
						if cb.URL != "" {
							callbackURIs = append(callbackURIs, cb.URL)
						}
					}
					callbackURIs = append(callbackURIs, requiredCallbackURI)

					// Try to update the application
					ctx, cancel := common.CreateContext()
					updateReq := &domain.UpdateApplicationRequest{
						CallbackURIs: callbackURIs,
					}
					_, err := client.UpdateApplication(ctx, appID, updateReq)
					cancel()

					if err != nil {
						// If update fails (e.g., sandbox limitation), provide manual instructions
						_, _ = common.Yellow.Printf("  Could not add callback URI automatically: %v\n", err)
						fmt.Printf("  Please add this callback URI manually in the Nylas dashboard:\n")
						fmt.Printf("    %s\n", requiredCallbackURI)
						fmt.Println()
						fmt.Printf("  Dashboard: https://dashboard.nylas.com/applications\n")
						fmt.Printf("  Navigate to: Your App → Settings → Callback URIs → Add URI\n")
					} else {
						_, _ = common.Green.Printf("  ✓ Added callback URI: %s\n", requiredCallbackURI)
					}
				} else {
					_, _ = common.Green.Println("✓ Callback URI already configured")
				}
			}

			// Auto-detect existing grants from Nylas API
			fmt.Println()
			fmt.Println("Checking for existing grants...")

			client := nylasadapter.NewHTTPClient()
			client.SetRegion(region)
			client.SetCredentials(clientID, "", apiKey)

			ctx, cancel := common.CreateContext()
			defer cancel()

			grants, err := client.ListGrants(ctx)
			if err != nil {
				_, _ = common.Yellow.Printf("  Could not fetch grants: %v\n", err)
				fmt.Println()
				fmt.Println("Next steps:")
				fmt.Println("  nylas auth login    Authenticate with your email provider")
				return nil
			}

			if len(grants) == 0 {
				fmt.Println("  No existing grants found")
				fmt.Println()
				fmt.Println("Next steps:")
				fmt.Println("  nylas auth login    Authenticate with your email provider")
				return nil
			}

			// Get grant store to save grants locally
			grantStore, err := createGrantStore()
			if err != nil {
				_, _ = common.Yellow.Printf("  Could not save grants locally: %v\n", err)
				return nil
			}

			// First pass: Add all valid grants without setting default
			var validGrants []domain.Grant
			for _, grant := range grants {
				if !grant.IsValid() {
					continue
				}

				grantInfo := domain.GrantInfo{
					ID:       grant.ID,
					Email:    grant.Email,
					Provider: grant.Provider,
				}

				if err := grantStore.SaveGrant(grantInfo); err != nil {
					continue
				}

				validGrants = append(validGrants, grant)
				_, _ = common.Green.Printf("  ✓ Added %s (%s)\n", grant.Email, grant.Provider.DisplayName())
			}

			if len(validGrants) == 0 {
				fmt.Println("  No valid grants found")
				fmt.Println()
				fmt.Println("Next steps:")
				fmt.Println("  nylas auth login    Authenticate with your email provider")
				return nil
			}

			// Second pass: Set default grant
			var defaultGrantID string
			if len(validGrants) == 1 {
				// Single grant - auto-select as default
				defaultGrantID = validGrants[0].ID
				_ = grantStore.SetDefaultGrant(defaultGrantID)
				fmt.Println()
				_, _ = common.Green.Printf("✓ Set %s as default account\n", validGrants[0].Email)
			} else {
				// Multiple grants - let user choose default
				fmt.Println()
				fmt.Println("Select default account:")
				for i, grant := range validGrants {
					fmt.Printf("  [%d] %s (%s)\n", i+1, grant.Email, grant.Provider.DisplayName())
				}
				fmt.Println()
				fmt.Print("Select default account (1-", len(validGrants), "): ")
				input, _ := reader.ReadString('\n')
				choice := strings.TrimSpace(input)

				var selected int
				if _, err := fmt.Sscanf(choice, "%d", &selected); err != nil || selected < 1 || selected > len(validGrants) {
					// If invalid selection, default to first
					_, _ = common.Yellow.Printf("Invalid selection, defaulting to %s\n", validGrants[0].Email)
					defaultGrantID = validGrants[0].ID
				} else {
					defaultGrantID = validGrants[selected-1].ID
				}

				_ = grantStore.SetDefaultGrant(defaultGrantID)
				selectedGrant := validGrants[0]
				for _, g := range validGrants {
					if g.ID == defaultGrantID {
						selectedGrant = g
						break
					}
				}
				_, _ = common.Green.Printf("✓ Set %s as default account\n", selectedGrant.Email)
			}

			fmt.Println()
			fmt.Printf("Added %d grant(s). Run 'nylas auth list' to see all accounts.\n", len(validGrants))

			// Update config file with default grant and grants list
			cfg.DefaultGrant = defaultGrantID
			cfg.Grants = make([]domain.GrantInfo, len(validGrants))
			for i, grant := range validGrants {
				cfg.Grants[i] = domain.GrantInfo{
					ID:       grant.ID,
					Email:    grant.Email,
					Provider: grant.Provider,
				}
			}
			if err := configStore.Save(cfg); err != nil {
				_, _ = common.Yellow.Printf("  Warning: Could not update config file: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&region, "region", "r", "us", "API region (us or eu)")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Nylas Client ID (auto-detected if not provided)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Nylas API Key")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset all configuration")

	return cmd
}

// getAppClientID returns the client ID for an application.
// It checks both ID and ApplicationID fields since the API may use either.
func getAppClientID(app domain.Application) string {
	if app.ApplicationID != "" {
		return app.ApplicationID
	}
	return app.ID
}

// getAppDisplayName returns a human-readable display name for an application.
func getAppDisplayName(app domain.Application) string {
	clientID := getAppClientID(app)
	env := app.Environment
	if env == "" {
		env = "production"
	}

	region := app.Region
	if region == "" {
		region = "us"
	}

	// Truncate client ID for display if too long
	displayID := clientID
	if len(displayID) > 20 {
		displayID = displayID[:17] + "..."
	}

	return fmt.Sprintf("%s (%s, %s)", displayID, env, region)
}

// sanitizeAPIKey removes all characters that are invalid in HTTP headers.
// This handles cases where pasting API keys includes invisible characters
// like carriage returns, newlines, or other control characters.
func sanitizeAPIKey(key string) string {
	var result strings.Builder
	result.Grow(len(key))

	for _, r := range key {
		// Only keep printable ASCII characters (space through tilde)
		// This excludes control characters, newlines, carriage returns, etc.
		if r >= ' ' && r <= '~' {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}
