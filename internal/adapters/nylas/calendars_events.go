package nylas

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nylas/cli/internal/domain"
)

func (c *HTTPClient) GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return nil, err
	}
	result, err := c.GetEventsWithCursor(ctx, grantID, calendarID, params)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// GetEventsWithCursor retrieves events with pagination cursor support.
func (c *HTTPClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	if params == nil {
		params = &domain.EventQueryParams{Limit: 10}
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/events", c.baseURL, grantID)
	queryURL := NewQueryBuilder().
		Add("calendar_id", calendarID).
		AddInt("limit", params.Limit).
		Add("page_token", params.PageToken).
		AddInt64("start", params.Start).
		AddInt64("end", params.End).
		Add("title", params.Title).
		Add("location", params.Location).
		AddBool("show_cancelled", params.ShowCancelled).
		AddBool("expand_recurring", params.ExpandRecurring).
		AddBoolPtr("busy", params.Busy).
		Add("order_by", params.OrderBy).
		BuildURL(baseURL)

	resp, err := c.doJSONRequest(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data       []eventResponse `json:"data"`
		NextCursor string          `json:"next_cursor,omitempty"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &domain.EventListResponse{
		Data: convertEvents(result.Data),
		Pagination: domain.Pagination{
			NextCursor: result.NextCursor,
			HasMore:    result.NextCursor != "",
		},
	}, nil
}

// GetEvent retrieves a single event by ID.
func (c *HTTPClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	if err := validateRequired("grant ID", grantID); err != nil {
		return nil, err
	}
	if err := validateRequired("calendar ID", calendarID); err != nil {
		return nil, err
	}
	if err := validateRequired("event ID", eventID); err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("%s/v3/grants/%s/events/%s", c.baseURL, grantID, eventID)
	queryURL := NewQueryBuilder().Add("calendar_id", calendarID).BuildURL(baseURL)

	var result struct {
		Data eventResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrEventNotFound); err != nil {
		return nil, err
	}

	event := convertEvent(result.Data)
	return &event, nil
}

// CreateEvent creates a new event.
func (c *HTTPClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	baseURL := fmt.Sprintf("%s/v3/grants/%s/events", c.baseURL, grantID)
	queryURL := NewQueryBuilder().Add("calendar_id", calendarID).BuildURL(baseURL)

	payload := map[string]any{
		"title": req.Title,
		"when":  req.When,
	}

	if req.Description != "" {
		payload["description"] = req.Description
	}
	if req.Location != "" {
		payload["location"] = req.Location
	}
	if len(req.Participants) > 0 {
		payload["participants"] = req.Participants
	}
	payload["busy"] = req.Busy
	if req.Visibility != "" {
		payload["visibility"] = req.Visibility
	}
	if len(req.Recurrence) > 0 {
		payload["recurrence"] = req.Recurrence
	}
	if req.Conferencing != nil {
		payload["conferencing"] = req.Conferencing
	}
	if req.Reminders != nil {
		payload["reminders"] = req.Reminders
	}
	if len(req.Metadata) > 0 {
		payload["metadata"] = req.Metadata
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data eventResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	event := convertEvent(result.Data)
	return &event, nil
}

// UpdateEvent updates an existing event.
func (c *HTTPClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	baseURL := fmt.Sprintf("%s/v3/grants/%s/events/%s", c.baseURL, grantID, eventID)
	queryURL := NewQueryBuilder().Add("calendar_id", calendarID).BuildURL(baseURL)

	payload := make(map[string]any)
	if req.Title != nil {
		payload["title"] = *req.Title
	}
	if req.Description != nil {
		payload["description"] = *req.Description
	}
	if req.Location != nil {
		payload["location"] = *req.Location
	}
	if req.When != nil {
		payload["when"] = req.When
	}
	if len(req.Participants) > 0 {
		payload["participants"] = req.Participants
	}
	if req.Busy != nil {
		payload["busy"] = *req.Busy
	}
	if req.Visibility != nil {
		payload["visibility"] = *req.Visibility
	}
	if len(req.Recurrence) > 0 {
		payload["recurrence"] = req.Recurrence
	}
	if req.Conferencing != nil {
		payload["conferencing"] = req.Conferencing
	}
	if req.Reminders != nil {
		payload["reminders"] = req.Reminders
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data eventResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	event := convertEvent(result.Data)
	return &event, nil
}

// DeleteEvent deletes an event.
func (c *HTTPClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	baseURL := fmt.Sprintf("%s/v3/grants/%s/events/%s", c.baseURL, grantID, eventID)
	queryURL := NewQueryBuilder().Add("calendar_id", calendarID).BuildURL(baseURL)
	return c.doDelete(ctx, queryURL)
}

// SendRSVP sends an RSVP response to an event invitation.
func (c *HTTPClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	baseURL := fmt.Sprintf("%s/v3/grants/%s/events/%s/send-rsvp", c.baseURL, grantID, eventID)
	queryURL := NewQueryBuilder().Add("calendar_id", calendarID).BuildURL(baseURL)

	payload := map[string]any{
		"status": req.Status,
	}
	if req.Comment != "" {
		payload["comment"] = req.Comment
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload, http.StatusOK, http.StatusAccepted)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// GetFreeBusy retrieves free/busy information.
func (c *HTTPClient) GetFreeBusy(ctx context.Context, grantID string, freeBusyReq *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/calendars/free-busy", c.baseURL, grantID)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, freeBusyReq)
	if err != nil {
		return nil, err
	}

	var result domain.FreeBusyResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAvailability retrieves availability information.
func (c *HTTPClient) GetAvailability(ctx context.Context, availReq *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	queryURL := fmt.Sprintf("%s/v3/calendars/availability", c.baseURL)

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, availReq)
	if err != nil {
		return nil, err
	}

	var result domain.AvailabilityResponse
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CreateVirtualCalendarGrant creates a virtual calendar grant (account).
