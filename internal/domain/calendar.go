package domain

import "time"

// Calendar represents a calendar from Nylas.
type Calendar struct {
	ID          string `json:"id"`
	GrantID     string `json:"grant_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
	ReadOnly    bool   `json:"read_only"`
	IsPrimary   bool   `json:"is_primary,omitempty"`
	IsOwner     bool   `json:"is_owner,omitempty"`
	HexColor    string `json:"hex_color,omitempty"`
	Object      string `json:"object,omitempty"`
}

// Event represents a calendar event from Nylas.
type Event struct {
	ID            string            `json:"id"`
	GrantID       string            `json:"grant_id"`
	CalendarID    string            `json:"calendar_id"`
	Title         string            `json:"title"`
	Description   string            `json:"description,omitempty"`
	Location      string            `json:"location,omitempty"`
	When          EventWhen         `json:"when"`
	Participants  []Participant     `json:"participants,omitempty"`
	Organizer     *Participant      `json:"organizer,omitempty"`
	Status        string            `json:"status,omitempty"` // confirmed, cancelled, tentative
	Busy          bool              `json:"busy"`
	ReadOnly      bool              `json:"read_only"`
	Visibility    string            `json:"visibility,omitempty"` // public, private
	Recurrence    []string          `json:"recurrence,omitempty"`
	Conferencing  *Conferencing     `json:"conferencing,omitempty"`
	Reminders     *Reminders        `json:"reminders,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	MasterEventID string            `json:"master_event_id,omitempty"`
	ICalUID       string            `json:"ical_uid,omitempty"`
	HtmlLink      string            `json:"html_link,omitempty"`
	CreatedAt     time.Time         `json:"created_at,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at,omitempty"`
	Object        string            `json:"object,omitempty"`
}

// EventWhen represents when an event occurs.
type EventWhen struct {
	// For timespan events
	StartTime     int64  `json:"start_time,omitempty"`
	EndTime       int64  `json:"end_time,omitempty"`
	StartTimezone string `json:"start_timezone,omitempty"`
	EndTimezone   string `json:"end_timezone,omitempty"`

	// For date events (all-day)
	Date    string `json:"date,omitempty"`
	EndDate string `json:"end_date,omitempty"`

	// For datespan events (multi-day all-day)
	StartDate string `json:"start_date,omitempty"`
	// EndDate is shared with date events

	Object string `json:"object,omitempty"` // timespan, date, datespan
}

// StartDateTime returns the start time as a time.Time.
// If StartTimezone is specified, the time is returned in that timezone.
// Returns zero time if no valid start time is set or if date parsing fails.
// This is a convenience getter for deserialized API data.
func (w EventWhen) StartDateTime() time.Time {
	if w.StartTime > 0 {
		t := time.Unix(w.StartTime, 0)
		// If a timezone is specified, convert to that timezone
		if w.StartTimezone != "" {
			loc, err := time.LoadLocation(w.StartTimezone)
			if err == nil {
				t = t.In(loc)
			}
		}
		return t
	}
	if w.Date != "" {
		// Date strings from API are pre-validated; ignore parse errors
		t, _ := time.Parse("2006-01-02", w.Date)
		return t
	}
	if w.StartDate != "" {
		// Date strings from API are pre-validated; ignore parse errors
		t, _ := time.Parse("2006-01-02", w.StartDate)
		return t
	}
	return time.Time{}
}

// EndDateTime returns the end time as a time.Time.
// If EndTimezone is specified, the time is returned in that timezone.
// Returns zero time if no valid end time is set or if date parsing fails.
// This is a convenience getter for deserialized API data.
func (w EventWhen) EndDateTime() time.Time {
	if w.EndTime > 0 {
		t := time.Unix(w.EndTime, 0)
		// If a timezone is specified, convert to that timezone
		if w.EndTimezone != "" {
			loc, err := time.LoadLocation(w.EndTimezone)
			if err == nil {
				t = t.In(loc)
			}
		}
		return t
	}
	if w.EndDate != "" {
		// Date strings from API are pre-validated; ignore parse errors
		t, _ := time.Parse("2006-01-02", w.EndDate)
		return t
	}
	if w.Date != "" {
		// Date strings from API are pre-validated; ignore parse errors
		t, _ := time.Parse("2006-01-02", w.Date)
		return t
	}
	return time.Time{}
}

// IsAllDay returns true if this is an all-day event.
func (w EventWhen) IsAllDay() bool {
	return w.Object == "date" || w.Object == "datespan" || w.Date != "" || w.StartDate != ""
}

