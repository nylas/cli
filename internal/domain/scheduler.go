package domain

import (
	"errors"
	"fmt"
	"time"
)

// SchedulerConfiguration represents a scheduling configuration (meeting type)
type SchedulerConfiguration struct {
	ID                  string                     `json:"id,omitempty"`
	Name                string                     `json:"name"`
	Slug                string                     `json:"slug,omitempty"`
	RequiresSessionAuth bool                       `json:"requires_session_auth,omitempty"`
	Participants        []ConfigurationParticipant `json:"participants"`
	Availability        AvailabilityRules          `json:"availability"`
	EventBooking        EventBooking               `json:"event_booking"`
	Scheduler           SchedulerSettings          `json:"scheduler"`
	AppearanceSettings  *AppearanceSettings        `json:"appearance,omitempty"`
	CreatedAt           *time.Time                 `json:"created_at,omitempty"`
	ModifiedAt          *time.Time                 `json:"modified_at,omitempty"`
}

// ConfigurationParticipant represents a participant in a scheduler configuration
type ConfigurationParticipant struct {
	Email        string                    `json:"email"`
	Name         string                    `json:"name,omitempty"`
	IsOrganizer  bool                      `json:"is_organizer,omitempty"`
	Availability ConfigurationAvailability `json:"availability,omitempty"`
	Booking      *ParticipantBooking       `json:"booking,omitempty"`
}

// ConfigurationAvailability holds participant availability settings
type ConfigurationAvailability struct {
	CalendarIDs []string    `json:"calendar_ids,omitempty"`
	OpenHours   []OpenHours `json:"open_hours,omitempty"`
}

// ParticipantBooking represents booking calendar settings for a participant
type ParticipantBooking struct {
	CalendarID string `json:"calendar_id"`
}

// OpenHours represents available hours
type OpenHours struct {
	Days     []int    `json:"days"`  // [1, 2, 3, 4, 5] for Monday-Friday (0=Sunday, 1=Monday, ..., 6=Saturday)
	Start    string   `json:"start"` // "09:00"
	End      string   `json:"end"`   // "17:00"
	Timezone string   `json:"timezone,omitempty"`
	ExDates  []string `json:"exdates,omitempty"` // Excluded dates
}

// AvailabilityRules defines availability rules for scheduling
type AvailabilityRules struct {
	DurationMinutes    int                 `json:"duration_minutes"`
	IntervalMinutes    int                 `json:"interval_minutes,omitempty"`
	RoundTo            int                 `json:"round_to,omitempty"`
	AvailabilityMethod string              `json:"availability_method,omitempty"` // "max-fairness", "max-availability"
	Buffer             *AvailabilityBuffer `json:"buffer,omitempty"`
}

// AvailabilityBuffer represents buffer time before/after meetings
type AvailabilityBuffer struct {
	Before int `json:"before,omitempty"`
	After  int `json:"after,omitempty"`
}

// EventBooking represents event booking settings
type EventBooking struct {
	Title           string                `json:"title"`
	Description     string                `json:"description,omitempty"`
	Location        string                `json:"location,omitempty"`
	Timezone        string                `json:"timezone,omitempty"`
	BookingType     string                `json:"booking_type,omitempty"` // "booking", "organizer-confirmation"
	Conferencing    *ConferencingSettings `json:"conferencing,omitempty"`
	DisableEmails   bool                  `json:"disable_emails,omitempty"`
	ReminderMinutes []int                 `json:"reminder_minutes,omitempty"`
	Metadata        map[string]string     `json:"metadata,omitempty"`
}

// ConferencingSettings represents video conferencing settings
type ConferencingSettings struct {
	Provider   string               `json:"provider"` // "Google Meet", "Zoom", "Microsoft Teams"
	Autocreate bool                 `json:"autocreate,omitempty"`
	Details    *ConferencingDetails `json:"details,omitempty"` // Reuses ConferencingDetails from calendar.go
}

// SchedulerSettings represents scheduler UI settings
type SchedulerSettings struct {
	AvailableDaysInFuture int            `json:"available_days_in_future,omitempty"`
	MinBookingNotice      int            `json:"min_booking_notice,omitempty"`
	MinCancellationNotice int            `json:"min_cancellation_notice,omitempty"`
	ConfirmationMethod    string         `json:"confirmation_method,omitempty"` // "automatic", "manual"
	ReschedulingURL       string         `json:"rescheduling_url,omitempty"`
	CancellationURL       string         `json:"cancellation_url,omitempty"`
	AdditionalFields      map[string]any `json:"additional_fields,omitempty"`
	CancellationPolicy    string         `json:"cancellation_policy,omitempty"`
}

// AppearanceSettings represents UI customization settings
type AppearanceSettings struct {
	CompanyName     string `json:"company_name,omitempty"`
	Logo            string `json:"logo,omitempty"`
	Color           string `json:"color,omitempty"`
	SubmitText      string `json:"submit_text,omitempty"`
	ThankYouMessage string `json:"thank_you_message,omitempty"`
}

