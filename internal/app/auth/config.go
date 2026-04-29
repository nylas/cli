package auth

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	storedGrantsKey       = "grants"
	storedDefaultGrantKey = "default_grant"
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
func (s *ConfigService) SetupConfig(region, clientID, clientSecret, apiKey, orgID string) (err error) {
	previousClientID, _ := s.secrets.Get(ports.KeyClientID)
	previousAPIKey, _ := s.secrets.Get(ports.KeyAPIKey)
	credentialsChanged := haveCredentialsChanged(previousClientID, clientID, previousAPIKey, apiKey)

	secretSnapshots := []secretSnapshot{
		newSecretSnapshot(s.secrets, ports.KeyClientID),
		newSecretSnapshot(s.secrets, ports.KeyClientSecret),
		newSecretSnapshot(s.secrets, ports.KeyAPIKey),
		newSecretSnapshot(s.secrets, ports.KeyOrgID),
		newSecretSnapshot(s.secrets, storedGrantsKey),
		newSecretSnapshot(s.secrets, storedDefaultGrantKey),
	}

	// Preserve existing non-auth settings when refreshing credentials.
	cfg, err := s.config.Load()
	if err != nil || cfg == nil {
		cfg = domain.DefaultConfig()
	}
	previousCfg := cloneConfig(cfg)

	defer func() {
		if err == nil {
			return
		}

		if rollbackErr := rollbackSetupState(s.config, s.secrets, previousCfg, secretSnapshots); rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
		}
	}()

	// Save credentials to secret store
	if err = s.secrets.Set(ports.KeyClientID, clientID); err != nil {
		return err
	}
	if clientSecret != "" {
		if err = s.secrets.Set(ports.KeyClientSecret, clientSecret); err != nil {
			return err
		}
	}
	if err = s.secrets.Set(ports.KeyAPIKey, apiKey); err != nil {
		return err
	}
	if orgID != "" {
		if err = s.secrets.Set(ports.KeyOrgID, orgID); err != nil {
			return err
		}
	}

	cfg.Grants = nil
	if credentialsChanged {
		cfg.DefaultGrant = ""
		if err = clearStoredGrantState(s.secrets); err != nil {
			return err
		}
	}
	if region != "" {
		cfg.Region = region
	}
	if err = s.config.Save(cfg); err != nil {
		return err
	}

	return nil
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
	}

	// Get client_id from keystore
	clientID, _ := s.secrets.Get(ports.KeyClientID)
	status.ClientID = clientID

	// Get org_id from keystore
	orgID, _ := s.secrets.Get(ports.KeyOrgID)
	status.OrgID = orgID

	// Check for secrets
	_, err = s.secrets.Get(ports.KeyAPIKey)
	status.HasAPIKey = err == nil

	_, err = s.secrets.Get(ports.KeyClientSecret)
	status.HasClientSecret = err == nil

	status.IsConfigured = s.IsConfigured()
	status.DefaultGrant = cfg.DefaultGrant

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

// GetOrgID retrieves the organization ID from keystore.
func (s *ConfigService) GetOrgID() (string, error) {
	return s.secrets.Get(ports.KeyOrgID)
}

// ResetConfig clears all configuration and secrets.
func (s *ConfigService) ResetConfig() error {
	// Delete all secrets
	_ = s.secrets.Delete(ports.KeyClientID)
	_ = s.secrets.Delete(ports.KeyClientSecret)
	_ = s.secrets.Delete(ports.KeyAPIKey)
	_ = s.secrets.Delete(ports.KeyOrgID)
	_ = clearStoredGrantState(s.secrets)

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

func haveCredentialsChanged(previousClientID, clientID, previousAPIKey, apiKey string) bool {
	return strings.TrimSpace(previousClientID) != strings.TrimSpace(clientID) ||
		strings.TrimSpace(previousAPIKey) != strings.TrimSpace(apiKey)
}

func clearStoredGrantState(secrets ports.SecretStore) error {
	if err := secrets.Delete(storedGrantsKey); err != nil {
		return err
	}
	if err := secrets.Delete(storedDefaultGrantKey); err != nil {
		return err
	}
	return nil
}

type secretSnapshot struct {
	key     string
	value   string
	present bool
}

func newSecretSnapshot(secrets ports.SecretStore, key string) secretSnapshot {
	value, err := secrets.Get(key)
	return secretSnapshot{
		key:     key,
		value:   value,
		present: err == nil,
	}
}

func restoreSecretSnapshots(secrets ports.SecretStore, snapshots []secretSnapshot) error {
	var restoreErr error
	for _, snapshot := range snapshots {
		if snapshot.present {
			if err := secrets.Set(snapshot.key, snapshot.value); err != nil {
				restoreErr = errors.Join(restoreErr, err)
			}
			continue
		}
		if err := secrets.Delete(snapshot.key); err != nil {
			restoreErr = errors.Join(restoreErr, err)
		}
	}

	return restoreErr
}

func rollbackSetupState(config ports.ConfigStore, secrets ports.SecretStore, previousCfg *domain.Config, snapshots []secretSnapshot) error {
	rollbackErr := restoreSecretSnapshots(secrets, snapshots)
	if previousCfg != nil {
		if err := config.Save(previousCfg); err != nil {
			rollbackErr = errors.Join(rollbackErr, err)
		}
	}

	return rollbackErr
}

func cloneConfig(cfg *domain.Config) *domain.Config {
	if cfg == nil {
		return nil
	}

	raw, err := json.Marshal(cfg)
	if err != nil {
		cloned := *cfg
		return &cloned
	}

	var cloned domain.Config
	if err := json.Unmarshal(raw, &cloned); err != nil {
		cloned := *cfg
		return &cloned
	}

	return &cloned
}
