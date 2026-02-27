package mcp

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// executeListCalendars returns all calendars for the resolved grant.
func (s *Server) executeListCalendars(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendars, err := s.client.GetCalendars(ctx, grantID)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	result := make([]map[string]any, 0, len(calendars))
	for _, cal := range calendars {
		result = append(result, map[string]any{
			"id":          cal.ID,
			"name":        cal.Name,
			"description": cal.Description,
			"timezone":    cal.Timezone,
			"read_only":   cal.ReadOnly,
			"is_primary":  cal.IsPrimary,
			"hex_color":   cal.HexColor,
		})
	}
	return toolSuccess(result)
}

// executeGetCalendar returns a single calendar by ID.
func (s *Server) executeGetCalendar(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendarID := getString(args, "calendar_id", "")
	if calendarID == "" {
		return toolError("calendar_id is required")
	}

	cal, err := s.client.GetCalendar(ctx, grantID, calendarID)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":          cal.ID,
		"name":        cal.Name,
		"description": cal.Description,
		"timezone":    cal.Timezone,
		"read_only":   cal.ReadOnly,
		"is_primary":  cal.IsPrimary,
		"hex_color":   cal.HexColor,
	})
}

// executeCreateCalendar creates a new calendar.
func (s *Server) executeCreateCalendar(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	name := getString(args, "name", "")
	if name == "" {
		return toolError("name is required")
	}

	req := &domain.CreateCalendarRequest{
		Name:        name,
		Description: getString(args, "description", ""),
		Location:    getString(args, "location", ""),
		Timezone:    getString(args, "timezone", ""),
	}

	cal, err := s.client.CreateCalendar(ctx, grantID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":     cal.ID,
		"name":   cal.Name,
		"status": "created",
	})
}

// executeUpdateCalendar updates a calendar's metadata.
func (s *Server) executeUpdateCalendar(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendarID := getString(args, "calendar_id", "")
	if calendarID == "" {
		return toolError("calendar_id is required")
	}

	req := &domain.UpdateCalendarRequest{}
	if v := getString(args, "name", ""); v != "" {
		req.Name = &v
	}
	if v := getString(args, "description", ""); v != "" {
		req.Description = &v
	}
	if v := getString(args, "location", ""); v != "" {
		req.Location = &v
	}
	if v := getString(args, "timezone", ""); v != "" {
		req.Timezone = &v
	}

	cal, err := s.client.UpdateCalendar(ctx, grantID, calendarID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":     cal.ID,
		"name":   cal.Name,
		"status": "updated",
	})
}

// executeDeleteCalendar deletes a calendar.
func (s *Server) executeDeleteCalendar(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendarID := getString(args, "calendar_id", "")
	if calendarID == "" {
		return toolError("calendar_id is required")
	}

	if err := s.client.DeleteCalendar(ctx, grantID, calendarID); err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccessText("Deleted calendar " + calendarID)
}

// executeListEvents returns events for a calendar.
func (s *Server) executeListEvents(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendarID := getString(args, "calendar_id", "primary")

	params := &domain.EventQueryParams{
		Limit: clampLimit(args, "limit", 200),
	}
	if v := getString(args, "title", ""); v != "" {
		params.Title = v
	}
	if v := getInt64(args, "start", 0); v > 0 {
		params.Start = v
	}
	if v := getInt64(args, "end", 0); v > 0 {
		params.End = v
	}
	if b := getBool(args, "expand_recurring"); b != nil {
		params.ExpandRecurring = *b
	}
	if b := getBool(args, "show_cancelled"); b != nil {
		params.ShowCancelled = *b
	}
	if pageToken := getString(args, "page_token", ""); pageToken != "" {
		params.PageToken = pageToken
	}

	resp, err := s.client.GetEventsWithCursor(ctx, grantID, calendarID, params)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	result := make([]map[string]any, 0, len(resp.Data))
	for _, ev := range resp.Data {
		result = append(result, map[string]any{
			"id":                 ev.ID,
			"title":              ev.Title,
			"status":             ev.Status,
			"when":               formatEventWhen(ev.When),
			"location":           ev.Location,
			"busy":               ev.Busy,
			"participants_count": len(ev.Participants),
		})
	}

	return toolSuccess(map[string]any{
		"data":        result,
		"next_cursor": resp.Pagination.NextCursor,
	})
}

