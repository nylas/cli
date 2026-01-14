package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// folderResponse represents an API folder response.
type folderResponse struct {
	ID              string   `json:"id"`
	GrantID         string   `json:"grant_id"`
	Name            string   `json:"name"`
	SystemFolder    any      `json:"system_folder"` // Can be string or bool depending on provider
	ParentID        string   `json:"parent_id"`
	BackgroundColor string   `json:"background_color"`
	TextColor       string   `json:"text_color"`
	TotalCount      int      `json:"total_count"`
	UnreadCount     int      `json:"unread_count"`
	ChildIDs        []string `json:"child_ids"`
	Attributes      []string `json:"attributes"`
}

// GetFolders retrieves all folders for a grant.
func (c *HTTPClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/folders", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []folderResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return convertFolders(result.Data), nil
}

// GetFolder retrieves a single folder by ID.
func (c *HTTPClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("folder ID", folderID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/folders/%s", c.baseURL, grantID, folderID)

	var result struct {
		Data folderResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: folder not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}

	folder := convertFolder(result.Data)
	return &folder, nil
}

// CreateFolder creates a new folder.
func (c *HTTPClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/folders", c.baseURL, grantID)

	payload := map[string]any{
		"name": req.Name,
	}
	if req.ParentID != "" {
		payload["parent_id"] = req.ParentID
	}
	if req.BackgroundColor != "" {
		payload["background_color"] = req.BackgroundColor
	}
	if req.TextColor != "" {
		payload["text_color"] = req.TextColor
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data folderResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	folder := convertFolder(result.Data)
	return &folder, nil
}

// UpdateFolder updates an existing folder.
func (c *HTTPClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("folder ID", folderID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/folders/%s", c.baseURL, grantID, folderID)

	payload := make(map[string]any, 4) // Pre-allocate for up to 4 fields
	if req.Name != "" {
		payload["name"] = req.Name
	}
	if req.ParentID != "" {
		payload["parent_id"] = req.ParentID
	}
	if req.BackgroundColor != "" {
		payload["background_color"] = req.BackgroundColor
	}
	if req.TextColor != "" {
		payload["text_color"] = req.TextColor
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data folderResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	folder := convertFolder(result.Data)
	return &folder, nil
}

// DeleteFolder deletes a folder.
func (c *HTTPClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	if err := validateRequired("grant ID", grantID); err != nil {
		return err
	}
	if err := validateRequired("folder ID", folderID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/grants/%s/folders/%s", c.baseURL, grantID, folderID)
	return c.doDelete(ctx, queryURL)
}

// convertFolders converts API folder responses to domain models.
func convertFolders(folders []folderResponse) []domain.Folder {
	return util.Map(folders, convertFolder)
}

// convertFolder converts an API folder response to domain model.
func convertFolder(f folderResponse) domain.Folder {
	// SystemFolder can be a string or bool depending on provider
	var systemFolder string
	switch v := f.SystemFolder.(type) {
	case string:
		systemFolder = v
	case bool:
		if v {
			systemFolder = "true"
		}
		// If false, leave as empty string
	}

	return domain.Folder{
		ID:              f.ID,
		GrantID:         f.GrantID,
		Name:            f.Name,
		SystemFolder:    systemFolder,
		ParentID:        f.ParentID,
		BackgroundColor: f.BackgroundColor,
		TextColor:       f.TextColor,
		TotalCount:      f.TotalCount,
		UnreadCount:     f.UnreadCount,
		ChildIDs:        f.ChildIDs,
		Attributes:      f.Attributes,
	}
}
