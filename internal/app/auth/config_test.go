package auth

import (
	"testing"

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
func (m *mockSecretStore) Name() string             { return "mock" }

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
