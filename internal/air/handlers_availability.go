package air

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ====================================
// AVAILABILITY & FIND TIME HANDLERS
// ====================================

// AvailabilityRequest represents a request to find available times.
type AvailabilityRequest struct {
	StartTime       int64    `json:"start_time"`
	EndTime         int64    `json:"end_time"`
	DurationMinutes int      `json:"duration_minutes"`
	Participants    []string `json:"participants"` // Email addresses
	IntervalMinutes int      `json:"interval_minutes,omitempty"`
}

// AvailabilityResponse represents available meeting slots.
type AvailabilityResponse struct {
	Slots   []AvailableSlotResponse `json:"slots"`
	Message string                  `json:"message,omitempty"`
}

// AvailableSlotResponse represents a single available time slot.
type AvailableSlotResponse struct {
	StartTime int64    `json:"start_time"`
	EndTime   int64    `json:"end_time"`
	Emails    []string `json:"emails,omitempty"`
}

// FreeBusyRequest represents a request to get free/busy info.
type FreeBusyRequest struct {
	StartTime int64    `json:"start_time"`
	EndTime   int64    `json:"end_time"`
	Emails    []string `json:"emails"`
}

// FreeBusyResponse represents free/busy data for participants.
type FreeBusyResponse struct {
	Data []FreeBusyCalendarResponse `json:"data"`
}

// FreeBusyCalendarResponse represents a calendar's busy times.
type FreeBusyCalendarResponse struct {
	Email     string             `json:"email"`
	TimeSlots []TimeSlotResponse `json:"time_slots"`
}

// TimeSlotResponse represents a busy or free time slot.
type TimeSlotResponse struct {
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	Status    string `json:"status"` // busy, free
}

// ConflictsResponse represents conflicting events.
type ConflictsResponse struct {
	Conflicts []EventConflict `json:"conflicts"`
	HasMore   bool            `json:"has_more"`
}

// EventConflict represents a scheduling conflict.
type EventConflict struct {
	Event1 EventResponse `json:"event1"`
	Event2 EventResponse `json:"event2"`
}

