package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListLists(ctx context.Context) ([]domain.AgentList, error) {
	return []domain.AgentList{
		{
			ID:             "list-1",
			Name:           "Blocked domains",
			Type:           "domain",
			ItemsCount:     2,
			ApplicationID:  "app-123",
			OrganizationID: "org-123",
		},
	}, nil
}

func (m *MockClient) GetList(ctx context.Context, listID string) (*domain.AgentList, error) {
	return &domain.AgentList{
		ID:             listID,
		Name:           "Blocked domains",
		Type:           "domain",
		ItemsCount:     2,
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
	}, nil
}

func (m *MockClient) CreateList(ctx context.Context, payload map[string]any) (*domain.AgentList, error) {
	list := &domain.AgentList{ID: "list-new", ItemsCount: 0}
	if name, ok := payload["name"].(string); ok {
		list.Name = name
	}
	if listType, ok := payload["type"].(string); ok {
		list.Type = listType
	}
	if description, ok := payload["description"].(string); ok {
		list.Description = description
	}
	return list, nil
}

func (m *MockClient) UpdateList(ctx context.Context, listID string, payload map[string]any) (*domain.AgentList, error) {
	list := &domain.AgentList{ID: listID, Type: "domain"}
	if name, ok := payload["name"].(string); ok {
		list.Name = name
	}
	if description, ok := payload["description"].(string); ok {
		list.Description = description
	}
	return list, nil
}

func (m *MockClient) DeleteList(ctx context.Context, listID string) error {
	return nil
}

func (m *MockClient) GetListItems(ctx context.Context, listID string) ([]string, error) {
	return []string{"spam.com", "junk.net"}, nil
}

func (m *MockClient) AddListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return &domain.AgentList{ID: listID, Type: "domain", ItemsCount: 2 + len(items)}, nil
}

func (m *MockClient) RemoveListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return &domain.AgentList{ID: listID, Type: "domain", ItemsCount: max(0, 2-len(items))}, nil
}