// executeGetEvent returns a single event by ID.
func (s *Server) executeGetEvent(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	eventID := getString(args, "event_id", "")
	if eventID == "" {
		return toolError("event_id is required")
	}
	calendarID := getString(args, "calendar_id", "primary")

	event, err := s.client.GetEvent(ctx, grantID, calendarID, eventID)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	participants := make([]map[string]any, 0, len(event.Participants))
	for _, p := range event.Participants {
		participants = append(participants, map[string]any{
			"name":   p.Name,
			"email":  p.Email,
			"status": p.Status,
		})
	}

	result := map[string]any{
		"id":           event.ID,
		"title":        event.Title,
		"description":  event.Description,
		"location":     event.Location,
		"when":         formatEventWhen(event.When),
		"status":       event.Status,
		"busy":         event.Busy,
		"visibility":   event.Visibility,
		"participants": participants,
		"recurrence":   event.Recurrence,
		"html_link":    event.HtmlLink,
		"ical_uid":     event.ICalUID,
	}

	if event.Organizer != nil {
		result["organizer"] = map[string]any{
			"name":  event.Organizer.Name,
			"email": event.Organizer.Email,
		}
	}

	if event.Conferencing != nil {
		conf := map[string]any{
			"provider": event.Conferencing.Provider,
		}
		if event.Conferencing.Details != nil {
			conf["url"] = event.Conferencing.Details.URL
		}
		result["conferencing"] = conf
	}

	if event.Reminders != nil {
		result["reminders"] = event.Reminders
	}

	return toolSuccess(result)
}

// executeCreateEvent creates a new calendar event.
func (s *Server) executeCreateEvent(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	calendarID := getString(args, "calendar_id", "primary")

	title := getString(args, "title", "")
	if title == "" {
		return toolError("title is required")
	}

	req := &domain.CreateEventRequest{
		Title:       title,
		Description: getString(args, "description", ""),
		Location:    getString(args, "location", ""),
	}

	if st := getInt64(args, "start_time", 0); st > 0 {
		et := getInt64(args, "end_time", 0)
		if et == 0 {
			return toolError("end_time is required when start_time is provided")
		}
		if et <= st {
			return toolError("end_time must be after start_time")
		}
		req.When = domain.EventWhen{
			StartTime: st,
			EndTime:   et,
		}
	} else if sd := getString(args, "start_date", ""); sd != "" {
		ed := getString(args, "end_date", "")
		if ed == "" {
			return toolError("end_date is required when start_date is provided")
		}
		req.When = domain.EventWhen{
			StartDate: sd,
			EndDate:   ed,
		}
	} else {
		return toolError("start_time or start_date is required")
	}

	req.Participants = parseEventParticipants(args)

	if b := getBool(args, "busy"); b != nil {
		req.Busy = *b
	}
	if v := getString(args, "visibility", ""); v != "" {
		req.Visibility = v
	}

	if url := getString(args, "conferencing_url", ""); url != "" {
		req.Conferencing = &domain.Conferencing{
			Details: &domain.ConferencingDetails{URL: url},
		}
	}

	if reminders := parseReminders(args); reminders != nil {
		req.Reminders = reminders
	}

	event, err := s.client.CreateEvent(ctx, grantID, calendarID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":     event.ID,
		"title":  event.Title,
		"status": "created",
	})
}

// executeUpdateEvent updates an existing calendar event.
func (s *Server) executeUpdateEvent(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	eventID := getString(args, "event_id", "")
	if eventID == "" {
		return toolError("event_id is required")
	}
	calendarID := getString(args, "calendar_id", "primary")

	req := &domain.UpdateEventRequest{}

	if v := getString(args, "title", ""); v != "" {
		req.Title = &v
	}
	if v := getString(args, "description", ""); v != "" {
		req.Description = &v
	}
	if v := getString(args, "location", ""); v != "" {
		req.Location = &v
	}
	if v := getString(args, "visibility", ""); v != "" {
		req.Visibility = &v
	}

	if st := getInt64(args, "start_time", 0); st > 0 {
		et := getInt64(args, "end_time", 0)
		if et == 0 {
			return toolError("end_time is required when start_time is provided")
		}
		if et <= st {
			return toolError("end_time must be after start_time")
		}
		when := domain.EventWhen{
			StartTime: st,
			EndTime:   et,
		}
		req.When = &when
	} else if sd := getString(args, "start_date", ""); sd != "" {
		ed := getString(args, "end_date", "")
		if ed == "" {
			return toolError("end_date is required when start_date is provided")
		}
		when := domain.EventWhen{
			StartDate: sd,
			EndDate:   ed,
		}
		req.When = &when
	}

	if participants := parseEventParticipants(args); len(participants) > 0 {
		req.Participants = participants
	}

	if b := getBool(args, "busy"); b != nil {
		req.Busy = b
	}

	if url := getString(args, "conferencing_url", ""); url != "" {
		req.Conferencing = &domain.Conferencing{
			Details: &domain.ConferencingDetails{URL: url},
		}
	}

	if reminders := parseReminders(args); reminders != nil {
		req.Reminders = reminders
	}

	event, err := s.client.UpdateEvent(ctx, grantID, calendarID, eventID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"id":     event.ID,
		"title":  event.Title,
		"status": "updated",
	})
}

// executeDeleteEvent deletes a calendar event.
func (s *Server) executeDeleteEvent(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	eventID := getString(args, "event_id", "")
	if eventID == "" {
		return toolError("event_id is required")
	}
	calendarID := getString(args, "calendar_id", "primary")

	if err := s.client.DeleteEvent(ctx, grantID, calendarID, eventID); err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccessText("Deleted event " + eventID)
}

