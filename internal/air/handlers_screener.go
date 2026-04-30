package air

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

// ScreenedSender represents a sender pending approval
type ScreenedSender struct {
	Email       string    `json:"email"`
	Name        string    `json:"name,omitempty"`
	Domain      string    `json:"domain"`
	FirstSeen   time.Time `json:"firstSeen"`
	EmailCount  int       `json:"emailCount"`
	SampleSubj  string    `json:"sampleSubject,omitempty"`
	Status      string    `json:"status"`                // pending, allowed, blocked
	Destination string    `json:"destination,omitempty"` // inbox, feed, paper_trail
}

// ScreenerStore manages screened senders
type ScreenerStore struct {
	senders map[string]*ScreenedSender
	mu      sync.RWMutex
}

var screenerStore = &ScreenerStore{
	senders: make(map[string]*ScreenedSender),
}

// handleGetScreenedSenders returns pending senders
func (s *Server) handleGetScreenedSenders(w http.ResponseWriter, r *http.Request) {
	screenerStore.mu.RLock()
	defer screenerStore.mu.RUnlock()

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	senders := make([]*ScreenedSender, 0)
	for _, sender := range screenerStore.senders {
		if sender.Status == status {
			senders = append(senders, sender)
		}
	}

	httputil.WriteJSON(w, http.StatusOK, senders)
}

// handleScreenerAllow allows a sender
func (s *Server) handleScreenerAllow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Destination string `json:"destination"` // inbox, feed, paper_trail
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Destination == "" {
		req.Destination = "inbox"
	}

	screenerStore.mu.Lock()
	defer screenerStore.mu.Unlock()

	if sender, ok := screenerStore.senders[req.Email]; ok {
		sender.Status = "allowed"
		sender.Destination = req.Destination
	} else {
		screenerStore.senders[req.Email] = &ScreenedSender{
			Email:       req.Email,
			Status:      "allowed",
			Destination: req.Destination,
			FirstSeen:   time.Now(),
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "allowed", "destination": req.Destination})
}

// handleScreenerBlock blocks a sender
func (s *Server) handleScreenerBlock(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	screenerStore.mu.Lock()
	defer screenerStore.mu.Unlock()

	if sender, ok := screenerStore.senders[req.Email]; ok {
		sender.Status = "blocked"
	} else {
		screenerStore.senders[req.Email] = &ScreenedSender{
			Email:     req.Email,
			Status:    "blocked",
			FirstSeen: time.Now(),
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "blocked"})
}

// handleAddToScreener adds a new sender for screening
func (s *Server) handleAddToScreener(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Name    string `json:"name,omitempty"`
		Subject string `json:"subject,omitempty"`
	}

	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	domain := extractDomain(req.Email)

	screenerStore.mu.Lock()
	defer screenerStore.mu.Unlock()

	if sender, ok := screenerStore.senders[req.Email]; ok {
		sender.EmailCount++
		if req.Subject != "" {
			sender.SampleSubj = req.Subject
		}
	} else {
		screenerStore.senders[req.Email] = &ScreenedSender{
			Email:      req.Email,
			Name:       req.Name,
			Domain:     domain,
			FirstSeen:  time.Now(),
			EmailCount: 1,
			SampleSubj: req.Subject,
			Status:     "pending",
		}
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "pending"})
}

// IsSenderAllowed reports whether a sender has been explicitly allowed and,
// if so, the routing destination ("inbox", "feed", "paper_trail").
//
// Only "allowed" senders pass — pending senders need screening, and blocked
// senders are rejected. Unknown senders default to needing screening too.
func IsSenderAllowed(email string) (bool, string) {
	screenerStore.mu.RLock()
	defer screenerStore.mu.RUnlock()

	sender, ok := screenerStore.senders[email]
	if !ok {
		return false, ""
	}
	if sender.Status == "allowed" {
		return true, sender.Destination
	}
	return false, ""
}
