package dashboard

import (
	"context"
	"errors"
	"testing"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceLoginStoresTokensAndDefersMFA(t *testing.T) {
	t.Parallel()

	t.Run("success stores tokens", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		require.NoError(t, store.Set(ports.KeyDashboardAppID, "stale-app"))
		require.NoError(t, store.Set(ports.KeyDashboardAppRegion, "eu"))
		require.NoError(t, store.Set(ports.KeyDashboardOrgPublicID, "stale-org"))
		mock := &dashboardadapter.MockAccountClient{
			LoginFn: func(_ context.Context, email, password, orgPublicID string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
				assert.Equal(t, "user@example.com", email)
				assert.Equal(t, "secret", password)
				assert.Equal(t, "org-1", orgPublicID)
				return &domain.DashboardAuthResponse{
					UserToken: "user-token",
					OrgToken:  "org-token",
					User:      domain.DashboardUser{PublicID: "user-1"},
				}, nil, nil
			},
		}

		svc := NewAuthService(mock, store)
		auth, mfa, err := svc.Login(context.Background(), "user@example.com", "secret", "org-1")
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Nil(t, mfa)

		storedUserToken, _ := store.Get(ports.KeyDashboardUserToken)
		assert.Equal(t, "user-token", storedUserToken)
		storedUserID, _ := store.Get(ports.KeyDashboardUserPublicID)
		assert.Equal(t, "user-1", storedUserID)
		storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
		assert.Empty(t, storedOrgID)
		appID, _ := store.Get(ports.KeyDashboardAppID)
		assert.Empty(t, appID)
		appRegion, _ := store.Get(ports.KeyDashboardAppRegion)
		assert.Empty(t, appRegion)
	})

	t.Run("mfa response does not store tokens", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		mock := &dashboardadapter.MockAccountClient{
			LoginFn: func(_ context.Context, _, _, _ string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
				return nil, &domain.DashboardMFARequired{
					User: domain.DashboardUser{PublicID: "user-1"},
				}, nil
			},
		}

		svc := NewAuthService(mock, store)
		auth, mfa, err := svc.Login(context.Background(), "user@example.com", "secret", "")
		require.NoError(t, err)
		assert.Nil(t, auth)
		assert.NotNil(t, mfa)

		storedUserToken, _ := store.Get(ports.KeyDashboardUserToken)
		assert.Empty(t, storedUserToken)
	})
}

func TestAuthServiceStoresTokensForVerificationAndMFA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func(t *testing.T, svc *AuthService, store ports.SecretStore)
		mock *dashboardadapter.MockAccountClient
	}{
		{
			name: "verify email code",
			mock: &dashboardadapter.MockAccountClient{
				VerifyEmailCodeFn: func(_ context.Context, email, code, region string) (*domain.DashboardAuthResponse, error) {
					assert.Equal(t, "user@example.com", email)
					assert.Equal(t, "123456", code)
					assert.Equal(t, "us", region)
					return &domain.DashboardAuthResponse{
						UserToken: "user-token",
						OrgToken:  "org-token",
						User:      domain.DashboardUser{PublicID: "user-1"},
					}, nil
				},
			},
			run: func(t *testing.T, svc *AuthService, store ports.SecretStore) {
				resp, err := svc.VerifyEmailCode(context.Background(), "user@example.com", "123456", "us")
				require.NoError(t, err)
				assert.Equal(t, "user-token", resp.UserToken)
			},
		},
		{
			name: "complete mfa",
			mock: &dashboardadapter.MockAccountClient{
				LoginMFAFn: func(_ context.Context, userPublicID, code, orgPublicID string) (*domain.DashboardAuthResponse, error) {
					assert.Equal(t, "user-1", userPublicID)
					assert.Equal(t, "654321", code)
					assert.Equal(t, "org-1", orgPublicID)
					return &domain.DashboardAuthResponse{
						UserToken: "user-token",
						OrgToken:  "org-token",
						User:      domain.DashboardUser{PublicID: "user-1"},
					}, nil
				},
			},
			run: func(t *testing.T, svc *AuthService, store ports.SecretStore) {
				resp, err := svc.CompleteMFA(context.Background(), "user-1", "654321", "org-1")
				require.NoError(t, err)
				assert.Equal(t, "org-token", resp.OrgToken)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newMemSecretStore()
			svc := NewAuthService(tt.mock, store)
			tt.run(t, svc, store)

			storedUserToken, _ := store.Get(ports.KeyDashboardUserToken)
			assert.Equal(t, "user-token", storedUserToken)
			storedOrgToken, _ := store.Get(ports.KeyDashboardOrgToken)
			assert.Equal(t, "org-token", storedOrgToken)
		})
	}
}

