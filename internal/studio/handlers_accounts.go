package studio

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nylas/cli/internal/domain"
)

func (s *Server) routeAccounts(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "/api/accounts")
	moveID, isMove := strings.CutSuffix(id, "/move")
	switch {
	case r.Method == http.MethodPost && id == "":
		s.handleAccountCreate(w, r)
	case r.Method == http.MethodPost && isMove && moveID != "":
		s.handleAccountMove(w, r, moveID)
	case r.Method == http.MethodPatch && id != "":
		s.handleAccountPatch(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		s.handleAccountDelete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAccountCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email       string `json:"email"`
		Name        string `json:"name"`
		AppPassword string `json:"app_password"`
		WorkspaceID string `json:"workspace_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	body.Email = strings.TrimSpace(body.Email)
	if body.Email == "" {
		writeError(w, http.StatusBadRequest, "account email is required")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if utf8.RuneCountInString(body.Name) > 256 {
		writeError(w, http.StatusBadRequest, "name must be 256 characters or fewer")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	account, err := s.nylasClient.CreateAgentAccount(ctx, body.Email, body.Name, body.AppPassword, strings.TrimSpace(body.WorkspaceID))
	if err != nil {
		writeMutationError(w, "Failed to create agent account", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusCreated, account.ID)
}

func (s *Server) handleAccountPatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		AppPassword string  `json:"app_password"`
		Name        *string `json:"name"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	appPassword := strings.TrimSpace(body.AppPassword)
	nameProvided := body.Name != nil
	var name string
	if nameProvided {
		name = strings.TrimSpace(*body.Name)
		if utf8.RuneCountInString(name) > 256 {
			writeError(w, http.StatusBadRequest, "name must be 256 characters or fewer")
			return
		}
	}
	if appPassword == "" && !nameProvided {
		writeError(w, http.StatusBadRequest, "app_password or name is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	// The grant PATCH requires the account email alongside the new password.
	account, err := s.nylasClient.GetAgentAccount(ctx, id)
	if err != nil {
		writeMutationError(w, "Failed to load agent account", err)
		return
	}

	// Preserve the existing name unless the caller overrides it: the grant
	// update replaces the full record, so an omitted name would clear it.
	effectiveName := account.Name
	if nameProvided {
		effectiveName = name
	}

	if _, err := s.nylasClient.UpdateAgentAccount(ctx, id, account.Email, effectiveName, appPassword); err != nil {
		writeMutationError(w, "Failed to update agent account", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

// handleAccountMove reassigns the grant to the target workspace. A single
// assign moves the grant even when it belongs to another workspace; removal
// is never sent because remove_grants strands the grant in no workspace.
func (s *Server) handleAccountMove(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		WorkspaceID string `json:"workspace_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	body.WorkspaceID = strings.TrimSpace(body.WorkspaceID)
	if body.WorkspaceID == "" {
		writeError(w, http.StatusBadRequest, "workspace_id is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if _, err := s.nylasClient.AssignWorkspaceGrants(ctx, body.WorkspaceID, &domain.WorkspaceAssignRequest{
		AssignGrants: []string{id},
	}); err != nil {
		writeMutationError(w, "Failed to move agent account", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handleAccountDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if err := s.nylasClient.DeleteAgentAccount(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete agent account", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}
