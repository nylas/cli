package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantService_ListGrantsUsesLiveAPIAndRefreshesLocalCache(t *testing.T) {
	grantStore := newMockGrantStore()
	configStore := newMockConfigStore()
	client := nylas.NewMockClient()

	grantStore.grants["stale-local"] = domain.GrantInfo{
		ID:       "stale-local",
		Email:    "stale@example.com",
		Provider: domain.ProviderGoogle,
	}
	grantStore.defaultGrant = "grant-2"
	client.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
		return []domain.Grant{
			{ID: "grant-1", Email: "one@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
			{ID: "grant-2", Email: "two@example.com", Provider: domain.ProviderMicrosoft, GrantStatus: "invalid"},
		}, nil
	}

	svc := NewGrantService(client, grantStore, configStore)

	got, err := svc.ListGrants(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.True(t, client.ListGrantsCalled)
	assert.Equal(t, "grant-1", got[0].ID)
	assert.Equal(t, "valid", got[0].Status)
	assert.False(t, got[0].IsDefault)
	assert.Equal(t, "grant-2", got[1].ID)
	assert.Equal(t, "invalid", got[1].Status)
	assert.True(t, got[1].IsDefault)

	assert.NotContains(t, grantStore.grants, "stale-local")
	assert.Equal(t, domain.GrantInfo{
		ID:       "grant-1",
		Email:    "one@example.com",
		Provider: domain.ProviderGoogle,
	}, grantStore.grants["grant-1"])
	assert.Equal(t, domain.GrantInfo{
		ID:       "grant-2",
		Email:    "two@example.com",
		Provider: domain.ProviderMicrosoft,
	}, grantStore.grants["grant-2"])
}

func TestGrantService_ListGrantsLiveFailureDoesNotReturnLocalStaleData(t *testing.T) {
	grantStore := newMockGrantStore()
	configStore := newMockConfigStore()
	client := nylas.NewMockClient()
	networkErr := errors.New("network down")

	grantStore.grants["stale-local"] = domain.GrantInfo{
		ID:       "stale-local",
		Email:    "stale@example.com",
		Provider: domain.ProviderGoogle,
	}
	client.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
		return nil, networkErr
	}

	svc := NewGrantService(client, grantStore, configStore)

	got, err := svc.ListGrants(context.Background())
	require.ErrorIs(t, err, networkErr)
	assert.Nil(t, got)
}

func TestGrantService_ListGrantsClearsStaleConfigDefault(t *testing.T) {
	grantStore := newMockGrantStore()
	configStore := newMockConfigStore()
	configStore.config.DefaultGrant = "missing-default"
	client := nylas.NewMockClient()
	client.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
		return []domain.Grant{
			{ID: "grant-1", Email: "one@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
		}, nil
	}

	svc := NewGrantService(client, grantStore, configStore)

	got, err := svc.ListGrants(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.False(t, got[0].IsDefault)
	assert.Empty(t, configStore.config.DefaultGrant)
	defaultGrant, err := grantStore.GetDefaultGrant()
	assert.Empty(t, defaultGrant)
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
}

func TestGrantService_CachedGrantCountUsesGrantStore(t *testing.T) {
	grantStore := newMockGrantStore()
	grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "one@example.com"}
	grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "two@example.com"}

	svc := NewGrantService(nylas.NewMockClient(), grantStore, newMockConfigStore())

	assert.Equal(t, 2, svc.CachedGrantCount())
}

func TestGrantService_SwitchGrant(t *testing.T) {
	t.Run("updates both local cache and config file", func(t *testing.T) {
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

		// Verify local cache was updated
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
		client.GetGrantFunc = func(ctx context.Context, grantID string) (*domain.Grant, error) {
			return nil, domain.ErrGrantNotFound
		}

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

	t.Run("returns error if config save fails and leaves local cache unchanged", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := &failingSaveConfigStore{config: &domain.Config{DefaultGrant: "grant-1"}}
		client := nylas.NewMockClient()

		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrant("grant-2")

		require.Error(t, err)

		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})
}

func TestGrantService_SwitchGrantByEmail(t *testing.T) {
	t.Run("updates both local cache and config file", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()
		client := nylas.NewMockClient()

		// Set up existing grants
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore.config.DefaultGrant = "grant-1"
		client.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
			return []domain.Grant{
				{ID: "grant-1", Email: "user1@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
				{ID: "grant-2", Email: "user2@example.com", Provider: domain.ProviderMicrosoft, GrantStatus: "valid"},
			}, nil
		}

		svc := NewGrantService(client, grantStore, configStore)

		err := svc.SwitchGrantByEmail("user2@example.com")

		require.NoError(t, err)

		// Verify local cache was updated
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
		client.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
			return []domain.Grant{
				{ID: "grant-1", Email: "user1@example.com", Provider: domain.ProviderGoogle, GrantStatus: "valid"},
			}, nil
		}

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
		assert.Equal(t, "grant-1", configStore.config.DefaultGrant)
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
		assert.Equal(t, "grant-1", configStore.config.DefaultGrant)
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
