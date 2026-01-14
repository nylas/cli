package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GetCalendars returns demo calendars.
func (d *DemoClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	return []domain.Calendar{
		{ID: "primary", Name: "Personal", IsPrimary: true, HexColor: "#4285F4"},
		{ID: "work", Name: "Work", IsPrimary: false, HexColor: "#0F9D58"},
		{ID: "family", Name: "Family", IsPrimary: false, HexColor: "#DB4437"},
	}, nil
}

// GetCalendar returns a demo calendar.
func (d *DemoClient) GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error) {
	return &domain.Calendar{ID: calendarID, Name: "Demo Calendar", IsPrimary: true}, nil
}

// CreateCalendar simulates creating a calendar.
func (d *DemoClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	return &domain.Calendar{
		ID:          "new-demo-calendar",
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		Timezone:    req.Timezone,
	}, nil
}

// UpdateCalendar simulates updating a calendar.
func (d *DemoClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	cal := &domain.Calendar{ID: calendarID}
	if req.Name != nil {
		cal.Name = *req.Name
	}
	if req.Description != nil {
		cal.Description = *req.Description
	}
	return cal, nil
}

// DeleteCalendar simulates deleting a calendar.
func (d *DemoClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	return nil
}
