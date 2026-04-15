package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// PolicyClient defines policy management operations.
type PolicyClient interface {
	ListPolicies(ctx context.Context) ([]domain.Policy, error)
	GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error)
	CreatePolicy(ctx context.Context, payload map[string]any) (*domain.Policy, error)
	UpdatePolicy(ctx context.Context, policyID string, payload map[string]any) (*domain.Policy, error)
	DeletePolicy(ctx context.Context, policyID string) error
}
