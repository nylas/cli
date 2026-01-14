package domain

import (
	"testing"
)

// =============================================================================
// Calendar Type Tests
// =============================================================================

func TestCalendar_Creation(t *testing.T) {
	cal := Calendar{
		ID:          "cal-123",
		GrantID:     "grant-456",
		Name:        "Work Calendar",
		Description: "My work events",
		Location:    "Office",
		Timezone:    "America/New_York",
		ReadOnly:    false,
		IsPrimary:   true,
		IsOwner:     true,
		HexColor:    "#4285f4",
	}

	if cal.ID != "cal-123" {
		t.Errorf("Calendar.ID = %q, want %q", cal.ID, "cal-123")
	}
	if cal.Name != "Work Calendar" {
		t.Errorf("Calendar.Name = %q, want %q", cal.Name, "Work Calendar")
	}
	if !cal.IsPrimary {
		t.Error("Calendar.IsPrimary should be true")
	}
	if !cal.IsOwner {
		t.Error("Calendar.IsOwner should be true")
	}
}

// =============================================================================
// TimeSlot Tests
// =============================================================================

func TestTimeSlot_Creation(t *testing.T) {
	slot := TimeSlot{
		StartTime: 1704067200,
		EndTime:   1704070800,
		Status:    "busy",
	}

	if slot.StartTime != 1704067200 {
		t.Errorf("TimeSlot.StartTime = %d, want %d", slot.StartTime, 1704067200)
	}
	if slot.Status != "busy" {
		t.Errorf("TimeSlot.Status = %q, want %q", slot.Status, "busy")
	}
}

// =============================================================================
// Participant Tests
// =============================================================================

func TestParticipant_Creation(t *testing.T) {
	tests := []struct {
		name        string
		participant Participant
		wantEmail   string
		wantStatus  string
	}{
		{
			name: "creates participant with all fields",
			participant: Participant{
				Person:  Person{Name: "John Doe", Email: "john@example.com"},
				Status:  "yes",
				Comment: "Looking forward to it",
			},
			wantEmail:  "john@example.com",
			wantStatus: "yes",
		},
		{
			name: "creates participant with minimal fields",
			participant: Participant{
				Person: Person{Email: "jane@example.com"},
			},
			wantEmail:  "jane@example.com",
			wantStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.participant.Email != tt.wantEmail {
				t.Errorf("Participant.Email = %q, want %q", tt.participant.Email, tt.wantEmail)
			}
			if tt.participant.Status != tt.wantStatus {
				t.Errorf("Participant.Status = %q, want %q", tt.participant.Status, tt.wantStatus)
			}
		})
	}
}

// =============================================================================
// Conferencing Tests
// =============================================================================

func TestConferencing_Creation(t *testing.T) {
	conf := Conferencing{
		Provider: "Google Meet",
		Details: &ConferencingDetails{
			URL:         "https://meet.google.com/abc-defg-hij",
			MeetingCode: "abc-defg-hij",
			Password:    "secret123",
			Phone:       []string{"+1-555-123-4567"},
		},
	}

	if conf.Provider != "Google Meet" {
		t.Errorf("Conferencing.Provider = %q, want %q", conf.Provider, "Google Meet")
	}
	if conf.Details == nil {
		t.Fatal("Conferencing.Details should not be nil")
	}
	if conf.Details.URL != "https://meet.google.com/abc-defg-hij" {
		t.Errorf("ConferencingDetails.URL = %q, want expected URL", conf.Details.URL)
	}
	if len(conf.Details.Phone) != 1 {
		t.Errorf("ConferencingDetails.Phone length = %d, want 1", len(conf.Details.Phone))
	}
}

// =============================================================================
// Reminders Tests
// =============================================================================

func TestReminders_Creation(t *testing.T) {
	reminders := Reminders{
		UseDefault: false,
		Overrides: []Reminder{
			{ReminderMinutes: 10, ReminderMethod: "popup"},
			{ReminderMinutes: 60, ReminderMethod: "email"},
		},
	}

	if reminders.UseDefault {
		t.Error("Reminders.UseDefault should be false")
	}
	if len(reminders.Overrides) != 2 {
		t.Fatalf("Reminders.Overrides length = %d, want 2", len(reminders.Overrides))
	}
	if reminders.Overrides[0].ReminderMinutes != 10 {
		t.Errorf("Reminder.ReminderMinutes = %d, want 10", reminders.Overrides[0].ReminderMinutes)
	}
}

