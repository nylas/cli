package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error) {
	return []domain.Grant{
		{
			ID:          "grant-demo-1",
			Provider:    "google",
			Email:       "demo1@example.com",
			GrantStatus: "valid",
		},
		{
			ID:          "grant-demo-2",
			Provider:    "microsoft",
			Email:       "demo2@example.com",
			GrantStatus: "valid",
		},
	}, nil
}

func (d *DemoClient) GetGrantStats(ctx context.Context) (*domain.GrantStats, error) {
	return &domain.GrantStats{
		Total:      10,
		Valid:      8,
		Invalid:    2,
		ByProvider: map[string]int{"google": 6, "microsoft": 4},
		ByStatus:   map[string]int{"valid": 8, "invalid": 2},
	}, nil
}

// Virtual Calendar Demo Implementations

func (d *DemoClient) CreateVirtualCalendarGrant(ctx context.Context, email string) (*domain.VirtualCalendarGrant, error) {
	return &domain.VirtualCalendarGrant{
		ID:          "vcal-demo-1",
		Provider:    "virtual-calendar",
		Email:       email,
		GrantStatus: "valid",
		CreatedAt:   1704067200,
		UpdatedAt:   1704067200,
	}, nil
}

func (d *DemoClient) ListVirtualCalendarGrants(ctx context.Context) ([]domain.VirtualCalendarGrant, error) {
	return []domain.VirtualCalendarGrant{
		{
			ID:          "vcal-demo-1",
			Provider:    "virtual-calendar",
			Email:       "conference-room-a@demo.com",
			GrantStatus: "valid",
			CreatedAt:   1704067200,
			UpdatedAt:   1704067200,
		},
		{
			ID:          "vcal-demo-2",
			Provider:    "virtual-calendar",
			Email:       "conference-room-b@demo.com",
			GrantStatus: "valid",
			CreatedAt:   1704153600,
			UpdatedAt:   1704153600,
		},
		{
			ID:          "vcal-demo-3",
			Provider:    "virtual-calendar",
			Email:       "resource-projector@demo.com",
			GrantStatus: "valid",
			CreatedAt:   1704240000,
			UpdatedAt:   1704240000,
		},
	}, nil
}

func (d *DemoClient) GetVirtualCalendarGrant(ctx context.Context, grantID string) (*domain.VirtualCalendarGrant, error) {
	return &domain.VirtualCalendarGrant{
		ID:          grantID,
		Provider:    "virtual-calendar",
		Email:       "conference-room-a@demo.com",
		GrantStatus: "valid",
		CreatedAt:   1704067200,
		UpdatedAt:   1704067200,
	}, nil
}

func (d *DemoClient) DeleteVirtualCalendarGrant(ctx context.Context, grantID string) error {
	return nil
}

// Recurring Event Demo Implementations

func (d *DemoClient) GetRecurringEventInstances(ctx context.Context, grantID, calendarID, masterEventID string, params *domain.EventQueryParams) ([]domain.Event, error) {
	now := time.Now()
	return []domain.Event{
		{
			ID:            "event-demo-instance-1",
			GrantID:       grantID,
			CalendarID:    calendarID,
			Title:         "Weekly Demo Team Meeting",
			Description:   "Recurring team sync - Instance 1",
			MasterEventID: masterEventID,
			When: domain.EventWhen{
				StartTime: now.Unix(),
				EndTime:   now.Add(1 * time.Hour).Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			Status:     "confirmed",
			Busy:       true,
		},
		{
			ID:            "event-demo-instance-2",
			GrantID:       grantID,
			CalendarID:    calendarID,
			Title:         "Weekly Demo Team Meeting",
			Description:   "Recurring team sync - Instance 2",
			MasterEventID: masterEventID,
			When: domain.EventWhen{
				StartTime: now.Add(7 * 24 * time.Hour).Unix(),
				EndTime:   now.Add(7*24*time.Hour + 1*time.Hour).Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			Status:     "confirmed",
			Busy:       true,
		},
		{
			ID:            "event-demo-instance-3",
			GrantID:       grantID,
			CalendarID:    calendarID,
			Title:         "Weekly Demo Team Meeting",
			Description:   "Recurring team sync - Instance 3",
			MasterEventID: masterEventID,
			When: domain.EventWhen{
				StartTime: now.Add(14 * 24 * time.Hour).Unix(),
				EndTime:   now.Add(14*24*time.Hour + 1*time.Hour).Unix(),
				Object:    "timespan",
			},
			Recurrence: []string{"RRULE:FREQ=WEEKLY;BYDAY=MO"},
			Status:     "confirmed",
			Busy:       true,
		},
	}, nil
}

func (d *DemoClient) UpdateRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string, req *domain.UpdateEventRequest) (*domain.Event, error) {
	now := time.Now()
	title := "Updated Demo Recurring Event Instance"
	if req.Title != nil {
		title = *req.Title
	}
	return &domain.Event{
		ID:         eventID,
		GrantID:    grantID,
		CalendarID: calendarID,
		Title:      title,
		When: domain.EventWhen{
			StartTime: now.Unix(),
			EndTime:   now.Add(1 * time.Hour).Unix(),
			Object:    "timespan",
		},
		Status: "confirmed",
		Busy:   true,
	}, nil
}

func (d *DemoClient) DeleteRecurringEventInstance(ctx context.Context, grantID, calendarID, eventID string) error {
	return nil
}
