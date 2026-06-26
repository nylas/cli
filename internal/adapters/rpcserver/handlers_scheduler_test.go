package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeSchedulerClient struct {
	ports.SchedulerClient

	listSchedulerConfigurations func(context.Context) ([]domain.SchedulerConfiguration, error)
	getSchedulerConfiguration   func(context.Context, string) (*domain.SchedulerConfiguration, error)
	getSchedulerSession         func(context.Context, string) (*domain.SchedulerSession, error)
	getBooking                  func(context.Context, string) (*domain.Booking, error)
	listGroupEvents             func(context.Context, string, string, string, int64, int64) ([]domain.GroupEvent, error)

	configID   string
	sessionID  string
	bookingID  string
	grantID    string
	calendarID string
	startTime  int64
	endTime    int64
}

func (f *fakeSchedulerClient) ListSchedulerConfigurations(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
	if f.listSchedulerConfigurations == nil {
		return nil, errors.New("unexpected ListSchedulerConfigurations")
	}
	return f.listSchedulerConfigurations(ctx)
}

func (f *fakeSchedulerClient) GetSchedulerConfiguration(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
	f.configID = configID
	if f.getSchedulerConfiguration == nil {
		return nil, errors.New("unexpected GetSchedulerConfiguration")
	}
	return f.getSchedulerConfiguration(ctx, configID)
}

func (f *fakeSchedulerClient) GetSchedulerSession(ctx context.Context, sessionID string) (*domain.SchedulerSession, error) {
	f.sessionID = sessionID
	if f.getSchedulerSession == nil {
		return nil, errors.New("unexpected GetSchedulerSession")
	}
	return f.getSchedulerSession(ctx, sessionID)
}

func (f *fakeSchedulerClient) GetBooking(ctx context.Context, bookingID string) (*domain.Booking, error) {
	f.bookingID = bookingID
	if f.getBooking == nil {
		return nil, errors.New("unexpected GetBooking")
	}
	return f.getBooking(ctx, bookingID)
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
			name:   "scheduler.config.list returns configurations",
			method: "scheduler.config.list",
			params: `{}`,
			client: &fakeSchedulerClient{
				listSchedulerConfigurations: func(ctx context.Context) ([]domain.SchedulerConfiguration, error) {
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
			name:   "scheduler.config.get client error maps to internal error",
			method: "scheduler.config.get",
			params: `{"config_id":"config-1"}`,
			client: &fakeSchedulerClient{
				getSchedulerConfiguration: func(ctx context.Context, configID string) (*domain.SchedulerConfiguration, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
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
			params: `{"booking_id":"booking-1"}`,
			client: &fakeSchedulerClient{
				getBooking: func(ctx context.Context, bookingID string) (*domain.Booking, error) {
					return &domain.Booking{BookingID: bookingID, Title: "Intro", Status: "confirmed"}, nil
				},
			},
			assert: func(t *testing.T, client *fakeSchedulerClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
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
