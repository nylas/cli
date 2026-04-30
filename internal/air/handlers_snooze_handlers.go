package air

import (
	"cmp"
	"net/http"
	"slices"
	"time"
)

// =============================================================================
// Snooze HTTP Handlers
// =============================================================================

// handleSnooze handles snooze operations.
func (s *Server) handleSnooze(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listSnoozedEmails(w, r)
	case http.MethodPost:
		s.snoozeEmail(w, r)
	case http.MethodDelete:
		s.unsnoozeEmail(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listSnoozedEmails returns all snoozed emails.
func (s *Server) listSnoozedEmails(w http.ResponseWriter, _ *http.Request) {
	s.snoozeMu.RLock()
	snoozed := make([]SnoozedEmail, 0, len(s.snoozedEmails))
	now := time.Now().Unix()
	for _, se := range s.snoozedEmails {
		if se.SnoozeUntil > now {
			snoozed = append(snoozed, se)
		}
	}
	s.snoozeMu.RUnlock()

	// Sort by snooze time (soonest first)
	slices.SortFunc(snoozed, func(a, b SnoozedEmail) int {
		return cmp.Compare(a.SnoozeUntil, b.SnoozeUntil)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"snoozed": snoozed,
		"count":   len(snoozed),
	})
}

// snoozeEmail snoozes an email until a specific time.
func (s *Server) snoozeEmail(w http.ResponseWriter, r *http.Request) {
	var req SnoozeRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	if req.EmailID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Email ID required"})
		return
	}

	var snoozeUntil int64
	if req.SnoozeUntil > 0 {
		snoozeUntil = req.SnoozeUntil
	} else if req.Duration != "" {
		parsed, err := parseNaturalDuration(req.Duration)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid duration: " + err.Error(),
			})
			return
		}
		snoozeUntil = parsed
	} else {
		// Default: snooze for 1 hour
		snoozeUntil = time.Now().Add(time.Hour).Unix()
	}

	// Validate snooze time is in the future
	if snoozeUntil <= time.Now().Unix() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Snooze time must be in the future",
		})
		return
	}

	snoozed := SnoozedEmail{
		EmailID:     req.EmailID,
		SnoozeUntil: snoozeUntil,
		CreatedAt:   time.Now().Unix(),
	}

	func() {
		s.snoozeMu.Lock()
		defer s.snoozeMu.Unlock()
		if s.snoozedEmails == nil {
			s.snoozedEmails = make(map[string]SnoozedEmail)
		}
		s.snoozedEmails[req.EmailID] = snoozed
	}()

	writeJSON(w, http.StatusOK, SnoozeResponse{
		Success:     true,
		EmailID:     req.EmailID,
		SnoozeUntil: snoozeUntil,
		Message:     "Email snoozed until " + time.Unix(snoozeUntil, 0).Format("Mon Jan 2 3:04 PM"),
	})
}

// unsnoozeEmail removes the snooze from an email.
func (s *Server) unsnoozeEmail(w http.ResponseWriter, r *http.Request) {
	emailID := r.URL.Query().Get("email_id")
	if emailID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Email ID required"})
		return
	}

	func() {
		s.snoozeMu.Lock()
		defer s.snoozeMu.Unlock()
		delete(s.snoozedEmails, emailID)
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"success":  true,
		"email_id": emailID,
	})
}
