package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	if m.GetEventsFunc != nil {
		return m.GetEventsFunc(ctx, grantID, calendarID, params)
	}
	return []domain.Event{}, nil
}

// GetEventsWithCursor retrieves events with pagination.
func (m *MockClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	return &domain.EventListResponse{Data: []domain.Event{}}, nil
}

// GetEvent retrieves a single event.
func (m *MockClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(ctx, grantID, calendarID, eventID)
	}
	return &domain.Event{
		ID:         eventID,
		CalendarID: calendarID,
		Title:      "Test Event",
	}, nil
}

// CreateEvent creates a new event.
func (m *MockClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	if m.CreateEventFunc != nil {
		return m.CreateEventFunc(ctx, grantID, calendarID, req)
	}
	return &domain.Event{
		ID:         "new-event-id",
		CalendarID: calendarID,
		Title:      req.Title,
	}, nil
}

// UpdateEvent updates an existing event.
func (m *MockClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	if m.UpdateEventFunc != nil {
		return m.UpdateEventFunc(ctx, grantID, calendarID, eventID, req)
	}
	event := &domain.Event{
		ID:         eventID,
		CalendarID: calendarID,
	}
	if req.Title != nil {
		event.Title = *req.Title
	}
	return event, nil
}

// DeleteEvent deletes an event.
func (m *MockClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	if m.DeleteEventFunc != nil {
		return m.DeleteEventFunc(ctx, grantID, calendarID, eventID)
	}
	return nil
}

// SendRSVP sends an RSVP response to an event invitation.
func (m *MockClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	return nil
}

// GetFreeBusy retrieves free/busy information.
func (m *MockClient) GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	now := req.StartTime
	result := &domain.FreeBusyResponse{
		Data: make([]domain.FreeBusyCalendar, len(req.Emails)),
	}
	for i, email := range req.Emails {
		result.Data[i] = domain.FreeBusyCalendar{
			Email: email,
			TimeSlots: []domain.TimeSlot{
				{
					StartTime: now + 3600, // 1 hour from start
					EndTime:   now + 7200, // 2 hours from start
					Status:    "busy",
				},
			},
		}
	}
	return result, nil
}

// GetAvailability finds available meeting times.
func (m *MockClient) GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	duration := int64(req.DurationMinutes * 60)
	result := &domain.AvailabilityResponse{
		Data: domain.AvailabilityData{
			TimeSlots: []domain.AvailableSlot{
				{
					StartTime: req.StartTime + 7200,
					EndTime:   req.StartTime + 7200 + duration,
				},
				{
					StartTime: req.StartTime + 14400,
					EndTime:   req.StartTime + 14400 + duration,
				},
			},
		},
	}
	return result, nil
}

// GetContacts retrieves contacts.
