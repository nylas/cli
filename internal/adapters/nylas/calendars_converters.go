package nylas

import (
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

func convertCalendars(cals []calendarResponse) []domain.Calendar {
	return util.Map(cals, convertCalendar)
}

// convertCalendar converts an API calendar response to domain model.
func convertCalendar(c calendarResponse) domain.Calendar {
	return domain.Calendar{
		ID:          c.ID,
		GrantID:     c.GrantID,
		Name:        c.Name,
		Description: c.Description,
		Location:    c.Location,
		Timezone:    c.Timezone,
		ReadOnly:    c.ReadOnly,
		IsPrimary:   c.IsPrimary,
		IsOwner:     c.IsOwner,
		HexColor:    c.HexColor,
		Object:      c.Object,
	}
}

// convertEvents converts API event responses to domain models.
func convertEvents(events []eventResponse) []domain.Event {
	return util.Map(events, convertEvent)
}

// convertEvent converts an API event response to domain model.
func convertEvent(e eventResponse) domain.Event {
	participants := util.Map(e.Participants, func(p struct {
		Name    string `json:"name"`
		Email   string `json:"email"`
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}) domain.Participant {
		return domain.Participant{
			Person:  domain.Person{Name: p.Name, Email: p.Email},
			Status:  p.Status,
			Comment: p.Comment,
		}
	})

	var organizer *domain.Participant
	if e.Organizer != nil {
		organizer = &domain.Participant{
			Person:  domain.Person{Name: e.Organizer.Name, Email: e.Organizer.Email},
			Status:  e.Organizer.Status,
			Comment: e.Organizer.Comment,
		}
	}

	var conferencing *domain.Conferencing
	if e.Conferencing != nil {
		conferencing = &domain.Conferencing{
			Provider: e.Conferencing.Provider,
		}
		if e.Conferencing.Details != nil {
			conferencing.Details = &domain.ConferencingDetails{
				URL:         e.Conferencing.Details.URL,
				MeetingCode: e.Conferencing.Details.MeetingCode,
				Password:    e.Conferencing.Details.Password,
				Phone:       e.Conferencing.Details.Phone,
			}
		}
	}

	var reminders *domain.Reminders
	if e.Reminders != nil {
		overrides := util.Map(e.Reminders.Overrides, func(o struct {
			ReminderMinutes int    `json:"reminder_minutes"`
			ReminderMethod  string `json:"reminder_method"`
		}) domain.Reminder {
			return domain.Reminder{
				ReminderMinutes: o.ReminderMinutes,
				ReminderMethod:  o.ReminderMethod,
			}
		})
		reminders = &domain.Reminders{
			UseDefault: e.Reminders.UseDefault,
			Overrides:  overrides,
		}
	}

	return domain.Event{
		ID:          e.ID,
		GrantID:     e.GrantID,
		CalendarID:  e.CalendarID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		When: domain.EventWhen{
			StartTime:     e.When.StartTime,
			EndTime:       e.When.EndTime,
			StartTimezone: e.When.StartTimezone,
			EndTimezone:   e.When.EndTimezone,
			Date:          e.When.Date,
			EndDate:       e.When.EndDate,
			StartDate:     e.When.StartDate,
			Object:        e.When.Object,
		},
		Participants:  participants,
		Organizer:     organizer,
		Status:        e.Status,
		Busy:          e.Busy,
		ReadOnly:      e.ReadOnly,
		Visibility:    e.Visibility,
		Recurrence:    e.Recurrence,
		Conferencing:  conferencing,
		Reminders:     reminders,
		MasterEventID: e.MasterEventID,
		ICalUID:       e.ICalUID,
		HtmlLink:      e.HtmlLink,
		CreatedAt:     time.Unix(e.CreatedAt, 0),
		UpdatedAt:     time.Unix(e.UpdatedAt, 0),
		Object:        e.Object,
	}
}
