package nylas

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

func (c *HTTPClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars", c.baseURL, url.PathEscape(grantID))

	var result struct {
		Data []calendarResponse `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return convertCalendars(result.Data), nil
}

// GetCalendar retrieves a single calendar by ID.
func (c *HTTPClient) GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(calendarID))

	var result struct {
		Data calendarResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrCalendarNotFound); err != nil {
		return nil, err
	}

	cal := convertCalendar(result.Data)
	return &cal, nil
}

// CreateCalendar creates a new calendar.
func (c *HTTPClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars", c.baseURL, url.PathEscape(grantID))

	payload := map[string]any{
		"name": req.Name,
	}
	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.Location != "" {
		payload["location"] = req.Location
	}
	if req.Timezone != "" {
		payload["timezone"] = req.Timezone
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data calendarResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	cal := convertCalendar(result.Data)
	return &cal, nil
}

// UpdateCalendar updates an existing calendar.
func (c *HTTPClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(calendarID))

	payload := make(map[string]any)
	if req.Name != nil {
		payload["name"] = *req.Name
	}
	if req.Description != nil {
		payload["description"] = *req.Description
	}
	if req.Location != nil {
		payload["location"] = *req.Location
	}
	if req.Timezone != nil {
		payload["timezone"] = *req.Timezone
	}
	if req.HexColor != nil {
		payload["hex_color"] = *req.HexColor
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data calendarResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	cal := convertCalendar(result.Data)
	return &cal, nil
}

// DeleteCalendar deletes a calendar.
func (c *HTTPClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	if err := validateRequired("grant ID", grantID); err != nil {
		return err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(calendarID))
	return c.doDelete(ctx, queryURL)
}

// GetEvents retrieves events for a calendar.
