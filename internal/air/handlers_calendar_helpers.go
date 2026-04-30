package air

import (
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// eventToResponse converts a domain event to an API response.
func eventToResponse(e domain.Event) EventResponse {
	resp := EventResponse{
		ID:          e.ID,
		CalendarID:  e.CalendarID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartTime:   e.When.StartTime,
		EndTime:     e.When.EndTime,
		Timezone:    e.When.StartTimezone,
		IsAllDay:    e.When.IsAllDay(),
		Status:      e.Status,
		Busy:        e.Busy,
		HtmlLink:    e.HtmlLink,
	}

	// Handle all-day events. Skip the override when parsing fails — the
	// upstream-provided StartTime/EndTime is more useful than a year-1 Unix
	// timestamp would be.
	if resp.IsAllDay {
		switch {
		case e.When.Date != "":
			if t, err := time.Parse("2006-01-02", e.When.Date); err == nil {
				resp.StartTime = t.Unix()
				resp.EndTime = t.Add(24 * time.Hour).Unix()
			}
		case e.When.StartDate != "":
			if st, err := time.Parse("2006-01-02", e.When.StartDate); err == nil {
				resp.StartTime = st.Unix()
			}
			if e.When.EndDate != "" {
				if et, err := time.Parse("2006-01-02", e.When.EndDate); err == nil {
					resp.EndTime = et.Unix()
				}
			}
		}
	}

	// Convert participants
	for _, p := range e.Participants {
		resp.Participants = append(resp.Participants, EventParticipantResponse{
			Name:   p.Name,
			Email:  p.Email,
			Status: p.Status,
		})
	}

	// Convert conferencing
	if e.Conferencing != nil && e.Conferencing.Details != nil {
		resp.Conferencing = &ConferencingResponse{
			Provider: e.Conferencing.Provider,
			URL:      e.Conferencing.Details.URL,
		}
	}

	return resp
}

// cachedEventToResponse converts a cached event to response format.
func cachedEventToResponse(e *cache.CachedEvent) EventResponse {
	return EventResponse{
		ID:          e.ID,
		CalendarID:  e.CalendarID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartTime:   e.StartTime.Unix(),
		EndTime:     e.EndTime.Unix(),
		IsAllDay:    e.AllDay,
		Status:      e.Status,
		Busy:        e.Busy,
	}
}

// demoEvents returns demo event data.
func demoEvents() []EventResponse {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return []EventResponse{
		{
			ID:          "demo-event-001",
			CalendarID:  "primary",
			Title:       "Team Standup",
			Description: "Daily team sync",
			Location:    "",
			StartTime:   today.Add(9 * time.Hour).Unix(),
			EndTime:     today.Add(9*time.Hour + 30*time.Minute).Unix(),
			Timezone:    "America/New_York",
			IsAllDay:    false,
			Status:      "confirmed",
			Busy:        true,
			Participants: []EventParticipantResponse{
				{Name: "Sarah Chen", Email: "sarah@example.com", Status: "yes"},
				{Name: "Alex Johnson", Email: "alex@example.com", Status: "yes"},
			},
			Conferencing: &ConferencingResponse{
				Provider: "Google Meet",
				URL:      "https://meet.google.com/abc-defg-hij",
			},
		},
		{
			ID:          "demo-event-002",
			CalendarID:  "work",
			Title:       "Product Review",
			Description: "Weekly product roadmap review with stakeholders",
			Location:    "Conference Room A",
			StartTime:   today.Add(14 * time.Hour).Unix(),
			EndTime:     today.Add(15 * time.Hour).Unix(),
			Timezone:    "America/New_York",
			IsAllDay:    false,
			Status:      "confirmed",
			Busy:        true,
			Participants: []EventParticipantResponse{
				{Name: "Product Team", Email: "product@example.com", Status: "yes"},
			},
		},
		{
			ID:          "demo-event-003",
			CalendarID:  "primary",
			Title:       "Lunch with Client",
			Description: "Discuss Q1 partnership opportunities",
			Location:    "Cafe Milano",
			StartTime:   today.Add(12 * time.Hour).Unix(),
			EndTime:     today.Add(13 * time.Hour).Unix(),
			Timezone:    "America/New_York",
			IsAllDay:    false,
			Status:      "confirmed",
			Busy:        true,
		},
		{
			ID:          "demo-event-004",
			CalendarID:  "primary",
			Title:       "Focus Time",
			Description: "Deep work - no meetings",
			StartTime:   today.Add(15 * time.Hour).Unix(),
			EndTime:     today.Add(17 * time.Hour).Unix(),
			Timezone:    "America/New_York",
			IsAllDay:    false,
			Status:      "confirmed",
			Busy:        true,
		},
		{
			ID:         "demo-event-005",
			CalendarID: "holidays",
			Title:      "Christmas Day",
			StartTime:  time.Date(now.Year(), 12, 25, 0, 0, 0, 0, now.Location()).Unix(),
			EndTime:    time.Date(now.Year(), 12, 26, 0, 0, 0, 0, now.Location()).Unix(),
			Timezone:   "America/New_York",
			IsAllDay:   true,
			Status:     "confirmed",
			Busy:       false,
		},
	}
}
