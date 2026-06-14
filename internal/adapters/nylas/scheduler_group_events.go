package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// groupEventsBaseURL builds the grant + configuration scoped group-events URL.
func (c *HTTPClient) groupEventsBaseURL(grantID, configID string) string {
	return fmt.Sprintf("%s/v3/grants/%s/scheduling/configurations/%s/group-events",
		c.baseURL, url.PathEscape(grantID), url.PathEscape(configID))
}

// ListGroupEvents retrieves the group events for a configuration within a time
// window. The API requires calendar_id, start_time, and end_time query params.
func (c *HTTPClient) ListGroupEvents(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return nil, err
	}
	if startTime <= 0 || endTime <= 0 {
		return nil, fmt.Errorf("%w: start and end time are required to list group events", domain.ErrInvalidInput)
	}

	queryURL := NewQueryBuilder().
		Add("calendar_id", calendarID).
		AddInt64("start_time", startTime).
		AddInt64("end_time", endTime).
		BuildURL(c.groupEventsBaseURL(grantID, configID))

	var result struct {
		Data []domain.GroupEvent `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// CreateGroupEvent creates a group event under a configuration.
func (c *HTTPClient) CreateGroupEvent(ctx context.Context, grantID, configID string, req *domain.CreateGroupEventRequest) ([]domain.GroupEvent, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: group event request cannot be nil", domain.ErrInvalidInput)
	}

	resp, err := c.doJSONRequest(ctx, "POST", c.groupEventsBaseURL(grantID, configID), req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.GroupEvent `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// UpdateGroupEvent updates a group event.
func (c *HTTPClient) UpdateGroupEvent(ctx context.Context, grantID, configID, eventID string, req *domain.UpdateGroupEventRequest) ([]domain.GroupEvent, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}
	if err := validateRequired("event ID", eventID); err != nil {
		return nil, err
	}
	if req == nil {
		return nil, fmt.Errorf("%w: group event request cannot be nil", domain.ErrInvalidInput)
	}

	queryURL := fmt.Sprintf("%s/%s", c.groupEventsBaseURL(grantID, configID), url.PathEscape(eventID))
	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.GroupEvent `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// DeleteGroupEvent deletes a group event.
func (c *HTTPClient) DeleteGroupEvent(ctx context.Context, grantID, configID, eventID string) error {
	if err := validateRequired("grant ID", grantID); err != nil {
		return err
	}
	if err := validateRequired("configuration ID", configID); err != nil {
		return err
	}
	if err := validateRequired("event ID", eventID); err != nil {
		return err
	}

	queryURL := fmt.Sprintf("%s/%s", c.groupEventsBaseURL(grantID, configID), url.PathEscape(eventID))
	return c.doDelete(ctx, queryURL)
}

// ImportGroupEvents imports existing provider events as group events. This
// endpoint is configuration-scoped (not grant-scoped).
func (c *HTTPClient) ImportGroupEvents(ctx context.Context, configID string, items []domain.ImportGroupEventItem) ([]domain.GroupEvent, error) {
	if err := validateRequired("configuration ID", configID); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%w: at least one event to import is required", domain.ErrInvalidInput)
	}

	queryURL := fmt.Sprintf("%s/v3/scheduling/configurations/%s/import-group-events", c.baseURL, url.PathEscape(configID))
	// Per the OpenAPI spec (requests/scheduling/import_group_event.yaml), the
	// request body is a bare JSON array of import items, not a wrapped object.
	resp, err := c.doJSONRequest(ctx, "POST", queryURL, items)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []domain.GroupEvent `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
