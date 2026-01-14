package domain

import (
	"testing"
	"time"
)

// =============================================================================
// SchedulerConfiguration Tests
// =============================================================================

func TestSchedulerConfiguration_Creation(t *testing.T) {
	now := time.Now()
	config := SchedulerConfiguration{
		ID:                  "config-123",
		Name:                "30-Minute Meeting",
		Slug:                "30-min-meeting",
		RequiresSessionAuth: true,
		Participants: []ConfigurationParticipant{
			{
				Email:       "host@example.com",
				Name:        "Host User",
				IsOrganizer: true,
				Availability: ConfigurationAvailability{
					CalendarIDs: []string{"cal-primary"},
					OpenHours: []OpenHours{
						{
							Days:     []int{1, 2, 3, 4, 5},
							Start:    "09:00",
							End:      "17:00",
							Timezone: "America/New_York",
						},
					},
				},
				Booking: &ParticipantBooking{
					CalendarID: "cal-primary",
				},
			},
		},
		Availability: AvailabilityRules{
			DurationMinutes:    30,
			IntervalMinutes:    15,
			RoundTo:            15,
			AvailabilityMethod: "max-availability",
			Buffer: &AvailabilityBuffer{
				Before: 5,
				After:  5,
			},
		},
		EventBooking: EventBooking{
			Title:           "Meeting with {{guest_name}}",
			Description:     "A 30-minute meeting",
			Location:        "Video call",
			Timezone:        "America/New_York",
			BookingType:     "booking",
			DisableEmails:   false,
			ReminderMinutes: []int{15, 60},
		},
		Scheduler: SchedulerSettings{
			AvailableDaysInFuture: 30,
			MinBookingNotice:      60,
			MinCancellationNotice: 60,
			ConfirmationMethod:    "automatic",
		},
		AppearanceSettings: &AppearanceSettings{
			CompanyName:     "Acme Corp",
			Color:           "#4285f4",
			SubmitText:      "Book Meeting",
			ThankYouMessage: "Thanks for booking!",
		},
		CreatedAt:  &now,
		ModifiedAt: &now,
	}

	if config.Name != "30-Minute Meeting" {
		t.Errorf("SchedulerConfiguration.Name = %q, want %q", config.Name, "30-Minute Meeting")
	}
	if config.Slug != "30-min-meeting" {
		t.Errorf("SchedulerConfiguration.Slug = %q, want %q", config.Slug, "30-min-meeting")
	}
	if !config.RequiresSessionAuth {
		t.Error("SchedulerConfiguration.RequiresSessionAuth should be true")
	}
	if len(config.Participants) != 1 {
		t.Errorf("SchedulerConfiguration.Participants length = %d, want 1", len(config.Participants))
	}
	if config.Availability.DurationMinutes != 30 {
		t.Errorf("AvailabilityRules.DurationMinutes = %d, want 30", config.Availability.DurationMinutes)
	}
}

// =============================================================================
// ConfigurationParticipant Tests
// =============================================================================

func TestConfigurationParticipant_Creation(t *testing.T) {
	participant := ConfigurationParticipant{
		Email:       "participant@example.com",
		Name:        "Participant Name",
		IsOrganizer: false,
		Availability: ConfigurationAvailability{
			CalendarIDs: []string{"cal-1", "cal-2"},
			OpenHours: []OpenHours{
				{
					Days:  []int{1, 2, 3, 4, 5},
					Start: "08:00",
					End:   "18:00",
				},
			},
		},
		Booking: &ParticipantBooking{
			CalendarID: "cal-1",
		},
	}

	if participant.Email != "participant@example.com" {
		t.Errorf("ConfigurationParticipant.Email = %q, want %q", participant.Email, "participant@example.com")
	}
	if participant.IsOrganizer {
		t.Error("ConfigurationParticipant.IsOrganizer should be false")
	}
	if len(participant.Availability.CalendarIDs) != 2 {
		t.Errorf("ConfigurationAvailability.CalendarIDs length = %d, want 2", len(participant.Availability.CalendarIDs))
	}
}

// =============================================================================
// OpenHours Tests
// =============================================================================

