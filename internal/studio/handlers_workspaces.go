package studio

import (
	"net/http"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

func (s *Server) routeWorkspaces(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "/api/workspaces")
	switch {
	case r.Method == http.MethodPost && id == "":
		s.handleWorkspaceCreate(w, r)
	case r.Method == http.MethodPatch && id != "":
		s.handleWorkspacePatch(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		s.handleWorkspaceDelete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleWorkspaceCreate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name   string `json:"name"`
		Domain string `json:"domain"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "workspace name is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	ws, err := s.nylasClient.CreateWorkspace(ctx, &domain.CreateWorkspaceRequest{
		Name:   body.Name,
		Domain: strings.TrimSpace(body.Domain),
	})
	if err != nil {
		writeMutationError(w, "Failed to create workspace", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusCreated, ws.ID)
}

func (s *Server) handleWorkspacePatch(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		PolicyID      *string  `json:"policy_id"`
		AddRuleIDs    []string `json:"add_rule_ids"`
		RemoveRuleIDs []string `json:"remove_rule_ids"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if body.PolicyID == nil && len(body.AddRuleIDs) == 0 && len(body.RemoveRuleIDs) == 0 {
		writeError(w, http.StatusBadRequest, "nothing to update")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	update := &domain.UpdateWorkspaceRequest{PolicyID: body.PolicyID}
	if len(body.AddRuleIDs) > 0 || len(body.RemoveRuleIDs) > 0 {
		// Rule changes are read-modify-write: the API replaces the whole
		// rule_ids list, so the current attachments must be fetched first.
		// Policy-only patches skip the extra round-trip.
		ws, err := s.nylasClient.GetWorkspace(ctx, id)
		if err != nil {
			writeMutationError(w, "Failed to load workspace", err)
			return
		}
		ruleIDs := applyRuleIDChanges(ws.RulesIDs, body.AddRuleIDs, body.RemoveRuleIDs)
		update.RulesIDs = &ruleIDs
	}

	if _, err := s.nylasClient.UpdateWorkspace(ctx, id, update); err != nil {
		writeMutationError(w, "Failed to update workspace", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handleWorkspaceDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if err := s.nylasClient.DeleteWorkspace(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete workspace", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

// workspaceRulesUpdate wraps a full rule_ids array as a workspace update.
func workspaceRulesUpdate(ruleIDs []string) *domain.UpdateWorkspaceRequest {
	return &domain.UpdateWorkspaceRequest{RulesIDs: &ruleIDs}
}

// applyRuleIDChanges produces the full rule_ids array for the workspace PATCH:
// the API replaces the whole list, so removals and additions are applied to
// the current attachment set, deduplicated, order-preserving.
func applyRuleIDChanges(current, add, remove []string) []string {
	out := make([]string, 0, len(current)+len(add))
	for _, id := range current {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(remove, id) {
			continue
		}
		out = append(out, id)
	}
	for _, id := range add {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(out, id) || slices.Contains(remove, id) {
			continue
		}
		out = append(out, id)
	}
	return out
}
