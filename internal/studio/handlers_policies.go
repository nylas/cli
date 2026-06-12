package studio

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
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
// ceiling-bounded editor.
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

	if !s.validateAgainstCeiling(ctx, w, payload) {
		return
	}

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

	if !s.requireMutablePolicy(ctx, w, id) {
		return
	}
	if !s.validateAgainstCeiling(ctx, w, payload) {
		return
	}

	if _, err := s.nylasClient.UpdatePolicy(ctx, id, payload); err != nil {
		writeMutationError(w, "Failed to update policy", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

func (s *Server) handlePolicyDelete(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if !s.requireMutablePolicy(ctx, w, id) {
		return
	}

	if err := s.nylasClient.DeletePolicy(ctx, id); err != nil {
		writeMutationError(w, "Failed to delete policy", err)
		return
	}

	s.respondMutation(ctx, w, http.StatusOK, id)
}

// planCeilingPolicyID identifies the plan-ceiling policy: the one attached to
// the default workspace. It is immutable and its limits bound custom policies.
func (s *Server) planCeilingPolicyID(ctx context.Context) (string, error) {
	workspaces, err := s.nylasClient.ListWorkspaces(ctx)
	if err != nil {
		return "", err
	}
	for _, ws := range workspaces {
		if ws.Default {
			return strings.TrimSpace(ws.PolicyID), nil
		}
	}
	return "", nil
}

// requireMutablePolicy rejects writes against the plan-ceiling policy.
func (s *Server) requireMutablePolicy(ctx context.Context, w http.ResponseWriter, id string) bool {
	ceilingID, err := s.planCeilingPolicyID(ctx)
	if err != nil {
		writeMutationError(w, "Failed to resolve plan ceiling policy", err)
		return false
	}
	if ceilingID != "" && id == ceilingID {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error":   "default_policy_immutable",
			"message": "The default policy is your plan ceiling and cannot be modified",
		})
		return false
	}
	return true
}

// validateAgainstCeiling rejects numeric limits exceeding the plan ceiling.
func (s *Server) validateAgainstCeiling(ctx context.Context, w http.ResponseWriter, payload map[string]any) bool {
	limits, _ := payload["limits"].(map[string]any)
	if len(limits) == 0 {
		return true
	}

	ceilingID, err := s.planCeilingPolicyID(ctx)
	if err != nil || ceilingID == "" {
		// No resolvable ceiling: let the API enforce its own bounds.
		return true
	}
	ceiling, err := s.nylasClient.GetPolicy(ctx, ceilingID)
	if err != nil || ceiling == nil || ceiling.Limits == nil {
		return true
	}

	for field, max := range ceilingLimitValues(ceiling.Limits) {
		requested, ok := limits[field].(float64)
		if !ok {
			continue
		}
		if requested > max {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error":   "above_plan_ceiling",
				"message": fmt.Sprintf("%s exceeds the plan ceiling (%v > %v)", field, requested, max),
			})
			return false
		}
	}
	return true
}

// ceilingLimitValues flattens the ceiling policy's numeric limits into the
// wire field names used by policy payloads.
func ceilingLimitValues(limits *domain.PolicyLimits) map[string]float64 {
	out := make(map[string]float64, 7)
	if limits.LimitAttachmentSizeInBytes != nil {
		out["limit_attachment_size_limit"] = float64(*limits.LimitAttachmentSizeInBytes)
	}
	if limits.LimitAttachmentCount != nil {
		out["limit_attachment_count_limit"] = float64(*limits.LimitAttachmentCount)
	}
	if limits.LimitSizeTotalMimeInBytes != nil {
		out["limit_size_total_mime"] = float64(*limits.LimitSizeTotalMimeInBytes)
	}
	if limits.LimitStorageTotalInBytes != nil {
		out["limit_storage_total"] = float64(*limits.LimitStorageTotalInBytes)
	}
	if limits.LimitCountDailyMessagePerGrant != nil {
		out["limit_count_daily_message_per_grant"] = float64(*limits.LimitCountDailyMessagePerGrant)
	}
	if limits.LimitInboxRetentionPeriodInDays != nil {
		out["limit_inbox_retention_period"] = float64(*limits.LimitInboxRetentionPeriodInDays)
	}
	if limits.LimitSpamRetentionPeriodInDays != nil {
		out["limit_spam_retention_period"] = float64(*limits.LimitSpamRetentionPeriodInDays)
	}
	return out
}
