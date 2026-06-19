package dashboard

import (
	"context"
	"errors"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// DomainService handles inbox/agent-account domain management via dashboard-account.
type DomainService struct {
	account ports.DashboardAccountClient
	secrets ports.SecretStore
}

// NewDomainService creates a new dashboard domain service.
func NewDomainService(account ports.DashboardAccountClient, secrets ports.SecretStore) *DomainService {
	return &DomainService{
		account: account,
		secrets: secrets,
	}
}

// ListDomains lists domains for the active dashboard organization.
func (s *DomainService) ListDomains(ctx context.Context, limit int, pageToken string) (domain.DashboardInboxDomainPage, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (domain.DashboardInboxDomainPage, error) {
		return s.account.ListDomains(ctx, limit, pageToken, userToken, orgToken)
	})
}

// GetDomain retrieves a domain by ID or domain address.
func (s *DomainService) GetDomain(ctx context.Context, domainIDOrAddress, region string) (*domain.DashboardInboxDomain, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
		return s.account.GetDomain(ctx, domainIDOrAddress, region, userToken, orgToken)
	})
}

// CheckAvailability checks whether a domain address is already used in the active org.
func (s *DomainService) CheckAvailability(ctx context.Context, domainAddress string) (*domain.DashboardInboxDomainAvailability, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardInboxDomainAvailability, error) {
		return s.account.CheckDomainAvailability(ctx, domainAddress, userToken, orgToken)
	})
}

// CreateDomain creates/registers a domain in dashboard-account.
func (s *DomainService) CreateDomain(ctx context.Context, input domain.DashboardCreateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
		return s.account.CreateDomain(ctx, input, userToken, orgToken)
	})
}

// UpdateDomain updates a domain's display name.
func (s *DomainService) UpdateDomain(ctx context.Context, domainID, region string, input domain.DashboardUpdateInboxDomainInput) (*domain.DashboardInboxDomain, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardInboxDomain, error) {
		return s.account.UpdateDomain(ctx, domainID, region, input, userToken, orgToken)
	})
}

// DeleteDomain deletes a domain.
func (s *DomainService) DeleteDomain(ctx context.Context, domainID, region string) (bool, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (bool, error) {
		return s.account.DeleteDomain(ctx, domainID, region, userToken, orgToken)
	})
}

// GetDomainInfo returns DNS-record info for a verification type.
func (s *DomainService) GetDomainInfo(ctx context.Context, domainID, region, verificationType string) (*domain.DashboardDomainVerificationResult, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardDomainVerificationResult, error) {
		return s.account.GetDomainInfo(ctx, domainID, region, verificationType, userToken, orgToken)
	})
}

// VerifyDomain triggers DNS/authentication verification for a domain.
func (s *DomainService) VerifyDomain(ctx context.Context, domainID, region string, input domain.DashboardVerifyInboxDomainInput) (*domain.DashboardDomainVerificationResult, error) {
	return withDomainSessionRetry(ctx, s, func(userToken, orgToken string) (*domain.DashboardDomainVerificationResult, error) {
		return s.account.VerifyDomain(ctx, domainID, region, input, userToken, orgToken)
	})
}

func (s *DomainService) loadTokens() (userToken, orgToken string, err error) {
	return loadDashboardTokens(s.secrets)
}

func withDomainSessionRetry[T any](ctx context.Context, s *DomainService, call func(userToken, orgToken string) (T, error)) (T, error) {
	userToken, orgToken, err := s.loadTokens()
	var zero T
	if err != nil {
		return zero, err
	}

	result, err := call(userToken, orgToken)
	if !errors.Is(err, domain.ErrDashboardSessionExpired) {
		return result, err
	}

	userToken, orgToken, err = NewAuthService(s.account, s.secrets).refreshTokens(ctx, userToken, orgToken)
	if err != nil {
		return zero, err
	}
	return call(userToken, orgToken)
}
