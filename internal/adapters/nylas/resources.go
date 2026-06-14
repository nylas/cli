package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// ListRoomResources retrieves the bookable room and equipment resources a grant
// has access to (GET /v3/grants/{id}/resources).
func (c *HTTPClient) ListRoomResources(ctx context.Context, grantID string) ([]domain.RoomResource, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/resources", c.baseURL, url.PathEscape(grantID))

	var result struct {
		Data []domain.RoomResource `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}
