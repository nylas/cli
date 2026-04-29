package auth

import (
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSecretStore is a simple in-memory secret store for testing.
type mockSecretStore struct {
	data map[string]string
}

func newMockSecretStore() *mockSecretStore {
	return &mockSecretStore{data: make(map[string]string)}
}

func (m *mockSecretStore) Set(key, value string) error { m.data[key] = value; return nil }
func (m *mockSecretStore) Get(key string) (string, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return "", nil
}
func (m *mockSecretStore) Delete(key string) error { delete(m.data, key); return nil }
func (m *mockSecretStore) IsAvailable() bool       { return true }
func (m *mockSecretStore) Name() string            { return "mock" }

type failingSecretStore struct {
	*mockSecretStore
	failDeleteKey string
}

func (s *failingSecretStore) Delete(key string) error {
	if key == s.failDeleteKey {
		return errors.New("delete failed")
	}
	return s.mockSecretStore.Delete(key)
}

func TestConfigService_ResetConfig(t *testing.T) {
	t.Run("clears only API credentials", func(t *testing.T) {
		secrets := newMockSecretStore()
		configStore := newMockConfigStore()

		// Populate API credentials
		secrets.data[ports.KeyClientID] = "client-123"
		secrets.data[ports.KeyClientSecret] = "secret-456"
		secrets.data[ports.KeyAPIKey] = "nyl_abc"
		secrets.data[ports.KeyOrgID] = "org-789"

		// Populate dashboard credentials (should NOT be cleared)
		secrets.data[ports.KeyDashboardUserToken] = "user-token"
		secrets.data[ports.KeyDashboardAppID] = "app-id"

		svc := NewConfigService(configStore, secrets)

		err := svc.ResetConfig()
		require.NoError(t, err)

		// API credentials should be cleared
		assert.Empty(t, secrets.data[ports.KeyClientID])
		assert.Empty(t, secrets.data[ports.KeyClientSecret])
		assert.Empty(t, secrets.data[ports.KeyAPIKey])
		assert.Empty(t, secrets.data[ports.KeyOrgID])

		// Dashboard credentials should be untouched
		assert.Equal(t, "user-token", secrets.data[ports.KeyDashboardUserToken])
		assert.Equal(t, "app-id", secrets.data[ports.KeyDashboardAppID])
	})
}

func TestConfigService_GetStatusIgnoresConfigGrantList(t *testing.T) {
	secrets := newMockSecretStore()
	configStore := newMockConfigStore()
	configStore.config = &domain.Config{
		Region:       "us",
		DefaultGrant: "grant-1",
		Grants: []domain.GrantInfo{
			{ID: "grant-1", Email: "one@example.com"},
			{ID: "grant-2", Email: "two@example.com"},
		},
	}

	svc := NewConfigService(configStore, secrets)

	status, err := svc.GetStatus()
	require.NoError(t, err)
	assert.Zero(t, status.GrantCount)
	assert.Equal(t, "grant-1", status.DefaultGrant)
}

func TestConfigService_SetupConfig_PreservesExistingSettings(t *testing.T) {
	secrets := newMockSecretStore()
	secrets.data[ports.KeyClientID] = "old-client"
	secrets.data[ports.KeyAPIKey] = "old-api-key"
	secrets.data[storedGrantsKey] = `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`
	secrets.data[storedDefaultGrantKey] = "grant-123"
	configStore := newMockConfigStore()
	configStore.config = &domain.Config{
		Region:       "eu",
		CallbackPort: 7777,
		DefaultGrant: "grant-123",
		Grants: []domain.GrantInfo{{
			ID:       "grant-123",
			Email:    "user@example.com",
			Provider: domain.ProviderGoogle,
		}},
		TUITheme: "oled",
		WorkingHours: &domain.WorkingHoursConfig{
			Default: &domain.DaySchedule{
				Enabled: true,
				Start:   "08:00",
				End:     "16:00",
			},
		},
		AI: &domain.AIConfig{
			DefaultProvider: "openai",
		},
		GPG: &domain.GPGConfig{
			DefaultKey: "ABC123",
			AutoSign:   true,
		},
	}

	svc := NewConfigService(configStore, secrets)

	err := svc.SetupConfig("us", "client-123", "", "nyl_abc", "org-789")
	require.NoError(t, err)

	assert.Equal(t, "client-123", secrets.data[ports.KeyClientID])
	assert.Equal(t, "nyl_abc", secrets.data[ports.KeyAPIKey])
	assert.Equal(t, "org-789", secrets.data[ports.KeyOrgID])

	cfg, err := configStore.Load()
	require.NoError(t, err)

	assert.Equal(t, "us", cfg.Region)
	assert.Equal(t, 7777, cfg.CallbackPort)
	assert.Empty(t, cfg.DefaultGrant)
	assert.Empty(t, cfg.Grants)
	assert.Equal(t, "oled", cfg.TUITheme)
	require.NotNil(t, cfg.WorkingHours)
	require.NotNil(t, cfg.WorkingHours.Default)
	assert.Equal(t, "08:00", cfg.WorkingHours.Default.Start)
	require.NotNil(t, cfg.AI)
	assert.Equal(t, "openai", cfg.AI.DefaultProvider)
	require.NotNil(t, cfg.GPG)
	assert.Equal(t, "ABC123", cfg.GPG.DefaultKey)
	assert.Empty(t, secrets.data[storedGrantsKey])
	assert.Empty(t, secrets.data[storedDefaultGrantKey])
}

func TestConfigService_SetupConfig_PreservesGrantStateWhenCredentialsDoNotChange(t *testing.T) {
	secrets := newMockSecretStore()
	secrets.data[ports.KeyClientID] = "client-123"
	secrets.data[ports.KeyAPIKey] = "nyl_abc"
	secrets.data[storedGrantsKey] = `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`
	secrets.data[storedDefaultGrantKey] = "grant-123"

	configStore := newMockConfigStore()
	configStore.config = &domain.Config{
		Region:       "eu",
		CallbackPort: 7777,
		DefaultGrant: "grant-123",
		Grants: []domain.GrantInfo{{
			ID:       "grant-123",
			Email:    "user@example.com",
			Provider: domain.ProviderGoogle,
		}},
		TUITheme: "oled",
	}

	svc := NewConfigService(configStore, secrets)

	err := svc.SetupConfig("us", "client-123", "", "nyl_abc", "org-789")
	require.NoError(t, err)

	cfg, err := configStore.Load()
	require.NoError(t, err)

	assert.Equal(t, "grant-123", cfg.DefaultGrant)
	assert.Empty(t, cfg.Grants)
	assert.Equal(t, "grant-123", secrets.data[storedDefaultGrantKey])
	assert.NotEmpty(t, secrets.data[storedGrantsKey])
}

func TestConfigService_SetupConfig_RollsBackOnConfigSaveFailure(t *testing.T) {
	secrets := newMockSecretStore()
	secrets.data[ports.KeyClientID] = "old-client"
	secrets.data[ports.KeyClientSecret] = "old-secret"
	secrets.data[ports.KeyAPIKey] = "old-api-key"
	secrets.data[ports.KeyOrgID] = "old-org"
	secrets.data[storedGrantsKey] = `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`
	secrets.data[storedDefaultGrantKey] = "grant-123"

	configStore := &failingSetupConfigStore{
		config: &domain.Config{
			Region:       "eu",
			CallbackPort: 7777,
			DefaultGrant: "grant-123",
			Grants: []domain.GrantInfo{{
				ID:       "grant-123",
				Email:    "user@example.com",
				Provider: domain.ProviderGoogle,
			}},
			TUITheme: "oled",
		},
		err: errors.New("disk full"),
	}

	svc := NewConfigService(configStore, secrets)

	err := svc.SetupConfig("us", "client-123", "", "new-api-key", "org-789")
	require.Error(t, err)

	assert.Equal(t, "old-client", secrets.data[ports.KeyClientID])
	assert.Equal(t, "old-secret", secrets.data[ports.KeyClientSecret])
	assert.Equal(t, "old-api-key", secrets.data[ports.KeyAPIKey])
	assert.Equal(t, "old-org", secrets.data[ports.KeyOrgID])
	assert.Equal(t, `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`, secrets.data[storedGrantsKey])
	assert.Equal(t, "grant-123", secrets.data[storedDefaultGrantKey])

	cfg, loadErr := configStore.Load()
	require.NoError(t, loadErr)
	assert.Equal(t, "eu", cfg.Region)
	assert.Equal(t, "grant-123", cfg.DefaultGrant)
	require.Len(t, cfg.Grants, 1)
	assert.Equal(t, "grant-123", cfg.Grants[0].ID)
	assert.Equal(t, "oled", cfg.TUITheme)
}

func TestConfigService_SetupConfig_ClearsGrantStateWhenPreviousAPIKeyIsMissing(t *testing.T) {
	secrets := newMockSecretStore()
	secrets.data[ports.KeyClientID] = "client-123"
	secrets.data[storedGrantsKey] = `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`
	secrets.data[storedDefaultGrantKey] = "grant-123"

	configStore := newMockConfigStore()
	configStore.config = &domain.Config{
		Region:       "eu",
		CallbackPort: 7777,
		DefaultGrant: "grant-123",
		Grants: []domain.GrantInfo{{
			ID:       "grant-123",
			Email:    "user@example.com",
			Provider: domain.ProviderGoogle,
		}},
		TUITheme: "oled",
	}

	svc := NewConfigService(configStore, secrets)

	err := svc.SetupConfig("us", "client-123", "", "nyl_new", "org-789")
	require.NoError(t, err)

	cfg, loadErr := configStore.Load()
	require.NoError(t, loadErr)

	assert.Equal(t, "us", cfg.Region)
	assert.Empty(t, cfg.DefaultGrant)
	assert.Empty(t, cfg.Grants)
	assert.Equal(t, "oled", cfg.TUITheme)
	assert.Equal(t, "nyl_new", secrets.data[ports.KeyAPIKey])
	assert.Empty(t, secrets.data[storedGrantsKey])
	assert.Empty(t, secrets.data[storedDefaultGrantKey])
}

func TestConfigService_SetupConfig_RollsBackOnSecretStoreFailure(t *testing.T) {
	baseSecrets := newMockSecretStore()
	baseSecrets.data[ports.KeyClientID] = "old-client"
	baseSecrets.data[ports.KeyAPIKey] = "old-api-key"
	baseSecrets.data[ports.KeyOrgID] = "old-org"
	baseSecrets.data[storedGrantsKey] = `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`
	baseSecrets.data[storedDefaultGrantKey] = "grant-123"

	secrets := &failingSecretStore{
		mockSecretStore: baseSecrets,
		failDeleteKey:   storedDefaultGrantKey,
	}

	configStore := newMockConfigStore()
	configStore.config = &domain.Config{
		Region:       "eu",
		CallbackPort: 7777,
		DefaultGrant: "grant-123",
		Grants: []domain.GrantInfo{{
			ID:       "grant-123",
			Email:    "user@example.com",
			Provider: domain.ProviderGoogle,
		}},
		TUITheme: "oled",
	}

	svc := NewConfigService(configStore, secrets)

	err := svc.SetupConfig("us", "client-123", "", "new-api-key", "org-789")
	require.Error(t, err)

	assert.Equal(t, "old-client", baseSecrets.data[ports.KeyClientID])
	assert.Equal(t, "old-api-key", baseSecrets.data[ports.KeyAPIKey])
	assert.Equal(t, "old-org", baseSecrets.data[ports.KeyOrgID])
	assert.Equal(t, `[{"id":"grant-123","email":"user@example.com","provider":"google"}]`, baseSecrets.data[storedGrantsKey])
	assert.Equal(t, "grant-123", baseSecrets.data[storedDefaultGrantKey])

	cfg, loadErr := configStore.Load()
	require.NoError(t, loadErr)
	assert.Equal(t, "eu", cfg.Region)
	assert.Equal(t, "grant-123", cfg.DefaultGrant)
	require.Len(t, cfg.Grants, 1)
	assert.Equal(t, "grant-123", cfg.Grants[0].ID)
	assert.Equal(t, "oled", cfg.TUITheme)
}

type failingSetupConfigStore struct {
	config *domain.Config
	err    error
}

func (m *failingSetupConfigStore) Load() (*domain.Config, error) {
	return cloneConfig(m.config), nil
}

func (m *failingSetupConfigStore) Save(cfg *domain.Config) error {
	if m.err != nil {
		return m.err
	}
	m.config = cloneConfig(cfg)
	return nil
}

func (m *failingSetupConfigStore) Path() string {
	return "/tmp/test-config.yaml"
}

func (m *failingSetupConfigStore) Exists() bool {
	return m.config != nil
}
