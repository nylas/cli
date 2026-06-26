package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeCalendarClient struct {
	ports.CalendarClient

	getCalendars        func(context.Context, string) ([]domain.Calendar, error)
	getEventsWithCursor func(context.Context, string, string, *domain.EventQueryParams) (*domain.EventListResponse, error)
	getEvent            func(context.Context, string, string, string) (*domain.Event, error)
}

func (f *fakeCalendarClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	if f.getCalendars == nil {
		return nil, errors.New("unexpected GetCalendars")
	}
	return f.getCalendars(ctx, grantID)
}

func (f *fakeCalendarClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	if f.getEventsWithCursor == nil {
		return nil, errors.New("unexpected GetEventsWithCursor")
	}
	return f.getEventsWithCursor(ctx, grantID, calendarID, params)
}

func (f *fakeCalendarClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	if f.getEvent == nil {
		return nil, errors.New("unexpected GetEvent")
	}
	return f.getEvent(ctx, grantID, calendarID, eventID)
}

func TestRegisterCalendarHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeCalendarClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:         "calendar.list returns calendars",
			method:       "calendar.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarClient{
				getCalendars: func(ctx context.Context, grantID string) ([]domain.Calendar, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					return []domain.Calendar{{ID: "cal-1", Name: "Work"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Calendars []domain.Calendar `json:"calendars"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Calendars) != 1 || result.Calendars[0].ID != "cal-1" {
					t.Fatalf("calendars = %#v, want cal-1", result.Calendars)
				}
			},
		},
		{
			name:         "event.list defaults calendar and forwards query params",
			method:       "event.list",
			params:       `{"grant_id":"request-grant","limit":25,"page_token":"cursor-1","updated_after":1710000000,"start":1710000100,"end":1710000200}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarClient{
				getEventsWithCursor: func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
					if grantID != "request-grant" {
						t.Fatalf("grantID = %q, want request-grant", grantID)
					}
					if calendarID != "primary" {
						t.Fatalf("calendarID = %q, want primary", calendarID)
					}
					if params.Limit != 25 || params.PageToken != "cursor-1" || params.UpdatedAfter != 1710000000 || params.Start != 1710000100 || params.End != 1710000200 {
						t.Fatalf("params = %+v, want forwarded query params", params)
					}
					return &domain.EventListResponse{
						Data:       []domain.Event{{ID: "event-1", Title: "Sync"}},
						Pagination: domain.Pagination{NextCursor: "cursor-2", HasMore: true},
					}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Events     []domain.Event `json:"events"`
					NextCursor string         `json:"next_cursor"`
					HasMore    bool           `json:"has_more"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Events) != 1 || result.Events[0].ID != "event-1" || result.NextCursor != "cursor-2" || !result.HasMore {
					t.Fatalf("result = %+v, want event-1 cursor-2 has_more", result)
				}
			},
		},
		{
			name:         "event.list uses request calendar",
			method:       "event.list",
			params:       `{"calendar_id":"cal-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarClient{
				getEventsWithCursor: func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
					if calendarID != "cal-1" {
						t.Fatalf("calendarID = %q, want cal-1", calendarID)
					}
					return &domain.EventListResponse{}, nil
				},
			},
			assert: requireNoRPCError,
		},
		{
			name:   "event.get returns event",
			method: "event.get",
			params: `{"grant_id":"grant-1","calendar_id":"cal-1","event_id":"event-1"}`,
			client: &fakeCalendarClient{
				getEvent: func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
					if grantID != "grant-1" || calendarID != "cal-1" || eventID != "event-1" {
						t.Fatalf("args = %q %q %q, want grant-1 cal-1 event-1", grantID, calendarID, eventID)
					}
					return &domain.Event{ID: "event-1", Title: "Sync"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var event domain.Event
				unmarshalResult(t, resp, &event)
				if event.ID != "event-1" || event.Title != "Sync" {
					t.Fatalf("event = %#v, want event-1 Sync", event)
				}
			},
		},
		{
			name:   "calendar.list missing grant returns invalid params",
			method: "calendar.list",
			params: `{}`,
			client: &fakeCalendarClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "event.get missing event_id returns invalid params",
			method:       "event.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "malformed params returns invalid params",
			method:       "event.list",
			params:       `"bad"`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "event.get",
			params:       `{"event_id":"event-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarClient{
				getEvent: func(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
					if calendarID != "primary" {
						t.Fatalf("calendarID = %q, want primary", calendarID)
					}
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterCalendarHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchCalendarRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func dispatchCalendarRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}
