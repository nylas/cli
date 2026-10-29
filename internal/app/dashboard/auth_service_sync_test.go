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

		err := svc.storeTokens(context.Background(), &domain.DashboardAuthResponse{
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

		err := svc.storeTokens(context.Background(), &domain.DashboardAuthResponse{
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
