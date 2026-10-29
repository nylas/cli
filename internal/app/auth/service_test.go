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

func (m *mockGrantStore) ReplaceGrants(grants []domain.GrantInfo) error {
	m.grants = make(map[string]domain.GrantInfo, len(grants))
	for _, grant := range grants {
		m.grants[grant.ID] = grant
	}
	if m.defaultGrant != "" {
		if _, ok := m.grants[m.defaultGrant]; !ok {
			m.defaultGrant = ""
		}
	}
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
		configStore.config.Grants = []domain.GrantInfo{{ID: "old-deleted-grant", Email: "user@example.com"}}
		configStore.config.DefaultGrant = "old-deleted-grant"

		// Set up a default grant that doesn't exist in grants list
		grantStore.defaultGrant = "old-deleted-grant"

		svc := &Service{
			grantStore: grantStore,
			config:     configStore,
		}

		// Call autoSwitchDefault with no grants
		require.NoError(t, svc.autoSwitchDefault())

		// Default should be cleared
		_, err := grantStore.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
		assert.Empty(t, configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
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
		require.NoError(t, svc.autoSwitchDefault())

		// A default should be set (one of the remaining grants)
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Contains(t, []string{"grant-1", "grant-2"}, defaultID)
		assert.Equal(t, defaultID, configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})
}

func TestService_syncConfigWithGrantStoreStoresOnlyDefaultGrant(t *testing.T) {
	grantStore := newMockGrantStore()
	grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "one@example.com"}
	grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "two@example.com"}
	grantStore.defaultGrant = "grant-2"
	configStore := newMockConfigStore()

	svc := &Service{
		grantStore: grantStore,
		config:     configStore,
	}

	require.NoError(t, svc.syncConfigWithGrantStore())

	assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
	assert.Empty(t, configStore.config.Grants)
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
		require.NoError(t, svc.autoSwitchDefault())

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

// mockOAuthServer implements ports.OAuthServer for testing
type mockOAuthServer struct {
	redirectURI   string
	code          string
	expectedState string
	setState      string
	startErr      error
	waitErr       error
	startCalled   bool
	stopCalled    bool
}

func (m *mockOAuthServer) Start() error {
	m.startCalled = true
	return m.startErr
}

func (m *mockOAuthServer) Stop() error {
	m.stopCalled = true
	return nil
}

func (m *mockOAuthServer) GetRedirectURI() string {
	return m.redirectURI
}

func (m *mockOAuthServer) SetExpectedState(state string) {
	m.setState = state
}

func (m *mockOAuthServer) WaitForCallback(ctx context.Context, expectedState string) (string, error) {
	m.expectedState = expectedState
	if m.waitErr != nil {
		return "", m.waitErr
	}
	return m.code, nil
}

// mockBrowser implements ports.Browser for testing
type mockBrowser struct {
	openedURL string
	openErr   error
	onOpen    func()
}

func (m *mockBrowser) Open(url string) error {
	m.openedURL = url
	if m.onOpen != nil {
		m.onOpen()
	}
	return m.openErr
}

func TestNewService(t *testing.T) {
	client := nylas.NewMockClient()
	grantStore := newMockGrantStore()
	configStore := newMockConfigStore()
	server := &mockOAuthServer{}
	browser := &mockBrowser{}

	svc := NewService(client, grantStore, configStore, server, browser)

	assert.NotNil(t, svc)
	assert.Equal(t, client, svc.client)
	assert.Equal(t, grantStore, svc.grantStore)
	assert.Equal(t, configStore, svc.config)
	assert.Equal(t, server, svc.server)
	assert.Equal(t, browser, svc.browser)
}

