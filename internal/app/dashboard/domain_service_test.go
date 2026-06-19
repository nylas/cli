package dashboard

import (
	"context"
	"testing"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDomainServiceForwardsDashboardTokens(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	seedTokens(store, "user-token", "org-token")

	var calls []string
	mock := &dashboardadapter.MockAccountClient{
		ListDomainsFn: func(_ context.Context, limit int, pageToken, userToken, orgToken string) (domain.DashboardInboxDomainPage, error) {
			calls = append(calls, "list")
			assert.Equal(t, 25, limit)
			assert.Equal(t, "cursor", pageToken)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return domain.DashboardInboxDomainPage{
				Domains:    []domain.DashboardInboxDomain{{ID: "dom_1"}},
				NextCursor: "next",
			}, nil
		},
		GetDomainFn: func(_ context.Context, domainIDOrAddress, region, userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
			calls = append(calls, "get")
			assert.Equal(t, "example.com", domainIDOrAddress)
			assert.Equal(t, "us", region)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardInboxDomain{ID: "dom_1"}, nil
		},
		CheckDomainAvailabilityFn: func(_ context.Context, domainAddress, userToken, orgToken string) (*domain.DashboardInboxDomainAvailability, error) {
			calls = append(calls, "check")
			assert.Equal(t, "example.com", domainAddress)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardInboxDomainAvailability{Available: true}, nil
		},
		CreateDomainFn: func(_ context.Context, input domain.DashboardCreateInboxDomainInput, userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
			calls = append(calls, "create")
			assert.Equal(t, "example.com", input.DomainAddress)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardInboxDomain{ID: "dom_new"}, nil
		},
		UpdateDomainFn: func(_ context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput, userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
			calls = append(calls, "update")
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "us", region)
			assert.Equal(t, "Renamed", input.Name)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardInboxDomain{ID: "dom_1", Name: "Renamed"}, nil
		},
		DeleteDomainFn: func(_ context.Context, domainID, region, userToken, orgToken string) (bool, error) {
			calls = append(calls, "delete")
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "us", region)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return true, nil
		},
		GetDomainInfoFn: func(_ context.Context, domainID, region, verificationType, userToken, orgToken string) (*domain.DashboardDomainVerificationResult, error) {
			calls = append(calls, "info")
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "us", region)
			assert.Equal(t, "mx", verificationType)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardDomainVerificationResult{Status: "pending"}, nil
		},
		VerifyDomainFn: func(_ context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput, userToken, orgToken string) (*domain.DashboardDomainVerificationResult, error) {
			calls = append(calls, "verify")
			assert.Equal(t, "dom_1", domainID)
			assert.Equal(t, "us", region)
			assert.Equal(t, "mx", input.Type)
			assert.Equal(t, "user-token", userToken)
			assert.Equal(t, "org-token", orgToken)
			return &domain.DashboardDomainVerificationResult{Status: "done"}, nil
		},
	}

	svc := NewDomainService(mock, store)
	ctx := context.Background()

	page, err := svc.ListDomains(ctx, 25, "cursor")
	require.NoError(t, err)
	assert.Equal(t, "next", page.NextCursor)
	_, err = svc.GetDomain(ctx, "example.com", "us")
	require.NoError(t, err)
	_, err = svc.CheckAvailability(ctx, "example.com")
	require.NoError(t, err)
	_, err = svc.CreateDomain(ctx, domain.DashboardCreateInboxDomainInput{DomainAddress: "example.com"})
	require.NoError(t, err)
	_, err = svc.UpdateDomain(ctx, "dom_1", "us", domain.DashboardUpdateInboxDomainInput{Name: "Renamed"})
	require.NoError(t, err)
	_, err = svc.DeleteDomain(ctx, "dom_1", "us")
	require.NoError(t, err)
	_, err = svc.GetDomainInfo(ctx, "dom_1", "us", "mx")
	require.NoError(t, err)
	_, err = svc.VerifyDomain(ctx, "dom_1", "us", domain.DashboardVerifyInboxDomainInput{Type: "mx"})
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"list", "get", "check", "create", "update", "delete", "info", "verify"}, calls)
}

func TestDomainServiceReturnsNotLoggedInWhenTokensMissing(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	svc := NewDomainService(&dashboardadapter.MockAccountClient{}, store)

	_, err := svc.ListDomains(context.Background(), 25, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrDashboardNotLoggedIn)
}

func TestDomainServiceRefreshesExpiredSessionAndRetries(t *testing.T) {
	t.Parallel()

	store := newMemSecretStore()
	seedTokens(store, "user-token-old", "org-token-old")

	checkCalls := 0
	mock := &dashboardadapter.MockAccountClient{
		CheckDomainAvailabilityFn: func(_ context.Context, domainAddress, userToken, orgToken string) (*domain.DashboardInboxDomainAvailability, error) {
			checkCalls++
			assert.Equal(t, "example.com", domainAddress)
			if checkCalls == 1 {
				assert.Equal(t, "user-token-old", userToken)
				assert.Equal(t, "org-token-old", orgToken)
				return nil, domain.NewDashboardAPIError(401, "INVALID_SESSION", "Invalid or expired session")
			}
			assert.Equal(t, "user-token-new", userToken)
			assert.Equal(t, "org-token-new", orgToken)
			return &domain.DashboardInboxDomainAvailability{DomainAddress: domainAddress, Available: true}, nil
		},
		RefreshFn: func(_ context.Context, userToken, orgToken string) (*domain.DashboardRefreshResponse, error) {
			assert.Equal(t, "user-token-old", userToken)
			assert.Equal(t, "org-token-old", orgToken)
			return &domain.DashboardRefreshResponse{
				UserToken: "user-token-new",
				OrgToken:  "org-token-new",
			}, nil
		},
	}

	svc := NewDomainService(mock, store)
	result, err := svc.CheckAvailability(context.Background(), "example.com")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Available)
	assert.Equal(t, 2, checkCalls)

	storedUserToken, err := store.Get(ports.KeyDashboardUserToken)
	require.NoError(t, err)
	assert.Equal(t, "user-token-new", storedUserToken)
	storedOrgToken, err := store.Get(ports.KeyDashboardOrgToken)
	require.NoError(t, err)
	assert.Equal(t, "org-token-new", storedOrgToken)
}
