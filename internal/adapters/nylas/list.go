package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

type listListResponse struct {
	Data       []domain.AgentList `json:"data"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type listResponse struct {
	Data domain.AgentList `json:"data"`
}

// agentListItem is one entry of GET /v3/lists/{id}/items; only the normalized
// value is exposed to callers.
type agentListItem struct {
	Value string `json:"value"`
}

type listItemsResponse struct {
	Data       []agentListItem `json:"data"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// ListLists lists all lists available to the authenticated application.
func (c *HTTPClient) ListLists(ctx context.Context) ([]domain.AgentList, error) {
	baseURL := fmt.Sprintf("%s/v3/lists", c.baseURL)
	pageToken := ""
	lists := make([]domain.AgentList, 0)

	for {
		queryBuilder := NewQueryBuilder()
		if pageToken != "" {
			queryBuilder.Add("page_token", pageToken)
		}
		queryURL := queryBuilder.BuildURL(baseURL)

		var result listListResponse
		if err := c.doGet(ctx, queryURL, &result); err != nil {
			return nil, err
		}

		lists = append(lists, result.Data...)

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate lists: repeated cursor %q", result.NextCursor)
		}
		pageToken = result.NextCursor
	}

	return lists, nil
}

// GetList retrieves a list by ID.
func (c *HTTPClient) GetList(ctx context.Context, listID string) (*domain.AgentList, error) {
	queryURL := fmt.Sprintf("%s/v3/lists/%s", c.baseURL, url.PathEscape(listID))

	var result listResponse
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrListNotFound); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateList creates a new list.
func (c *HTTPClient) CreateList(ctx context.Context, payload map[string]any) (*domain.AgentList, error) {
	queryURL := fmt.Sprintf("%s/v3/lists", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result listResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// UpdateList updates an existing list's metadata (name, description).
func (c *HTTPClient) UpdateList(ctx context.Context, listID string, payload map[string]any) (*domain.AgentList, error) {
	queryURL := fmt.Sprintf("%s/v3/lists/%s", c.baseURL, url.PathEscape(listID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result listResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Data.ID == "" {
		result.Data.ID = listID
	}

	return &result.Data, nil
}

// DeleteList deletes a list by ID.
func (c *HTTPClient) DeleteList(ctx context.Context, listID string) error {
	queryURL := fmt.Sprintf("%s/v3/lists/%s", c.baseURL, url.PathEscape(listID))
	return c.doDelete(ctx, queryURL)
}

// GetListItems retrieves all items of a list, following pagination.
func (c *HTTPClient) GetListItems(ctx context.Context, listID string) ([]string, error) {
	baseURL := fmt.Sprintf("%s/v3/lists/%s/items", c.baseURL, url.PathEscape(listID))
	pageToken := ""
	items := make([]string, 0)

	for {
		queryBuilder := NewQueryBuilder()
		if pageToken != "" {
			queryBuilder.Add("page_token", pageToken)
		}
		queryURL := queryBuilder.BuildURL(baseURL)

		var result listItemsResponse
		if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrListNotFound); err != nil {
			return nil, err
		}

		for _, item := range result.Data {
			items = append(items, item.Value)
		}

		if result.NextCursor == "" {
			break
		}
		if result.NextCursor == pageToken {
			return nil, fmt.Errorf("failed to paginate list items: repeated cursor %q", result.NextCursor)
		}
		pageToken = result.NextCursor
	}

	return items, nil
}

// AddListItems adds items to a list (up to 1000 per request). Values are
// normalized and validated against the list's type by the API.
func (c *HTTPClient) AddListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return c.modifyListItems(ctx, "POST", listID, items)
}

// RemoveListItems removes items from a list.
func (c *HTTPClient) RemoveListItems(ctx context.Context, listID string, items []string) (*domain.AgentList, error) {
	return c.modifyListItems(ctx, "DELETE", listID, items)
}

func (c *HTTPClient) modifyListItems(ctx context.Context, method, listID string, items []string) (*domain.AgentList, error) {
	queryURL := fmt.Sprintf("%s/v3/lists/%s/items", c.baseURL, url.PathEscape(listID))

	resp, err := c.doJSONRequest(ctx, method, queryURL, map[string]any{"items": items})
	if err != nil {
		return nil, err
	}

	var result listResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	if result.Data.ID == "" {
		result.Data.ID = listID
	}

	return &result.Data, nil
}