// CreateSchedulerConfigurationRequest represents a request to create a scheduler configuration
type CreateSchedulerConfigurationRequest struct {
	Name                string                     `json:"name"`
	Slug                string                     `json:"slug,omitempty"`
	RequiresSessionAuth bool                       `json:"requires_session_auth,omitempty"`
	Participants        []ConfigurationParticipant `json:"participants"`
	Availability        AvailabilityRules          `json:"availability"`
	EventBooking        EventBooking               `json:"event_booking"`
	Scheduler           SchedulerSettings          `json:"scheduler"`
	AppearanceSettings  *AppearanceSettings        `json:"appearance,omitempty"`
}

// UpdateSchedulerConfigurationRequest represents a request to update a scheduler configuration
type UpdateSchedulerConfigurationRequest struct {
	Name                *string                    `json:"name,omitempty"`
	Slug                *string                    `json:"slug,omitempty"`
	RequiresSessionAuth *bool                      `json:"requires_session_auth,omitempty"`
	Participants        []ConfigurationParticipant `json:"participants,omitempty"`
	Availability        *AvailabilityRules         `json:"availability,omitempty"`
	EventBooking        *EventBooking              `json:"event_booking,omitempty"`
	Scheduler           *SchedulerSettings         `json:"scheduler,omitempty"`
	AppearanceSettings  *AppearanceSettings        `json:"appearance,omitempty"`
}

// SchedulerSession represents a scheduling session
type SchedulerSession struct {
	SessionID       string    `json:"session_id"`
	ConfigurationID string    `json:"configuration_id"`
	BookingURL      string    `json:"booking_url,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
}

// CreateSchedulerSessionRequest represents a request to create a scheduler session
type CreateSchedulerSessionRequest struct {
	ConfigurationID  string         `json:"configuration_id"`
	TimeToLive       int            `json:"time_to_live,omitempty"` // Session TTL in minutes (max 30)
	Slug             string         `json:"slug,omitempty"`
	AdditionalFields map[string]any `json:"additional_fields,omitempty"`
}

// Validate checks the fields the Nylas v3 spec requires on session creation:
// the configuration is identified by configuration_id OR slug, and
// time_to_live is capped at 30 minutes (0 means unset; the API defaults to 5).
// It is the single source of truth shared by the adapter and RPC entrypoints.
func (r *CreateSchedulerSessionRequest) Validate() error {
	if r == nil {
		return errors.New("session request is required")
	}
	if r.ConfigurationID == "" && r.Slug == "" {
		return errors.New("configuration_id or slug is required")
	}
	if r.TimeToLive < 0 || r.TimeToLive > 30 {
		return fmt.Errorf("time_to_live must be between 0 and 30 minutes (0 uses the server default), got %d", r.TimeToLive)
	}
	return nil
}

// Booking represents a scheduled booking
type Booking struct {
	BookingID        string               `json:"booking_id"`
	EventID          string               `json:"event_id,omitempty"`
	Title            string               `json:"title"`
	Organizer        Participant          `json:"organizer"`              // Reuses Participant from calendar.go
	Participants     []Participant        `json:"participants,omitempty"` // Reuses Participant from calendar.go
	StartTime        time.Time            `json:"start_time"`
	EndTime          time.Time            `json:"end_time"`
	Status           string               `json:"status"` // "confirmed", "cancelled", "pending"
	Description      string               `json:"description,omitempty"`
	Location         string               `json:"location,omitempty"`
	Timezone         string               `json:"timezone,omitempty"`
	Conferencing     *ConferencingDetails `json:"conferencing,omitempty"` // Reuses ConferencingDetails from calendar.go
	AdditionalFields map[string]any       `json:"additional_fields,omitempty"`
	Metadata         map[string]string    `json:"metadata,omitempty"`
	CreatedAt        time.Time            `json:"created_at,omitempty"`
	UpdatedAt        time.Time            `json:"updated_at,omitempty"`
}

// ConfirmBookingRequest represents a request to confirm or cancel a pending
// booking. Per the Nylas v3 spec, salt and status are required; the salt is
// extracted from the booking reference in the organizer confirmation link.
type ConfirmBookingRequest struct {
	Salt               string `json:"salt"`
	Status             string `json:"status"` // "confirmed" or "cancelled"
	CancellationReason string `json:"cancellation_reason,omitempty"`
}

// Validate checks the fields the Nylas v3 spec requires on a confirm/cancel
// request: a salt and a "confirmed" or "cancelled" status. It is the single
// source of truth shared by the adapter and RPC entrypoints.
func (r *ConfirmBookingRequest) Validate() error {
	if r.Salt == "" {
		return errors.New("salt is required")
	}
	if r.Status != "confirmed" && r.Status != "cancelled" {
		return fmt.Errorf("status must be 'confirmed' or 'cancelled', got %q", r.Status)
	}
	return nil
}

// RescheduleBookingRequest is the Nylas v3 reschedule payload. Per the spec
// (booking_update.yaml) it carries only the new start and end times; timezone
// and reason are not part of the request model.
type RescheduleBookingRequest struct {
	StartTime int64 `json:"start_time"` // Unix timestamp for new start time
	EndTime   int64 `json:"end_time"`   // Unix timestamp for new end time
}