// Validate checks that EventWhen has exactly one valid time specification.
func (w EventWhen) Validate() error {
	hasTimespan := w.StartTime > 0 && w.EndTime > 0
	hasDate := w.Date != ""
	hasDatespan := w.StartDate != "" && w.EndDate != ""

	count := 0
	if hasTimespan {
		count++
	}
	if hasDate {
		count++
	}
	if hasDatespan {
		count++
	}

	if count == 0 {
		return ErrInvalidInput
	}
	if count > 1 {
		return ErrInvalidInput
	}
	return nil
}

// IsTimezoneLocked returns true if the event has timezone locking enabled.
func (e Event) IsTimezoneLocked() bool {
	if e.Metadata == nil {
		return false
	}
	return e.Metadata["timezone_locked"] == "true"
}

// GetLockedTimezone returns the locked timezone for the event, if set.
// Returns empty string if timezone is not locked.
func (e Event) GetLockedTimezone() string {
	if !e.IsTimezoneLocked() {
		return ""
	}
	// Return the event's start timezone if locked
	if !e.When.IsAllDay() && e.When.StartTimezone != "" {
		return e.When.StartTimezone
	}
	return ""
}

// Participant represents an event participant.
// Embeds Person for name/email and adds RSVP status.
type Participant struct {
	Person
	Status  string `json:"status,omitempty"` // yes, no, maybe, noreply
	Comment string `json:"comment,omitempty"`
}

// Conferencing represents video conferencing details.
type Conferencing struct {
	Provider string               `json:"provider,omitempty"` // Google Meet, Zoom, etc.
	Details  *ConferencingDetails `json:"details,omitempty"`
}

// ConferencingDetails contains conferencing URLs and info.
type ConferencingDetails struct {
	URL         string   `json:"url,omitempty"`
	MeetingCode string   `json:"meeting_code,omitempty"`
	Password    string   `json:"password,omitempty"`
	Phone       []string `json:"phone,omitempty"`
}

// Reminders represents event reminders.
type Reminders struct {
	UseDefault bool       `json:"use_default"`
	Overrides  []Reminder `json:"overrides,omitempty"`
}

// Reminder represents a single reminder.
type Reminder struct {
	ReminderMinutes int    `json:"reminder_minutes"`
	ReminderMethod  string `json:"reminder_method,omitempty"` // email, popup
}

// EventQueryParams for filtering events.
type EventQueryParams struct {
	Limit           int    `json:"limit,omitempty"`
	PageToken       string `json:"page_token,omitempty"`
	CalendarID      string `json:"calendar_id,omitempty"`
	Title           string `json:"title,omitempty"`
	Location        string `json:"location,omitempty"`
	ShowCancelled   bool   `json:"show_cancelled,omitempty"`
	Start           int64  `json:"start,omitempty"` // Unix timestamp
	End             int64  `json:"end,omitempty"`   // Unix timestamp
	MetadataPair    string `json:"metadata_pair,omitempty"`
	Busy            *bool  `json:"busy,omitempty"`
	OrderBy         string `json:"order_by,omitempty"` // start, end
	ExpandRecurring bool   `json:"expand_recurring,omitempty"`
}

