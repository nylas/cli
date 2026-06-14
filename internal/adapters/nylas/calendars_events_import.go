package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// importEventsDefaultLimit mirrors the API default page size for the import
// endpoint (max allowed is 500).
const importEventsDefaultLimit = 50

// ImportEvents bulk-reads events from a calendar over a time window
// (GET /v3/grants/{id}/events/import), including expanded recurring instances.
// Unlike GetEvents this endpoint is intended for migration/export and requires
// a calendar ID. When start/end are omitted the API defaults to now .. +1 month.
func (c *HTTPClient) ImportEvents(ctx context.Context, grantID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if params == nil || params.CalendarID == "" {
		return nil, fmt.Errorf("%w: calendar ID is required for import", domain.ErrInvalidInput)
	}
	// Don't mutate the caller's params: a shared/reused EventQueryParams must
	// not be silently altered by this call.
	limit := params.Limit
	if limit <= 0 {
		limit = importEventsDefaultLimit
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/events/import", c.baseURL, url.PathEscape(grantID))
	queryURL := NewQueryBuilder().
		Add("calendar_id", params.CalendarID).
		AddInt("limit", limit).
		Add("page_token", params.PageToken).
		AddInt64("start", params.Start).
		AddInt64("end", params.End).
		BuildURL(baseURL)

	var result struct {
		Data []eventResponse `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return convertEvents(result.Data), nil
}
