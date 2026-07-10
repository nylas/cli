package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeSchedulerClient struct {
	ports.SchedulerClient

	listSchedulerConfigurations func(context.Context, string) ([]domain.SchedulerConfiguration, error)
	getSchedulerConfiguration   func(context.Context, string, string) (*domain.SchedulerConfiguration, error)
	getSchedulerSession         func(context.Context, string) (*domain.SchedulerSession, error)
	getBooking                  func(context.Context, string, string) (*domain.Booking, error)
	listGroupEvents             func(context.Context, string, string, string, int64, int64) ([]domain.GroupEvent, error)

	configID        string
	configurationID string
	sessionID       string
	bookingID       string
	grantID         string
	calendarID      string
	startTime       int64
	endTime         int64
	cancelReason    string
	confirmReq      *domain.ConfirmBookingRequest
	sessionReq      *domain.CreateSchedulerSessionRequest

	rescheduleBooking func(context.Context, string, string, *domain.RescheduleBookingRequest) (*domain.Booking, error)
}

func (f *fakeSchedulerClient) RescheduleBooking(ctx context.Context, configurationID, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
	f.configurationID = configurationID
	f.bookingID = bookingID
	if f.rescheduleBooking == nil {
		return nil, errors.New("unexpected RescheduleBooking")
	}
	return f.rescheduleBooking(ctx, configurationID, bookingID, req)
}

func (f *fakeSchedulerClient) CancelBooking(ctx context.Context, configurationID, bookingID, reason string) error {
	f.configurationID = configurationID
	f.bookingID = bookingID
	f.cancelReason = reason
	return nil
}

func (f *fakeSchedulerClient) ConfirmBooking(ctx context.Context, configurationID, bookingID string, req *domain.ConfirmBookingRequest) (*domain.Booking, error) {
	f.configurationID = configurationID
	f.bookingID = bookingID
	f.confirmReq = req
	return &domain.Booking{BookingID: bookingID}, nil
}

func (f *fakeSchedulerClient) CreateSchedulerSession(ctx context.Context, req *domain.CreateSchedulerSessionRequest) (*domain.SchedulerSession, error) {
	f.sessionReq = req
	return &domain.SchedulerSession{SessionID: "session-1"}, nil
}

func (f *fakeSchedulerClient) ListSchedulerConfigurations(ctx context.Context, grantID string) ([]domain.SchedulerConfiguration, error) {
	f.grantID = grantID
	if f.listSchedulerConfigurations == nil {
		return nil, errors.New("unexpected ListSchedulerConfigurations")
	}
	return f.listSchedulerConfigurations(ctx, grantID)
}

func (f *fakeSchedulerClient) GetSchedulerConfiguration(ctx context.Context, grantID, configID string) (*domain.SchedulerConfiguration, error) {
	f.grantID = grantID
	f.configID = configID
	if f.getSchedulerConfiguration == nil {
		return nil, errors.New("unexpected GetSchedulerConfiguration")
	}
	return f.getSchedulerConfiguration(ctx, grantID, configID)
}

func (f *fakeSchedulerClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	f.sessionID = sessionID
	if f.getSchedulerSession == nil {
		return nil, errors.New("unexpected GetSchedulerSession")
	}
	return f.getSchedulerSession(ctx, sessionID)
}

func (f *fakeSchedulerClient) GetBooking(ctx context.Context, configurationID, bookingID string) (*domain.Booking, error) {
	f.configurationID = configurationID
	f.bookingID = bookingID
	if f.getBooking == nil {
		return nil, errors.New("unexpected GetBooking")
	}
	return f.getBooking(ctx, configurationID, bookingID)
}

func (f *fakeSchedulerClient) ListGroupEvents(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error) {
	f.grantID = grantID
	f.configID = configID
	f.calendarID = calendarID
	f.startTime = startTime
	f.endTime = endTime
	if f.listGroupEvents == nil {
		return nil, errors.New("unexpected ListGroupEvents")
	}
	return f.listGroupEvents(ctx, grantID, configID, calendarID, startTime, endTime)
}

func TestRegisterSchedulerHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeSchedulerClient
		assert       func(*testing.T, *fakeSchedulerClient, rpcTestResponse)
	}{
		{
			name:   "scheduler.config.list forwards grant and returns configurations",
			method: "scheduler.config.list",
			params: `{"grant_id":"grant-1"}`,
			client: &fakeSchedulerClient{
				listSchedulerConfigurations: func(ctx context.Context, grantID string) ([]domain.SchedulerConfiguration, error) {
					return []domain.SchedulerConfiguration{{ID: "config-1", Name: "Intro"}}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result schedulerConfigListResult
				unmarshalResult(t, resp, &result)
				if len(result.Configurations) != 1 || result.Configurations[0].ID != "config-1" {
					t.Fatalf("configurations = %+v, want config-1", result.Configurations)
				}
				if client.grantID != "grant-1" {
					t.Fatalf("grantID = %q, want grant-1", client.grantID)
				}
			},
		},
		{
			name:         "scheduler.config.list uses default grant when omitted",
			method:       "scheduler.config.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeSchedulerClient{
				listSchedulerConfigurations: func(ctx context.Context, grantID string) ([]domain.SchedulerConfiguration, error) {
					return nil, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "default-grant" {
					t.Fatalf("grantID = %q, want default-grant", client.grantID)
				}
			},
		},
		{
			name:   "scheduler.config.list requires a grant",
			method: "scheduler.config.list",
			params: `{}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.grantID != "" {
					t.Fatalf("ListSchedulerConfigurations called with grant %q, want no call", client.grantID)
				}
			},
		},
		{
			name:   "scheduler.config.get requires config_id",
			method: "scheduler.config.get",
			params: `{}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.configID != "" {
					t.Fatalf("GetSchedulerConfiguration called with config %q, want no call", client.configID)
				}
			},
		},
		{
			name:   "scheduler.config.get forwards grant and maps client error",
			method: "scheduler.config.get",
			params: `{"grant_id":"grant-1","config_id":"config-1"}`,
			client: &fakeSchedulerClient{
				getSchedulerConfiguration: func(ctx context.Context, grantID, configID string) (*domain.SchedulerConfiguration, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.grantID != "grant-1" {
					t.Fatalf("grantID = %q, want grant-1", client.grantID)
				}
				if client.configID != "config-1" {
					t.Fatalf("configID = %q, want config-1", client.configID)
				}
			},
		},
		{
			name:   "scheduler.session.get returns session",
			method: "scheduler.session.get",
			params: `{"session_id":"session-1"}`,
			client: &fakeSchedulerClient{
				getSchedulerSession: func(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
					return &domain.SchedulerSession{SessionID: sessionID, ConfigurationID: "config-1"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sessionID != "session-1" {
					t.Fatalf("sessionID = %q, want session-1", client.sessionID)
				}

				var session domain.SchedulerSession
				unmarshalResult(t, resp, &session)
				if session.SessionID != "session-1" || session.ConfigurationID != "config-1" {
					t.Fatalf("session = %+v, want session-1 config-1", session)
				}
			},
		},
		{
			name:   "scheduler.session.get requires session_id",
			method: "scheduler.session.get",
			params: `{}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.sessionID != "" {
					t.Fatalf("GetSchedulerSession called with session %q, want no call", client.sessionID)
				}
			},
		},
		{
			name:   "scheduler.booking.get returns booking",
			method: "scheduler.booking.get",
			params: `{"configuration_id":"config-1","booking_id":"booking-1"}`,
			client: &fakeSchedulerClient{
				getBooking: func(ctx context.Context, configurationID, bookingID string) (*domain.Booking, error) {
					return &domain.Booking{BookingID: bookingID, Title: "Intro", Status: "confirmed"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.configurationID != "config-1" {
					t.Fatalf("configurationID = %q, want config-1", client.configurationID)
				}
				if client.bookingID != "booking-1" {
					t.Fatalf("bookingID = %q, want booking-1", client.bookingID)
				}

				var booking domain.Booking
				unmarshalResult(t, resp, &booking)
				if booking.BookingID != "booking-1" || booking.Title != "Intro" {
					t.Fatalf("booking = %+v, want booking-1 Intro", booking)
				}
			},
		},
		{
			name:   "scheduler.booking.get requires booking_id",
			method: "scheduler.booking.get",
			params: `{}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.bookingID != "" {
					t.Fatalf("GetBooking called with booking %q, want no call", client.bookingID)
				}
			},
		},
		{
			// Booking endpoints authenticate with a session token minted from the
			// configuration, so configuration_id is required before any API call.
			name:   "scheduler.booking.get requires configuration_id",
			method: "scheduler.booking.get",
			params: `{"booking_id":"booking-1"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.configurationID != "" {
					t.Fatalf("GetBooking called with configuration %q, want no call", client.configurationID)
				}
			},
		},
		{
			name:   "scheduler.booking.confirm requires salt",
			method: "scheduler.booking.confirm",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","status":"confirmed"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.bookingID != "" {
					t.Fatalf("ConfirmBooking called with booking %q, want no call", client.bookingID)
				}
			},
		},
		{
			name:   "scheduler.booking.confirm requires valid status",
			method: "scheduler.booking.confirm",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","salt":"s4lt","status":"bogus"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.bookingID != "" {
					t.Fatalf("ConfirmBooking called with booking %q, want no call", client.bookingID)
				}
			},
		},
		{
			// Declining via confirm carries the reason too; the legacy "reason"
			// alias must keep working just like scheduler.booking.cancel's.
			name:   "scheduler.booking.confirm accepts legacy reason field",
			method: "scheduler.booking.confirm",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","salt":"s4lt","status":"cancelled","reason":"organizer conflict"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.confirmReq == nil || client.confirmReq.CancellationReason != "organizer conflict" {
					t.Fatalf("confirmReq = %+v, want legacy reason forwarded as cancellation_reason", client.confirmReq)
				}
			},
		},
		{
			name:   "scheduler.booking.confirm prefers cancellation_reason over legacy reason",
			method: "scheduler.booking.confirm",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","salt":"s4lt","status":"cancelled","cancellation_reason":"spec","reason":"legacy"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.confirmReq == nil || client.confirmReq.CancellationReason != "spec" {
					t.Fatalf("confirmReq = %+v, want cancellation_reason to win", client.confirmReq)
				}
			},
		},
		{
			// The session TTL param was renamed ttl -> time_to_live to match the
			// Nylas v3 spec; older RPC clients still sending "ttl" must not have
			// their TTL silently dropped.
			name:   "scheduler.session.create accepts legacy ttl field",
			method: "scheduler.session.create",
			params: `{"configuration_id":"config-1","ttl":25}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sessionReq == nil || client.sessionReq.TimeToLive != 25 {
					t.Fatalf("sessionReq = %+v, want legacy ttl forwarded as time_to_live", client.sessionReq)
				}
			},
		},
		{
			// Trust boundary: malformed session params must be rejected before
			// they reach the Nylas API (spec: configuration_id OR slug required).
			name:   "scheduler.session.create requires configuration_id or slug",
			method: "scheduler.session.create",
			params: `{"time_to_live":10}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.sessionReq != nil {
					t.Fatalf("CreateSchedulerSession called with %+v, want no call", client.sessionReq)
				}
			},
		},
		{
			name:   "scheduler.session.create accepts slug without configuration_id",
			method: "scheduler.session.create",
			params: `{"slug":"my-page"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sessionReq == nil || client.sessionReq.Slug != "my-page" {
					t.Fatalf("sessionReq = %+v, want slug forwarded", client.sessionReq)
				}
			},
		},
		{
			// The spec caps time_to_live at 30 minutes.
			name:   "scheduler.session.create rejects ttl above the 30-minute cap",
			method: "scheduler.session.create",
			params: `{"configuration_id":"config-1","time_to_live":31}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.sessionReq != nil {
					t.Fatalf("CreateSchedulerSession called with %+v, want no call", client.sessionReq)
				}
			},
		},
		{
			name:   "scheduler.session.create prefers time_to_live over legacy ttl",
			method: "scheduler.session.create",
			params: `{"configuration_id":"config-1","time_to_live":10,"ttl":25}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sessionReq == nil || client.sessionReq.TimeToLive != 10 {
					t.Fatalf("sessionReq = %+v, want time_to_live to win", client.sessionReq)
				}
			},
		},
		{
			name:   "scheduler.booking.cancel accepts legacy reason field",
			method: "scheduler.booking.cancel",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","reason":"customer no-show"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.cancelReason != "customer no-show" {
					t.Fatalf("cancelReason = %q, want the legacy reason field to be forwarded", client.cancelReason)
				}
			},
		},
		{
			name:   "scheduler.booking.cancel prefers cancellation_reason over legacy reason",
			method: "scheduler.booking.cancel",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","cancellation_reason":"spec","reason":"legacy"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.cancelReason != "spec" {
					t.Fatalf("cancelReason = %q, want cancellation_reason to win", client.cancelReason)
				}
			},
		},
		{
			// A stray legacy "reason" on a status:"confirmed" payload was ignored
			// before the alias existed; it must not start flowing to the API as a
			// cancellation_reason on a confirmation.
			name:   "scheduler.booking.confirm ignores legacy reason unless cancelling",
			method: "scheduler.booking.confirm",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","salt":"s4lt","status":"confirmed","reason":"stray note"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.confirmReq == nil || client.confirmReq.CancellationReason != "" {
					t.Fatalf("confirmReq = %+v, want stray legacy reason dropped on confirm", client.confirmReq)
				}
			},
		},
		{
			// The reschedule PATCH applied but the read-back could not verify the
			// record: a typed partial success, not a failure — the client must get
			// the port's partial booking plus an explicit warning, so it stays
			// distinguishable from a verified success.
			name:   "scheduler.booking.reschedule returns booking and warning when read-back fails",
			method: "scheduler.booking.reschedule",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","start_time":1704067200,"end_time":1704070800}`,
			client: &fakeSchedulerClient{
				rescheduleBooking: func(ctx context.Context, configurationID, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
					return &domain.Booking{BookingID: bookingID, StartTime: time.Unix(req.StartTime, 0), EndTime: time.Unix(req.EndTime, 0)},
						fmt.Errorf("%w: transient failure", domain.ErrBookingReadBackFailed)
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result schedulerBookingRescheduleResult
				unmarshalResult(t, resp, &result)
				if result.BookingID != "booking-1" {
					t.Fatalf("booking = %+v, want booking-1 returned as partial success", result.Booking)
				}
				if result.StartTime.Unix() != 1704067200 || result.EndTime.Unix() != 1704070800 {
					t.Fatalf("times = %v %v, want the applied request times", result.StartTime, result.EndTime)
				}
				if result.Warning == "" {
					t.Fatal("Warning is empty, want the read-back failure surfaced")
				}
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(resp.Result, &raw); err != nil {
					t.Fatalf("unmarshal raw result: %v", err)
				}
				if _, ok := raw["warning"]; !ok {
					t.Fatal("warning key absent from JSON, want it present on partial success")
				}
			},
		},
		{
			// A fully verified reschedule must NOT carry a warning.
			name:   "scheduler.booking.reschedule omits warning on verified success",
			method: "scheduler.booking.reschedule",
			params: `{"configuration_id":"config-1","booking_id":"booking-1","start_time":1704067200,"end_time":1704070800}`,
			client: &fakeSchedulerClient{
				rescheduleBooking: func(ctx context.Context, configurationID, bookingID string, req *domain.RescheduleBookingRequest) (*domain.Booking, error) {
					return &domain.Booking{BookingID: bookingID, Status: "confirmed"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result schedulerBookingRescheduleResult
				unmarshalResult(t, resp, &result)
				if result.BookingID != "booking-1" || result.Status != "confirmed" {
					t.Fatalf("booking = %+v, want the read-back record", result.Booking)
				}
				// Assert on the raw JSON: omitempty must drop the key entirely, an
				// empty-string decode cannot distinguish absent from "".
				var raw map[string]json.RawMessage
				if err := json.Unmarshal(resp.Result, &raw); err != nil {
					t.Fatalf("unmarshal raw result: %v", err)
				}
				if _, ok := raw["warning"]; ok {
					t.Fatalf("warning key present in JSON (%s), want it omitted on verified success", raw["warning"])
				}
			},
		},
		{
			// Mirror the CLI guard: reschedule must reject zero/invalid times
			// before minting a session or calling the API.
			name:   "scheduler.booking.reschedule requires valid times",
			method: "scheduler.booking.reschedule",
			params: `{"configuration_id":"config-1","booking_id":"booking-1"}`,
			client: &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "scheduler.groupEvent.list forwards grant and returns events",
			method: "scheduler.groupEvent.list",
			params: `{"grant_id":"grant-1","config_id":"config-1","calendar_id":"cal-1","start_time":1710000000,"end_time":1710003600}`,
			client: &fakeSchedulerClient{
				listGroupEvents: func(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error) {
					return []domain.GroupEvent{{ID: "event-1", CalendarID: calendarID, Title: "Workshop"}}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.grantID != "grant-1" || client.configID != "config-1" || client.calendarID != "cal-1" {
					t.Fatalf("args = %q %q %q, want grant-1 config-1 cal-1", client.grantID, client.configID, client.calendarID)
				}
				if client.startTime != 1710000000 || client.endTime != 1710003600 {
					t.Fatalf("times = %d %d, want forwarded window", client.startTime, client.endTime)
				}

				var result schedulerGroupEventResult
				unmarshalResult(t, resp, &result)
				if len(result.Events) != 1 || result.Events[0].ID != "event-1" {
					t.Fatalf("events = %+v, want event-1", result.Events)
				}
			},
		},
		{
			name:         "scheduler.groupEvent.list requires calendar_id",
			method:       "scheduler.groupEvent.list",
			params:       `{"config_id":"config-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.grantID != "" {
					t.Fatalf("ListGroupEvents called with grant %q, want no call", client.grantID)
				}
			},
		},
		{
			name:         "scheduler.groupEvent.list requires time window",
			method:       "scheduler.groupEvent.list",
			params:       `{"config_id":"config-1","calendar_id":"cal-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeSchedulerClient{},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.grantID != "" {
					t.Fatalf("ListGroupEvents called with grant %q, want no call", client.grantID)
				}
			},
		},
		{
			name:   "scheduler.groupEvent.list client error maps to internal error",
			method: "scheduler.groupEvent.list",
			params: `{"grant_id":"grant-1","config_id":"config-1","calendar_id":"cal-1","start_time":1710000000,"end_time":1710003600}`,
			client: &fakeSchedulerClient{
				listGroupEvents: func(ctx context.Context, grantID, configID, calendarID string, startTime, endTime int64) ([]domain.GroupEvent, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.grantID != "grant-1" {
					t.Fatalf("grantID = %q, want grant-1", client.grantID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterSchedulerHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchCalendarRequest(t, d, tt.method, tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}
