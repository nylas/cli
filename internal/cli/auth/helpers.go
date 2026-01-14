package auth

import (
	"os"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/adapters/oauth"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/ports"
)

// createDependencies creates all the common dependencies.
func createDependencies() (ports.ConfigStore, ports.SecretStore, ports.GrantStore, error) {
	configStore := config.NewDefaultFileStore()

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, nil, nil, err
	}

	grantStore := keyring.NewGrantStore(secretStore)

	return configStore, secretStore, grantStore, nil
}

// createConfigService creates the config service.
func createConfigService() (*authapp.ConfigService, ports.ConfigStore, ports.SecretStore, error) {
	configStore, secretStore, _, err := createDependencies()
	if err != nil {
		return nil, nil, nil, err
	}
	return authapp.NewConfigService(configStore, secretStore), configStore, secretStore, nil
}

// getCredentialsFromEnv gets credentials from environment variables only.
func getCredentialsFromEnv() (apiKey, clientID, clientSecret string) {
	apiKey = os.Getenv("NYLAS_API_KEY")
	clientID = os.Getenv("NYLAS_CLIENT_ID")
	clientSecret = os.Getenv("NYLAS_CLIENT_SECRET")
	return
}

// getCredentialsWithEnvFallback gets credentials from env vars first, then keyring.
func getCredentialsWithEnvFallback(secretStore ports.SecretStore) (apiKey, clientID, clientSecret string) {
	// Check environment variables first (highest priority)
	apiKey, clientID, clientSecret = getCredentialsFromEnv()

	// If API key not in env, try keyring/file store
	if apiKey == "" && secretStore != nil {
		apiKey, _ = secretStore.Get(ports.KeyAPIKey)
		if clientID == "" {
			clientID, _ = secretStore.Get(ports.KeyClientID)
		}
		if clientSecret == "" {
			clientSecret, _ = secretStore.Get(ports.KeyClientSecret)
		}
	}
	return
}

// createGrantService creates the grant service.
func createGrantService() (*authapp.GrantService, *authapp.ConfigService, error) {
	configStore, secretStore, grantStore, err := createDependencies()
	if err != nil {
		return nil, nil, err
	}

	configSvc := authapp.NewConfigService(configStore, secretStore)

	// Create Nylas client
	client := nylasadapter.NewHTTPClient()

	// Set credentials if available
	cfg, _ := configStore.Load()
	client.SetRegion(cfg.Region)

	apiKey, clientID, clientSecret := getCredentialsWithEnvFallback(secretStore)
	client.SetCredentials(clientID, clientSecret, apiKey)

	return authapp.NewGrantService(client, grantStore, configStore), configSvc, nil
}

// createGrantStore creates just the grant store for local operations.
func createGrantStore() (ports.GrantStore, error) {
	_, _, grantStore, err := createDependencies()
	if err != nil {
		return nil, err
	}
	return grantStore, nil
}

// createAuthService creates the auth service.
func createAuthService() (*authapp.Service, *authapp.ConfigService, error) {
	configStore, secretStore, grantStore, err := createDependencies()
	if err != nil {
		return nil, nil, err
	}

	configSvc := authapp.NewConfigService(configStore, secretStore)

	// Create Nylas client
	client := nylasadapter.NewHTTPClient()

	// Set credentials if available
	cfg, _ := configStore.Load()
	client.SetRegion(cfg.Region)

	apiKey, clientID, clientSecret := getCredentialsWithEnvFallback(secretStore)
	client.SetCredentials(clientID, clientSecret, apiKey)

	// Create OAuth server
	oauthServer := oauth.NewCallbackServer(cfg.CallbackPort)

	// Create browser
	browserAdapter := browser.NewDefaultBrowser()

	return authapp.NewService(client, grantStore, configStore, oauthServer, browserAdapter), configSvc, nil
}