func TestAuthServiceLogoutStatusAndSSOPoll(t *testing.T) {
	t.Parallel()

	t.Run("logout clears local state even if server logout fails", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		seedTokens(store, "user-token", "org-token")
		require.NoError(t, store.Set(ports.KeyDashboardUserPublicID, "user-1"))
		require.NoError(t, store.Set(ports.KeyDashboardOrgPublicID, "org-1"))
		require.NoError(t, store.Set(ports.KeyDashboardAppID, "app-1"))
		require.NoError(t, store.Set(ports.KeyDashboardAppRegion, "us"))

		mock := &dashboardadapter.MockAccountClient{
			LogoutFn: func(_ context.Context, userToken, orgToken string) error {
				assert.Equal(t, "user-token", userToken)
				assert.Equal(t, "org-token", orgToken)
				return errors.New("network down")
			},
		}

		svc := NewAuthService(mock, store)
		require.NoError(t, svc.Logout(context.Background()))

		status := svc.GetStatus()
		assert.False(t, status.LoggedIn)
		assert.False(t, status.HasOrgToken)
		appID, _ := store.Get(ports.KeyDashboardAppID)
		assert.Empty(t, appID)
	})

	t.Run("logout returns local deletion failures", func(t *testing.T) {
		t.Parallel()

		store := &failingSecretStore{
			memSecretStore: newMemSecretStore(),
			failDeleteKey:  ports.KeyDashboardOrgToken,
		}
		seedTokens(store, "user-token", "org-token")

		svc := NewAuthService(&dashboardadapter.MockAccountClient{
			LogoutFn: func(_ context.Context, _, _ string) error { return nil },
		}, store)
		err := svc.Logout(context.Background())

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clear "+ports.KeyDashboardOrgToken)

		userToken, getErr := store.Get(ports.KeyDashboardUserToken)
		require.NoError(t, getErr)
		assert.Empty(t, userToken)
		orgToken, getErr := store.Get(ports.KeyDashboardOrgToken)
		require.NoError(t, getErr)
		assert.Equal(t, "org-token", orgToken)
	})

	t.Run("SSOPoll stores credentials only on complete", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		mock := &dashboardadapter.MockAccountClient{
			SSOPollFn: func(_ context.Context, flowID, orgPublicID string) (*domain.DashboardSSOPollResponse, error) {
				assert.Equal(t, "flow-1", flowID)
				assert.Equal(t, "org-1", orgPublicID)
				return &domain.DashboardSSOPollResponse{
					Status: domain.SSOStatusComplete,
					Auth: &domain.DashboardAuthResponse{
						UserToken: "user-token",
						OrgToken:  "org-token",
						User:      domain.DashboardUser{PublicID: "user-1"},
					},
				}, nil
			},
		}

		svc := NewAuthService(mock, store)
		resp, err := svc.SSOPoll(context.Background(), "flow-1", "org-1")
		require.NoError(t, err)
		require.NotNil(t, resp.Auth)
		assert.True(t, svc.IsLoggedIn())
		status := svc.GetStatus()
		assert.Equal(t, "user-1", status.UserID)
		assert.True(t, status.HasOrgToken)
	})
}

func TestAuthServiceStoreTokensRollsBackOnFailure(t *testing.T) {
	t.Parallel()

	store := &failingSecretStore{
		memSecretStore: newMemSecretStore(),
		failSetKey:     ports.KeyDashboardUserPublicID,
	}
	require.NoError(t, store.Set(ports.KeyDashboardUserToken, "stale-user"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgToken, "stale-org-token"))
	require.NoError(t, store.Set(ports.KeyDashboardOrgPublicID, "stale-org-id"))
	require.NoError(t, store.Set(ports.KeyDashboardAppID, "stale-app"))
	require.NoError(t, store.Set(ports.KeyDashboardAppRegion, "eu"))

	svc := NewAuthService(&dashboardadapter.MockAccountClient{}, store)
	err := svc.storeTokens(&domain.DashboardAuthResponse{
		UserToken: "user-new",
		OrgToken:  "org-new",
		User:      domain.DashboardUser{PublicID: "user-new"},
		Organizations: []domain.DashboardOrganization{
			{PublicID: "org-1"},
		},
	})

	require.Error(t, err)

	storedUserToken, _ := store.Get(ports.KeyDashboardUserToken)
	assert.Equal(t, "stale-user", storedUserToken)
	storedOrgToken, _ := store.Get(ports.KeyDashboardOrgToken)
	assert.Equal(t, "stale-org-token", storedOrgToken)
	storedOrgID, _ := store.Get(ports.KeyDashboardOrgPublicID)
	assert.Equal(t, "stale-org-id", storedOrgID)
	appID, _ := store.Get(ports.KeyDashboardAppID)
	assert.Equal(t, "stale-app", appID)
	appRegion, _ := store.Get(ports.KeyDashboardAppRegion)
	assert.Equal(t, "eu", appRegion)
}
