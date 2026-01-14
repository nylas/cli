package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// GetEvents returns demo events.
func (d *DemoClient) GetEvents(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	return d.getDemoEvents(), nil
}

// GetEventsWithCursor returns demo events with pagination.
func (d *DemoClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	return &domain.EventListResponse{Data: d.getDemoEvents()}, nil
}

func (d *DemoClient) getDemoEvents() []domain.Event {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return []domain.Event{
		{
			ID:         "event-001",
			CalendarID: "primary",
			Title:      "Team Standup",
			When: domain.EventWhen{
				StartTime: today.Add(9 * time.Hour).Unix(),
				EndTime:   today.Add(9*time.Hour + 15*time.Minute).Unix(),
			},
			Status:   "confirmed",
			Location: "Conference Room A",
			Participants: []domain.Participant{
				{Person: domain.Person{Name: "Sarah Chen", Email: "sarah@company.com"}, Status: "yes"},
				{Person: domain.Person{Name: "Mike Johnson", Email: "mike@company.com"}, Status: "yes"},
				{Person: domain.Person{Name: "Demo User", Email: "demo@example.com"}, Status: "yes"},
			},
		},
		{
			ID:         "event-002",
			CalendarID: "primary",
			Title:      "1:1 with Manager",
			When: domain.EventWhen{
				StartTime: today.Add(11 * time.Hour).Unix(),
				EndTime:   today.Add(11*time.Hour + 30*time.Minute).Unix(),
			},
			Status:       "confirmed",
			Location:     "Google Meet",
			Conferencing: &domain.Conferencing{Provider: "Google Meet", Details: &domain.ConferencingDetails{URL: "https://meet.google.com/abc-defg-hij"}},
		},
		{
			ID:         "event-003",
			CalendarID: "primary",
			Title:      "Lunch Break",
			When: domain.EventWhen{
				StartTime: today.Add(12 * time.Hour).Unix(),
				EndTime:   today.Add(13 * time.Hour).Unix(),
			},
			Status: "confirmed",
		},
		{
			ID:         "event-004",
			CalendarID: "work",
			Title:      "Project Review",
			When: domain.EventWhen{
				StartTime: today.Add(14 * time.Hour).Unix(),
				EndTime:   today.Add(15 * time.Hour).Unix(),
			},
			Status:      "confirmed",
			Location:    "Main Office - Room 302",
			Description: "Quarterly project review with stakeholders",
			Participants: []domain.Participant{
				{Person: domain.Person{Name: "Product Team", Email: "product@company.com"}, Status: "yes"},
				{Person: domain.Person{Name: "Engineering", Email: "eng@company.com"}, Status: "maybe"},
			},
		},
		{
			ID:         "event-005",
			CalendarID: "primary",
			Title:      "Dentist Appointment",
			When: domain.EventWhen{
				StartTime: today.Add(24*time.Hour + 10*time.Hour).Unix(),
				EndTime:   today.Add(24*time.Hour + 11*time.Hour).Unix(),
			},
			Status:   "confirmed",
			Location: "123 Health St, Suite 400",
		},
		{
			ID:         "event-006",
			CalendarID: "family",
			Title:      "Birthday Party - Mom",
			When: domain.EventWhen{
				StartTime: today.Add(3*24*time.Hour + 18*time.Hour).Unix(),
				EndTime:   today.Add(3*24*time.Hour + 21*time.Hour).Unix(),
			},
			Status:   "confirmed",
			Location: "Family Home",
		},
		{
			ID:         "event-007",
			CalendarID: "primary",
			Title:      "Gym Session",
			When: domain.EventWhen{
				StartTime: today.Add(24*time.Hour + 7*time.Hour).Unix(),
				EndTime:   today.Add(24*time.Hour + 8*time.Hour).Unix(),
			},
			Status:   "confirmed",
			Location: "Downtown Fitness",
		},
	}
}

// GetEvent returns a demo event.
func (d *DemoClient) GetEvent(ctx context.Context, grantID, calendarID, eventID string) (*domain.Event, error) {
	events := d.getDemoEvents()
	for _, event := range events {
		if event.ID == eventID {
			return &event, nil
		}
	}
	return &events[0], nil
}

// CreateEvent simulates creating an event.
func (d *DemoClient) CreateEvent(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
	return &domain.Event{ID: "new-event", CalendarID: calendarID, Title: req.Title}, nil
}

// UpdateEvent simulates updating an event.
func (d *DemoClient) UpdateEvent(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	event := &domain.Event{ID: eventID, CalendarID: calendarID}
	if req.Title != nil {
		event.Title = *req.Title
	}
	return event, nil
}

// DeleteEvent simulates deleting an event.
func (d *DemoClient) DeleteEvent(ctx context.Context, grantID, calendarID, eventID string) error {
	return nil
}

// SendRSVP simulates sending an RSVP response.
func (d *DemoClient) SendRSVP(ctx context.Context, grantID, calendarID, eventID string, req *domain.SendRSVPRequest) error {
	return nil
}

// GetFreeBusy returns demo free/busy information.
func (d *DemoClient) GetFreeBusy(ctx context.Context, grantID string, req *domain.FreeBusyRequest) (*domain.FreeBusyResponse, error) {
	result := &domain.FreeBusyResponse{
		Data: make([]domain.FreeBusyCalendar, len(req.Emails)),
	}
	for i, email := range req.Emails {
		result.Data[i] = domain.FreeBusyCalendar{
			Email: email,
			TimeSlots: []domain.TimeSlot{
				{StartTime: req.StartTime + 3600, EndTime: req.StartTime + 7200, Status: "busy"},
			},
		}
	}
	return result, nil
}

// GetAvailability returns demo availability slots.
func (d *DemoClient) GetAvailability(ctx context.Context, req *domain.AvailabilityRequest) (*domain.AvailabilityResponse, error) {
	duration := int64(req.DurationMinutes * 60)
	return &domain.AvailabilityResponse{
		Data: domain.AvailabilityData{
			TimeSlots: []domain.AvailableSlot{
				{StartTime: req.StartTime + 7200, EndTime: req.StartTime + 7200 + duration},
				{StartTime: req.StartTime + 14400, EndTime: req.StartTime + 14400 + duration},
			},
		},
	}, nil
}
