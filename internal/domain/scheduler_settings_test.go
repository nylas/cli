package domain

import (
	"testing"
	"time"
)

// =============================================================================
// SchedulerSettings Tests
// =============================================================================

func TestSchedulerSettings_Creation(t *testing.T) {
	settings := SchedulerSettings{
		AvailableDaysInFuture: 60,
		MinBookingNotice:      120,
		MinCancellationNotice: 1440,
		ConfirmationMethod:    "manual",
		ReschedulingURL:       "https://scheduler.example.com/reschedule",
		CancellationURL:       "https://scheduler.example.com/cancel",
		AdditionalFields: map[string]any{
			"phone":   "required",
			"company": "optional",
		},
		CancellationPolicy: "24 hours notice required",
	}

	if settings.AvailableDaysInFuture != 60 {
		t.Errorf("SchedulerSettings.AvailableDaysInFuture = %d, want 60", settings.AvailableDaysInFuture)
	}
	if settings.ConfirmationMethod != "manual" {
		t.Errorf("SchedulerSettings.ConfirmationMethod = %q, want %q", settings.ConfirmationMethod, "manual")
	}
	if settings.CancellationPolicy == "" {
		t.Error("SchedulerSettings.CancellationPolicy should not be empty")
	}
}

// =============================================================================
// AppearanceSettings Tests
// =============================================================================

func TestAppearanceSettings_Creation(t *testing.T) {
	appearance := AppearanceSettings{
		CompanyName:     "Tech Startup Inc",
		Logo:            "https://example.com/logo.png",
		Color:           "#00ff00",
		SubmitText:      "Schedule Now",
		ThankYouMessage: "Your meeting has been scheduled!",
	}

	if appearance.CompanyName != "Tech Startup Inc" {
		t.Errorf("AppearanceSettings.CompanyName = %q, want %q", appearance.CompanyName, "Tech Startup Inc")
	}
	if appearance.Color != "#00ff00" {
		t.Errorf("AppearanceSettings.Color = %q, want %q", appearance.Color, "#00ff00")
	}
}

// =============================================================================
// SchedulerSession Tests
// =============================================================================

func TestSchedulerSession_Creation(t *testing.T) {
	now := time.Now()
	session := SchedulerSession{
		SessionID:       "session-123",
		ConfigurationID: "config-456",
		BookingURL:      "https://scheduler.example.com/book/session-123",
		CreatedAt:       now,
		ExpiresAt:       now.Add(24 * time.Hour),
	}

	if session.SessionID != "session-123" {
		t.Errorf("SchedulerSession.SessionID = %q, want %q", session.SessionID, "session-123")
	}
	if session.ConfigurationID != "config-456" {
		t.Errorf("SchedulerSession.ConfigurationID = %q, want %q", session.ConfigurationID, "config-456")
	}
	if session.BookingURL == "" {
		t.Error("SchedulerSession.BookingURL should not be empty")
	}
}

// =============================================================================
// CreateSchedulerSessionRequest Tests
// =============================================================================

func TestCreateSchedulerSessionRequest_Creation(t *testing.T) {
	req := CreateSchedulerSessionRequest{
		ConfigurationID: "config-789",
		TimeToLive:      60,
		Slug:            "quick-chat",
		AdditionalFields: map[string]any{
			"email":   "guest@example.com",
			"company": "Guest Corp",
		},
	}

	if req.ConfigurationID != "config-789" {
		t.Errorf("CreateSchedulerSessionRequest.ConfigurationID = %q, want %q", req.ConfigurationID, "config-789")
	}
	if req.TimeToLive != 60 {
		t.Errorf("CreateSchedulerSessionRequest.TimeToLive = %d, want 60", req.TimeToLive)
	}
}

// =============================================================================
// Booking Tests
// =============================================================================

