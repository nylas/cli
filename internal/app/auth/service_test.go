package auth

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGrantStore implements ports.GrantStore for testing
type mockGrantStore struct {
	grants       map[string]domain.GrantInfo
	defaultGrant string
}

func newMockGrantStore() *mockGrantStore {
	return &mockGrantStore{
		grants: make(map[string]domain.GrantInfo),
	}
}

func (m *mockGrantStore) SaveGrant(info domain.GrantInfo) error {
	m.grants[info.ID] = info
	return nil
}

func (m *mockGrantStore) GetGrant(grantID string) (*domain.GrantInfo, error) {
	if grant, ok := m.grants[grantID]; ok {
		return &grant, nil
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	for _, grant := range m.grants {
		if grant.Email == email {
			return &grant, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) ListGrants() ([]domain.GrantInfo, error) {
	grants := make([]domain.GrantInfo, 0, len(m.grants))
	for _, grant := range m.grants {
		grants = append(grants, grant)
	}
	return grants, nil
}

func (m *mockGrantStore) DeleteGrant(grantID string) error {
	delete(m.grants, grantID)
	return nil
}

func (m *mockGrantStore) SetDefaultGrant(grantID string) error {
	m.defaultGrant = grantID
	return nil
}

func (m *mockGrantStore) GetDefaultGrant() (string, error) {
	if m.defaultGrant == "" {
		return "", domain.ErrNoDefaultGrant
	}
	return m.defaultGrant, nil
}

func (m *mockGrantStore) ClearGrants() error {
	m.grants = make(map[string]domain.GrantInfo)
	m.defaultGrant = ""
	return nil
}

// mockConfigStore implements ports.ConfigStore for testing
type mockConfigStore struct {
	config *domain.Config
}

func newMockConfigStore() *mockConfigStore {
	return &mockConfigStore{
		config: &domain.Config{Region: "us"},
	}
}

func (m *mockConfigStore) Load() (*domain.Config, error) {
	return m.config, nil
}

func (m *mockConfigStore) Save(cfg *domain.Config) error {
	m.config = cfg
	return nil
}

func (m *mockConfigStore) Path() string {
	return "/tmp/test-config.yaml"
}

func (m *mockConfigStore) Exists() bool {
	return m.config != nil
}

func TestService_autoSwitchDefault(t *testing.T) {
	t.Run("clears default when no grants remain", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()

		// Set up a default grant that doesn't exist in grants list
		grantStore.defaultGrant = "old-deleted-grant"

		svc := &Service{
			grantStore: grantStore,
			config:     configStore,
		}

		// Call autoSwitchDefault with no grants
		svc.autoSwitchDefault()

		// Default should be cleared
		_, err := grantStore.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	})

	t.Run("sets first remaining grant as default", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()

		// Add some grants
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}

		svc := &Service{
			grantStore: grantStore,
			config:     configStore,
		}

		// Call autoSwitchDefault
		svc.autoSwitchDefault()

		// A default should be set (one of the remaining grants)
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Contains(t, []string{"grant-1", "grant-2"}, defaultID)
	})
}

func TestService_FirstGrantBecomesDefault(t *testing.T) {
	t.Run("first login sets grant as default", func(t *testing.T) {
		grantStore := newMockGrantStore()

		// Simulate what happens in Login() - no default exists
		_, err := grantStore.GetDefaultGrant()
		require.ErrorIs(t, err, domain.ErrNoDefaultGrant)

		// Save the grant
		grant := domain.GrantInfo{ID: "new-grant-1", Email: "user@example.com", Provider: domain.ProviderGoogle}
		require.NoError(t, grantStore.SaveGrant(grant))

		// Set as default since no default exists (this is the Login logic)
		if _, err := grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
			require.NoError(t, grantStore.SetDefaultGrant(grant.ID))
		}

		// Verify the grant is now default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "new-grant-1", defaultID)
	})

	t.Run("logout and new login sets new grant as default", func(t *testing.T) {
		grantStore := newMockGrantStore()
		configStore := newMockConfigStore()

		svc := &Service{
			grantStore: grantStore,
			config:     configStore,
		}

		// First login - grant1 becomes default
		grant1 := domain.GrantInfo{ID: "grant-1", Email: "user1@example.com", Provider: domain.ProviderGoogle}
		require.NoError(t, grantStore.SaveGrant(grant1))
		require.NoError(t, grantStore.SetDefaultGrant(grant1.ID))

		// Verify grant1 is default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)

		// Logout - delete grant and auto-switch (simulating Logout behavior)
		require.NoError(t, grantStore.DeleteGrant("grant-1"))
		svc.autoSwitchDefault()

		// Verify default is cleared (no grants remain)
		_, err = grantStore.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)

		// Second login - grant2 should become default
		grant2 := domain.GrantInfo{ID: "grant-2", Email: "user2@example.com", Provider: domain.ProviderGoogle}
		require.NoError(t, grantStore.SaveGrant(grant2))

		// This is the Login logic - set as default if no default exists
		if _, err := grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
			require.NoError(t, grantStore.SetDefaultGrant(grant2.ID))
		}

		// Verify grant2 is now default
		defaultID, err = grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)
	})

	t.Run("second login does not override existing default", func(t *testing.T) {
		grantStore := newMockGrantStore()

		// First login - grant1 becomes default
		grant1 := domain.GrantInfo{ID: "grant-1", Email: "user1@example.com", Provider: domain.ProviderGoogle}
		require.NoError(t, grantStore.SaveGrant(grant1))
		require.NoError(t, grantStore.SetDefaultGrant(grant1.ID))

		// Second login without logout - grant2 should NOT become default
		grant2 := domain.GrantInfo{ID: "grant-2", Email: "user2@example.com", Provider: domain.ProviderMicrosoft}
		require.NoError(t, grantStore.SaveGrant(grant2))

		// This is the Login logic - only set default if none exists
		if _, err := grantStore.GetDefaultGrant(); err == domain.ErrNoDefaultGrant {
			require.NoError(t, grantStore.SetDefaultGrant(grant2.ID))
		}

		// Verify grant1 is still default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
	})
}
