package chat

import (
	"sync"
	"sync/atomic"
	"time"
)

// approvalTimeout is how long to wait for user approval before auto-rejecting.
const approvalTimeout = 5 * time.Minute

// gatedTools lists tools that require user approval before execution.
var gatedTools = map[string]bool{
	"send_email":   true,
	"create_event": true,
}

// IsGated returns true if the tool requires user approval.
func IsGated(toolName string) bool {
	return gatedTools[toolName]
}

// ApprovalDecision is the user's response to an approval request.
type ApprovalDecision struct {
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// PendingApproval represents a tool call waiting for user approval.
type PendingApproval struct {
	ID      string         `json:"id"`
	Tool    string         `json:"tool"`
	Args    map[string]any `json:"args"`
	Preview map[string]any `json:"preview"`
	ch      chan ApprovalDecision
}

// Wait blocks until the user approves/rejects or the timeout expires.
func (pa *PendingApproval) Wait() (ApprovalDecision, bool) {
	select {
	case decision := <-pa.ch:
		return decision, true
	case <-time.After(approvalTimeout):
		return ApprovalDecision{Approved: false, Reason: "timed out"}, false
	}
}

// ApprovalStore manages pending approval requests.
type ApprovalStore struct {
	pending sync.Map
	counter atomic.Int64
}

// NewApprovalStore creates a new ApprovalStore.
func NewApprovalStore() *ApprovalStore {
	return &ApprovalStore{}
}

// Create registers a new pending approval and returns it.
func (s *ApprovalStore) Create(call ToolCall, preview map[string]any) *PendingApproval {
	id := s.nextID()
	pa := &PendingApproval{
		ID:      id,
		Tool:    call.Name,
		Args:    call.Args,
		Preview: preview,
		ch:      make(chan ApprovalDecision, 1), // buffered so sender never blocks
	}
	s.pending.Store(id, pa)
	return pa
}

// Resolve sends a decision for a pending approval. Returns false if not found.
func (s *ApprovalStore) Resolve(id string, decision ApprovalDecision) bool {
	val, ok := s.pending.LoadAndDelete(id)
	if !ok {
		return false
	}
	pa := val.(*PendingApproval)
	pa.ch <- decision
	return true
}

// nextID generates a sequential approval ID.
func (s *ApprovalStore) nextID() string {
	n := s.counter.Add(1)
	return "approval_" + itoa(n)
}

// itoa converts int64 to string without importing strconv.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10)
		i--
		n /= 10
	}
	return string(buf[i+1:])
}

// BuildPreview creates a human-readable preview of a gated tool call.
func BuildPreview(call ToolCall) map[string]any {
	preview := make(map[string]any)

	switch call.Name {
	case "send_email":
		if to, ok := call.Args["to"].(string); ok {
			preview["to"] = to
		}
		if subj, ok := call.Args["subject"].(string); ok {
			preview["subject"] = subj
		}
		if body, ok := call.Args["body"].(string); ok {
			if len(body) > 200 {
				body = body[:200] + "..."
			}
			preview["body"] = body
		}
	case "create_event":
		if title, ok := call.Args["title"].(string); ok {
			preview["title"] = title
		}
		if start, ok := call.Args["start_time"].(string); ok {
			preview["start_time"] = start
		}
		if end, ok := call.Args["end_time"].(string); ok {
			preview["end_time"] = end
		}
		if desc, ok := call.Args["description"].(string); ok {
			preview["description"] = desc
		}
	}

	return preview
}
