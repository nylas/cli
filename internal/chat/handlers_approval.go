package chat

import (
	"net/http"

	"github.com/nylas/cli/internal/httputil"
)

// approvalRequest is the body for approve/reject endpoints.
type approvalRequest struct {
	ApprovalID string `json:"approval_id"`
	Reason     string `json:"reason,omitempty"`
}

// handleApprove approves a pending tool call.
// POST /api/chat/approve
func (s *Server) handleApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req approvalRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ApprovalID == "" {
		http.Error(w, "approval_id is required", http.StatusBadRequest)
		return
	}

	if !s.approvals.Resolve(req.ApprovalID, ApprovalDecision{Approved: true}) {
		http.Error(w, "approval not found or already resolved", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// handleReject rejects a pending tool call.
// POST /api/chat/reject
func (s *Server) handleReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req approvalRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ApprovalID == "" {
		http.Error(w, "approval_id is required", http.StatusBadRequest)
		return
	}

	reason := req.Reason
	if reason == "" {
		reason = "rejected by user"
	}

	if !s.approvals.Resolve(req.ApprovalID, ApprovalDecision{Approved: false, Reason: reason}) {
		http.Error(w, "approval not found or already resolved", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
