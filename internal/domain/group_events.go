package domain

// GroupEventParticipant is a participant included in a Scheduler group event.
type GroupEventParticipant struct {
	Name        string `json:"name,omitempty"`
	Email       string `json:"email"`
	IsOrganizer bool   `json:"is_organizer,omitempty"`
}

// GroupEventWhen is the time span (start/end + timezones) of a group event.
type GroupEventWhen struct {
	StartTime     int64  `json:"start_time,omitempty"`
	EndTime       int64  `json:"end_time,omitempty"`
	StartTimezone string `json:"start_timezone,omitempty"`
	EndTimezone   string `json:"end_timezone,omitempty"`
}

// GroupEvent is a Nylas Scheduler group event. Group events live under a
// Scheduler Configuration and let multiple participants book a shared slot.
type GroupEvent struct {
	ID           string                  `json:"id,omitempty"`
	CalendarID   string                  `json:"calendar_id,omitempty"`
	Title        string                  `json:"title,omitempty"`
	Description  string                  `json:"description,omitempty"`
	Location     string                  `json:"location,omitempty"`
	Capacity     int                     `json:"capacity,omitempty"`
	Participants []GroupEventParticipant `json:"participants,omitempty"`
	When         *GroupEventWhen         `json:"when,omitempty"`
}

// CreateGroupEventRequest creates a group event. calendar_id, capacity, title,
// and when are required. Participants uses omitempty: when none are supplied the
// field is omitted (not sent as null) so the API falls back to the organizer,
// per the documented "if you don't specify a participant, Nylas uses the event
// organizer instead" behavior.
type CreateGroupEventRequest struct {
	CalendarID   string                  `json:"calendar_id"`
	Title        string                  `json:"title"`
	Capacity     int                     `json:"capacity"`
	Description  string                  `json:"description,omitempty"`
	Location     string                  `json:"location,omitempty"`
	Participants []GroupEventParticipant `json:"participants,omitempty"`
	When         *GroupEventWhen         `json:"when"`
}

// UpdateGroupEventRequest updates a group event. All fields are optional; only
// non-zero fields are sent.
type UpdateGroupEventRequest struct {
	CalendarID   string                  `json:"calendar_id,omitempty"`
	Title        string                  `json:"title,omitempty"`
	Capacity     int                     `json:"capacity,omitempty"`
	Description  string                  `json:"description,omitempty"`
	Location     string                  `json:"location,omitempty"`
	Participants []GroupEventParticipant `json:"participants,omitempty"`
	When         *GroupEventWhen         `json:"when,omitempty"`
}

// ImportGroupEventItem describes one provider event to import as a group event.
type ImportGroupEventItem struct {
	CalendarID   string                      `json:"calendar_id"`
	EventID      string                      `json:"event_id"`
	Capacity     int                         `json:"capacity,omitempty"`
	Participants []GroupEventParticipant     `json:"participants,omitempty"`
	Exceptions   []ImportGroupEventException `json:"exceptions,omitempty"`
}

// ImportGroupEventException overrides capacity for a specific instance when
// importing a recurring event.
type ImportGroupEventException struct {
	EventID  string `json:"event_id,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}