func TestOpenHours_Creation(t *testing.T) {
	tests := []struct {
		name      string
		openHours OpenHours
	}{
		{
			name: "weekday hours",
			openHours: OpenHours{
				Days:     []int{1, 2, 3, 4, 5},
				Start:    "09:00",
				End:      "17:00",
				Timezone: "America/Los_Angeles",
			},
		},
		{
			name: "weekend hours with excluded dates",
			openHours: OpenHours{
				Days:     []int{0, 6},
				Start:    "10:00",
				End:      "14:00",
				Timezone: "America/New_York",
				ExDates:  []string{"2024-12-25", "2024-01-01"},
			},
		},
		{
			name: "split day hours",
			openHours: OpenHours{
				Days:  []int{1, 3, 5},
				Start: "14:00",
				End:   "20:00",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.openHours.Days) == 0 {
				t.Error("OpenHours.Days should not be empty")
			}
			if tt.openHours.Start == "" {
				t.Error("OpenHours.Start should not be empty")
			}
			if tt.openHours.End == "" {
				t.Error("OpenHours.End should not be empty")
			}
		})
	}
}

// =============================================================================
// AvailabilityRules Tests
// =============================================================================

func TestAvailabilityRules_Creation(t *testing.T) {
	rules := AvailabilityRules{
		DurationMinutes:    45,
		IntervalMinutes:    15,
		RoundTo:            30,
		AvailabilityMethod: "max-fairness",
		Buffer: &AvailabilityBuffer{
			Before: 10,
			After:  5,
		},
	}

	if rules.DurationMinutes != 45 {
		t.Errorf("AvailabilityRules.DurationMinutes = %d, want 45", rules.DurationMinutes)
	}
	if rules.AvailabilityMethod != "max-fairness" {
		t.Errorf("AvailabilityRules.AvailabilityMethod = %q, want %q", rules.AvailabilityMethod, "max-fairness")
	}
	if rules.Buffer == nil {
		t.Fatal("AvailabilityRules.Buffer should not be nil")
	}
	if rules.Buffer.Before != 10 {
		t.Errorf("AvailabilityBuffer.Before = %d, want 10", rules.Buffer.Before)
	}
}

// =============================================================================
// AvailabilityBuffer Tests
// =============================================================================

func TestAvailabilityBuffer_Creation(t *testing.T) {
	buffer := AvailabilityBuffer{
		Before: 15,
		After:  10,
	}

	if buffer.Before != 15 {
		t.Errorf("AvailabilityBuffer.Before = %d, want 15", buffer.Before)
	}
	if buffer.After != 10 {
		t.Errorf("AvailabilityBuffer.After = %d, want 10", buffer.After)
	}
}

// =============================================================================
// EventBooking Tests
// =============================================================================

func TestEventBooking_Creation(t *testing.T) {
	booking := EventBooking{
		Title:       "Consultation with {{guest_name}}",
		Description: "A consultation meeting to discuss your needs",
		Location:    "Zoom",
		Timezone:    "Europe/London",
		BookingType: "organizer-confirmation",
		Conferencing: &ConferencingSettings{
			Provider:   "Zoom",
			Autocreate: true,
		},
		DisableEmails:   false,
		ReminderMinutes: []int{10, 30, 1440},
		Metadata: map[string]string{
			"booking_type": "consultation",
		},
	}

	if booking.Title == "" {
		t.Error("EventBooking.Title should not be empty")
	}
	if booking.BookingType != "organizer-confirmation" {
		t.Errorf("EventBooking.BookingType = %q, want %q", booking.BookingType, "organizer-confirmation")
	}
	if booking.Conferencing == nil {
		t.Fatal("EventBooking.Conferencing should not be nil")
	}
	if booking.Conferencing.Provider != "Zoom" {
		t.Errorf("ConferencingSettings.Provider = %q, want %q", booking.Conferencing.Provider, "Zoom")
	}
	if len(booking.ReminderMinutes) != 3 {
		t.Errorf("EventBooking.ReminderMinutes length = %d, want 3", len(booking.ReminderMinutes))
	}
}

// =============================================================================
// ConferencingSettings Tests
// =============================================================================

func TestConferencingSettings_Creation(t *testing.T) {
	tests := []struct {
		name     string
		settings ConferencingSettings
	}{
		{
			name: "Google Meet auto-create",
			settings: ConferencingSettings{
				Provider:   "Google Meet",
				Autocreate: true,
			},
		},
		{
			name: "Zoom with details",
			settings: ConferencingSettings{
				Provider:   "Zoom",
				Autocreate: false,
				Details: &ConferencingDetails{
					URL:         "https://zoom.us/j/123456789",
					MeetingCode: "123456789",
					Password:    "password123",
				},
			},
		},
		{
			name: "Microsoft Teams",
			settings: ConferencingSettings{
				Provider:   "Microsoft Teams",
				Autocreate: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.settings.Provider == "" {
				t.Error("ConferencingSettings.Provider should not be empty")
			}
		})
	}
}
