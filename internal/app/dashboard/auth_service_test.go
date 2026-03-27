package dashboard

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
)

// memSecretStore is a simple in-memory implementation of ports.SecretStore for
// use in tests only. It never touches the OS keyring.
type memSecretStore struct {
	data map[string]string
}

func newMemSecretStore() *memSecretStore {
	return &memSecretStore{data: make(map[string]string)}
}

func (m *memSecretStore) Set(key, value string) error {
	m.data[key] = value
	return nil
}

func (m *memSecretStore) Get(key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", nil
	}
	return v, nil
}

func (m *memSecretStore) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *memSecretStore) IsAvailable() bool { return true }

func (m *memSecretStore) Name() string { return "mem" }

type failingSecretStore struct {
	*memSecretStore
	failSetKey string
}

func (f *failingSecretStore) Set(key, value string) error {
	if key == f.failSetKey {
		return errors.New("set failed")
	}
	return f.memSecretStore.Set(key, value)
}

// seedTokens pre-populates userToken (and optionally orgToken) so that
// loadTokens() succeeds without going through a full Login flow.
func seedTokens(s ports.SecretStore, userToken, orgToken string) {
	_ = s.Set(ports.KeyDashboardUserToken, userToken)
	if orgToken != "" {
		_ = s.Set(ports.KeyDashboardOrgToken, orgToken)
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_GetCurrentSession
// ---------------------------------------------------------------------------

func TestAuthService_GetCurrentSession(t *testing.T) {
	t.Parallel()

	sessionResp := &domain.DashboardSessionResponse{
		User:       domain.DashboardUser{PublicID: "user-1"},
		CurrentOrg: "org-1",
	}

	tests := []struct {
		name        string
		seedUser    string
		seedOrg     string
		mockFn      func(ctx context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error)
		wantErr     bool
		wantErrIs   error
		wantSession *domain.DashboardSessionResponse
		// verify the tokens forwarded to the mock
		wantUserToken string
		wantOrgToken  string
	}{
		{
			name:          "passes stored tokens to account client",
			seedUser:      "ut-abc",
			seedOrg:       "ot-xyz",
			wantUserToken: "ut-abc",
			wantOrgToken:  "ot-xyz",
			mockFn: func(_ context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error) {
				return sessionResp, nil
			},
			wantSession: sessionResp,
		},
		{
			name:          "works without org token",
			seedUser:      "ut-abc",
			wantUserToken: "ut-abc",
			wantOrgToken:  "",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return sessionResp, nil
			},
			wantSession: sessionResp,
		},
		{
			name:      "returns ErrDashboardNotLoggedIn when no user token stored",
			wantErr:   true,
			wantErrIs: domain.ErrDashboardNotLoggedIn,
		},
		{
			name:     "propagates error from account client",
			seedUser: "ut-abc",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return nil, errors.New("upstream error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newMemSecretStore()
			seedTokens(store, tt.seedUser, tt.seedOrg)

			// capture forwarded tokens
			var gotUserToken, gotOrgToken string
			mock := &dashboardadapter.MockAccountClient{
				GetCurrentSessionFn: func(ctx context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error) {
					gotUserToken = userToken
					gotOrgToken = orgToken
					if tt.mockFn != nil {
						return tt.mockFn(ctx, userToken, orgToken)
					}
					return &domain.DashboardSessionResponse{}, nil
				},
			}

			svc := NewAuthService(mock, store)
			got, err := svc.GetCurrentSession(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				assert.Nil(t, got)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantSession, got)

			if tt.seedUser != "" {
				assert.Equal(t, tt.wantUserToken, gotUserToken)
				assert.Equal(t, tt.wantOrgToken, gotOrgToken)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_SwitchOrg
// ---------------------------------------------------------------------------

func TestAuthService_SwitchOrg(t *testing.T) {
	t.Parallel()

	switchResp := &domain.DashboardSwitchOrgResponse{
		OrgToken: "new-org-token",
		Org:      domain.DashboardSwitchOrgOrg{PublicID: "org-new", Name: "New Corp"},
	}

	tests := []struct {
		name            string
		seedUser        string
		seedOrg         string
		orgPublicID     string
		mockFn          func(ctx context.Context, orgPublicID, userToken, orgToken string) (*domain.DashboardSwitchOrgResponse, error)
		setupStore      func(s *memSecretStore)
		wantErr         bool
		wantErrIs       error
		wantOrgToken    string
		wantOrgPublicID string
		wantAppIDGone   bool
		wantResp        *domain.DashboardSwitchOrgResponse
	}{
		{
			name:        "stores new org token and clears active app",
			seedUser:    "ut-abc",
			seedOrg:     "ot-old",
			orgPublicID: "org-new",
			mockFn: func(_ context.Context, _, _, _ string) (*domain.DashboardSwitchOrgResponse, error) {
				return switchResp, nil
			},
			wantOrgToken:    "new-org-token",
			wantOrgPublicID: "org-new",
			wantAppIDGone:   true,
			wantResp:        switchResp,
		},
		{
			name:        "stores org public ID from response",
			seedUser:    "ut-abc",
			orgPublicID: "org-new",
			mockFn: func(_ context.Context, _, _, _ string) (*domain.DashboardSwitchOrgResponse, error) {
				return &domain.DashboardSwitchOrgResponse{
					OrgToken: "t1",
					Org:      domain.DashboardSwitchOrgOrg{PublicID: "org-stored"},
				}, nil
			},
			wantOrgPublicID: "org-stored",
		},
		{
			name:        "skips storing empty org token",
			seedUser:    "ut-abc",
			orgPublicID: "org-new",
			mockFn: func(_ context.Context, _, _, _ string) (*domain.DashboardSwitchOrgResponse, error) {
				return &domain.DashboardSwitchOrgResponse{
					OrgToken: "",
					Org:      domain.DashboardSwitchOrgOrg{PublicID: "org-stored"},
				}, nil
			},
			wantOrgToken:    "",
			wantOrgPublicID: "org-stored",
		},
		{
			name:      "returns ErrDashboardNotLoggedIn when no user token",
			wantErr:   true,
			wantErrIs: domain.ErrDashboardNotLoggedIn,
		},
		{
			name:        "propagates account client error",
			seedUser:    "ut-abc",
			orgPublicID: "org-new",
			mockFn: func(_ context.Context, _, _, _ string) (*domain.DashboardSwitchOrgResponse, error) {
				return nil, errors.New("network failure")
			},
			wantErr: true,
		},
		{
			name:        "pre-existing app ID is deleted after switch",
			seedUser:    "ut-abc",
			orgPublicID: "org-new",
			setupStore: func(s *memSecretStore) {
				_ = s.Set(ports.KeyDashboardAppID, "app-old-123")
				_ = s.Set(ports.KeyDashboardAppRegion, "us")
			},
			mockFn: func(_ context.Context, _, _, _ string) (*domain.DashboardSwitchOrgResponse, error) {
				return switchResp, nil
			},
			wantAppIDGone:   true,
			wantOrgToken:    "new-org-token",
			wantOrgPublicID: "org-new",
			wantResp:        switchResp,
		},
		{
			name:        "forwards correct tokens to account client",
			seedUser:    "ut-user",
			seedOrg:     "ot-org",
			orgPublicID: "org-target",
			mockFn: func(_ context.Context, orgPublicID, userToken, orgToken string) (*domain.DashboardSwitchOrgResponse, error) {
				assert.Equal(t, "org-target", orgPublicID)
				assert.Equal(t, "ut-user", userToken)
				assert.Equal(t, "ot-org", orgToken)
				return switchResp, nil
			},
			wantOrgToken:    "new-org-token",
			wantOrgPublicID: "org-new",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newMemSecretStore()
			seedTokens(store, tt.seedUser, tt.seedOrg)
			if tt.setupStore != nil {
				tt.setupStore(store)
			}

			mock := &dashboardadapter.MockAccountClient{
				SwitchOrgFn: tt.mockFn,
			}

			svc := NewAuthService(mock, store)
			resp, err := svc.SwitchOrg(context.Background(), tt.orgPublicID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
				assert.Nil(t, resp)
				return
			}

			require.NoError(t, err)

			if tt.wantResp != nil {
				assert.Equal(t, tt.wantResp, resp)
			}

			// Verify stored org token.
			if tt.wantOrgToken != "" {
				storedOrgToken, _ := store.Get(ports.KeyDashboardOrgToken)
				assert.Equal(t, tt.wantOrgToken, storedOrgToken)
			}

			// Verify stored org public ID.
			if tt.wantOrgPublicID != "" {
				storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
				assert.Equal(t, tt.wantOrgPublicID, storedOrgID)
			}

			// Verify active app was cleared.
			if tt.wantAppIDGone {
				appID, _ := store.Get(ports.KeyDashboardAppID)
				assert.Empty(t, appID, "app ID should be cleared after org switch")
				appRegion, _ := store.Get(ports.KeyDashboardAppRegion)
				assert.Empty(t, appRegion, "app region should be cleared after org switch")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_SyncSessionOrg
// ---------------------------------------------------------------------------

func TestAuthService_SyncSessionOrg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		seedUser          string
		seedOrg           string
		storeFactory      func() ports.SecretStore
		mockFn            func(ctx context.Context, userToken, orgToken string) (*domain.DashboardSessionResponse, error)
		wantErr           bool
		wantErrIs         error
		wantOrgPublicID   string
		wantNoOrgPublicID bool
	}{
		{
			name:     "stores CurrentOrg.PublicID on success",
			seedUser: "ut-abc",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return &domain.DashboardSessionResponse{
					CurrentOrg: "org-synced",
				}, nil
			},
			wantOrgPublicID: "org-synced",
		},
		{
			name:     "returns error when GetCurrentSession fails",
			seedUser: "ut-abc",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return nil, errors.New("session fetch failed")
			},
			wantErr:           true,
			wantNoOrgPublicID: true,
		},
		{
			name:              "returns error when not logged in",
			wantErr:           true,
			wantErrIs:         domain.ErrDashboardNotLoggedIn,
			wantNoOrgPublicID: true,
		},
		{
			name:     "does not store empty CurrentOrg.PublicID",
			seedUser: "ut-abc",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return &domain.DashboardSessionResponse{
					CurrentOrg: "",
				}, nil
			},
			wantNoOrgPublicID: true,
		},
		{
			name:     "overwrites pre-existing org public ID with server value",
			seedUser: "ut-abc",
			seedOrg:  "ot-xyz",
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return &domain.DashboardSessionResponse{
					CurrentOrg: "org-from-server",
				}, nil
			},
			wantOrgPublicID: "org-from-server",
		},
		{
			name:     "returns error when storing synced org fails",
			seedUser: "ut-abc",
			storeFactory: func() ports.SecretStore {
				return &failingSecretStore{
					memSecretStore: newMemSecretStore(),
					failSetKey:     ports.KeyDashboardOrgPublicID,
				}
			},
			mockFn: func(_ context.Context, _, _ string) (*domain.DashboardSessionResponse, error) {
				return &domain.DashboardSessionResponse{
					CurrentOrg: "org-synced",
				}, nil
			},
			wantErr:           true,
			wantNoOrgPublicID: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var store ports.SecretStore
			if tt.storeFactory != nil {
				store = tt.storeFactory()
			} else {
				store = newMemSecretStore()
			}
			seedTokens(store, tt.seedUser, tt.seedOrg)

			mock := &dashboardadapter.MockAccountClient{
				GetCurrentSessionFn: tt.mockFn,
			}

			svc := NewAuthService(mock, store)
			err := svc.SyncSessionOrg(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				}
			} else {
				require.NoError(t, err)
			}

			stored, _ := store.Get(ports.KeyDashboardOrgPublicID)
			if tt.wantOrgPublicID != "" {
				assert.Equal(t, tt.wantOrgPublicID, stored)
			}
			if tt.wantNoOrgPublicID {
				assert.Empty(t, stored)
			}
		})
	}
}

func TestAuthServiceStoreTokens(t *testing.T) {
	t.Parallel()

	t.Run("stores org when there is exactly one organization", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		svc := NewAuthService(&dashboardadapter.MockAccountClient{}, store)

		err := svc.storeTokens(&domain.DashboardAuthResponse{
			UserToken: "user-token",
			User:      domain.DashboardUser{PublicID: "user-1"},
			Organizations: []domain.DashboardOrganization{
				{PublicID: "org-only"},
			},
		})

		require.NoError(t, err)

		storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
		assert.Equal(t, "org-only", storedOrgID)
	})

	t.Run("does not guess active org when multiple organizations exist", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		svc := NewAuthService(&dashboardadapter.MockAccountClient{}, store)

		err := svc.storeTokens(&domain.DashboardAuthResponse{
			UserToken: "user-token",
			User:      domain.DashboardUser{PublicID: "user-1"},
			Organizations: []domain.DashboardOrganization{
				{PublicID: "org-1"},
				{PublicID: "org-2"},
			},
		})

		require.NoError(t, err)

		storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
		assert.Empty(t, storedOrgID)
	})
}
