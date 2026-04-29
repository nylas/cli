package air

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// ReplyLaterItem represents an email in the reply later queue
type ReplyLaterItem struct {
	EmailID     string    `json:"emailId"`
	Subject     string    `json:"subject"`
	From        string    `json:"from"`
	AddedAt     time.Time `json:"addedAt"`
	RemindAt    time.Time `json:"remindAt,omitzero"`
	DraftID     string    `json:"draftId,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	Priority    int       `json:"priority"` // 1=high, 2=medium, 3=low
	IsCompleted bool      `json:"isCompleted"`
}

// replyLaterStore holds reply later items
type replyLaterStore struct {
	items map[string]*ReplyLaterItem // emailID -> item
	mu    sync.RWMutex
}

var rlStore = &replyLaterStore{
	items: make(map[string]*ReplyLaterItem),
}

// handleReplyLaterRoute dispatches reply later requests by method
func (s *Server) handleReplyLaterRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetReplyLaterItems(w, r)
	case http.MethodPost:
		s.handleAddToReplyLater(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetReplyLaterItems returns all reply later items
func (s *Server) handleGetReplyLaterItems(w http.ResponseWriter, r *http.Request) {
	rlStore.mu.RLock()
	defer rlStore.mu.RUnlock()

	showCompleted := ParseBool(r.URL.Query(), "completed")

	items := make([]*ReplyLaterItem, 0)
	for _, item := range rlStore.items {
		if showCompleted || !item.IsCompleted {
			items = append(items, item)
		}
	}

	httputil.WriteJSON(w, http.StatusOK, items)
}

// handleAddToReplyLater adds an email to reply later queue
func (s *Server) handleAddToReplyLater(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailID  string `json:"emailId"`
		Subject  string `json:"subject"`
		From     string `json:"from"`
		RemindIn string `json:"remindIn,omitempty"` // "1h", "1d", "1w"
		Priority int    `json:"priority,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.EmailID == "" {
		http.Error(w, "emailId is required", http.StatusBadRequest)
		return
	}

	item := &ReplyLaterItem{
		EmailID:     req.EmailID,
		Subject:     req.Subject,
		From:        req.From,
		AddedAt:     time.Now(),
		Notes:       req.Notes,
		Priority:    req.Priority,
		IsCompleted: false,
	}

	if item.Priority == 0 {
		item.Priority = 2 // Default medium
	}

	// Parse remind time
	if req.RemindIn != "" {
		switch req.RemindIn {
		case "1h":
			item.RemindAt = time.Now().Add(1 * time.Hour)
		case "4h":
			item.RemindAt = time.Now().Add(4 * time.Hour)
		case "1d":
			item.RemindAt = time.Now().Add(24 * time.Hour)
		case "3d":
			item.RemindAt = time.Now().Add(72 * time.Hour)
		case "1w":
			item.RemindAt = time.Now().Add(168 * time.Hour)
		}
	}

	rlStore.mu.Lock()
	rlStore.items[req.EmailID] = item
	rlStore.mu.Unlock()

	httputil.WriteJSON(w, http.StatusOK, item)
}

// handleUpdateReplyLater updates a reply later item
func (s *Server) handleUpdateReplyLater(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EmailID     string `json:"emailId"`
		DraftID     string `json:"draftId,omitempty"`
		Notes       string `json:"notes,omitempty"`
		Priority    int    `json:"priority,omitempty"`
		IsCompleted bool   `json:"isCompleted"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rlStore.mu.Lock()
	defer rlStore.mu.Unlock()

	item, ok := rlStore.items[req.EmailID]
	if !ok {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	if req.DraftID != "" {
		item.DraftID = req.DraftID
	}
	if req.Notes != "" {
		item.Notes = req.Notes
	}
	if req.Priority > 0 {
		item.Priority = req.Priority
	}
	item.IsCompleted = req.IsCompleted

	httputil.WriteJSON(w, http.StatusOK, item)
}

// handleRemoveFromReplyLater removes an email from reply later queue
func (s *Server) handleRemoveFromReplyLater(w http.ResponseWriter, r *http.Request) {
	emailID := r.URL.Query().Get("emailId")
	if emailID == "" {
		http.Error(w, "emailId is required", http.StatusBadRequest)
		return
	}

	rlStore.mu.Lock()
	delete(rlStore.items, emailID)
	rlStore.mu.Unlock()

	w.WriteHeader(http.StatusNoContent)
}

// GetPendingReminders returns items with reminders due
func GetPendingReminders() []*ReplyLaterItem {
	rlStore.mu.RLock()
	defer rlStore.mu.RUnlock()

	now := time.Now()
	pending := make([]*ReplyLaterItem, 0)

	for _, item := range rlStore.items {
		if !item.IsCompleted && !item.RemindAt.IsZero() && item.RemindAt.Before(now) {
			pending = append(pending, item)
		}
	}

	return pending
}