func TestBooking_Creation(t *testing.T) {
	now := time.Now()
	booking := Booking{
		BookingID: "booking-123",
		EventID:   "event-456",
		Title:     "Strategy Meeting",
		Organizer: Participant{
			Person: Person{Name: "Host User", Email: "host@example.com"},
		},
		Participants: []Participant{
			{Person: Person{Name: "Guest User", Email: "guest@example.com"}, Status: "yes"},
		},
		StartTime:   now.Add(24 * time.Hour),
		EndTime:     now.Add(25 * time.Hour),
		Status:      "confirmed",
		Description: "Discuss Q1 strategy",
		Location:    "Conference Room A",
		Timezone:    "America/New_York",
		Conferencing: &ConferencingDetails{
			URL:         "https://meet.google.com/abc-defg-hij",
			MeetingCode: "abc-defg-hij",
		},
		AdditionalFields: map[string]any{
			"guest_phone": "+1-555-123-4567",
		},
		Metadata: map[string]string{
			"source": "website",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if booking.BookingID != "booking-123" {
		t.Errorf("Booking.BookingID = %q, want %q", booking.BookingID, "booking-123")
	}
	if booking.Status != "confirmed" {
		t.Errorf("Booking.Status = %q, want %q", booking.Status, "confirmed")
	}
	if booking.Organizer.Email != "host@example.com" {
		t.Errorf("Booking.Organizer.Email = %q, want %q", booking.Organizer.Email, "host@example.com")
	}
	if len(booking.Participants) != 1 {
		t.Errorf("Booking.Participants length = %d, want 1", len(booking.Participants))
	}
}

func TestBooking_StatusValues(t *testing.T) {
	tests := []struct {
		name   string
		status string
	}{
		{"confirmed", "confirmed"},
		{"cancelled", "cancelled"},
		{"pending", "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			booking := Booking{Status: tt.status}
			if booking.Status != tt.status {
				t.Errorf("Booking.Status = %q, want %q", booking.Status, tt.status)
			}
		})
	}
}

// =============================================================================
// ConfirmBookingRequest Tests
// =============================================================================

func TestConfirmBookingRequest_Creation(t *testing.T) {
	tests := []struct {
		name string
		req  ConfirmBookingRequest
	}{
		{
			name: "confirm booking",
			req: ConfirmBookingRequest{
				Status: "confirmed",
			},
		},
		{
			name: "cancel booking with reason",
			req: ConfirmBookingRequest{
				Status: "cancelled",
				Reason: "Schedule conflict",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.Status == "" {
				t.Error("ConfirmBookingRequest.Status should not be empty")
			}
		})
	}
}

// =============================================================================
// RescheduleBookingRequest Tests
// =============================================================================

func TestRescheduleBookingRequest_Creation(t *testing.T) {
	now := time.Now()
	req := RescheduleBookingRequest{
		StartTime: now.Add(48 * time.Hour).Unix(),
		EndTime:   now.Add(49 * time.Hour).Unix(),
		Timezone:  "America/Chicago",
		Reason:    "Guest requested different time",
	}

	if req.StartTime == 0 {
		t.Error("RescheduleBookingRequest.StartTime should not be zero")
	}
	if req.EndTime == 0 {
		t.Error("RescheduleBookingRequest.EndTime should not be zero")
	}
	if req.Timezone != "America/Chicago" {
		t.Errorf("RescheduleBookingRequest.Timezone = %q, want %q", req.Timezone, "America/Chicago")
	}
}

// =============================================================================
// SchedulerPage Tests
// =============================================================================

func TestSchedulerPage_Creation(t *testing.T) {
	now := time.Now()
	page := SchedulerPage{
		ID:              "page-123",
		ConfigurationID: "config-456",
		Name:            "Book a Demo",
		Slug:            "book-demo",
		URL:             "https://scheduler.example.com/book-demo",
		CustomDomain:    "booking.example.com",
		CreatedAt:       now.Add(-7 * 24 * time.Hour),
		ModifiedAt:      now,
	}

	if page.Name != "Book a Demo" {
		t.Errorf("SchedulerPage.Name = %q, want %q", page.Name, "Book a Demo")
	}
	if page.Slug != "book-demo" {
		t.Errorf("SchedulerPage.Slug = %q, want %q", page.Slug, "book-demo")
	}
	if page.CustomDomain != "booking.example.com" {
		t.Errorf("SchedulerPage.CustomDomain = %q, want %q", page.CustomDomain, "booking.example.com")
	}
}

// =============================================================================
// CreateSchedulerPageRequest Tests
// =============================================================================

func TestCreateSchedulerPageRequest_Creation(t *testing.T) {
	req := CreateSchedulerPageRequest{
		ConfigurationID: "config-789",
		Name:            "Sales Meeting",
		Slug:            "sales-meeting",
		CustomDomain:    "meetings.sales.example.com",
	}

	if req.ConfigurationID != "config-789" {
		t.Errorf("CreateSchedulerPageRequest.ConfigurationID = %q, want %q", req.ConfigurationID, "config-789")
	}
	if req.Name != "Sales Meeting" {
		t.Errorf("CreateSchedulerPageRequest.Name = %q, want %q", req.Name, "Sales Meeting")
	}
}

// =============================================================================
// UpdateSchedulerPageRequest Tests
// =============================================================================

func TestUpdateSchedulerPageRequest_Creation(t *testing.T) {
	name := "Updated Page Name"
	slug := "updated-slug"
	customDomain := "new.example.com"

	req := UpdateSchedulerPageRequest{
		Name:         &name,
		Slug:         &slug,
		CustomDomain: &customDomain,
	}

	if req.Name == nil || *req.Name != "Updated Page Name" {
		t.Errorf("UpdateSchedulerPageRequest.Name = %v, want %q", req.Name, "Updated Page Name")
	}
	if req.Slug == nil || *req.Slug != "updated-slug" {
		t.Errorf("UpdateSchedulerPageRequest.Slug = %v, want %q", req.Slug, "updated-slug")
	}
}

// =============================================================================
// CreateSchedulerConfigurationRequest Tests
// =============================================================================

func TestCreateSchedulerConfigurationRequest_Creation(t *testing.T) {
	req := CreateSchedulerConfigurationRequest{
		Name: "New Configuration",
		Slug: "new-config",
		Participants: []ConfigurationParticipant{
			{Email: "host@example.com", IsOrganizer: true},
		},
		Availability: AvailabilityRules{
			DurationMinutes: 60,
		},
		EventBooking: EventBooking{
			Title: "Meeting",
		},
		Scheduler: SchedulerSettings{
			AvailableDaysInFuture: 14,
		},
	}

	if req.Name != "New Configuration" {
		t.Errorf("CreateSchedulerConfigurationRequest.Name = %q, want %q", req.Name, "New Configuration")
	}
	if len(req.Participants) != 1 {
		t.Errorf("CreateSchedulerConfigurationRequest.Participants length = %d, want 1", len(req.Participants))
	}
}

// =============================================================================
// UpdateSchedulerConfigurationRequest Tests
// =============================================================================

func TestUpdateSchedulerConfigurationRequest_Creation(t *testing.T) {
	name := "Updated Configuration"
	requiresAuth := false

	req := UpdateSchedulerConfigurationRequest{
		Name:                &name,
		RequiresSessionAuth: &requiresAuth,
		Availability: &AvailabilityRules{
			DurationMinutes: 45,
		},
	}

	if req.Name == nil || *req.Name != "Updated Configuration" {
		t.Errorf("UpdateSchedulerConfigurationRequest.Name = %v, want %q", req.Name, "Updated Configuration")
	}
	if req.RequiresSessionAuth == nil || *req.RequiresSessionAuth {
		t.Error("UpdateSchedulerConfigurationRequest.RequiresSessionAuth should be false")
	}
	if req.Availability == nil {
		t.Fatal("UpdateSchedulerConfigurationRequest.Availability should not be nil")
	}
	if req.Availability.DurationMinutes != 45 {
		t.Errorf("AvailabilityRules.DurationMinutes = %d, want 45", req.Availability.DurationMinutes)
	}
}
