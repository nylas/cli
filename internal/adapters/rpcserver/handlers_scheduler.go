package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type schedulerConfigListResult struct {
	Configurations []domain.SchedulerConfiguration `json:"configurations"`
}

type schedulerConfigListParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type schedulerConfigGetParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ConfigID string `json:"config_id"`
}

type schedulerConfigCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateSchedulerConfigurationRequest
}

type schedulerConfigUpdateParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ConfigID string `json:"config_id"`
	domain.UpdateSchedulerConfigurationRequest
}

type schedulerSessionCreateParams struct {
	domain.CreateSchedulerSessionRequest
}

type schedulerSessionGetParams struct {
	SessionID string `json:"session_id"`
}

// Booking endpoints authenticate with a Scheduler session token minted from the
// configuration, so every booking RPC requires configuration_id.
type schedulerBookingGetParams struct {
	ConfigurationID string `json:"configuration_id"`
	BookingID       string `json:"booking_id"`
}

type schedulerBookingConfirmParams struct {
	ConfigurationID string `json:"configuration_id"`
	BookingID       string `json:"booking_id"`
	domain.ConfirmBookingRequest
}

type schedulerBookingRescheduleParams struct {
	ConfigurationID string `json:"configuration_id"`
	BookingID       string `json:"booking_id"`
	domain.RescheduleBookingRequest
}

type schedulerBookingCancelParams struct {
	ConfigurationID    string `json:"configuration_id"`
	BookingID          string `json:"booking_id"`
	CancellationReason string `json:"cancellation_reason,omitempty"`
	// Reason is the legacy field name; accepted so older clients keep working.
	Reason string `json:"reason,omitempty"`
}

// cancellationReason returns the cancellation reason, preferring the spec field
// over the legacy "reason" alias.
func (p schedulerBookingCancelParams) cancellationReason() string {
	if p.CancellationReason != "" {
		return p.CancellationReason
	}
	return p.Reason
}

type schedulerBookingCancelResult struct {
	Cancelled bool `json:"cancelled"`
}

type schedulerGroupEventListParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	ConfigID   string `json:"config_id"`
	CalendarID string `json:"calendar_id"`
	StartTime  int64  `json:"start_time,omitempty"`
	EndTime    int64  `json:"end_time,omitempty"`
}

type schedulerGroupEventCreateParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ConfigID string `json:"config_id"`
	domain.CreateGroupEventRequest
}

type schedulerGroupEventUpdateParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ConfigID string `json:"config_id"`
	EventID  string `json:"event_id"`
	domain.UpdateGroupEventRequest
}

type schedulerGroupEventDeleteParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ConfigID string `json:"config_id"`
	EventID  string `json:"event_id"`
}

type schedulerGroupEventImportParams struct {
	ConfigID string                        `json:"config_id"`
	Items    []domain.ImportGroupEventItem `json:"items"`
}

type schedulerGroupEventResult struct {
	Events []domain.GroupEvent `json:"events"`
}

