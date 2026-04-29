package otp

import (
	"os"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	nylasadapter "github.com/nylas/cli/internal/adapters/nylas"
	otpapp "github.com/nylas/cli/internal/app/otp"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

// createOTPService creates the OTP service.
func createOTPService() (*otpapp.Service, error) {
	configStore := config.NewDefaultFileStore()

	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, err
	}

	grantStore, err := common.NewDefaultGrantStore()
	if err != nil {
		return nil, err
	}

	// Create Nylas client
	client := nylasadapter.NewHTTPClient()

	// Set credentials - check env vars first
	cfg, _ := configStore.Load()
	client.SetRegion(cfg.Region)

	apiKey := os.Getenv("NYLAS_API_KEY")
	clientID := os.Getenv("NYLAS_CLIENT_ID")
	clientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// If API key not in env, try keyring/file store
	if apiKey == "" {
		apiKey, _ = secretStore.Get(ports.KeyAPIKey)
		if clientID == "" {
			clientID, _ = secretStore.Get(ports.KeyClientID)
		}
		if clientSecret == "" {
			clientSecret, _ = secretStore.Get(ports.KeyClientSecret)
		}
	}

	client.SetCredentials(clientID, clientSecret, apiKey)

	return otpapp.NewService(client, grantStore, configStore), nil
}
