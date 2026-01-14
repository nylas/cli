package nylas

// calendarResponse represents an API calendar response.
type calendarResponse struct {
	ID          string `json:"id"`
	GrantID     string `json:"grant_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Location    string `json:"location"`
	Timezone    string `json:"timezone"`
	ReadOnly    bool   `json:"read_only"`
	IsPrimary   bool   `json:"is_primary"`
	IsOwner     bool   `json:"is_owner"`
	HexColor    string `json:"hex_color"`
	Object      string `json:"object"`
}

// eventResponse represents an API event response.
type eventResponse struct {
	ID          string `json:"id"`
	GrantID     string `json:"grant_id"`
	CalendarID  string `json:"calendar_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Location    string `json:"location"`
	When        struct {
		StartTime     int64  `json:"start_time,omitempty"`
		EndTime       int64  `json:"end_time,omitempty"`
		StartTimezone string `json:"start_timezone,omitempty"`
		EndTimezone   string `json:"end_timezone,omitempty"`
		Date          string `json:"date,omitempty"`
		EndDate       string `json:"end_date,omitempty"`
		StartDate     string `json:"start_date,omitempty"`
		Object        string `json:"object,omitempty"`
	} `json:"when"`
	Participants []struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Status  string `json:"status"`
		Comment string `json:"comment"`
	} `json:"participants"`
	Organizer *struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Status  string `json:"status"`
		Comment string `json:"comment"`
	} `json:"organizer"`
	Status       string   `json:"status"`
	Busy         bool     `json:"busy"`
	ReadOnly     bool     `json:"read_only"`
	Visibility   string   `json:"visibility"`
	Recurrence   []string `json:"recurrence"`
	Conferencing *struct {
		Provider string `json:"provider"`
		Details  *struct {
			URL         string   `json:"url"`
			MeetingCode string   `json:"meeting_code"`
			Password    string   `json:"password"`
			Phone       []string `json:"phone"`
		} `json:"details"`
	} `json:"conferencing"`
	Reminders *struct {
		UseDefault bool `json:"use_default"`
		Overrides  []struct {
			ReminderMinutes int    `json:"reminder_minutes"`
			ReminderMethod  string `json:"reminder_method"`
		} `json:"overrides"`
	} `json:"reminders"`
	MasterEventID string `json:"master_event_id"`
	ICalUID       string `json:"ical_uid"`
	HtmlLink      string `json:"html_link"`
	CreatedAt     int64  `json:"created_at"`
	UpdatedAt     int64  `json:"updated_at"`
	Object        string `json:"object"`
}

// GetCalendars retrieves all calendars for a grant.