func TestService_Logout(t *testing.T) {
	t.Run("successful logout revokes and deletes grant", func(t *testing.T) {
		client := nylas.NewMockClient()
		grantStore := newMockGrantStore()
		grantStore.grants["grant-123"] = domain.GrantInfo{ID: "grant-123", Email: "user@example.com"}
		grantStore.defaultGrant = "grant-123"
		configStore := newMockConfigStore()
		configStore.config.Grants = []domain.GrantInfo{{ID: "grant-123", Email: "user@example.com"}}
		configStore.config.DefaultGrant = "grant-123"

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		err := svc.Logout(context.Background())

		require.NoError(t, err)

		// Verify grant was revoked
		assert.True(t, client.RevokeGrantCalled)

		// Verify grant was deleted
		_, err = grantStore.GetGrant("grant-123")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify default was cleared
		_, err = grantStore.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
		assert.Empty(t, configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})

	t.Run("no default grant returns error", func(t *testing.T) {
		grantStore := newMockGrantStore()
		svc := NewService(nylas.NewMockClient(), grantStore, newMockConfigStore(), &mockOAuthServer{}, &mockBrowser{})

		err := svc.Logout(context.Background())

		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	})

	t.Run("revoke error propagates", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.RevokeGrantFunc = func(ctx context.Context, grantID string) error {
			return errors.New("revoke failed")
		}

		grantStore := newMockGrantStore()
		grantStore.grants["grant-123"] = domain.GrantInfo{ID: "grant-123"}
		grantStore.defaultGrant = "grant-123"

		svc := NewService(client, grantStore, newMockConfigStore(), &mockOAuthServer{}, &mockBrowser{})

		err := svc.Logout(context.Background())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "revoke failed")
	})

	t.Run("grant not found on revoke is ignored", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.RevokeGrantFunc = func(ctx context.Context, grantID string) error {
			return domain.ErrGrantNotFound
		}

		grantStore := newMockGrantStore()
		grantStore.grants["grant-123"] = domain.GrantInfo{ID: "grant-123"}
		grantStore.defaultGrant = "grant-123"

		svc := NewService(client, grantStore, newMockConfigStore(), &mockOAuthServer{}, &mockBrowser{})

		err := svc.Logout(context.Background())

		require.NoError(t, err)

		// Grant should still be deleted locally
		_, err = grantStore.GetGrant("grant-123")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)
	})

	t.Run("auto-switches to another grant", func(t *testing.T) {
		client := nylas.NewMockClient()
		grantStore := newMockGrantStore()
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore := newMockConfigStore()
		configStore.config.Grants = []domain.GrantInfo{
			{ID: "grant-1", Email: "user1@example.com"},
			{ID: "grant-2", Email: "user2@example.com"},
		}
		configStore.config.DefaultGrant = "grant-1"

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		err := svc.Logout(context.Background())

		require.NoError(t, err)

		// Verify grant-1 was deleted
		_, err = grantStore.GetGrant("grant-1")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify default switched to grant-2
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)
		assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})
}

func TestService_LogoutGrant(t *testing.T) {
	t.Run("logs out specific grant", func(t *testing.T) {
		client := nylas.NewMockClient()
		grantStore := newMockGrantStore()
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore := newMockConfigStore()
		configStore.config.Grants = []domain.GrantInfo{
			{ID: "grant-1", Email: "user1@example.com"},
			{ID: "grant-2", Email: "user2@example.com"},
		}
		configStore.config.DefaultGrant = "grant-1"

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		err := svc.LogoutGrant(context.Background(), "grant-2")

		require.NoError(t, err)

		// Verify grant-2 was deleted
		_, err = grantStore.GetGrant("grant-2")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify grant-1 is still default
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)
		assert.Equal(t, "grant-1", configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})

	t.Run("logging out default grant switches to another", func(t *testing.T) {
		client := nylas.NewMockClient()
		grantStore := newMockGrantStore()
		grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
		grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
		grantStore.defaultGrant = "grant-1"
		configStore := newMockConfigStore()
		configStore.config.Grants = []domain.GrantInfo{
			{ID: "grant-1", Email: "user1@example.com"},
			{ID: "grant-2", Email: "user2@example.com"},
		}
		configStore.config.DefaultGrant = "grant-1"

		svc := NewService(client, grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

		err := svc.LogoutGrant(context.Background(), "grant-1")

		require.NoError(t, err)

		// Verify grant-1 was deleted
		_, err = grantStore.GetGrant("grant-1")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// Verify default switched to grant-2
		defaultID, err := grantStore.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)
		assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
		assert.Empty(t, configStore.config.Grants)
	})

	t.Run("grant not found on revoke is ignored", func(t *testing.T) {
		client := nylas.NewMockClient()
		client.RevokeGrantFunc = func(ctx context.Context, grantID string) error {
			return domain.ErrGrantNotFound
		}
		grantStore := newMockGrantStore()
		grantStore.grants["grant-123"] = domain.GrantInfo{ID: "grant-123"}

		svc := NewService(client, grantStore, newMockConfigStore(), &mockOAuthServer{}, &mockBrowser{})

		err := svc.LogoutGrant(context.Background(), "grant-123")

		require.NoError(t, err)

		// Grant should still be deleted locally
		_, err = grantStore.GetGrant("grant-123")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)
	})
}

func TestService_RemoveLocalGrant(t *testing.T) {
	grantStore := newMockGrantStore()
	grantStore.grants["grant-1"] = domain.GrantInfo{ID: "grant-1", Email: "user1@example.com"}
	grantStore.grants["grant-2"] = domain.GrantInfo{ID: "grant-2", Email: "user2@example.com"}
	grantStore.defaultGrant = "grant-1"
	configStore := newMockConfigStore()
	configStore.config.Grants = []domain.GrantInfo{
		{ID: "grant-1", Email: "user1@example.com"},
		{ID: "grant-2", Email: "user2@example.com"},
	}
	configStore.config.DefaultGrant = "grant-1"

	svc := NewService(nylas.NewMockClient(), grantStore, configStore, &mockOAuthServer{}, &mockBrowser{})

	err := svc.RemoveLocalGrant("grant-1")
	require.NoError(t, err)

	defaultID, err := grantStore.GetDefaultGrant()
	require.NoError(t, err)
	assert.Equal(t, "grant-2", defaultID)
	assert.Equal(t, "grant-2", configStore.config.DefaultGrant)
	assert.Empty(t, configStore.config.Grants)
}
