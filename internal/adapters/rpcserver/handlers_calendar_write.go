package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type eventCreateParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	domain.CreateEventRequest
}

type eventUpdateParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
	domain.UpdateEventRequest
}

type eventDeleteParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
}

type eventRSVPParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
	domain.SendRSVPRequest
}

type eventRSVPResult struct {
	OK bool `json:"ok"`
}

func RegisterCalendarWriteHandlers(d *Dispatcher, client ports.CalendarClient, defaultGrant string) {
	d.Register("event.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		event, err := client.CreateEvent(ctx, grantID, calendarIDOrPrimary(p.CalendarID), &p.CreateEventRequest)
		if err != nil {
			return nil, fmt.Errorf("event.create: %w", err)
		}
		return event, nil
	})

	d.Register("event.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventUpdateParams
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

		event, err := client.UpdateEvent(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.EventID, &p.UpdateEventRequest)
		if err != nil {
			return nil, fmt.Errorf("event.update: %w", err)
		}
		return event, nil
	})

	d.Register("event.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventDeleteParams
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

		if err := client.DeleteEvent(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.EventID); err != nil {
			return nil, fmt.Errorf("event.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("event.rsvp", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventRSVPParams
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

		if err := client.SendRSVP(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.EventID, &p.SendRSVPRequest); err != nil {
			return nil, fmt.Errorf("event.rsvp: %w", err)
		}
		return eventRSVPResult{OK: true}, nil
	})
}

func calendarIDOrPrimary(calendarID string) string {
	if calendarID != "" {
		return calendarID
	}
	return "primary"
}