func RegisterSchedulerHandlers(d *Dispatcher, client ports.SchedulerClient, defaultGrant string) {
	d.Register("scheduler.config.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerConfigListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		configurations, err := client.ListSchedulerConfigurations(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("scheduler.config.list: %w", err)
		}
		return schedulerConfigListResult{Configurations: configurations}, nil
	})

	d.Register("scheduler.config.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerConfigGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		config, err := client.GetSchedulerConfiguration(ctx, grantID, p.ConfigID)
		if err != nil {
			return nil, fmt.Errorf("scheduler.config.get: %w", err)
		}
		return config, nil
	})

	d.Register("scheduler.config.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerConfigCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		config, err := client.CreateSchedulerConfiguration(ctx, grantID, &p.CreateSchedulerConfigurationRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.config.create: %w", err)
		}
		return config, nil
	})

	d.Register("scheduler.config.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerConfigUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		config, err := client.UpdateSchedulerConfiguration(ctx, grantID, p.ConfigID, &p.UpdateSchedulerConfigurationRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.config.update: %w", err)
		}
		return config, nil
	})

	d.Register("scheduler.config.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerConfigGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteSchedulerConfiguration(ctx, grantID, p.ConfigID); err != nil {
			return nil, fmt.Errorf("scheduler.config.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("scheduler.session.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerSessionCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		session, err := client.CreateSchedulerSession(ctx, &p.CreateSchedulerSessionRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.session.create: %w", err)
		}
		return session, nil
	})

	d.Register("scheduler.session.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerSessionGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.SessionID == "" {
			return nil, NewRPCError(InvalidParams, "session_id required", nil)
		}

		session, err := client.GetSchedulerSession(ctx, p.SessionID)
		if err != nil {
			return nil, fmt.Errorf("scheduler.session.get: %w", err)
		}
		return session, nil
	})

	d.Register("scheduler.booking.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerBookingGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.BookingID == "" {
			return nil, NewRPCError(InvalidParams, "booking_id required", nil)
		}
		if p.ConfigurationID == "" {
			return nil, NewRPCError(InvalidParams, "configuration_id required", nil)
		}

		booking, err := client.GetBooking(ctx, p.ConfigurationID, p.BookingID)
		if err != nil {
			return nil, fmt.Errorf("scheduler.booking.get: %w", err)
		}
		return booking, nil
	})

	d.Register("scheduler.booking.confirm", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerBookingConfirmParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.BookingID == "" {
			return nil, NewRPCError(InvalidParams, "booking_id required", nil)
		}
		if p.ConfigurationID == "" {
			return nil, NewRPCError(InvalidParams, "configuration_id required", nil)
		}
		// The Nylas v3 spec requires salt + status on the confirm payload.
		if err := p.Validate(); err != nil {
			return nil, NewRPCError(InvalidParams, err.Error(), nil)
		}

		booking, err := client.ConfirmBooking(ctx, p.ConfigurationID, p.BookingID, &p.ConfirmBookingRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.booking.confirm: %w", err)
		}
		return booking, nil
	})

	d.Register("scheduler.booking.reschedule", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerBookingRescheduleParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.BookingID == "" {
			return nil, NewRPCError(InvalidParams, "booking_id required", nil)
		}
		if p.ConfigurationID == "" {
			return nil, NewRPCError(InvalidParams, "configuration_id required", nil)
		}
		// Mirror the CLI's guard: reschedule requires valid new start/end times.
		if p.StartTime == 0 || p.EndTime == 0 {
			return nil, NewRPCError(InvalidParams, "start_time and end_time are required", nil)
		}
		if p.EndTime <= p.StartTime {
			return nil, NewRPCError(InvalidParams, "end_time must be after start_time", nil)
		}

		booking, err := client.RescheduleBooking(ctx, p.ConfigurationID, p.BookingID, &p.RescheduleBookingRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.booking.reschedule: %w", err)
		}
		return booking, nil
	})

	d.Register("scheduler.booking.cancel", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerBookingCancelParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.BookingID == "" {
			return nil, NewRPCError(InvalidParams, "booking_id required", nil)
		}
		if p.ConfigurationID == "" {
			return nil, NewRPCError(InvalidParams, "configuration_id required", nil)
		}

		if err := client.CancelBooking(ctx, p.ConfigurationID, p.BookingID, p.cancellationReason()); err != nil {
			return nil, fmt.Errorf("scheduler.booking.cancel: %w", err)
		}
		return schedulerBookingCancelResult{Cancelled: true}, nil
	})

	d.Register("scheduler.groupEvent.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerGroupEventListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		if p.CalendarID == "" {
			return nil, NewRPCError(InvalidParams, "calendar_id required", nil)
		}
		if p.StartTime <= 0 || p.EndTime <= 0 {
			return nil, NewRPCError(InvalidParams, "start_time and end_time required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		events, err := client.ListGroupEvents(ctx, grantID, p.ConfigID, p.CalendarID, p.StartTime, p.EndTime)
		if err != nil {
			return nil, fmt.Errorf("scheduler.groupEvent.list: %w", err)
		}
		return schedulerGroupEventResult{Events: events}, nil
	})

	d.Register("scheduler.groupEvent.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerGroupEventCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		events, err := client.CreateGroupEvent(ctx, grantID, p.ConfigID, &p.CreateGroupEventRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.groupEvent.create: %w", err)
		}
		return schedulerGroupEventResult{Events: events}, nil
	})

	d.Register("scheduler.groupEvent.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerGroupEventUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		if p.EventID == "" {
			return nil, NewRPCError(InvalidParams, "event_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		events, err := client.UpdateGroupEvent(ctx, grantID, p.ConfigID, p.EventID, &p.UpdateGroupEventRequest)
		if err != nil {
			return nil, fmt.Errorf("scheduler.groupEvent.update: %w", err)
		}
		return schedulerGroupEventResult{Events: events}, nil
	})

	d.Register("scheduler.groupEvent.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerGroupEventDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}
		if p.EventID == "" {
			return nil, NewRPCError(InvalidParams, "event_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteGroupEvent(ctx, grantID, p.ConfigID, p.EventID); err != nil {
			return nil, fmt.Errorf("scheduler.groupEvent.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("scheduler.groupEvent.import", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p schedulerGroupEventImportParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ConfigID == "" {
			return nil, NewRPCError(InvalidParams, "config_id required", nil)
		}

		events, err := client.ImportGroupEvents(ctx, p.ConfigID, p.Items)
		if err != nil {
			return nil, fmt.Errorf("scheduler.groupEvent.import: %w", err)
		}
		return schedulerGroupEventResult{Events: events}, nil
	})
}
