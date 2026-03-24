package dashboard

import (
	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/ports"
)

// CreateAuthService creates the dashboard auth service chain (exported for setup wizard).
func CreateAuthService() (*dashboardapp.AuthService, ports.SecretStore, error) {
	return createAuthService()
}

// CreateAppService creates the dashboard app management service (exported for setup wizard).
func CreateAppService() (*dashboardapp.AppService, error) {
	return createAppService()
}

// RunSSO executes the SSO device-code flow (exported for setup wizard).
func RunSSO(provider, mode string, privacyAccepted bool) error {
	return runSSO(provider, mode, privacyAccepted)
}

// AcceptPrivacyPolicy prompts for privacy policy acceptance (exported for setup wizard).
func AcceptPrivacyPolicy() error {
	return acceptPrivacyPolicy()
}

// ActivateAPIKey stores an API key in the keyring and configures the CLI (exported for setup wizard).
func ActivateAPIKey(apiKey, clientID, region string) error {
	return activateAPIKey(apiKey, clientID, region)
}

// GetActiveOrgID retrieves the active organization ID (exported for setup wizard).
func GetActiveOrgID() (string, error) {
	return getActiveOrgID()
}

// ReadLine prompts for a line of text input (exported for setup wizard).
func ReadLine(prompt string) (string, error) {
	return readLine(prompt)
}
