package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type calendarListParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type calendarListResult struct {
	Calendars []domain.Calendar `json:"calendars"`
}

type eventListParams struct {
	GrantID      string `json:"grant_id,omitempty"`
	CalendarID   string `json:"calendar_id,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	PageToken    string `json:"page_token,omitempty"`
	UpdatedAfter int64  `json:"updated_after,omitempty"`
	Start        int64  `json:"start,omitempty"`
	End          int64  `json:"end,omitempty"`
}

type eventListResult struct {
	Events     []domain.Event `json:"events"`
	NextCursor string         `json:"next_cursor"`
	HasMore    bool           `json:"has_more"`
}

type eventGetParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
}

func RegisterCalendarHandlers(d *Dispatcher, client ports.CalendarClient, defaultGrant string) {
	d.Register("calendar.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		calendars, err := client.GetCalendars(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("calendar.list: %w", err)
		}
		return calendarListResult{Calendars: calendars}, nil
	})

	d.Register("event.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}
		calendarID := p.CalendarID
		if calendarID == "" {
			calendarID = "primary"
		}

		resp, err := client.GetEventsWithCursor(ctx, grantID, calendarID, &domain.EventQueryParams{
			Limit:        p.Limit,
			PageToken:    p.PageToken,
			UpdatedAfter: p.UpdatedAfter,
			Start:        p.Start,
			End:          p.End,
		})
		if err != nil {
			return nil, fmt.Errorf("event.list: %w", err)
		}
		return eventListResult{
			Events:     resp.Data,
			NextCursor: resp.Pagination.NextCursor,
			HasMore:    resp.Pagination.HasMore,
		}, nil
	})

	d.Register("event.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.EventID == "" {
			return nil, NewRPCError(InvalidParams, "event_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}
		calendarID := p.CalendarID
		if calendarID == "" {
			calendarID = "primary"
		}

		event, err := client.GetEvent(ctx, grantID, calendarID, p.EventID)
		if err != nil {
			return nil, fmt.Errorf("event.get: %w", err)
		}
		return event, nil
	})
}
