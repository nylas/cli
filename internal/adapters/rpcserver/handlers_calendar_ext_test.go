package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeCalendarExtClient struct {
	ports.CalendarClient

	getCalendar       func(context.Context, string, string) (*domain.Calendar, error)
	createCalendar    func(context.Context, string, *domain.CreateCalendarRequest) (*domain.Calendar, error)
	updateCalendar    func(context.Context, string, string, *domain.UpdateCalendarRequest) (*domain.Calendar, error)
	deleteCalendar    func(context.Context, string, string) error
	getFreeBusy       func(context.Context, string, *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error)
	getAvailability   func(context.Context, *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error)
	listRoomResources func(context.Context, string) ([]domain.RoomResource, error)
	importEvents      func(context.Context, string, *domain.EventQueryParams) ([]domain.Event, error)
	getRecurring      func(context.Context, string, string, string, *domain.EventQueryParams) ([]domain.Event, error)
	updateRecurring   func(context.Context, string, string, string, *domain.UpdateEventRequest) (*domain.Event, error)
	deleteRecurring   func(context.Context, string, string, string) error
	createVirtual     func(context.Context, string) (*domain.VirtualCalendarGrant, error)
	listVirtual       func(context.Context) ([]domain.VirtualCalendarGrant, error)
	getVirtual        func(context.Context, string) (*domain.VirtualCalendarGrant, error)
	deleteVirtual     func(context.Context, string) error
}

func (f *fakeCalendarExtClient) GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error) {
	if f.getCalendar == nil {
		return nil, errors.New("unexpected GetCalendar")
	}
	return f.getCalendar(ctx, grantID, calendarID)
}

func (f *fakeCalendarExtClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	if f.createCalendar == nil {
		return nil, errors.New("unexpected CreateCalendar")
	}
	return f.createCalendar(ctx, grantID, req)
}

func (f *fakeCalendarExtClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	if f.updateCalendar == nil {
		return nil, errors.New("unexpected UpdateCalendar")
	}
	return f.updateCalendar(ctx, grantID, calendarID, req)
}

func (f *fakeCalendarExtClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	if f.deleteCalendar == nil {
		return errors.New("unexpected DeleteCalendar")
	}
	return f.deleteCalendar(ctx, grantID, calendarID)
}

func (f *fakeCalendarExtClient) GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	if f.getFreeBusy == nil {
		return nil, errors.New("unexpected GetFreeBusy")
	}
	return f.getFreeBusy(ctx, grantID, req)
}

func (f *fakeCalendarExtClient) GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	if f.getAvailability == nil {
		return nil, errors.New("unexpected GetAvailability")
	}
	return f.getAvailability(ctx, req)
}

func (f *fakeCalendarExtClient) ListRoomResources(ctx context.Context, grantID string) ([]domain.RoomResource, error) {
	if f.listRoomResources == nil {
		return nil, errors.New("unexpected ListRoomResources")
	}
	return f.listRoomResources(ctx, grantID)
}

func (f *fakeCalendarExtClient) ImportEvents(ctx context.Context, grantID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if f.importEvents == nil {
		return nil, errors.New("unexpected ImportEvents")
	}
	return f.importEvents(ctx, grantID, params)
}

func (f *fakeCalendarExtClient) GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if f.getRecurring == nil {
		return nil, errors.New("unexpected GetRecurringEventInstances")
	}
	return f.getRecurring(ctx, grantID, calendarID, masterEventID, params)
}

func (f *fakeCalendarExtClient) UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	if f.updateRecurring == nil {
		return nil, errors.New("unexpected UpdateRecurringEventInstance")
	}
	return f.updateRecurring(ctx, grantID, calendarID, eventID, req)
}

func (f *fakeCalendarExtClient) DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error {
	if f.deleteRecurring == nil {
		return errors.New("unexpected DeleteRecurringEventInstance")
	}
	return f.deleteRecurring(ctx, grantID, calendarID, eventID)
}

func (f *fakeCalendarExtClient) CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error) {
	if f.createVirtual == nil {
		return nil, errors.New("unexpected CreateVirtualCalendarGrant")
	}
	return f.createVirtual(ctx, email)
}

