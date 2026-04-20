package dashboard

import (
	"context"
	"errors"
	"sync"
	"testing"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppServiceListApplications(t *testing.T) {
	t.Parallel()

	t.Run("region filter forwards single request", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		seedTokens(store, "user-token", "org-token")

		mock := &dashboardadapter.MockGatewayClient{
			ListApplicationsFn: func(_ context.Context, orgPublicID, region, userToken, orgToken string) ([]domain.GatewayApplication, error) {
				assert.Equal(t, "org-1", orgPublicID)
				assert.Equal(t, "eu", region)
				assert.Equal(t, "user-token", userToken)
				assert.Equal(t, "org-token", orgToken)
				return []domain.GatewayApplication{{ApplicationID: "app-eu", Region: "eu"}}, nil
			},
		}

		svc := NewAppService(mock, store)
		apps, err := svc.ListApplications(context.Background(), "org-1", "eu")
		require.NoError(t, err)
		require.Len(t, apps, 1)
		assert.Equal(t, "app-eu", apps[0].ApplicationID)
	})

	t.Run("merges regions, tolerates one failure, and deduplicates", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		seedTokens(store, "user-token", "org-token")

		mock := &dashboardadapter.MockGatewayClient{
			ListApplicationsFn: func(_ context.Context, _ string, region, _, _ string) ([]domain.GatewayApplication, error) {
				switch region {
				case "us":
					return []domain.GatewayApplication{
						{ApplicationID: "shared-app", Region: "us"},
						{ApplicationID: "us-only", Region: "us"},
					}, nil
				case "eu":
					return []domain.GatewayApplication{
						{ApplicationID: "shared-app", Region: "eu"},
					}, errors.New("eu unavailable")
				default:
					return nil, nil
				}
			},
		}

		svc := NewAppService(mock, store)
		apps, err := svc.ListApplications(context.Background(), "org-1", "")
		require.Error(t, err)
		var partialErr *domain.DashboardPartialResultError
		require.ErrorAs(t, err, &partialErr)
		require.Len(t, apps, 2)
		assert.ElementsMatch(t, []string{"shared-app", "us-only"}, []string{apps[0].ApplicationID, apps[1].ApplicationID})
	})

	t.Run("returns first error when both regions fail", func(t *testing.T) {
		t.Parallel()

		store := newMemSecretStore()
		seedTokens(store, "user-token", "org-token")

		mock := &dashboardadapter.MockGatewayClient{
			ListApplicationsFn: func(_ context.Context, _ string, region, _, _ string) ([]domain.GatewayApplication, error) {
				return nil, errors.New(region + " failed")
			},
		}

		svc := NewAppService(mock, store)
		apps, err := svc.ListApplications(context.Background(), "org-1", "")
		require.Error(t, err)
		assert.Nil(t, apps)
		assert.Contains(t, err.Error(), "failed to list applications")
	})
}

func TestAppServiceDeduplicateApps(t *testing.T) {
	t.Parallel()

	input := []domain.GatewayApplication{
		{ApplicationID: "app-1", Region: "us"},
		{ApplicationID: "app-1", Region: "eu"},
		{Region: "us", Environment: "sandbox", Branding: &domain.GatewayApplicationBrand{Name: "No ID"}},
		{Region: "us", Environment: "sandbox", Branding: &domain.GatewayApplicationBrand{Name: "No ID"}},
	}

	got := deduplicateApps(input)
	require.Len(t, got, 2)
	assert.Equal(t, "app-1", got[0].ApplicationID)
	assert.Equal(t, "No ID", got[1].Branding.Name)
}

func TestAppServiceManagementCalls(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	seedTokens(store, "user-token", "org-token")

	var mu sync.Mutex
	calls := make([]string, 0, 3)
	mock := &dashboardadapter.MockGatewayClient{
		CreateApplicationFn: func(_ context.Context, orgPublicID, region, name, userToken, orgToken string) (*domain.GatewayCreatedApplication, error) {
			mu.Lock()
			calls = append(calls, "createApp:"+orgPublicID+":"+region+":"+name+":"+userToken+":"+orgToken)
			mu.Unlock()
			return &domain.GatewayCreatedApplication{ApplicationID: "app-1"}, nil
		},
		ListAPIKeysFn: func(_ context.Context, appID, region, userToken, orgToken string) ([]domain.GatewayAPIKey, error) {
			mu.Lock()
			calls = append(calls, "listKeys:"+appID+":"+region+":"+userToken+":"+orgToken)
			mu.Unlock()
			return []domain.GatewayAPIKey{{ID: "key-1"}}, nil
		},
		CreateAPIKeyFn: func(_ context.Context, appID, region, name string, expiresInDays int, userToken, orgToken string) (*domain.GatewayCreatedAPIKey, error) {
			mu.Lock()
			calls = append(calls, "createKey:"+appID+":"+region+":"+name+":"+userToken+":"+orgToken)
			mu.Unlock()
			return &domain.GatewayCreatedAPIKey{ID: "key-2"}, nil
		},
	}

	svc := NewAppService(mock, store)

	app, err := svc.CreateApplication(context.Background(), "org-1", "us", "Primary")
	require.NoError(t, err)
	assert.Equal(t, "app-1", app.ApplicationID)

	keys, err := svc.ListAPIKeys(context.Background(), "app-1", "us")
	require.NoError(t, err)
	require.Len(t, keys, 1)

	key, err := svc.CreateAPIKey(context.Background(), "app-1", "us", "Nightly", 30)
	require.NoError(t, err)
	assert.Equal(t, "key-2", key.ID)

	assert.Contains(t, calls, "createApp:org-1:us:Primary:user-token:org-token")
	assert.Contains(t, calls, "listKeys:app-1:us:user-token:org-token")
	assert.Contains(t, calls, "createKey:app-1:us:Nightly:user-token:org-token")
}

func TestAppServiceReturnsNotLoggedInWhenTokensMissing(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	svc := NewAppService(&dashboardadapter.MockGatewayClient{}, store)

	_, err := svc.CreateApplication(context.Background(), "org-1", "us", "Primary")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDashboardNotLoggedIn)
}

func TestLoadDashboardTokensPropagatesSecretStoreFailures(t *testing.T) {
	t.Parallel()

	t.Run("user token load failure is returned", func(t *testing.T) {
		t.Parallel()

		store := &failingSecretStore{
			memSecretStore: newMemSecretStore(),
			failGetKey:     ports.KeyDashboardUserToken,
		}

		_, _, err := loadDashboardTokens(store)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load dashboard user token")
		assert.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	})

	t.Run("org token load failure is returned", func(t *testing.T) {
		t.Parallel()

		store := &failingSecretStore{
			memSecretStore: newMemSecretStore(),
			failGetKey:     ports.KeyDashboardOrgToken,
		}
		seedTokens(store, "user-token", "")

		_, _, err := loadDashboardTokens(store)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load dashboard organization token")
		assert.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	})
}
