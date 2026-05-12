package auth

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

var oauthProviders = map[domain.Provider]bool{
	domain.ProviderGoogle:    true,
	domain.ProviderMicrosoft: true,
	domain.ProviderEWS:       true,
}

func newLoginCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with an email provider",
		Long: `Authenticate with an email provider.

OAuth providers (opens browser):
  google     Google/Gmail
  microsoft  Microsoft/Outlook
  ews        Exchange on-premises (EWS)

Credential providers (prompts for credentials):
  icloud     iCloud (requires app-specific password)
  yahoo      Yahoo (requires app password)
  imap       Generic IMAP server`,
		Example: `  # Login with Google (default)
  nylas auth login

  # Login with Microsoft/Outlook
  nylas auth login --provider microsoft

  # Login with Exchange on-premises
  nylas auth login --provider ews

  # Login with iCloud
  nylas auth login --provider icloud

  # Login with Yahoo
  nylas auth login --provider yahoo

  # Login with a generic IMAP server
  nylas auth login --provider imap`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := parseLoginProvider(provider)
			if err != nil {
				return err
			}

			configSvc, _, _, err := createConfigService()
			if err != nil {
				return err
			}

			if !configSvc.IsConfigured() {
				return fmt.Errorf("nylas not configured - run 'nylas auth config' first")
			}

			if oauthProviders[p] {
				return loginOAuth(p)
			}
			return loginCredentials(p)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "google", "Email provider (google, microsoft, ews, icloud, yahoo, imap)")

	return cmd
}

func loginOAuth(provider domain.Provider) error {
	authSvc, _, err := createAuthService()
	if err != nil {
		return err
	}

	fmt.Println("Opening browser for authentication...")
	fmt.Println("Complete the sign-in process in your browser.")

	ctx, cancel := common.CreateLongContext()
	defer cancel()

	grant, err := authSvc.Login(ctx, provider)
	if err != nil {
		return err
	}

	printLoginSuccess(grant)
	return nil
}

func loginCredentials(provider domain.Provider) error {
	settings, err := promptCredentials(provider)
	if err != nil {
		return err
	}

	authSvc, _, err := createAuthService()
	if err != nil {
		return err
	}

	apiProvider := credentialAPIProvider(provider)

	ctx, cancel := common.CreateLongContext()
	defer cancel()

	var grant *domain.Grant
	err = common.RunWithSpinner("Authenticating...", func() error {
		grant, err = authSvc.LoginWithCredentials(ctx, apiProvider, settings)
		return err
	})
	if err != nil {
		return err
	}

	printLoginSuccess(grant)
	return nil
}

func promptCredentials(provider domain.Provider) (map[string]any, error) {
	switch provider {
	case domain.ProviderICloud:
		return promptICloudCredentials()
	case domain.ProviderYahoo:
		return promptYahooCredentials()
	case domain.ProviderIMAP:
		return promptIMAPCredentials()
	default:
		return nil, fmt.Errorf("unsupported credential provider: %s", provider)
	}
}

func promptICloudCredentials() (map[string]any, error) {
	fmt.Println()
	_, _ = common.Dim.Println("  iCloud requires an app-specific password.")
	_, _ = common.Dim.Println("  Generate one at: https://appleid.apple.com/account/manage")
	fmt.Println()

	username, err := common.InputPrompt("iCloud email", "")
	if err != nil {
		return nil, err
	}

	password, err := common.PasswordPrompt("App-specific password")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"username": strings.TrimSpace(username),
		"password": password,
	}, nil
}

func promptYahooCredentials() (map[string]any, error) {
	fmt.Println()
	_, _ = common.Dim.Println("  Yahoo requires an app password.")
	_, _ = common.Dim.Println("  Generate one at: https://login.yahoo.com/account/security/app-passwords")
	fmt.Println()

	email, err := common.InputPrompt("Yahoo email", "")
	if err != nil {
		return nil, err
	}

	password, err := common.PasswordPrompt("App password")
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"imap_username": strings.TrimSpace(email),
		"imap_password": password,
		"imap_host":     "imap.mail.yahoo.com",
		"imap_port":     993,
		"type":          "yahoo",
	}, nil
}

func promptIMAPCredentials() (map[string]any, error) {
	fmt.Println()

	username, err := common.InputPrompt("IMAP username (email)", "")
	if err != nil {
		return nil, err
	}

	password, err := common.PasswordPrompt("IMAP password")
	if err != nil {
		return nil, err
	}

	imapHost, err := common.InputPrompt("IMAP host", "")
	if err != nil {
		return nil, err
	}

	imapPort := promptPort("IMAP port", 993)

	settings := map[string]any{
		"imap_username": strings.TrimSpace(username),
		"imap_password": password,
		"imap_host":     strings.TrimSpace(imapHost),
		"imap_port":     imapPort,
	}

	addSMTP, err := common.ConfirmPrompt("Add SMTP settings for sending email?", true)
	if err == nil && addSMTP {
		smtpHost, smtpErr := common.InputPrompt("SMTP host", strings.TrimSpace(imapHost))
		if smtpErr == nil && smtpHost != "" {
			settings["smtp_host"] = strings.TrimSpace(smtpHost)
			settings["smtp_port"] = promptPort("SMTP port", 465)
		}
	}

	return settings, nil
}

// credentialAPIProvider maps domain providers to the API provider string.
// Yahoo uses "imap" as the API provider with a "type": "yahoo" setting.
func credentialAPIProvider(provider domain.Provider) string {
	if provider == domain.ProviderYahoo {
		return "imap"
	}
	return string(provider)
}

func promptPort(title string, defaultPort int) int {
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

func printLoginSuccess(grant *domain.Grant) {
	_, _ = common.Green.Printf("\n✓ Successfully authenticated!\n")
	fmt.Printf("  Email:    %s\n", grant.Email)
	fmt.Printf("  Provider: %s\n", grant.Provider.DisplayName())
	fmt.Printf("  Grant ID: %s\n", grant.ID)
}

func parseLoginProvider(provider string) (domain.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case string(domain.ProviderGoogle):
		return domain.ProviderGoogle, nil
	case string(domain.ProviderMicrosoft):
		return domain.ProviderMicrosoft, nil
	case string(domain.ProviderEWS):
		return domain.ProviderEWS, nil
	case string(domain.ProviderICloud):
		return domain.ProviderICloud, nil
	case string(domain.ProviderYahoo):
		return domain.ProviderYahoo, nil
	case string(domain.ProviderIMAP):
		return domain.ProviderIMAP, nil
	default:
		return "", common.NewUserError(
			fmt.Sprintf("invalid provider: %s", provider),
			"use 'google', 'microsoft', 'ews', 'icloud', 'yahoo', or 'imap'",
		)
	}
}
