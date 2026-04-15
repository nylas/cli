package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// RuleClient defines rule management operations.
type RuleClient interface {
	ListRules(ctx context.Context) ([]domain.Rule, error)
	GetRule(ctx context.Context, ruleID string) (*domain.Rule, error)
	CreateRule(ctx context.Context, payload map[string]any) (*domain.Rule, error)
	UpdateRule(ctx context.Context, ruleID string, payload map[string]any) (*domain.Rule, error)
	DeleteRule(ctx context.Context, ruleID string) error
}
