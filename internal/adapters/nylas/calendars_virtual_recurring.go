package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

func (c *HTTPClient) CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error) {
	queryURL := fmt.Sprintf("%s/v3/connect/custom", c.baseURL)

	payload := map[string]any{
		"provider": "virtual-calendar",
		"settings": map[string]any{
			"email": email,
		},
		"scope": []string{"calendar"},
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result domain.VirtualCalendarGrant
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// ListVirtualCalendarGrants lists all virtual calendar grants.
func (c *HTTPClient) ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error) {
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)
	queryURL := NewQueryBuilder().Add("provider", "virtual-calendar").BuildURL(baseURL)

	var result struct {
		Data []domain.VirtualCalendarGrant `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return result.Data, nil
}

// GetVirtualCalendarGrant retrieves a single virtual calendar grant by ID.
func (c *HTTPClient) GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s", c.baseURL, grantID)

	var result struct {
		Data domain.VirtualCalendarGrant `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: virtual calendar grant not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DeleteVirtualCalendarGrant deletes a virtual calendar grant.
func (c *HTTPClient) DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s", c.baseURL, grantID)
	return c.doDelete(ctx, queryURL)
}

// GetRecurringEventInstances retrieves all instances of a recurring event.
func (c *HTTPClient) GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if params == nil {
		params = &domain.EventQueryParams{
			ExpandRecurring: true,
			Limit:           50,
		}
	} else {
		params.ExpandRecurring = true
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/events", c.baseURL, grantID)
	queryURL := NewQueryBuilder().
		Add("calendar_id", calendarID).
		Add("event_id", masterEventID).
		AddBool("expand_recurring", true).
		AddInt("limit", params.Limit).
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

// UpdateRecurringEventInstance updates a single instance of a recurring event.
func (c *HTTPClient) UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	return c.UpdateEvent(ctx, grantID, calendarID, eventID, req)
}

// DeleteRecurringEventInstance deletes a single instance of a recurring event.
func (c *HTTPClient) DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error {
	return c.DeleteEvent(ctx, grantID, calendarID, eventID)
}

// convertCalendars converts API calendar responses to domain models.
