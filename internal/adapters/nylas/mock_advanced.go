package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	return []domain.Grant{
		{
			ID:          "grant-1",
			Provider:    "google",
			Email:       "user1@example.com",
			GrantStatus: "valid",
		},
		{
			ID:          "grant-2",
			Provider:    "microsoft",
			Email:       "user2@example.com",
			GrantStatus: "valid",
		},
	}, nil
}

func (m *MockClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	return &domain.GrantStats{
		Total:      10,
		Valid:      8,
		Invalid:    2,
		ByProvider: map[string]int{"google": 6, "microsoft": 4},
		ByStatus:   map[string]int{"valid": 8, "invalid": 2},
	}, nil
}

// Virtual Calendar operations

func (m *MockClient) CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error) {
	return &domain.VirtualCalendarGrant{
		ID:          "vcal-grant-1",
		Provider:    "virtual-calendar",
		Email:       email,
		GrantStatus: "valid",
		CreatedAt:   1704067200,
		UpdatedAt:   1704067200,
	}, nil
}

func (m *MockClient) ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error) {
	return []domain.VirtualCalendarGrant{
		{
			ID:          "vcal-grant-1",
			Provider:    "virtual-calendar",
			Email:       "room-a@example.com",
			GrantStatus: "valid",
			CreatedAt:   1704067200,
			UpdatedAt:   1704067200,
		},
		{
			ID:          "vcal-grant-2",
			Provider:    "virtual-calendar",
			Email:       "room-b@example.com",
			GrantStatus: "valid",
			CreatedAt:   1704153600,
			UpdatedAt:   1704153600,
		},
	}, nil
}

func (m *MockClient) GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error) {
	return &domain.VirtualCalendarGrant{
		ID:          grantID,
		Provider:    "virtual-calendar",
		Email:       "room-a@example.com",
		GrantStatus: "valid",
		CreatedAt:   1704067200,
		UpdatedAt:   1704067200,
	}, nil
}

func (m *MockClient) DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error {
	return nil
}

// Recurring Event operations

func (m *MockClient) GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	now := time.Now()
	return []domain.Event{
		{
			ID:            "event-instance-1",
			GrantID:       grantID,
			CalendarID:    calendarID,
			Title:         "Weekly Team Meeting (Instance 1)",
			MasterEventID: masterEventID,
			When: domain.EventWhen{
				StartTime: now.Unix(),
				EndTime:   now.Add(1 * time.Hour).Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			Status:     "confirmed",
		},
		{
			ID:            "event-instance-2",
			GrantID:       grantID,
			CalendarID:    calendarID,
			Title:         "Weekly Team Meeting (Instance 2)",
			MasterEventID: masterEventID,
			When: domain.EventWhen{
				StartTime: now.Add(7 * 24 * time.Hour).Unix(),
				EndTime:   now.Add(7*24*time.Hour + 1*time.Hour).Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			Status:     "confirmed",
		},
	}, nil
}

func (m *MockClient) UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	now := time.Now()
	event := &domain.Event{
		ID:         eventID,
		GrantID:    grantID,
		CalendarID: calendarID,
		Title:      "Updated Recurring Event Instance",
		When: domain.EventWhen{
			StartTime: now.Unix(),
			EndTime:   now.Add(1 * time.Hour).Unix(),
			Object:    "timespan",
		},
		Status: "confirmed",
	}
	if req.Title != nil {
		event.Title = *req.Title
	}
	return event, nil
}

func (m *MockClient) DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error {
	return nil
}