// executeSendRSVP sends an RSVP response to a calendar event invitation.
func (s *Server) executeSendRSVP(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)
	eventID := getString(args, "event_id", "")
	if eventID == "" {
		return toolError("event_id is required")
	}
	status := getString(args, "status", "")
	if status == "" {
		return toolError("status is required (yes, no, or maybe)")
	}
	if status != "yes" && status != "no" && status != "maybe" {
		return toolError("status must be yes, no, or maybe")
	}
	calendarID := getString(args, "calendar_id", "primary")
	comment := getString(args, "comment", "")

	req := &domain.SendRSVPRequest{
		Status:  status,
		Comment: comment,
	}

	if err := s.client.SendRSVP(ctx, grantID, calendarID, eventID, req); err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(map[string]any{
		"status":   "rsvp_sent",
		"event_id": eventID,
		"rsvp":     status,
	})
}

// executeGetFreeBusy returns free/busy information for the given emails.
func (s *Server) executeGetFreeBusy(ctx context.Context, args map[string]any) *ToolResponse {
	grantID := s.resolveGrantID(args)

	emails := getStringSlice(args, "emails")
	if len(emails) == 0 {
		return toolError("emails is required")
	}

	startTime := getInt64(args, "start_time", 0)
	if startTime == 0 {
		return toolError("start_time is required")
	}

	endTime := getInt64(args, "end_time", 0)
	if endTime == 0 {
		return toolError("end_time is required")
	}

	req := &domain.FreeBusyRequest{
		Emails:    emails,
		StartTime: startTime,
		EndTime:   endTime,
	}

	resp, err := s.client.GetFreeBusy(ctx, grantID, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(resp.Data)
}

// executeGetAvailability returns available meeting slots across multiple accounts.
func (s *Server) executeGetAvailability(ctx context.Context, args map[string]any) *ToolResponse {
	startTime := getInt64(args, "start_time", 0)
	if startTime == 0 {
		return toolError("start_time is required")
	}

	endTime := getInt64(args, "end_time", 0)
	if endTime == 0 {
		return toolError("end_time is required")
	}

	durationMinutes := getInt(args, "duration_minutes", 0)
	if durationMinutes == 0 {
		return toolError("duration_minutes is required")
	}

	participants := parseAvailabilityParticipants(args)
	if len(participants) == 0 {
		return toolError("at least one participant is required")
	}

	req := &domain.AvailabilityRequest{
		StartTime:       startTime,
		EndTime:         endTime,
		DurationMinutes: durationMinutes,
		Participants:    participants,
		IntervalMinutes: getInt(args, "interval_minutes", 0),
		RoundTo:         getInt(args, "round_to", 0),
	}

	resp, err := s.client.GetAvailability(ctx, req)
	if err != nil {
		return toolError(sanitizeError(err))
	}

	return toolSuccess(resp.Data)
}

// formatEventWhen converts an EventWhen into a map for tool responses.
func formatEventWhen(w domain.EventWhen) map[string]any {
	result := make(map[string]any)
	if w.StartTime > 0 {
		result["start_time"] = w.StartTime
		result["end_time"] = w.EndTime
	} else if w.Date != "" {
		result["date"] = w.Date
	} else if w.StartDate != "" {
		result["start_date"] = w.StartDate
		result["end_date"] = w.EndDate
	}
	return result
}

// parseEventParticipants extracts calendar event participant list from args.
func parseEventParticipants(args map[string]any) []domain.Participant {
	val, ok := args["participants"]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	result := make([]domain.Participant, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		email, _ := m["email"].(string)
		if email == "" {
			continue
		}
		name, _ := m["name"].(string)
		result = append(result, domain.Participant{
			Person: domain.Person{Name: name, Email: email},
		})
	}
	return result
}

// parseReminders extracts reminders from args.
func parseReminders(args map[string]any) *domain.Reminders {
	val, ok := args["reminders"]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	overrides := make([]domain.Reminder, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		minutes := getInt(m, "minutes", 0)
		method, _ := m["method"].(string)
		overrides = append(overrides, domain.Reminder{
			ReminderMinutes: minutes,
			ReminderMethod:  method,
		})
	}
	if len(overrides) == 0 {
		return nil
	}
	return &domain.Reminders{Overrides: overrides}
}

// parseAvailabilityParticipants extracts availability participants from args.
func parseAvailabilityParticipants(args map[string]any) []domain.AvailabilityParticipant {
	val, ok := args["participants"]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	result := make([]domain.AvailabilityParticipant, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		email, _ := m["email"].(string)
		if email == "" {
			continue
		}
		p := domain.AvailabilityParticipant{Email: email}
		if ids, ok := m["calendar_ids"].([]any); ok {
			for _, id := range ids {
				if s, ok := id.(string); ok {
					p.CalendarIDs = append(p.CalendarIDs, s)
				}
			}
		}
		result = append(result, p)
	}
	return result
}
