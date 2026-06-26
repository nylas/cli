package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type calendarGetParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id"`
}

type calendarCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateCalendarRequest
}

type calendarUpdateParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id"`
	domain.UpdateCalendarRequest
}

type calendarDeleteParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id"`
}

type calendarFreeBusyParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.FreeBusyRequest
}

type calendarAvailabilityParams struct {
	domain.AvailabilityRequest
}

type calendarResourcesParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type calendarResourcesResult struct {
	Resources []domain.RoomResource `json:"resources"`
}

type eventImportParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.EventQueryParams
}

type eventListPayloadResult struct {
	Events []domain.Event `json:"events"`
}

type eventRecurringListParams struct {
	GrantID       string `json:"grant_id,omitempty"`
	CalendarID    string `json:"calendar_id,omitempty"`
	MasterEventID string `json:"master_event_id"`
	domain.EventQueryParams
}

type eventRecurringUpdateParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
	domain.UpdateEventRequest
}

type eventRecurringDeleteParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	CalendarID string `json:"calendar_id,omitempty"`
	EventID    string `json:"event_id"`
}

type virtualCalendarCreateParams struct {
	Email string `json:"email"`
}

type virtualCalendarListResult struct {
	Grants []domain.VirtualCalendarGrant `json:"grants"`
}

type virtualCalendarIDParams struct {
	GrantID string `json:"grant_id"`
}

// RegisterCalendarExtHandlers registers calendar CRUD, availability/free-busy,
// recurring-instance, room resource, import, and virtual-calendar-grant methods.
func RegisterCalendarExtHandlers(d *Dispatcher, client ports.CalendarClient, defaultGrant string) {
	d.Register("calendar.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CalendarID == "" {
			return nil, NewRPCError(InvalidParams, "calendar_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		calendar, err := client.GetCalendar(ctx, grantID, p.CalendarID)
		if err != nil {
			return nil, fmt.Errorf("calendar.get: %w", err)
		}
		return calendar, nil
	})

	d.Register("calendar.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		calendar, err := client.CreateCalendar(ctx, grantID, &p.CreateCalendarRequest)
		if err != nil {
			return nil, fmt.Errorf("calendar.create: %w", err)
		}
		return calendar, nil
	})

	d.Register("calendar.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CalendarID == "" {
			return nil, NewRPCError(InvalidParams, "calendar_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		calendar, err := client.UpdateCalendar(ctx, grantID, p.CalendarID, &p.UpdateCalendarRequest)
		if err != nil {
			return nil, fmt.Errorf("calendar.update: %w", err)
		}
		return calendar, nil
	})

	d.Register("calendar.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CalendarID == "" {
			return nil, NewRPCError(InvalidParams, "calendar_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteCalendar(ctx, grantID, p.CalendarID); err != nil {
			return nil, fmt.Errorf("calendar.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("calendar.freeBusy", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarFreeBusyParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		result, err := client.GetFreeBusy(ctx, grantID, &p.FreeBusyRequest)
		if err != nil {
			return nil, fmt.Errorf("calendar.freeBusy: %w", err)
		}
		return result, nil
	})

	d.Register("calendar.availability", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarAvailabilityParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		result, err := client.GetAvailability(ctx, &p.AvailabilityRequest)
		if err != nil {
			return nil, fmt.Errorf("calendar.availability: %w", err)
		}
		return result, nil
	})

	d.Register("calendar.resources", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p calendarResourcesParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		resources, err := client.ListRoomResources(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("calendar.resources: %w", err)
		}
		return calendarResourcesResult{Resources: resources}, nil
	})

	d.Register("event.import", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventImportParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.CalendarID == "" {
			return nil, NewRPCError(InvalidParams, "calendar_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		events, err := client.ImportEvents(ctx, grantID, &p.EventQueryParams)
		if err != nil {
			return nil, fmt.Errorf("event.import: %w", err)
		}
		return eventListPayloadResult{Events: events}, nil
	})

	d.Register("event.recurring.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventRecurringListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.MasterEventID == "" {
			return nil, NewRPCError(InvalidParams, "master_event_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		events, err := client.GetRecurringEventInstances(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.MasterEventID, &p.EventQueryParams)
		if err != nil {
			return nil, fmt.Errorf("event.recurring.list: %w", err)
		}
		return eventListPayloadResult{Events: events}, nil
	})

	d.Register("event.recurring.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventRecurringUpdateParams
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

		event, err := client.UpdateRecurringEventInstance(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.EventID, &p.UpdateEventRequest)
		if err != nil {
			return nil, fmt.Errorf("event.recurring.update: %w", err)
		}
		return event, nil
	})

	d.Register("event.recurring.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p eventRecurringDeleteParams
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

		if err := client.DeleteRecurringEventInstance(ctx, grantID, calendarIDOrPrimary(p.CalendarID), p.EventID); err != nil {
			return nil, fmt.Errorf("event.recurring.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("calendar.virtual.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p virtualCalendarCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.Email == "" {
			return nil, NewRPCError(InvalidParams, "email required", nil)
		}

		grant, err := client.CreateVirtualCalendarGrant(ctx, p.Email)
		if err != nil {
			return nil, fmt.Errorf("calendar.virtual.create: %w", err)
		}
		return grant, nil
	})

	d.Register("calendar.virtual.list", func(ctx context.Context, _ json.RawMessage) (any, error) {
		grants, err := client.ListVirtualCalendarGrants(ctx)
		if err != nil {
			return nil, fmt.Errorf("calendar.virtual.list: %w", err)
		}
		return virtualCalendarListResult{Grants: grants}, nil
	})

	d.Register("calendar.virtual.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p virtualCalendarIDParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GrantID == "" {
			return nil, NewRPCError(InvalidParams, "grant_id required", nil)
		}

		grant, err := client.GetVirtualCalendarGrant(ctx, p.GrantID)
		if err != nil {
			return nil, fmt.Errorf("calendar.virtual.get: %w", err)
		}
		return grant, nil
	})

	d.Register("calendar.virtual.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p virtualCalendarIDParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GrantID == "" {
			return nil, NewRPCError(InvalidParams, "grant_id required", nil)
		}

		if err := client.DeleteVirtualCalendarGrant(ctx, p.GrantID); err != nil {
			return nil, fmt.Errorf("calendar.virtual.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}
