package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListLists(ctx context.Context) ([]domain.AgentList, error) {
	return []domain.AgentList{
		{
			ID:             "list-demo-1",
			Name:           "Demo blocked domains",
			Description:    "Domains blocked in the demo workspace",
			Type:           "domain",
			ItemsCount:     2,
			ApplicationID:  "app-demo",
			OrganizationID: "org-demo",
		},
		{
			ID:             "list-demo-2",
			Name:           "Demo VIP addresses",
			Type:           "address",
			ItemsCount:     1,
			ApplicationID:  "app-demo",
			OrganizationID: "org-demo",
		},
	}, nil
}

func (d *DemoClient) GetList(ctx context.Context, listID string) (*domain.AgentList, error) {
	return &domain.AgentList{
		ID:             listID,
		Name:           "Demo blocked domains",
		Description:    "Domains blocked in the demo workspace",
		Type:           "domain",
		ItemsCount:     2,
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) CreateList(ctx context.Context, payload map[string]any) (*domain.AgentList, error) {
	list := &domain.AgentList{ID: "list-demo-new", ApplicationID: "app-demo", OrganizationID: "org-demo"}
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

func (d *DemoClient) UpdateList(ctx context.Context, listID string, payload map[string]any) (*domain.AgentList, error) {
	list := &domain.AgentList{ID: listID, Type: "domain", ApplicationID: "app-demo", OrganizationID: "org-demo"}
	if name, ok := payload["name"].(string); ok {
		list.Name = name
	}
	if description, ok := payload["description"].(string); ok {
		list.Description = description
	}
	return list, nil
}

func (d *DemoClient) DeleteList(ctx context.Context, listID string) error {
	return nil
}

func (d *DemoClient) GetListItems(ctx context.Context, listID string) ([]string, error) {
	return []string{"spam-demo.com", "junk-demo.net"}, nil
}

func (d *DemoClient) AddListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return &domain.AgentList{ID: listID, Type: "domain", ItemsCount: 2 + len(items)}, nil
}

func (d *DemoClient) RemoveListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return &domain.AgentList{ID: listID, Type: "domain", ItemsCount: max(0, 2-len(items))}, nil
}