// =============================================================================
// VirtualCalendarGrant Tests
// =============================================================================

func TestVirtualCalendarGrant_Creation(t *testing.T) {
	grant := VirtualCalendarGrant{
		ID:          "vcal-123",
		Provider:    "virtual-calendar",
		Email:       "resource-room@company.com",
		GrantStatus: "valid",
		CreatedAt:   1704067200,
		UpdatedAt:   1704070800,
	}

	if grant.Provider != "virtual-calendar" {
		t.Errorf("VirtualCalendarGrant.Provider = %q, want %q", grant.Provider, "virtual-calendar")
	}
	if grant.GrantStatus != "valid" {
		t.Errorf("VirtualCalendarGrant.GrantStatus = %q, want %q", grant.GrantStatus, "valid")
	}
}

// =============================================================================
// EventQueryParams Tests
// =============================================================================

func TestEventQueryParams_Creation(t *testing.T) {
	busy := true
	params := EventQueryParams{
		Limit:           50,
		CalendarID:      "cal-123",
		Title:           "Team Meeting",
		Start:           1704067200,
		End:             1706745600,
		ShowCancelled:   true,
		Busy:            &busy,
		OrderBy:         "start",
		ExpandRecurring: true,
	}

	if params.Limit != 50 {
		t.Errorf("EventQueryParams.Limit = %d, want 50", params.Limit)
	}
	if params.CalendarID != "cal-123" {
		t.Errorf("EventQueryParams.CalendarID = %q, want %q", params.CalendarID, "cal-123")
	}
	if !params.ShowCancelled {
		t.Error("EventQueryParams.ShowCancelled should be true")
	}
	if params.Busy == nil || !*params.Busy {
		t.Error("EventQueryParams.Busy should be true")
	}
	if !params.ExpandRecurring {
		t.Error("EventQueryParams.ExpandRecurring should be true")
	}
}

// =============================================================================
// AvailabilityRequest Tests
// =============================================================================

func TestAvailabilityRequest_Creation(t *testing.T) {
	req := AvailabilityRequest{
		StartTime:       1704067200,
		EndTime:         1704153600,
		DurationMinutes: 30,
		Participants: []AvailabilityParticipant{
			{
				Email:       "user1@example.com",
				CalendarIDs: []string{"cal-1", "cal-2"},
			},
			{
				Email: "user2@example.com",
			},
		},
		IntervalMinutes: 15,
		RoundTo:         30,
	}

	if req.DurationMinutes != 30 {
		t.Errorf("AvailabilityRequest.DurationMinutes = %d, want 30", req.DurationMinutes)
	}
	if len(req.Participants) != 2 {
		t.Fatalf("AvailabilityRequest.Participants length = %d, want 2", len(req.Participants))
	}
	if req.Participants[0].Email != "user1@example.com" {
		t.Errorf("Participant.Email = %q, want %q", req.Participants[0].Email, "user1@example.com")
	}
	if len(req.Participants[0].CalendarIDs) != 2 {
		t.Errorf("Participant.CalendarIDs length = %d, want 2", len(req.Participants[0].CalendarIDs))
	}
}

// =============================================================================
// FreeBusyRequest Tests
// =============================================================================

func TestFreeBusyRequest_Creation(t *testing.T) {
	req := FreeBusyRequest{
		StartTime: 1704067200,
		EndTime:   1704153600,
		Emails:    []string{"user1@example.com", "user2@example.com"},
	}

	if req.StartTime != 1704067200 {
		t.Errorf("FreeBusyRequest.StartTime = %d, want %d", req.StartTime, 1704067200)
	}
	if len(req.Emails) != 2 {
		t.Errorf("FreeBusyRequest.Emails length = %d, want 2", len(req.Emails))
	}
}