// CreateEventRequest for creating a new event.
type CreateEventRequest struct {
	Title        string            `json:"title"`
	Description  string            `json:"description,omitempty"`
	Location     string            `json:"location,omitempty"`
	When         EventWhen         `json:"when"`
	Participants []Participant     `json:"participants,omitempty"`
	Busy         bool              `json:"busy"`
	Visibility   string            `json:"visibility,omitempty"`
	Recurrence   []string          `json:"recurrence,omitempty"`
	Conferencing *Conferencing     `json:"conferencing,omitempty"`
	Reminders    *Reminders        `json:"reminders,omitempty"`
	CalendarID   string            `json:"calendar_id,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// UpdateEventRequest for updating an event.
type UpdateEventRequest struct {
	Title        *string           `json:"title,omitempty"`
	Description  *string           `json:"description,omitempty"`
	Location     *string           `json:"location,omitempty"`
	When         *EventWhen        `json:"when,omitempty"`
	Participants []Participant     `json:"participants,omitempty"`
	Busy         *bool             `json:"busy,omitempty"`
	Visibility   *string           `json:"visibility,omitempty"`
	Recurrence   []string          `json:"recurrence,omitempty"`
	Conferencing *Conferencing     `json:"conferencing,omitempty"`
	Reminders    *Reminders        `json:"reminders,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// CalendarListResponse represents a paginated calendar list response.
type CalendarListResponse struct {
	Data       []Calendar `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// EventListResponse represents a paginated event list response.
type EventListResponse struct {
	Data       []Event    `json:"data"`
	Pagination Pagination `json:"pagination,omitempty"`
}

// FreeBusyRequest for checking availability.
type FreeBusyRequest struct {
	StartTime int64    `json:"start_time"` // Unix timestamp
	EndTime   int64    `json:"end_time"`   // Unix timestamp
	Emails    []string `json:"emails"`
}

// FreeBusyResponse represents availability data.
type FreeBusyResponse struct {
	Data []FreeBusyCalendar `json:"data"`
}

// FreeBusyCalendar represents a calendar's availability.
type FreeBusyCalendar struct {
	Email     string     `json:"email"`
	TimeSlots []TimeSlot `json:"time_slots,omitempty"`
	Object    string     `json:"object,omitempty"`
}

// TimeSlot represents a busy time slot.
type TimeSlot struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status,omitempty"` // busy, free
	Object    string `json:"object,omitempty"`
}

// AvailabilityRequest for finding available meeting times.
type AvailabilityRequest struct {
	StartTime       int64                     `json:"start_time"`
	EndTime         int64                     `json:"end_time"`
	DurationMinutes int                       `json:"duration_minutes"`
	Participants    []AvailabilityParticipant `json:"participants"`
	IntervalMinutes int                       `json:"interval_minutes,omitempty"`
	RoundTo         int                       `json:"round_to,omitempty"`
}

// AvailabilityParticipant represents a participant in availability check.
type AvailabilityParticipant struct {
	Email       string   `json:"email"`
	CalendarIDs []string `json:"calendar_ids,omitempty"`
}

// AvailabilityResponse contains available time slots.
type AvailabilityResponse struct {
	Data AvailabilityData `json:"data"`
}

// AvailabilityData contains the time slots data from availability API.
type AvailabilityData struct {
	TimeSlots []AvailableSlot `json:"time_slots"`
	Order     []string        `json:"order,omitempty"` // For round-robin scheduling
}

// AvailableSlot represents an available meeting slot.
type AvailableSlot struct {
	StartTime int64    `json:"start_time"`
	EndTime   int64    `json:"end_time"`
	Emails    []string `json:"emails,omitempty"`
}

// CreateCalendarRequest for creating a new calendar.
type CreateCalendarRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Timezone    string `json:"timezone,omitempty"`
}

// UpdateCalendarRequest for updating a calendar.
type UpdateCalendarRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Location    *string `json:"location,omitempty"`
	Timezone    *string `json:"timezone,omitempty"`
	HexColor    *string `json:"hex_color,omitempty"`
}

// SendRSVPRequest for responding to an event invitation.
type SendRSVPRequest struct {
	Status  string `json:"status"` // yes, no, maybe
	Comment string `json:"comment,omitempty"`
}

// VirtualCalendarGrant represents a virtual calendar account/grant.
type VirtualCalendarGrant struct {
	ID          string `json:"id"`
	Provider    string `json:"provider"` // Always "virtual-calendar"
	Email       string `json:"email"`    // Custom identifier
	GrantStatus string `json:"grant_status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// CreateVirtualCalendarGrantRequest for creating a virtual calendar grant.
type CreateVirtualCalendarGrantRequest struct {
	Provider string                       `json:"provider"` // Must be "virtual-calendar"
	Settings VirtualCalendarGrantSettings `json:"settings"`
	Scope    []string                     `json:"scope"` // ["calendar"]
}

// VirtualCalendarGrantSettings for virtual calendar grant creation.
type VirtualCalendarGrantSettings struct {
	Email string `json:"email"` // Custom identifier (not required to be email format)
}

// RecurringEventInfo provides information about a recurring event series.
type RecurringEventInfo struct {
	MasterEventID     string   `json:"master_event_id"`
	RecurrenceRule    []string `json:"recurrence"`
	OriginalStartTime *int64   `json:"original_start_time,omitempty"` // For modified instances
	ExpandRecurring   bool     `json:"expand_recurring,omitempty"`
}

// UpdateRecurringEventRequest for updating recurring event instances.
type UpdateRecurringEventRequest struct {
	UpdateEventRequest
	MasterEventID string `json:"master_event_id,omitempty"` // For instance updates
}
