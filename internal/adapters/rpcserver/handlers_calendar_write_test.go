package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeCalendarWriteClient struct {
	ports.CalendarClient

	createEvent func(context.Context, string, string, *domain.CreateEventRequest) (*domain.Event, error)
	updateEvent func(context.Context, string, string, string, *domain.UpdateEventRequest) (*domain.Event, error)
	deleteEvent func(context.Context, string, string, string) error
	sendRSVP    func(context.Context, string, string, string, *domain.SendRSVPRequest) error

	createGrantID    string
	createCalendarID string
	createReq        domain.CreateEventRequest
	updateGrantID    string
	updateCalendarID string
	updateEventID    string
	updateReq        domain.UpdateEventRequest
	deleteGrantID    string
	deleteCalendarID string
	deleteEventID    string
	rsvpGrantID      string
	rsvpCalendarID   string
	rsvpEventID      string
	rsvpReq          domain.SendRSVPRequest
}

func (f *fakeCalendarWriteClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	f.createGrantID = grantID
	f.createCalendarID = calendarID
	if req != nil {
		f.createReq = *req
	}
	if f.createEvent == nil {
		return nil, errors.New("unexpected CreateEvent")
	}
	return f.createEvent(ctx, grantID, calendarID, req)
}

func (f *fakeCalendarWriteClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	f.updateGrantID = grantID
	f.updateCalendarID = calendarID
	f.updateEventID = eventID
	if req != nil {
		f.updateReq = *req
	}
	if f.updateEvent == nil {
		return nil, errors.New("unexpected UpdateEvent")
	}
	return f.updateEvent(ctx, grantID, calendarID, eventID, req)
}

func (f *fakeCalendarWriteClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	f.deleteGrantID = grantID
	f.deleteCalendarID = calendarID
	f.deleteEventID = eventID
	if f.deleteEvent == nil {
		return errors.New("unexpected DeleteEvent")
	}
	return f.deleteEvent(ctx, grantID, calendarID, eventID)
}

func (f *fakeCalendarWriteClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	f.rsvpGrantID = grantID
	f.rsvpCalendarID = calendarID
	f.rsvpEventID = eventID
	if req != nil {
		f.rsvpReq = *req
	}
	if f.sendRSVP == nil {
		return errors.New("unexpected SendRSVP")
	}
	return f.sendRSVP(ctx, grantID, calendarID, eventID, req)
}

func TestRegisterCalendarWriteHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	title := "Updated sync"
	busy := true

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeCalendarWriteClient
		assert       func(*testing.T, *fakeCalendarWriteClient, rpcTestResponse)
	}{
		{
			name:         "event.create defaults calendar and forwards embedded request",
			method:       "event.create",
			params:       `{"title":"Sync","description":"Weekly","location":"Room 1","when":{"start_time":1710000000,"end_time":1710003600},"participants":[{"email":"ada@example.com","name":"Ada"}],"busy":true,"visibility":"private","recurrence":["RRULE:FREQ=WEEKLY"],"metadata":{"source":"rpc"}}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarWriteClient{
				createEvent: func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
					return &domain.Event{ID: "event-1", Title: req.Title}, nil
				},
			},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				if client.createGrantID != "default-grant" || client.createCalendarID != "primary" {
					t.Fatalf("create args = %q %q, want default-grant primary", client.createGrantID, client.createCalendarID)
				}
				if client.createReq.Title != "Sync" || client.createReq.Description != "Weekly" || client.createReq.Location != "Room 1" {
					t.Fatalf("create request = %+v, want embedded string fields", client.createReq)
				}
				if client.createReq.When.StartTime != 1710000000 || client.createReq.When.EndTime != 1710003600 {
					t.Fatalf("create when = %+v, want forwarded times", client.createReq.When)
				}
				if len(client.createReq.Participants) != 1 || client.createReq.Participants[0].Email != "ada@example.com" || client.createReq.Participants[0].Name != "Ada" {
					t.Fatalf("participants = %+v, want Ada", client.createReq.Participants)
				}
				if !client.createReq.Busy || client.createReq.Visibility != "private" || len(client.createReq.Recurrence) != 1 || client.createReq.Metadata["source"] != "rpc" {
					t.Fatalf("create request = %+v, want busy visibility recurrence metadata", client.createReq)
				}

				var event domain.Event
				unmarshalResult(t, resp, &event)
				if event.ID != "event-1" || event.Title != "Sync" {
					t.Fatalf("event = %+v, want event-1 Sync", event)
				}
			},
		},
		{
			name:   "event.create missing grant returns invalid params",
			method: "event.create",
			params: `{"title":"Sync"}`,
			client: &fakeCalendarWriteClient{},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.createGrantID != "" {
					t.Fatalf("CreateEvent called with grant %q, want no call", client.createGrantID)
				}
			},
		},
		{
			name:         "event.update requires event_id",
			method:       "event.update",
			params:       `{"title":"Sync"}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarWriteClient{},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.updateEventID != "" {
					t.Fatalf("UpdateEvent called with event %q, want no call", client.updateEventID)
				}
			},
		},
		{
			name:         "event.update forwards embedded request",
			method:       "event.update",
			params:       `{"grant_id":"grant-1","calendar_id":"cal-1","event_id":"event-1","title":"Updated sync","busy":true,"metadata":{"source":"rpc"}}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarWriteClient{
				updateEvent: func(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
					return &domain.Event{ID: eventID, Title: *req.Title}, nil
				},
			},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				if client.updateGrantID != "grant-1" || client.updateCalendarID != "cal-1" || client.updateEventID != "event-1" {
					t.Fatalf("update args = %q %q %q, want grant-1 cal-1 event-1", client.updateGrantID, client.updateCalendarID, client.updateEventID)
				}
				if client.updateReq.Title == nil || *client.updateReq.Title != title {
					t.Fatalf("Title = %v, want %q", client.updateReq.Title, title)
				}
				if client.updateReq.Busy == nil || *client.updateReq.Busy != busy || client.updateReq.Metadata["source"] != "rpc" {
					t.Fatalf("update request = %+v, want busy metadata", client.updateReq)
				}
			},
		},
		{
			name:         "event.delete requires event_id",
			method:       "event.delete",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarWriteClient{},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.deleteEventID != "" {
					t.Fatalf("DeleteEvent called with event %q, want no call", client.deleteEventID)
				}
			},
		},
		{
			name:         "event.delete returns deleted",
			method:       "event.delete",
			params:       `{"grant_id":"grant-1","calendar_id":"cal-1","event_id":"event-1"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarWriteClient{
				deleteEvent: func(ctx context.Context, grantID, calendarID, eventID string) error {
					return nil
				},
			},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				if client.deleteGrantID != "grant-1" || client.deleteCalendarID != "cal-1" || client.deleteEventID != "event-1" {
					t.Fatalf("delete args = %q %q %q, want grant-1 cal-1 event-1", client.deleteGrantID, client.deleteCalendarID, client.deleteEventID)
				}
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "event.rsvp requires event_id",
			method:       "event.rsvp",
			params:       `{"status":"yes"}`,
			defaultGrant: "default-grant",
			client:       &fakeCalendarWriteClient{},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.rsvpEventID != "" {
					t.Fatalf("SendRSVP called with event %q, want no call", client.rsvpEventID)
				}
			},
		},
		{
			name:         "event.rsvp forwards embedded request and returns ok",
			method:       "event.rsvp",
			params:       `{"grant_id":"grant-1","calendar_id":"cal-1","event_id":"event-1","status":"yes","comment":"See you there"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarWriteClient{
				sendRSVP: func(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
					return nil
				},
			},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				if client.rsvpGrantID != "grant-1" || client.rsvpCalendarID != "cal-1" || client.rsvpEventID != "event-1" {
					t.Fatalf("rsvp args = %q %q %q, want grant-1 cal-1 event-1", client.rsvpGrantID, client.rsvpCalendarID, client.rsvpEventID)
				}
				if client.rsvpReq.Status != "yes" || client.rsvpReq.Comment != "See you there" {
					t.Fatalf("rsvp request = %+v, want yes with comment", client.rsvpReq)
				}
				var result eventRSVPResult
				unmarshalResult(t, resp, &result)
				if !result.OK {
					t.Fatal("ok = false, want true")
				}
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "event.create",
			params:       `{"title":"Sync"}`,
			defaultGrant: "default-grant",
			client: &fakeCalendarWriteClient{
				createEvent: func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeCalendarWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterCalendarWriteHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchCalendarRequest(t, d, tt.method, tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}
