package studio

import (
	"net/http"
	"strings"
)

func (s *Server) routePolicies(w http.ResponseWriter, r *http.Request) {
	id := pathID(r, "/api/policies")
	switch {
	case r.Method == http.MethodGet && id != "":
		s.handlePolicyGet(w, r, id)
	case r.Method == http.MethodPost && id == "":
		s.handlePolicyCreate(w, r)
	case r.Method == http.MethodPatch && id != "":
		s.handlePolicyPatch(w, r, id)
	case r.Method == http.MethodDelete && id != "":
		s.handlePolicyDelete(w, r, id)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePolicyGet returns full policy detail (limits, spam detection) for the
// policy editor.
func (s *Server) handlePolicyGet(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	policy, err := s.nylasClient.GetPolicy(ctx, id)
	if err != nil {
		writeMutationError(w, "Failed to load policy", err)
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// Policy mutations forward to the API unchecked: the plan ceiling is the
// billing plan itself, enforced server-side by Nylas — limits above the plan
// maximum are rejected upstream, and omitted limits default to the plan
// maximum. No policy is special; a workspace with no policy simply runs at
// plan maximums.

func (s *Server) handlePolicyCreate(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if !decodeBody(w, r, &payload) {
		return
	}
	if name, _ := payload["name"].(string); strings.TrimSpace(name) == "" {
		writeError(w, http.StatusBadRequest, "policy name is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	policy, err := s.nylasClient.CreatePolicy(ctx, payload)
	if err != nil {
		writeMutationError(w, "Failed to create policy", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusCreated, policy.ID)
}

func (s *Server) handlePolicyPatch(w http.ResponseWriter, r *http.Request, id string) {
	var payload map[string]any
	if !decodeBody(w, r, &payload) {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if _, err := s.nylasClient.UpdatePolicy(ctx, id, payload); err != nil {
		writeMutationError(w, "Failed to update policy", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handlePolicyDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if err := s.nylasClient.DeletePolicy(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete policy", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}
