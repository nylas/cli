package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ListGrants returns demo grants.
func (d *DemoClient) ListGrants(ctx context.Context) ([]domain.Grant, error) {
	return []domain.Grant{
		{
			ID:          "demo-grant-001",
			Email:       "demo@example.com",
			Provider:    domain.ProviderGoogle,
			GrantStatus: "valid",
			CreatedAt:   domain.UnixTime{Time: time.Now().Add(-30 * 24 * time.Hour)},
		},
		{
			ID:          "demo-grant-002",
			Email:       "work@company.com",
			Provider:    domain.ProviderMicrosoft,
			GrantStatus: "valid",
			CreatedAt:   domain.UnixTime{Time: time.Now().Add(-7 * 24 * time.Hour)},
		},
	}, nil
}

// GetGrant returns a demo grant.
func (d *DemoClient) GetGrant(ctx context.Context, grantID string) (*domain.Grant, error) {
	return &domain.Grant{
		ID:          grantID,
		Email:       "demo@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}

// RevokeGrant is a no-op for demo client.
func (d *DemoClient) RevokeGrant(ctx context.Context, grantID string) error {
	return nil
}
