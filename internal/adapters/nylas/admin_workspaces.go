package nylas

import (
	"fmt"
	"net/url"

	"context"

	"github.com/nylas/cli/internal/domain"
)

// Workspace admin operations (/v3/workspaces).

// ListWorkspaces retrieves all workspaces.
func (c *HTTPClient) ListWorkspaces(ctx context.Context) ([]domain.Workspace, error) {
	queryURL := fmt.Sprintf("%s/v3/workspaces", c.baseURL)

	var result struct {
		Data []domain.Workspace `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetWorkspace retrieves a grant workspace.
func (c *HTTPClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	if err := validateRequired("workspace ID", workspaceID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/workspaces/%s", c.baseURL, url.PathEscape(workspaceID))

	var result struct {
		Data domain.Workspace `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrWorkspaceNotFound); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CreateWorkspace creates a new workspace.
func (c *HTTPClient) CreateWorkspace(ctx context.Context, req *domain.CreateWorkspaceRequest) (*domain.Workspace, error) {
	if req == nil {
		return nil, fmt.Errorf("create workspace request is required")
	}

	queryURL := fmt.Sprintf("%s/v3/workspaces", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Workspace `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateWorkspace updates workspace policy/rule attachments.
func (c *HTTPClient) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	if err := validateRequired("workspace ID", workspaceID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("update workspace request is required")
	}

	queryURL := fmt.Sprintf("%s/v3/workspaces/%s", c.baseURL, url.PathEscape(workspaceID))

	resp, err := c.doJSONRequest(ctx, "PATCH", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.Workspace `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// DeleteWorkspace deletes a workspace.
func (c *HTTPClient) DeleteWorkspace(ctx context.Context, workspaceID string) error {
	if err := validateRequired("workspace ID", workspaceID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/workspaces/%s", c.baseURL, url.PathEscape(workspaceID))
	return c.doDelete(ctx, queryURL)
}

// AssignWorkspaceGrants moves grants into or out of a workspace via the
// manual-assign endpoint. Assigning a grant moves it even if it currently
// belongs to another workspace; removing leaves it in no workspace.
func (c *HTTPClient) AssignWorkspaceGrants(ctx context.Context, workspaceID string, req *domain.WorkspaceAssignRequest) (*domain.WorkspaceAssignResult, error) {
	if err := validateRequired("workspace ID", workspaceID); err != nil {
		return nil, err
	}
	if req == nil || (len(req.AssignGrants) == 0 && len(req.RemoveGrants) == 0) {
		return nil, fmt.Errorf("at least one grant must be assigned or removed")
	}

	queryURL := fmt.Sprintf("%s/v3/workspaces/%s/manual-assign", c.baseURL, url.PathEscape(workspaceID))

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data domain.WorkspaceAssignResult `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return &result.Data, nil
}