// handleAvailability finds available meeting times.
func (s *Server) handleAvailability(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Demo mode: return mock availability
	if s.demoMode {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		writeJSON(w, http.StatusOK, AvailabilityResponse{
			Slots: []AvailableSlotResponse{
				{StartTime: today.Add(10 * time.Hour).Unix(), EndTime: today.Add(11 * time.Hour).Unix()},
				{StartTime: today.Add(14 * time.Hour).Unix(), EndTime: today.Add(15 * time.Hour).Unix()},
				{StartTime: today.Add(24*time.Hour + 9*time.Hour).Unix(), EndTime: today.Add(24*time.Hour + 10*time.Hour).Unix()},
				{StartTime: today.Add(24*time.Hour + 11*time.Hour).Unix(), EndTime: today.Add(24*time.Hour + 12*time.Hour).Unix()},
				{StartTime: today.Add(24*time.Hour + 15*time.Hour).Unix(), EndTime: today.Add(24*time.Hour + 16*time.Hour).Unix()},
			},
			Message: "Demo mode: showing sample availability",
		})
		return
	}

	// Check if configured
	if !s.requireConfig(w) {
		return
	}

	// Parse request
	var req AvailabilityRequest
	if r.Method == http.MethodPost {
		if !parseJSONBody(w, r, &req) {
			return
		}
	} else {
		// Parse from query params for GET
		query := r.URL.Query()
		if startStr := query.Get("start_time"); startStr != "" {
			req.StartTime, _ = strconv.ParseInt(startStr, 10, 64)
		}
		if endStr := query.Get("end_time"); endStr != "" {
			req.EndTime, _ = strconv.ParseInt(endStr, 10, 64)
		}
		if durationStr := query.Get("duration_minutes"); durationStr != "" {
			req.DurationMinutes, _ = strconv.Atoi(durationStr)
		}
		if participants := query.Get("participants"); participants != "" {
			req.Participants = strings.Split(participants, ",")
		}
		if intervalStr := query.Get("interval_minutes"); intervalStr != "" {
			req.IntervalMinutes, _ = strconv.Atoi(intervalStr)
		}
	}

	// Validate request
	if req.StartTime == 0 || req.EndTime == 0 {
		// Default to next 7 days
		now := time.Now()
		req.StartTime = now.Unix()
		req.EndTime = now.Add(7 * 24 * time.Hour).Unix()
	}
	if req.DurationMinutes == 0 {
		req.DurationMinutes = 30 // Default 30 minutes
	}
	if req.IntervalMinutes == 0 {
		req.IntervalMinutes = 15 // Default 15 min intervals
	}

	// Round times to 5-minute intervals (Nylas API requirement)
	req.StartTime = roundUpTo5Min(req.StartTime)
	req.EndTime = roundUpTo5Min(req.EndTime)

	// Get current user's email if no participants specified
	if len(req.Participants) == 0 {
		email := s.getCurrentUserEmail()
		if email != "" {
			req.Participants = []string{email}
		}
	}

	if len(req.Participants) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "At least one participant email is required",
		})
		return
	}

	// Build domain request
	domainReq := &domain.AvailabilityRequest{
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		DurationMinutes: req.DurationMinutes,
		IntervalMinutes: req.IntervalMinutes,
		Participants:    make([]domain.AvailabilityParticipant, 0, len(req.Participants)),
	}
	for _, email := range req.Participants {
		domainReq.Participants = append(domainReq.Participants, domain.AvailabilityParticipant{
			Email: strings.TrimSpace(email),
		})
	}

	// Call Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	result, err := s.nylasClient.GetAvailability(ctx, domainReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get availability: " + err.Error(),
		})
		return
	}

	// Convert to response
	resp := AvailabilityResponse{
		Slots: make([]AvailableSlotResponse, 0, len(result.Data.TimeSlots)),
	}
	for _, slot := range result.Data.TimeSlots {
		resp.Slots = append(resp.Slots, AvailableSlotResponse{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Emails:    slot.Emails,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleFreeBusy returns free/busy information for participants.
func (s *Server) handleFreeBusy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Special demo mode: return mock free/busy data
	if s.demoMode {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		writeJSON(w, http.StatusOK, FreeBusyResponse{
			Data: []FreeBusyCalendarResponse{
				{
					Email: "demo@example.com",
					TimeSlots: []TimeSlotResponse{
						{StartTime: today.Add(9 * time.Hour).Unix(), EndTime: today.Add(10 * time.Hour).Unix(), Status: "busy"},
						{StartTime: today.Add(12 * time.Hour).Unix(), EndTime: today.Add(13 * time.Hour).Unix(), Status: "busy"},
						{StartTime: today.Add(14 * time.Hour).Unix(), EndTime: today.Add(15 * time.Hour).Unix(), Status: "busy"},
					},
				},
			},
		})
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	// Parse request
	var req FreeBusyRequest
	if r.Method == http.MethodPost {
		if !parseJSONBody(w, r, &req) {
			return
		}
	} else {
		// Parse from query params for GET
		query := r.URL.Query()
		if startStr := query.Get("start_time"); startStr != "" {
			req.StartTime, _ = strconv.ParseInt(startStr, 10, 64)
		}
		if endStr := query.Get("end_time"); endStr != "" {
			req.EndTime, _ = strconv.ParseInt(endStr, 10, 64)
		}
		if emails := query.Get("emails"); emails != "" {
			req.Emails = strings.Split(emails, ",")
		}
	}

	// Validate and set defaults
	if req.StartTime == 0 || req.EndTime == 0 {
		// Default to next 7 days
		now := time.Now()
		req.StartTime = now.Unix()
		req.EndTime = now.Add(7 * 24 * time.Hour).Unix()
	}

	// Get current user's email if no emails specified
	if len(req.Emails) == 0 {
		email := s.getCurrentUserEmail()
		if email != "" {
			req.Emails = []string{email}
		}
	}

	if len(req.Emails) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "At least one email is required",
		})
		return
	}

	// Build domain request
	domainReq := &domain.FreeBusyRequest{
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Emails:    req.Emails,
	}

	// Call Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	result, err := s.nylasClient.GetFreeBusy(ctx, grantID, domainReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to get free/busy: " + err.Error(),
		})
		return
	}

	// Convert to response
	resp := FreeBusyResponse{
		Data: make([]FreeBusyCalendarResponse, 0, len(result.Data)),
	}
	for _, cal := range result.Data {
		calResp := FreeBusyCalendarResponse{
			Email:     cal.Email,
			TimeSlots: make([]TimeSlotResponse, 0, len(cal.TimeSlots)),
		}
		for _, slot := range cal.TimeSlots {
			calResp.TimeSlots = append(calResp.TimeSlots, TimeSlotResponse{
				StartTime: slot.StartTime,
				EndTime:   slot.EndTime,
				Status:    slot.Status,
			})
		}
		resp.Data = append(resp.Data, calResp)
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleConflicts detects scheduling conflicts in events.
func (s *Server) handleConflicts(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Special demo mode: return sample conflicts
	if s.demoMode {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		writeJSON(w, http.StatusOK, ConflictsResponse{
			Conflicts: []EventConflict{
				{
					Event1: EventResponse{
						ID:        "demo-event-conflict-1",
						Title:     "Team Meeting",
						StartTime: today.Add(14 * time.Hour).Unix(),
						EndTime:   today.Add(15 * time.Hour).Unix(),
						Busy:      true,
					},
					Event2: EventResponse{
						ID:        "demo-event-conflict-2",
						Title:     "Client Call",
						StartTime: today.Add(14*time.Hour + 30*time.Minute).Unix(),
						EndTime:   today.Add(15*time.Hour + 30*time.Minute).Unix(),
						Busy:      true,
					},
				},
			},
			HasMore: false,
		})
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	// Parse query params
	query := r.URL.Query()
	calendarID := query.Get("calendar_id")
	if calendarID == "" {
		calendarID = "primary"
	}

	// Parse time range
	var startTime, endTime int64
	if startStr := query.Get("start_time"); startStr != "" {
		startTime, _ = strconv.ParseInt(startStr, 10, 64)
	}
	if endStr := query.Get("end_time"); endStr != "" {
		endTime, _ = strconv.ParseInt(endStr, 10, 64)
	}

	// Default to current week
	if startTime == 0 || endTime == 0 {
		now := time.Now()
		weekday := int(now.Weekday())
		startOfWeek := now.AddDate(0, 0, -weekday).Truncate(24 * time.Hour)
		endOfWeek := startOfWeek.AddDate(0, 0, 7)
		startTime = startOfWeek.Unix()
		endTime = endOfWeek.Unix()
	}

	// Fetch events
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	params := &domain.EventQueryParams{
		Limit:           200,
		Start:           startTime,
		End:             endTime,
		ExpandRecurring: true,
	}

	result, err := s.nylasClient.GetEventsWithCursor(ctx, grantID, calendarID, params)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch events: " + err.Error(),
		})
		return
	}

	// Find conflicts (overlapping busy events)
	conflicts := findConflicts(result.Data)

	resp := ConflictsResponse{
		Conflicts: conflicts,
		HasMore:   false,
	}

	writeJSON(w, http.StatusOK, resp)
}

