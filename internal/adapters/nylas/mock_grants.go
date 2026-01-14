package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.Grant, error) {
	m.ExchangeCodeCalled = true
	if m.ExchangeCodeFunc != nil {
		return m.ExchangeCodeFunc(ctx, code, redirectURI)
	}
	return &domain.Grant{
		ID:          "mock-grant-id",
		Email:       "test@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}

// ListGrants lists all grants.
func (m *MockClient) ListGrants(ctx context.Context) ([]domain.Grant, error) {
	m.ListGrantsCalled = true
	if m.ListGrantsFunc != nil {
		return m.ListGrantsFunc(ctx)
	}
	return []domain.Grant{}, nil
}

// GetGrant retrieves a specific grant.
func (m *MockClient) GetGrant(ctx context.Context, grantID string) (*domain.Grant, error) {
	m.GetGrantCalled = true
	m.LastGrantID = grantID
	if m.GetGrantFunc != nil {
		return m.GetGrantFunc(ctx, grantID)
	}
	return &domain.Grant{
		ID:          grantID,
		Email:       "test@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}

// RevokeGrant revokes a grant.
func (m *MockClient) RevokeGrant(ctx context.Context, grantID string) error {
	m.RevokeGrantCalled = true
	m.LastGrantID = grantID
	if m.RevokeGrantFunc != nil {
		return m.RevokeGrantFunc(ctx, grantID)
	}
	return nil
}

// GetMessages retrieves recent messages.
