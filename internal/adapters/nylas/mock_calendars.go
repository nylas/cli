package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetCalendars(ctx context.Context, grantID string) ([]domain.Calendar, error) {
	if m.GetCalendarsFunc != nil {
		return m.GetCalendarsFunc(ctx, grantID)
	}
	return []domain.Calendar{
		{ID: "primary", Name: "Primary Calendar", IsPrimary: true},
	}, nil
}

// GetCalendar retrieves a single calendar.
func (m *MockClient) GetCalendar(ctx context.Context, grantID, calendarID string) (*domain.Calendar, error) {
	return &domain.Calendar{
		ID:        calendarID,
		Name:      "Test Calendar",
		IsPrimary: calendarID == "primary",
	}, nil
}

// CreateCalendar creates a new calendar.
func (m *MockClient) CreateCalendar(ctx context.Context, grantID string, req *domain.CreateCalendarRequest) (*domain.Calendar, error) {
	return &domain.Calendar{
		ID:          "new-calendar-id",
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		Timezone:    req.Timezone,
	}, nil
}

// UpdateCalendar updates an existing calendar.
func (m *MockClient) UpdateCalendar(ctx context.Context, grantID, calendarID string, req *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
	cal := &domain.Calendar{ID: calendarID}
	if req.Name != nil {
		cal.Name = *req.Name
	}
	if req.Description != nil {
		cal.Description = *req.Description
	}
	if req.Location != nil {
		cal.Location = *req.Location
	}
	if req.Timezone != nil {
		cal.Timezone = *req.Timezone
	}
	if req.HexColor != nil {
		cal.HexColor = *req.HexColor
	}
	return cal, nil
}

// DeleteCalendar deletes a calendar.
func (m *MockClient) DeleteCalendar(ctx context.Context, grantID, calendarID string) error {
	return nil
}

// GetEvents retrieves events.
