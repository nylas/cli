package domain

import "time"

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
	TimeToLive       int            `json:"ttl,omitempty"` // Session TTL in minutes
	Slug             string         `json:"slug,omitempty"`
	AdditionalFields map[string]any `json:"additional_fields,omitempty"`
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

// ConfirmBookingRequest represents a request to confirm a booking
type ConfirmBookingRequest struct {
	Status         string         `json:"status"` // "confirmed" or "cancelled"
	Reason         string         `json:"reason,omitempty"`
	AdditionalData map[string]any `json:"additional_data,omitempty"`
}

// RescheduleBookingRequest represents a request to reschedule a booking
type RescheduleBookingRequest struct {
	StartTime int64  `json:"start_time"`         // Unix timestamp for new start time
	EndTime   int64  `json:"end_time"`           // Unix timestamp for new end time
	Timezone  string `json:"timezone,omitempty"` // Timezone for the booking (e.g., "America/New_York")
	Reason    string `json:"reason,omitempty"`   // Reason for rescheduling
}

// SchedulerPage represents a hosted scheduling page
type SchedulerPage struct {
	ID              string    `json:"id,omitempty"`
	ConfigurationID string    `json:"configuration_id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	URL             string    `json:"url,omitempty"`
	CustomDomain    string    `json:"custom_domain,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	ModifiedAt      time.Time `json:"modified_at,omitempty"`
}

// CreateSchedulerPageRequest represents a request to create a scheduler page
type CreateSchedulerPageRequest struct {
	ConfigurationID string `json:"configuration_id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	CustomDomain    string `json:"custom_domain,omitempty"`
}

// UpdateSchedulerPageRequest represents a request to update a scheduler page
type UpdateSchedulerPageRequest struct {
	Name         *string `json:"name,omitempty"`
	Slug         *string `json:"slug,omitempty"`
	CustomDomain *string `json:"custom_domain,omitempty"`
}