// findConflicts detects overlapping events.
func findConflicts(events []domain.Event) []EventConflict {
	var conflicts []EventConflict

	// Filter to only busy events
	var busyEvents []domain.Event
	for _, e := range events {
		if e.Busy && e.Status != "cancelled" {
			busyEvents = append(busyEvents, e)
		}
	}

	// Check each pair for overlap
	for i := 0; i < len(busyEvents); i++ {
		for j := i + 1; j < len(busyEvents); j++ {
			e1, e2 := busyEvents[i], busyEvents[j]

			// Get start/end times
			start1, end1 := e1.When.StartTime, e1.When.EndTime
			start2, end2 := e2.When.StartTime, e2.When.EndTime

			// Handle all-day events. Only override the upstream timestamps on
			// successful parse; on malformed dates we'd otherwise produce a
			// year-1 Unix timestamp and detect bogus conflicts.
			start1, end1 = allDayBounds(e1.When, start1, end1)
			start2, end2 = allDayBounds(e2.When, start2, end2)

			// Check for overlap: start1 < end2 && start2 < end1
			if start1 < end2 && start2 < end1 {
				conflicts = append(conflicts, EventConflict{
					Event1: eventToResponse(e1),
					Event2: eventToResponse(e2),
				})
			}
		}
	}

	return conflicts
}

// allDayBounds returns the [start, end] Unix timestamps for an all-day or
// multi-day event. If the When value carries a date string we cannot parse,
// the caller-supplied fallback (start, end) is returned untouched so a
// malformed upstream date never collapses the window to year 1.
func allDayBounds(when domain.EventWhen, start, end int64) (int64, int64) {
	if !when.IsAllDay() {
		return start, end
	}
	switch {
	case when.Date != "":
		t, err := time.Parse("2006-01-02", when.Date)
		if err != nil {
			return start, end
		}
		return t.Unix(), t.Add(24 * time.Hour).Unix()
	case when.StartDate != "":
		t, err := time.Parse("2006-01-02", when.StartDate)
		if err != nil {
			return start, end
		}
		newStart := t.Unix()
		newEnd := newStart + 24*60*60
		if when.EndDate != "" {
			if et, err := time.Parse("2006-01-02", when.EndDate); err == nil {
				newEnd = et.Unix()
			}
		}
		return newStart, newEnd
	}
	return start, end
}

// roundUpTo5Min rounds a Unix timestamp up to the next 5-minute boundary.
// This is required by the Nylas API for availability requests.
func roundUpTo5Min(unixTime int64) int64 {
	const fiveMinutes = 5 * 60 // 300 seconds
	remainder := unixTime % fiveMinutes
	if remainder == 0 {
		return unixTime
	}
	return unixTime + (fiveMinutes - remainder)
}
