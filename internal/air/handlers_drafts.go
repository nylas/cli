package air

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// errSendGrantNotFound indicates the caller-supplied grant_id is not a
// stored, Air-supported grant for this user.
var errSendGrantNotFound = errors.New("grant not found or not supported by Air")

// handleDrafts handles POST /api/drafts (create) and GET /api/drafts (list).
func (s *Server) handleDrafts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListDrafts(w, r)
	case http.MethodPost:
		s.handleCreateDraft(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListDrafts returns all drafts.
func (s *Server) handleListDrafts(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, DraftsResponse{Drafts: demoDrafts()})
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	drafts, err := s.nylasClient.GetDrafts(ctx, grantID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch drafts: " + err.Error(),
		})
		return
	}

	// Convert to response format
	resp := DraftsResponse{
		Drafts: make([]DraftResponse, 0, len(drafts)),
	}
	for _, d := range drafts {
		resp.Drafts = append(resp.Drafts, draftToResponse(d))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleCreateDraft creates a new draft.
func (s *Server) handleCreateDraft(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, DraftResponse{ID: "demo-draft-new", Subject: "New Draft", Date: time.Now().Unix()})
	if grantID == "" {
		return
	}

	var req DraftRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	createReq := &domain.CreateDraftRequest{
		Subject:      req.Subject,
		Body:         req.Body,
		To:           participantsToEmail(req.To),
		Cc:           participantsToEmail(req.Cc),
		Bcc:          participantsToEmail(req.Bcc),
		ReplyToMsgID: req.ReplyToMsgID,
	}

	draft, err := s.nylasClient.CreateDraft(ctx, grantID, createReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create draft: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, draftToResponse(*draft))
}

// handleDraftByID handles single draft operations: GET, PUT, DELETE, and POST .../send.
func (s *Server) handleDraftByID(w http.ResponseWriter, r *http.Request) {
	// Parse draft ID from path: /api/drafts/{id} or /api/drafts/{id}/send
	path := strings.TrimPrefix(r.URL.Path, "/api/drafts/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Draft ID required", http.StatusBadRequest)
		return
	}
	draftID := parts[0]

	// Check for /send action
	if len(parts) > 1 && parts[1] == "send" && r.Method == http.MethodPost {
		s.handleSendDraft(w, r, draftID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetDraft(w, r, draftID)
	case http.MethodPut:
		s.handleUpdateDraft(w, r, draftID)
	case http.MethodDelete:
		s.handleDeleteDraft(w, r, draftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetDraft retrieves a single draft.
func (s *Server) handleGetDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	// Special demo mode handling: return specific draft or 404
	if s.demoMode {
		for _, d := range demoDrafts() {
			if d.ID == draftID {
				writeJSON(w, http.StatusOK, d)
				return
			}
		}
		writeError(w, http.StatusNotFound, "Draft not found")
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	draft, err := s.nylasClient.GetDraft(ctx, grantID, draftID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch draft: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, draftToResponse(*draft))
}

// handleUpdateDraft updates an existing draft.
func (s *Server) handleUpdateDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	grantID := s.withAuthGrant(w, DraftResponse{ID: draftID, Subject: "Updated Draft", Date: time.Now().Unix()})
	if grantID == "" {
		return
	}

	var req DraftRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	updateReq := &domain.CreateDraftRequest{
		Subject:      req.Subject,
		Body:         req.Body,
		To:           participantsToEmail(req.To),
		Cc:           participantsToEmail(req.Cc),
		Bcc:          participantsToEmail(req.Bcc),
		ReplyToMsgID: req.ReplyToMsgID,
	}

	draft, err := s.nylasClient.UpdateDraft(ctx, grantID, draftID, updateReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to update draft: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, draftToResponse(*draft))
}

// handleDeleteDraft deletes a draft.
func (s *Server) handleDeleteDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	grantID := s.withAuthGrant(w, UpdateEmailResponse{Success: true, Message: "Draft deleted (demo mode)"})
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	err := s.nylasClient.DeleteDraft(ctx, grantID, draftID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, UpdateEmailResponse{
			Success: false,
			Error:   "Failed to delete draft: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, UpdateEmailResponse{
		Success: true,
		Message: "Draft deleted",
	})
}

// handleSendDraft sends a draft.
func (s *Server) handleSendDraft(w http.ResponseWriter, r *http.Request, draftID string) {
	grantID := s.withAuthGrant(w, SendMessageResponse{Success: true, MessageID: "demo-sent-" + draftID, Message: "Email sent (demo mode)"})
	if grantID == "" {
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	msg, err := s.nylasClient.SendDraft(ctx, grantID, draftID, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SendMessageResponse{
			Success: false,
			Error:   "Failed to send draft: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, SendMessageResponse{
		Success:   true,
		MessageID: msg.ID,
		Message:   "Email sent successfully",
	})
}

// resolveSendGrantID returns the grant ID to send from. If the request supplies
// a grant_id it must match one of the user's stored, Air-supported grants
// (errSendGrantNotFound otherwise). When unset, the active default grant is
// used so older clients keep working.
func (s *Server) resolveSendGrantID(requestedGrantID, defaultGrantID string) (string, error) {
	if requestedGrantID == "" {
		return defaultGrantID, nil
	}
	supported, err := s.listSupportedGrants()
	if err != nil {
		return "", err
	}
	for i := range supported {
		if supported[i].ID == requestedGrantID {
			return requestedGrantID, nil
		}
	}
	return "", errSendGrantNotFound
}

// sendMessageForGrant sends via the per-grant /v3/grants/{id}/messages/send
// endpoint for every provider, populating From for Nylas-managed grants since
// the API rejects the request otherwise. Per-grant send is what archives the
// message to the sender's Sent folder; the transactional /v3/domains/...
// endpoint is a relay that does not archive, so it is *not* used here even
// for Nylas-provider grants.
func (s *Server) sendMessageForGrant(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	if len(req.From) == 0 {
		grant, err := s.nylasClient.GetGrant(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("fetch grant: %w", err)
		}
		if grant != nil && grant.Provider == domain.ProviderNylas && grant.Email != "" {
			req.From = []domain.EmailParticipant{{Email: grant.Email}}
		}
	}
	return s.nylasClient.SendMessage(ctx, grantID, req)
}

// handleSendMessage sends a message directly without creating a draft first.
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	defaultGrantID := s.withAuthGrant(w, SendMessageResponse{Success: true, MessageID: "demo-sent-" + time.Now().Format("20060102150405"), Message: "Email sent (demo mode)"})
	if defaultGrantID == "" {
		return
	}

	var req SendMessageRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	if len(req.To) == 0 {
		writeJSON(w, http.StatusBadRequest, SendMessageResponse{
			Success: false,
			Error:   "At least one recipient is required",
		})
		return
	}

	grantID, err := s.resolveSendGrantID(req.GrantID, defaultGrantID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errSendGrantNotFound) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, SendMessageResponse{
			Success: false,
			Error:   "Invalid sender account: " + err.Error(),
		})
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	sendReq := &domain.SendMessageRequest{
		Subject:      req.Subject,
		Body:         req.Body,
		To:           participantsToEmail(req.To),
		Cc:           participantsToEmail(req.Cc),
		Bcc:          participantsToEmail(req.Bcc),
		ReplyToMsgID: req.ReplyToMsgID,
	}

	msg, err := s.sendMessageForGrant(ctx, grantID, sendReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SendMessageResponse{
			Success: false,
			Error:   "Failed to send message: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, SendMessageResponse{
		Success:   true,
		MessageID: msg.ID,
		Message:   "Email sent successfully",
	})
}

