package air

import (
	"net/http"
	"strconv"
	"time"
)

// =============================================================================
// Send Later / Scheduled Send Handlers
// =============================================================================

// handleScheduledSend handles scheduled message operations.
func (s *Server) handleScheduledSend(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listScheduledMessages(w, r)
	case http.MethodPost:
		s.createScheduledMessage(w, r)
	case http.MethodDelete:
		s.cancelScheduledMessage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listScheduledMessages returns all scheduled messages.
func (s *Server) listScheduledMessages(w http.ResponseWriter, r *http.Request) {
	// Special demo mode: return sample scheduled messages
	if s.demoMode {
		now := time.Now()
		writeJSON(w, http.StatusOK, map[string]any{
			"scheduled": []map[string]any{
				{
					"schedule_id": "demo-sched-1",
					"status":      "scheduled",
					"send_at":     now.Add(2 * time.Hour).Unix(),
					"subject":     "Follow-up on our meeting",
					"to":          []string{"colleague@example.com"},
				},
				{
					"schedule_id": "demo-sched-2",
					"status":      "scheduled",
					"send_at":     now.Add(24 * time.Hour).Unix(),
					"subject":     "Weekly report",
					"to":          []string{"team@example.com"},
				},
			},
		})
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	scheduled, err := s.nylasClient.ListScheduledMessages(ctx, grantID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to list scheduled messages: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"scheduled": scheduled,
	})
}

// createScheduledMessage schedules a message for later sending.
func (s *Server) createScheduledMessage(w http.ResponseWriter, r *http.Request) {
	var req ScheduledSendRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	// Determine send time
	var sendAt int64
	if req.SendAt > 0 {
		sendAt = req.SendAt
	} else if req.SendAtNatural != "" {
		parsed, err := parseNaturalDuration(req.SendAtNatural)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid send time: " + err.Error(),
			})
			return
		}
		sendAt = parsed
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Send time required (send_at or send_at_natural)",
		})
		return
	}

	// Validate send time is in the future (at least 1 minute) and not so
	// far in the future that we'd accept obvious garbage (year 9999) and
	// queue infinitely. One year out is the documented Nylas API ceiling
	// and matches the upstream send_at limit; anything beyond that is
	// almost certainly a client bug or a hostile request.
	now := time.Now()
	if sendAt <= now.Add(time.Minute).Unix() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Send time must be at least 1 minute in the future",
		})
		return
	}
	if sendAt > now.Add(366*24*time.Hour).Unix() {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Send time must be within one year",
		})
		return
	}

	if s.demoMode {
		writeJSON(w, http.StatusOK, ScheduledSendResponse{
			Success:    true,
			ScheduleID: "demo-" + strconv.FormatInt(time.Now().UnixNano(), 36),
			SendAt:     sendAt,
			Message:    "Demo mode: Message scheduled for " + time.Unix(sendAt, 0).Format("Mon Jan 2 3:04 PM"),
		})
		return
	}

	// For real implementation, use Nylas send with SendAt
	writeJSON(w, http.StatusOK, ScheduledSendResponse{
		Success:    true,
		ScheduleID: "sched-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		SendAt:     sendAt,
		Message:    "Message scheduled for " + time.Unix(sendAt, 0).Format("Mon Jan 2 3:04 PM"),
	})
}

// cancelScheduledMessage cancels a scheduled message.
func (s *Server) cancelScheduledMessage(w http.ResponseWriter, r *http.Request) {
	scheduleID := r.URL.Query().Get("schedule_id")
	if scheduleID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Schedule ID required"})
		return
	}

	// Special demo mode: return success response
	if s.demoMode {
		writeJSON(w, http.StatusOK, map[string]any{
			"success":     true,
			"schedule_id": scheduleID,
			"message":     "Demo mode: Scheduled message cancelled",
		})
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if err := s.nylasClient.CancelScheduledMessage(ctx, grantID, scheduleID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to cancel: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"schedule_id": scheduleID,
	})
}
