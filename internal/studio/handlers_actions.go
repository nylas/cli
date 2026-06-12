package studio

import (
	"net/http"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

const testEmailCooldown = 30 * time.Second

// allowTestEmail enforces a per-grant cooldown so repeated clicks (or a
// misbehaving page) cannot burn plan send quota.
func (s *Server) allowTestEmail(grantID string) bool {
	s.testEmailMu.Lock()
	defer s.testEmailMu.Unlock()
	if last, ok := s.testEmailLast[grantID]; ok && time.Since(last) < testEmailCooldown {
		return false
	}
	if s.testEmailLast == nil {
		s.testEmailLast = make(map[string]time.Time)
	}
	s.testEmailLast[grantID] = time.Now()
	return true
}

// handleTestEmail sends a self-addressed test message from an agent account so
// users can confirm the mailbox works end to end.
func (s *Server) handleTestEmail(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var body struct {
		GrantID string `json:"grant_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	body.GrantID = strings.TrimSpace(body.GrantID)
	if body.GrantID == "" {
		writeError(w, http.StatusBadRequest, "grant_id is required")
		return
	}
	if !s.allowTestEmail(body.GrantID) {
		writeError(w, http.StatusTooManyRequests, "test email cooldown: wait 30 seconds between sends per account")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	account, err := s.nylasClient.GetAgentAccount(ctx, body.GrantID)
	if err != nil {
		writeMutationError(w, "Failed to load agent account", err)
		return
	}

	_, err = s.nylasClient.SendMessage(ctx, body.GrantID, &domain.SendMessageRequest{
		Subject: "Agent Studio test email",
		Body:    "This test message was sent from Agent Studio to confirm the mailbox works.",
		To:      []domain.EmailParticipant{{Email: account.Email}},
	})
	if err != nil {
		writeMutationError(w, "Failed to send test email", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent", "to": account.Email})
}
