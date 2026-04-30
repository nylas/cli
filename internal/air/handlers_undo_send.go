package air

import (
	"cmp"
	"encoding/json"
	"net/http"
	"slices"
	"time"
)

// =============================================================================
// Undo Send Handlers
// =============================================================================

// handleUndoSend handles undo send operations.
func (s *Server) handleUndoSend(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getUndoSendConfig(w, r)
	case http.MethodPut:
		s.updateUndoSendConfig(w, r)
	case http.MethodPost:
		s.undoSend(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePendingSends lists pending sends in grace period.
func (s *Server) handlePendingSends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.pendingSendMu.RLock()
	pending := make([]PendingSend, 0)
	now := time.Now().Unix()
	for _, ps := range s.pendingSends {
		if !ps.Cancelled && ps.SendAt > now {
			pending = append(pending, ps)
		}
	}
	s.pendingSendMu.RUnlock()

	// Sort by send time (soonest first)
	slices.SortFunc(pending, func(a, b PendingSend) int {
		return cmp.Compare(a.SendAt, b.SendAt)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"pending": pending,
		"count":   len(pending),
	})
}

// getUndoSendConfig returns the undo send configuration.
func (s *Server) getUndoSendConfig(w http.ResponseWriter, _ *http.Request) {
	config := s.getOrCreateUndoSendConfig()
	writeJSON(w, http.StatusOK, config)
}

// updateUndoSendConfig updates the undo send configuration.
func (s *Server) updateUndoSendConfig(w http.ResponseWriter, r *http.Request) {
	var config UndoSendConfig
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&config); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Validate grace period (5-60 seconds)
	if config.GracePeriodSec < 5 {
		config.GracePeriodSec = 5
	} else if config.GracePeriodSec > 60 {
		config.GracePeriodSec = 60
	}

	func() {
		s.undoSendMu.Lock()
		defer s.undoSendMu.Unlock()
		s.undoSendConfig = &config
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"config":  config,
	})
}

// undoSend cancels a pending send.
func (s *Server) undoSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MessageID string `json:"message_id"`
	}
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.MessageID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Message ID required"})
		return
	}

	s.pendingSendMu.Lock()
	defer s.pendingSendMu.Unlock()

	ps, exists := s.pendingSends[req.MessageID]
	if !exists {
		writeJSON(w, http.StatusNotFound, UndoSendResponse{
			Success: false,
			Error:   "Message not found or already sent",
		})
		return
	}

	now := time.Now().Unix()
	if ps.SendAt <= now {
		writeJSON(w, http.StatusBadRequest, UndoSendResponse{
			Success: false,
			Error:   "Grace period expired, message already sent",
		})
		return
	}

	// Mark as cancelled
	ps.Cancelled = true
	s.pendingSends[req.MessageID] = ps

	writeJSON(w, http.StatusOK, UndoSendResponse{
		Success:   true,
		MessageID: req.MessageID,
		Message:   "Message cancelled successfully",
	})
}

// getOrCreateUndoSendConfig returns the current undo send config.
func (s *Server) getOrCreateUndoSendConfig() UndoSendConfig {
	s.undoSendMu.RLock()
	if s.undoSendConfig != nil {
		config := *s.undoSendConfig
		s.undoSendMu.RUnlock()
		return config
	}
	s.undoSendMu.RUnlock()

	return UndoSendConfig{
		Enabled:        true,
		GracePeriodSec: 10,
	}
}
