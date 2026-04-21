package ui

import (
	"encoding/json"
	"net/http"
	"slices"

	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/domain"
)

func (s *Server) handleListGrants(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Demo mode: return sample grants
	if s.demoMode {
		writeJSON(w, http.StatusOK, GrantsResponse{
			Grants:       demoGrants(),
			DefaultGrant: demoDefaultGrant(),
		})
		return
	}

	grants, err := s.grantStore.ListGrants()
	if err != nil {
		writeJSON(w, http.StatusOK, GrantsResponse{Grants: []Grant{}})
		return
	}

	var grantList []Grant
	for _, g := range grants {
		grantList = append(grantList, grantFromDomain(g))
	}

	defaultID, _ := s.grantStore.GetDefaultGrant()

	writeJSON(w, http.StatusOK, GrantsResponse{
		Grants:       grantList,
		DefaultGrant: defaultID,
	})
}

// SetDefaultGrantRequest represents the request to set default grant.
type SetDefaultGrantRequest struct {
	GrantID string `json:"grant_id"`
}

// SetDefaultGrantResponse represents the response for setting default grant.
type SetDefaultGrantResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func (s *Server) handleSetDefaultGrant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Demo mode: simulate success
	if s.demoMode {
		writeJSON(w, http.StatusOK, SetDefaultGrantResponse{
			Success: true,
			Message: "Default account updated (demo mode)",
		})
		return
	}

	var req SetDefaultGrantRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SetDefaultGrantResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.GrantID == "" {
		writeJSON(w, http.StatusBadRequest, SetDefaultGrantResponse{
			Success: false,
			Error:   "Grant ID is required",
		})
		return
	}

	// Verify grant exists
	grants, err := s.grantStore.ListGrants()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SetDefaultGrantResponse{
			Success: false,
			Error:   "Failed to list grants",
		})
		return
	}

	// Use slices.ContainsFunc (Go 1.21+) for cleaner lookup
	found := slices.ContainsFunc(grants, func(g domain.GrantInfo) bool {
		return g.ID == req.GrantID
	})

	if !found {
		writeJSON(w, http.StatusNotFound, SetDefaultGrantResponse{
			Success: false,
			Error:   "Grant not found",
		})
		return
	}

	if err := authapp.PersistDefaultGrant(s.configStore, s.grantStore, req.GrantID); err != nil {
		writeJSON(w, http.StatusInternalServerError, SetDefaultGrantResponse{
			Success: false,
			Error:   "Failed to set default grant: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, SetDefaultGrantResponse{
		Success: true,
		Message: "Default account updated",
	})
}
