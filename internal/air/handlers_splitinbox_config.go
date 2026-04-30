package air

import (
	"encoding/json"
	"net/http"
	"strings"
)

// =============================================================================
// Split Inbox Config & VIP Management
// =============================================================================

// handleSplitInbox handles split inbox configuration and retrieval.
func (s *Server) handleSplitInbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getSplitInboxConfig(w, r)
	case http.MethodPut:
		s.updateSplitInboxConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSplitInboxConfig returns the split inbox configuration.
func (s *Server) getSplitInboxConfig(w http.ResponseWriter, _ *http.Request) {
	config := s.getOrCreateSplitInboxConfig()

	// Count emails per category (demo mode or from cache)
	counts := make(map[InboxCategory]int)
	counts[CategoryPrimary] = 50
	counts[CategoryVIP] = 5
	counts[CategoryNewsletters] = 20
	counts[CategoryUpdates] = 15
	counts[CategorySocial] = 8
	counts[CategoryPromotions] = 12

	writeJSON(w, http.StatusOK, SplitInboxResponse{
		Config:     config,
		Categories: counts,
	})
}

// updateSplitInboxConfig updates the split inbox configuration.
func (s *Server) updateSplitInboxConfig(w http.ResponseWriter, r *http.Request) {
	var config SplitInboxConfig
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&config); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	func() {
		s.splitInboxMu.Lock()
		defer s.splitInboxMu.Unlock()
		s.splitInboxConfig = &config
	}()

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"config":  config,
	})
}

// getOrCreateSplitInboxConfig returns the current split inbox config.
func (s *Server) getOrCreateSplitInboxConfig() SplitInboxConfig {
	s.splitInboxMu.RLock()
	if s.splitInboxConfig != nil {
		config := *s.splitInboxConfig
		s.splitInboxMu.RUnlock()
		return config
	}
	s.splitInboxMu.RUnlock()

	// Create default config
	return SplitInboxConfig{
		Enabled: true,
		Categories: []InboxCategory{
			CategoryPrimary, CategoryVIP, CategoryNewsletters,
			CategoryUpdates, CategorySocial, CategoryPromotions,
		},
		VIPSenders: []string{},
		Rules:      []CategoryRule{},
	}
}

// =============================================================================
// VIP Sender Management
// =============================================================================

// handleVIPSenders manages VIP sender list.
func (s *Server) handleVIPSenders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := s.getOrCreateSplitInboxConfig()
		writeJSON(w, http.StatusOK, map[string]any{
			"vip_senders": config.VIPSenders,
		})
	case http.MethodPost:
		var req struct {
			Email string `json:"email"`
		}
		if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
			return
		}
		s.addVIPSender(req.Email)
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "email": req.Email})
	case http.MethodDelete:
		email := r.URL.Query().Get("email")
		if email == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Email required"})
			return
		}
		s.removeVIPSender(email)
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "email": email})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// addVIPSender adds an email to the VIP list.
func (s *Server) addVIPSender(email string) {
	s.splitInboxMu.Lock()
	defer s.splitInboxMu.Unlock()

	if s.splitInboxConfig == nil {
		// Create default config inline to avoid deadlock
		s.splitInboxConfig = &SplitInboxConfig{
			Enabled: true,
			Categories: []InboxCategory{
				CategoryPrimary, CategoryVIP, CategoryNewsletters,
				CategoryUpdates, CategorySocial, CategoryPromotions,
			},
			VIPSenders: []string{},
			Rules:      []CategoryRule{},
		}
	}

	// Check if already exists
	for _, vip := range s.splitInboxConfig.VIPSenders {
		if strings.EqualFold(vip, email) {
			return
		}
	}
	s.splitInboxConfig.VIPSenders = append(s.splitInboxConfig.VIPSenders, email)
}

// removeVIPSender removes an email from the VIP list.
func (s *Server) removeVIPSender(email string) {
	s.splitInboxMu.Lock()
	defer s.splitInboxMu.Unlock()

	if s.splitInboxConfig == nil {
		return
	}

	filtered := make([]string, 0, len(s.splitInboxConfig.VIPSenders))
	for _, vip := range s.splitInboxConfig.VIPSenders {
		if !strings.EqualFold(vip, email) {
			filtered = append(filtered, vip)
		}
	}
	s.splitInboxConfig.VIPSenders = filtered
}