func (f *fakeCalendarExtClient) ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error) {
	if f.listVirtual == nil {
		return nil, errors.New("unexpected ListVirtualCalendarGrants")
	}
	return f.listVirtual(ctx)
}

func (f *fakeCalendarExtClient) GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error) {
	if f.getVirtual == nil {
		return nil, errors.New("unexpected GetVirtualCalendarGrant")
	}
	return f.getVirtual(ctx, grantID)
}

func (f *fakeCalendarExtClient) DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error {
	if f.deleteVirtual == nil {
		return errors.New("unexpected DeleteVirtualCalendarGrant")
	}
	return f.deleteVirtual(ctx, grantID)
}

func TestRegisterCalendarExtHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeCalendarExtClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:         "calendar.get returns calendar",
			method:       "calendar.get",
			params:       `{"calendar_id":"cal-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				getCalendar: func(_ context.Context, grantID, calendarID string) (*domain.Calendar, error) {
					if grantID != "default-grant" || calendarID != "cal-1" {
						t.Fatalf("args = %q/%q, want default-grant/cal-1", grantID, calendarID)
					}
					return &domain.Calendar{ID: "cal-1"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var cal domain.Calendar
				unmarshalResult(t, resp, &cal)
				if cal.ID != "cal-1" {
					t.Fatalf("calendar ID = %q, want cal-1", cal.ID)
				}
			},
		},
		{
			name:         "calendar.get missing calendar_id",
			method:       "calendar.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "calendar.create returns calendar",
			method:       "calendar.create",
			params:       `{"name":"Team"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				createCalendar: func(_ context.Context, _ string, _ *domain.CreateCalendarRequest) (*domain.Calendar, error) {
					return &domain.Calendar{ID: "cal-new"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var cal domain.Calendar
				unmarshalResult(t, resp, &cal)
				if cal.ID != "cal-new" {
					t.Fatalf("calendar ID = %q, want cal-new", cal.ID)
				}
			},
		},
		{
			name:         "calendar.update missing calendar_id",
			method:       "calendar.update",
			params:       `{"name":"x"}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "calendar.delete returns deleted",
			method:       "calendar.delete",
			params:       `{"calendar_id":"cal-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				deleteCalendar: func(_ context.Context, _, calendarID string) error {
					if calendarID != "cal-1" {
						t.Fatalf("calendarID = %q, want cal-1", calendarID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "calendar.freeBusy returns response",
			method:       "calendar.freeBusy",
			params:       `{"start_time":100,"end_time":200,"emails":["a@example.com"]}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				getFreeBusy: func(_ context.Context, grantID string, _ *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					return &domain.FreeBusyResponse{}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) { requireNoRPCError(t, resp) },
		},
		{
			name:   "calendar.availability returns response without grant",
			method: "calendar.availability",
			params: `{"start_time":100,"end_time":200}`,
			client: &fakeCalendarExtClient{
				getAvailability: func(context.Context, *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
					return &domain.AvailabilityResponse{}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) { requireNoRPCError(t, resp) },
		},
		{
			name:         "calendar.resources returns resources",
			method:       "calendar.resources",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				listRoomResources: func(context.Context, string) ([]domain.RoomResource, error) {
					return []domain.RoomResource{{Email: "room@example.com", Name: "Big Room"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result calendarResourcesResult
				unmarshalResult(t, resp, &result)
				if len(result.Resources) != 1 || result.Resources[0].Name != "Big Room" {
					t.Fatalf("resources = %+v, want one Big Room", result.Resources)
				}
			},
		},
		{
			name:         "event.import returns events",
			method:       "event.import",
			params:       `{"calendar_id":"cal-1","start":100,"end":200}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				importEvents: func(_ context.Context, _ string, params *domain.EventQueryParams) ([]domain.Event, error) {
					if params.CalendarID != "cal-1" {
						t.Fatalf("calendar_id = %q, want cal-1", params.CalendarID)
					}
					return []domain.Event{{ID: "ev-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result eventListPayloadResult
				unmarshalResult(t, resp, &result)
				if len(result.Events) != 1 || result.Events[0].ID != "ev-1" {
					t.Fatalf("events = %+v, want one ev-1", result.Events)
				}
			},
		},
		{
			name:         "event.import missing calendar_id",
			method:       "event.import",
			params:       `{"start":100,"end":200}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "event.recurring.list missing master_event_id",
			method:       "event.recurring.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "event.recurring.list returns instances with default calendar",
			method:       "event.recurring.list",
			params:       `{"master_event_id":"master-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				getRecurring: func(_ context.Context, _, calendarID, masterEventID string, _ *domain.EventQueryParams) ([]domain.Event, error) {
					if calendarID != "primary" || masterEventID != "master-1" {
						t.Fatalf("args = %q/%q, want primary/master-1", calendarID, masterEventID)
					}
					return []domain.Event{{ID: "inst-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result eventListPayloadResult
				unmarshalResult(t, resp, &result)
				if len(result.Events) != 1 {
					t.Fatalf("events = %+v, want one", result.Events)
				}
			},
		},
		{
			name:         "event.recurring.update missing event_id",
			method:       "event.recurring.update",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "event.recurring.delete returns deleted",
			method:       "event.recurring.delete",
			params:       `{"event_id":"ev-9"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				deleteRecurring: func(_ context.Context, _, calendarID, eventID string) error {
					if calendarID != "primary" || eventID != "ev-9" {
						t.Fatalf("args = %q/%q, want primary/ev-9", calendarID, eventID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:   "calendar.virtual.create requires email",
			method: "calendar.virtual.create",
			params: `{}`,
			client: &fakeCalendarExtClient{},
			assert: func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:   "calendar.virtual.create returns grant",
			method: "calendar.virtual.create",
			params: `{"email":"vcal@example.com"}`,
			client: &fakeCalendarExtClient{
				createVirtual: func(_ context.Context, email string) (*domain.VirtualCalendarGrant, error) {
					if email != "vcal@example.com" {
						t.Fatalf("email = %q, want vcal@example.com", email)
					}
					return &domain.VirtualCalendarGrant{ID: "vg-1", Email: email}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var grant domain.VirtualCalendarGrant
				unmarshalResult(t, resp, &grant)
				if grant.ID != "vg-1" {
					t.Fatalf("grant ID = %q, want vg-1", grant.ID)
				}
			},
		},
		{
			name:   "calendar.virtual.list returns grants",
			method: "calendar.virtual.list",
			params: `{}`,
			client: &fakeCalendarExtClient{
				listVirtual: func(context.Context) ([]domain.VirtualCalendarGrant, error) {
					return []domain.VirtualCalendarGrant{{ID: "vg-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result virtualCalendarListResult
				unmarshalResult(t, resp, &result)
				if len(result.Grants) != 1 {
					t.Fatalf("grants = %+v, want one", result.Grants)
				}
			},
		},
		{
			name:   "calendar.virtual.get missing grant_id",
			method: "calendar.virtual.get",
			params: `{}`,
			client: &fakeCalendarExtClient{},
			assert: func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:   "calendar.virtual.delete returns deleted",
			method: "calendar.virtual.delete",
			params: `{"grant_id":"vg-1"}`,
			client: &fakeCalendarExtClient{
				deleteVirtual: func(_ context.Context, grantID string) error {
					if grantID != "vg-1" {
						t.Fatalf("grantID = %q, want vg-1", grantID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "client error surfaces as internal error",
			method:       "calendar.get",
			params:       `{"calendar_id":"cal-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarExtClient{
				getCalendar: func(context.Context, string, string) (*domain.Calendar, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InternalError) },
		},
		{
			name:         "missing default grant errors",
			method:       "calendar.get",
			params:       `{"calendar_id":"cal-1"}`,
			defaultGrant: "",
			client:       &fakeCalendarExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterCalendarExtHandlers(d, tt.client, tt.defaultGrant)

			raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + tt.method + `","params":` + tt.params + `}`)
			got := d.Dispatch(context.Background(), raw)
			if got == nil {
				t.Fatal("Dispatch() = nil, want response")
			}
			var resp rpcTestResponse
			if err := json.Unmarshal(got, &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			tt.assert(t, resp)
		})
	}
}
