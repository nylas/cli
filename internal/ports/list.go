package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// ListClient defines list management operations for rule in_list conditions.
type ListClient interface {
	ListLists(ctx context.Context) ([]domain.AgentList, error)
	GetList(ctx context.Context, listID string) (*domain.AgentList, error)
	CreateList(ctx context.Context, payload map[string]any) (*domain.AgentList, error)
	UpdateList(ctx context.Context, listID string, payload map[string]any) (*domain.AgentList, error)
	DeleteList(ctx context.Context, listID string) error
	GetListItems(ctx context.Context, listID string) ([]string, error)
	AddListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error)
	RemoveListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error)
}
