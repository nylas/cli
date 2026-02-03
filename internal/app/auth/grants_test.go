package auth

import (
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantService_SwitchGrant(t *testing.T) {
	t.Run("updates both keyring and config file", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		// Set up existing grants
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore.config.DefaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrant("grant-2")

		require.NoError(t, err)

		// Verify keyring was updated
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)

		// Verify config file was updated
		assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
	})

	t.Run("returns error for non-existent grant", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.defaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrant("non-existent")

		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify default was not changed
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})

	t.Run("succeeds even if config save fails", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := &failingSaveConfigStore{config: &domain.Config{DefaultGrant: "grant-1"}}
		client := nylas.NewMockClient()

		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		// Should succeed - keyring is authoritative, config save failure is non-fatal
		err := svc.SwitchGrant("grant-2")

		require.NoError(t, err)

		// Verify keyring was updated
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)
	})
}

func TestGrantService_SwitchGrantByEmail(t *testing.T) {
	t.Run("updates both keyring and config file", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		// Set up existing grants
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore.config.DefaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrantByEmail("user2@example.com")

		require.NoError(t, err)

		// Verify keyring was updated
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)

		// Verify config file was updated
		assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
	})

	t.Run("returns error for non-existent email", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.defaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrantByEmail("nonexistent@example.com")

		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify default was not changed
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})
}

func TestGrantService_AddGrant(t *testing.T) {
	t.Run("adds grant and sets as default when requested", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.AddGrant("grant-1", "user@example.com", domain.ProviderGoogle, true)

		require.NoError(t, err)

		// Verify grant was saved
		grant, err := grantStore.GetGrant("grant-1")
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", grant.Email)

		// Verify it's set as default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})

	t.Run("auto-sets first grant as default", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		svc := NewGrantService(client, grantStore, configStore)

		// Add grant without setDefault=true, but it's the first grant
		err := svc.AddGrant("grant-1", "user@example.com", domain.ProviderGoogle, false)

		require.NoError(t, err)

		// Should still be set as default since it's the first
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})

	t.Run("does not override existing default", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		// Set up existing grant as default
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.defaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		// Add second grant without setDefault
		err := svc.AddGrant("grant-2", "user2@example.com", domain.ProviderMicrosoft, false)

		require.NoError(t, err)

		// Original default should be preserved
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})
}

// failingSaveConfigStore is a mock that fails on Save
type failingSaveConfigStore struct {
	config *domain.Config
}

func (m *failingSaveConfigStore) Load() (*domain.Config, error) {
	return m.config, nil
}

func (m *failingSaveConfigStore) Save(cfg *domain.Config) error {
	return domain.ErrInvalidInput // Simulate save failure
}

func (m *failingSaveConfigStore) Path() string {
	return "/tmp/test-config.yaml"
}

func (m *failingSaveConfigStore) Exists() bool {
	return true
}
