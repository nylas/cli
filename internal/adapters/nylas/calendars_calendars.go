package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

func (c *HTTPClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []calendarResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
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

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, grantID, calendarID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: calendar not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data calendarResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
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

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars", c.baseURL, grantID)

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

	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, grantID, calendarID)

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
	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/%s", c.baseURL, grantID, calendarID)
	return c.doDelete(ctx, queryURL)
}

// GetEvents retrieves events for a calendar.
