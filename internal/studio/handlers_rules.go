package studio

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
)

// attachRuleToWorkspace appends the rule to the workspace's rule_ids.
func (s *Server) attachRuleToWorkspace(ctx context.Context, workspaceID, ruleID string) error {
	ws, err := s.nylasClient.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	ruleIDs := applyRuleIDChanges(ws.RulesIDs, []string{ruleID}, nil)
	_, err = s.nylasClient.UpdateWorkspace(ctx, workspaceID, workspaceRulesUpdate(ruleIDs))
	return err
}

func (s *Server) routeRules(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "/api/rules")
	switch {
	case r.Method == http.MethodPost && id == "":
		s.handleRuleCreate(w, r)
	case r.Method == http.MethodPatch && id != "":
		s.handleRulePatch(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		s.handleRuleDelete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleRuleCreate(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if !decodeBody(w, r, &payload) {
		return
	}

	// workspace_id is studio routing, not part of the rule resource.
	workspaceID, _ := payload["workspace_id"].(string)
	workspaceID = strings.TrimSpace(workspaceID)
	delete(payload, "workspace_id")

	if name, _ := payload["name"].(string); strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "rule name is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	rule, err := s.nylasClient.CreateRule(ctx, payload)
	if err != nil {
		writeMutationError(w, "Failed to create rule", err)
		return
	}

	if workspaceID != "" {
		if err := s.attachRuleToWorkspace(ctx, workspaceID, rule.ID); err != nil {
			// Don't leave an orphaned rule behind: it would consume a plan
			// slot and clutter the palette.
			if cleanupErr := s.nylasClient.DeleteRule(ctx, rule.ID); cleanupErr != nil {
				slog.Error("studio: cleanup of unattached rule failed", "rule_id", rule.ID, "err", cleanupErr)
			}
			writeMutationError(w, "Failed to attach rule to workspace", err)
			return
		}
	}

	s.respondMutation(ctx, w, http.StatusCreated, rule.ID)
}

func (s *Server) handleRulePatch(w http.ResponseWriter, r *http.Request, id string) {
	var payload map[string]any
	if !decodeBody(w, r, &payload) {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if _, err := s.nylasClient.UpdateRule(ctx, id, payload); err != nil {
		writeMutationError(w, "Failed to update rule", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handleRuleDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	// Detach from any workspace that references the rule first; the API
	// rejects deleting attached rules, and stale rule_ids poison later writes.
	workspaces, err := s.nylasClient.ListWorkspaces(ctx)
	if err != nil {
		writeMutationError(w, "Failed to list workspaces for detach", err)
		return
	}
	for _, ws := range workspaces {
		if !containsTrimmed(ws.RulesIDs, id) {
			continue
		}
		ruleIDs := applyRuleIDChanges(ws.RulesIDs, nil, []string{id})
		if _, err := s.nylasClient.UpdateWorkspace(ctx, ws.ID, workspaceRulesUpdate(ruleIDs)); err != nil {
			writeMutationError(w, "Failed to detach rule from workspace", err)
			return
		}
	}

	if err := s.nylasClient.DeleteRule(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete rule", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func containsTrimmed(ids []string, target string) bool {
	for _, id := range ids {
		if strings.TrimSpace(id) == target {
			return true
		}
	}
	return false
}