// draftToResponse converts a domain draft to an API response.
func draftToResponse(d domain.Draft) DraftResponse {
	resp := DraftResponse{
		ID:      d.ID,
		Subject: d.Subject,
		Body:    d.Body,
		Date:    d.CreatedAt.Unix(),
	}

	for _, p := range d.To {
		resp.To = append(resp.To, EmailParticipantResponse{
			Name:  p.Name,
			Email: p.Email,
		})
	}
	for _, p := range d.Cc {
		resp.Cc = append(resp.Cc, EmailParticipantResponse{
			Name:  p.Name,
			Email: p.Email,
		})
	}
	for _, p := range d.Bcc {
		resp.Bcc = append(resp.Bcc, EmailParticipantResponse{
			Name:  p.Name,
			Email: p.Email,
		})
	}

	return resp
}

// participantsToEmail converts API participants to domain email participants.
func participantsToEmail(participants []EmailParticipantResponse) []domain.EmailParticipant {
	result := make([]domain.EmailParticipant, 0, len(participants))
	for _, p := range participants {
		result = append(result, domain.EmailParticipant{
			Name:  p.Name,
			Email: p.Email,
		})
	}
	return result
}

// demoDrafts returns demo draft data.
func demoDrafts() []DraftResponse {
	now := time.Now()
	return []DraftResponse{
		{
			ID:      "demo-draft-001",
			Subject: "Re: Project Update",
			Body:    "<p>Thanks for the update. I'll review and get back to you.</p>",
			To:      []EmailParticipantResponse{{Name: "Sarah Chen", Email: "sarah@example.com"}},
			Date:    now.Add(-1 * time.Hour).Unix(),
		},
		{
			ID:      "demo-draft-002",
			Subject: "Meeting Follow-up",
			Body:    "<p>Hi team,</p><p>Following up on our discussion...</p>",
			To:      []EmailParticipantResponse{{Name: "Team", Email: "team@example.com"}},
			Date:    now.Add(-2 * time.Hour).Unix(),
		},
	}
}

// demoFolders returns demo folder data.
func demoFolders() []FolderResponse {
	return []FolderResponse{
		{ID: "inbox", Name: "Inbox", SystemFolder: "inbox", TotalCount: 156, UnreadCount: 23},
		{ID: "sent", Name: "Sent", SystemFolder: "sent", TotalCount: 89, UnreadCount: 0},
		{ID: "drafts", Name: "Drafts", SystemFolder: "drafts", TotalCount: 3, UnreadCount: 0},
		{ID: "trash", Name: "Trash", SystemFolder: "trash", TotalCount: 12, UnreadCount: 0},
		{ID: "spam", Name: "Spam", SystemFolder: "spam", TotalCount: 5, UnreadCount: 0},
		{ID: "archive", Name: "Archive", SystemFolder: "archive", TotalCount: 234, UnreadCount: 0},
		{ID: "starred", Name: "Starred", SystemFolder: "", TotalCount: 8, UnreadCount: 0},
	}
}
