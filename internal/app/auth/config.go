package auth

import (
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// ConfigService handles configuration operations.
type ConfigService struct {
	config  ports.ConfigStore
	secrets ports.SecretStore
}

// NewConfigService creates a new config service.
func NewConfigService(config ports.ConfigStore, secrets ports.SecretStore) *ConfigService {
	return &ConfigService{
		config:  config,
		secrets: secrets,
	}
}

// SetupConfig saves the initial configuration.
func (s *ConfigService) SetupConfig(region, clientID, clientSecret, apiKey string) error {
	// Save credentials to secret store
	if err := s.secrets.Set(ports.KeyClientID, clientID); err != nil {
		return err
	}
	if clientSecret != "" {
		if err := s.secrets.Set(ports.KeyClientSecret, clientSecret); err != nil {
			return err
		}
	}
	if err := s.secrets.Set(ports.KeyAPIKey, apiKey); err != nil {
		return err
	}

	// Load existing config to preserve user settings (like working_hours)
	cfg, err := s.config.Load()
	if err != nil {
		cfg = domain.DefaultConfig()
	}
	cfg.Region = region
	return s.config.Save(cfg)
}

// IsConfigured checks if the application is configured.
func (s *ConfigService) IsConfigured() bool {
	// Check for client_id in keystore
	clientID, err := s.secrets.Get(ports.KeyClientID)
	if err != nil || clientID == "" {
		// Fall back to legacy config file check
		cfg, err := s.config.Load()
		if err != nil {
			return false
		}
		// Legacy configs might have client_id in config file
		// but new configs store it in keystore
		if cfg.Region == "" {
			return false
		}
	}

	// Check for API key
	apiKey, err := s.secrets.Get(ports.KeyAPIKey)
	return err == nil && apiKey != ""
}

// GetStatus returns the current configuration status.
func (s *ConfigService) GetStatus() (*domain.ConfigStatus, error) {
	cfg, err := s.config.Load()
	if err != nil {
		cfg = domain.DefaultConfig()
	}

	status := &domain.ConfigStatus{
		Region:      cfg.Region,
		SecretStore: s.secrets.Name(),
		ConfigPath:  s.config.Path(),
		// GrantCount is set by caller from grant store
	}

	// Get client_id from keystore
	clientID, _ := s.secrets.Get(ports.KeyClientID)
	status.ClientID = clientID

	// Check for secrets
	_, err = s.secrets.Get(ports.KeyAPIKey)
	status.HasAPIKey = err == nil

	_, err = s.secrets.Get(ports.KeyClientSecret)
	status.HasClientSecret = err == nil

	status.IsConfigured = s.IsConfigured()
	// DefaultGrant is set by caller from grant store

	return status, nil
}

// GetClientID retrieves the client ID from keystore.
func (s *ConfigService) GetClientID() (string, error) {
	return s.secrets.Get(ports.KeyClientID)
}

// GetAPIKey retrieves the API key from keystore.
func (s *ConfigService) GetAPIKey() (string, error) {
	return s.secrets.Get(ports.KeyAPIKey)
}

// GetClientSecret retrieves the client secret from keystore.
func (s *ConfigService) GetClientSecret() (string, error) {
	return s.secrets.Get(ports.KeyClientSecret)
}

// ResetConfig clears all configuration and secrets.
func (s *ConfigService) ResetConfig() error {
	// Delete all secrets
	_ = s.secrets.Delete(ports.KeyClientID)
	_ = s.secrets.Delete(ports.KeyClientSecret)
	_ = s.secrets.Delete(ports.KeyAPIKey)

	// Reset config to defaults
	return s.config.Save(domain.DefaultConfig())
}

// UpdateCallbackPort updates the OAuth callback port.
func (s *ConfigService) UpdateCallbackPort(port int) error {
	cfg, err := s.config.Load()
	if err != nil {
		cfg = domain.DefaultConfig()
	}
	cfg.CallbackPort = port
	return s.config.Save(cfg)
}

// HasKeystoreCredentials checks if credentials are stored in keystore.
func (s *ConfigService) HasKeystoreCredentials() bool {
	clientID, err := s.secrets.Get(ports.KeyClientID)
	if err != nil || clientID == "" {
		return false
	}
	apiKey, err := s.secrets.Get(ports.KeyAPIKey)
	return err == nil && apiKey != ""
}

// EnsureConfig ensures config exists, regenerating if necessary.
func (s *ConfigService) EnsureConfig() (*domain.Config, bool, error) {
	if s.config.Exists() {
		cfg, err := s.config.Load()
		return cfg, false, err
	}

	if !s.HasKeystoreCredentials() {
		return nil, false, domain.ErrNotConfigured
	}

	// Regenerate config from defaults
	cfg := domain.DefaultConfig()
	if err := s.config.Save(cfg); err != nil {
		return nil, false, err
	}
	return cfg, true, nil
}
