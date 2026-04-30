package air

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/nylas/cli/internal/air/cache"
	"github.com/nylas/cli/internal/domain"
)

// handleListEmails returns emails with optional filtering.
func (s *Server) handleListEmails(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Parse query parameters
	query := NewQueryParams(r.URL.Query())

	// Demo mode: filter the demo dataset by folder/unread/starred so users
	// can exercise the sidebar (Sent/Drafts/Archive/Trash) without a real
	// account. Without this, every folder showed the same Inbox set.
	if s.demoMode {
		filtered := filterDemoEmails(demoEmails(),
			query.Get("folder"),
			query.GetBool("unread"),
			query.GetBool("starred"),
		)
		writeJSON(w, http.StatusOK, EmailsResponse{Emails: filtered, HasMore: false})
		return
	}

	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	params := &domain.MessageQueryParams{
		Limit: query.GetLimit(50),
	}

	// Filter by folder
	folderID := query.Get("folder")
	if folderID != "" {
		params.In = []string{folderID}
	}

	// Filter by unread
	if query.GetBool("unread") {
		unreadBool := true
		params.Unread = &unreadBool
	}

	// Filter by starred
	if query.GetBool("starred") {
		starredBool := true
		params.Starred = &starredBool
	}

	// Search by sender email (from)
	fromFilter := query.Get("from")
	if fromFilter != "" {
		params.From = fromFilter
	}

	// Full-text search query
	searchQuery := query.Get("search")
	if searchQuery != "" {
		params.SearchQuery = searchQuery
	}

	// Cursor for pagination
	cursor := query.Get("cursor")
	if cursor != "" {
		params.PageToken = cursor
	}

	// Get account email for cache lookup
	accountEmail := s.getAccountEmail(grantID)

	// Try cache first (only for first page without complex filters).
	// Folder-filter caveat: background sync fetches the top-N messages with
	// no folder filter, so on busy inboxes the cache barely covers
	// Sent/Drafts/Archive. We short-circuit on a folder-filter hit only when
	// the cache returned at least a full page — otherwise the user sees a
	// stub of 1–2 messages instead of the real folder. from/search filters
	// operate on the full cached dataset, so they short-circuit as before.
	if cursor == "" && s.cacheAvailable() {
		var cached []*cache.CachedEmail
		if err := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			var err error
			cached, err = s.queryCachedEmails(store, params, folderID, fromFilter, searchQuery)
			return err
		}); err == nil && len(cached) > 0 {
			if folderID == "" || len(cached) >= params.Limit {
				writeJSON(w, http.StatusOK, cachedEmailsToResponse(cached, params.Limit))
				return
			}
		}
	}

	// Fetch messages from Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	result, err := s.nylasClient.GetMessagesWithCursor(ctx, grantID, params)
	if err != nil {
		// If offline and cache available, try cache as fallback
		if s.cacheAvailable() {
			var cached []*cache.CachedEmail
			if storeErr := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
				var cacheErr error
				cached, cacheErr = s.queryCachedEmails(store, params, folderID, fromFilter, searchQuery)
				return cacheErr
			}); storeErr == nil && len(cached) > 0 {
				resp := cachedEmailsToResponse(cached, params.Limit)
				resp.HasMore = false
				writeJSON(w, http.StatusOK, resp)
				return
			}
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch emails: " + err.Error(),
		})
		return
	}

	// Cache the results
	if s.cacheAvailable() {
		_ = s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			for i := range result.Data {
				_ = store.Put(domainMessageToCached(&result.Data[i]))
			}
			return nil
		})
	}

	// Convert to response format
	resp := EmailsResponse{
		Emails:     make([]EmailResponse, 0, len(result.Data)),
		NextCursor: result.Pagination.NextCursor,
		HasMore:    result.Pagination.HasMore,
	}
	for _, m := range result.Data {
		resp.Emails = append(resp.Emails, emailToResponse(m, false))
	}

	writeJSON(w, http.StatusOK, resp)
}

// handleEmailByID handles single email operations: GET, PUT, DELETE.
func (s *Server) handleEmailByID(w http.ResponseWriter, r *http.Request) {
	// Parse email ID from path: /api/emails/{id}[/{action}]
	path := strings.TrimPrefix(r.URL.Path, "/api/emails/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Email ID required", http.StatusBadRequest)
		return
	}
	emailID := parts[0]

	// Sub-resource: /api/emails/{id}/invite returns parsed iCalendar
	// invite data so the email preview can show a Gmail-style RSVP card.
	if len(parts) > 1 && parts[1] == "invite" {
		s.handleEmailInvite(w, r, emailID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetEmail(w, r, emailID)
	case http.MethodPut:
		s.handleUpdateEmail(w, r, emailID)
	case http.MethodDelete:
		s.handleDeleteEmail(w, r, emailID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetEmail retrieves a single email with full body.
func (s *Server) handleGetEmail(w http.ResponseWriter, r *http.Request, emailID string) {
	// Special demo mode: return specific email or 404
	if s.demoMode {
		for _, e := range demoEmails() {
			if e.ID == emailID {
				writeJSON(w, http.StatusOK, e)
				return
			}
		}
		writeError(w, http.StatusNotFound, "Email not found")
		return
	}
	grantID := s.withAuthGrant(w, nil) // Demo mode already handled above
	if grantID == "" {
		return
	}

	// Get account email for cache lookup
	accountEmail := s.getAccountEmail(grantID)

	// Try cache first
	if s.cacheAvailable() {
		var cached *cache.CachedEmail
		if err := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			var err error
			cached, err = store.Get(emailID)
			return err
		}); err == nil && cached != nil {
			resp := cachedEmailToResponse(cached)
			resp.Body = cached.BodyHTML // Include full body
			writeJSON(w, http.StatusOK, resp)
			return
		}
	}

	// Fetch message from Nylas API
	ctx, cancel := s.withTimeout(r)
	defer cancel()

	msg, err := s.nylasClient.GetMessage(ctx, grantID, emailID)
	if err != nil {
		// Try cache as fallback on error
		if s.cacheAvailable() {
			var cached *cache.CachedEmail
			if storeErr := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
				var cacheErr error
				cached, cacheErr = store.Get(emailID)
				return cacheErr
			}); storeErr == nil && cached != nil {
				resp := cachedEmailToResponse(cached)
				resp.Body = cached.BodyHTML
				writeJSON(w, http.StatusOK, resp)
				return
			}
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch email: " + err.Error(),
		})
		return
	}

	// Cache the result
	if s.cacheAvailable() {
		_ = s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			return store.Put(domainMessageToCached(msg))
		})
	}

	writeJSON(w, http.StatusOK, emailToResponse(*msg, true))
}

// handleUpdateEmail updates an email (mark read/unread, star/unstar).
func (s *Server) handleUpdateEmail(w http.ResponseWriter, r *http.Request, emailID string) {
	grantID := s.withAuthGrant(w, UpdateEmailResponse{Success: true, Message: "Email updated (demo mode)"})
	if grantID == "" {
		return
	}

	var req UpdateEmailRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	accountEmail := s.getAccountEmail(grantID)

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	updateReq := &domain.UpdateMessageRequest{
		Unread:  req.Unread,
		Starred: req.Starred,
		Folders: req.Folders,
	}

	if !s.IsOnline() {
		if err := s.enqueueMessageUpdate(grantID, accountEmail, emailID, updateReq); err == nil {
			s.updateCachedEmail(accountEmail, emailID, req.Unread, req.Starred, req.Folders)
			writeJSON(w, http.StatusOK, UpdateEmailResponse{
				Success: true,
				Message: "Email update queued until connection is restored",
			})
			return
		}
	}

	_, err := s.nylasClient.UpdateMessage(ctx, grantID, emailID, updateReq)
	if err != nil {
		if s.shouldQueueEmailAction(err) {
			if queueErr := s.enqueueMessageUpdate(grantID, accountEmail, emailID, updateReq); queueErr == nil {
				s.SetOnline(false)
				s.updateCachedEmail(accountEmail, emailID, req.Unread, req.Starred, req.Folders)
				writeJSON(w, http.StatusOK, UpdateEmailResponse{
					Success: true,
					Message: "Email update queued until connection is restored",
				})
				return
			}
		}
		writeJSON(w, http.StatusInternalServerError, UpdateEmailResponse{
			Success: false,
			Error:   "Failed to update email: " + err.Error(),
		})
		return
	}

	s.updateCachedEmail(accountEmail, emailID, req.Unread, req.Starred, req.Folders)

	writeJSON(w, http.StatusOK, UpdateEmailResponse{
		Success: true,
		Message: "Email updated",
	})
}

// handleDeleteEmail moves an email to trash.
func (s *Server) handleDeleteEmail(w http.ResponseWriter, r *http.Request, emailID string) {
	grantID := s.withAuthGrant(w, UpdateEmailResponse{Success: true, Message: "Email deleted (demo mode)"})
	if grantID == "" {
		return
	}

	accountEmail := s.getAccountEmail(grantID)

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	if !s.IsOnline() {
		if err := s.enqueueMessageDelete(grantID, accountEmail, emailID); err == nil {
			s.deleteCachedEmail(accountEmail, emailID)
			writeJSON(w, http.StatusOK, UpdateEmailResponse{
				Success: true,
				Message: "Email delete queued until connection is restored",
			})
			return
		}
	}

	err := s.nylasClient.DeleteMessage(ctx, grantID, emailID)
	if err != nil {
		if s.shouldQueueEmailAction(err) {
			if queueErr := s.enqueueMessageDelete(grantID, accountEmail, emailID); queueErr == nil {
				s.SetOnline(false)
				s.deleteCachedEmail(accountEmail, emailID)
				writeJSON(w, http.StatusOK, UpdateEmailResponse{
					Success: true,
					Message: "Email delete queued until connection is restored",
				})
				return
			}
		}
		writeJSON(w, http.StatusInternalServerError, UpdateEmailResponse{
			Success: false,
			Error:   "Failed to delete email: " + err.Error(),
		})
		return
	}

	s.deleteCachedEmail(accountEmail, emailID)

	writeJSON(w, http.StatusOK, UpdateEmailResponse{
		Success: true,
		Message: "Email deleted",
	})
}

func (s *Server) queryCachedEmails(store *cache.EmailStore, params *domain.MessageQueryParams, folderID, fromFilter, searchQuery string) ([]*cache.CachedEmail, error) {
	if searchQuery == "" && fromFilter == "" {
		return store.List(cache.ListOptions{
			Limit:       params.Limit,
			FolderID:    folderID,
			UnreadOnly:  params.Unread != nil && *params.Unread,
			StarredOnly: params.Starred != nil && *params.Starred,
		})
	}

	query := cache.ParseSearchQuery(searchQuery)
	if fromFilter != "" {
		query.From = fromFilter
	}
	if folderID != "" {
		query.In = folderID
	}
	if params.Unread != nil {
		query.IsUnread = params.Unread
	}
	if params.Starred != nil {
		query.IsStarred = params.Starred
	}

	return store.SearchWithQuery(query, params.Limit)
}

func cachedEmailsToResponse(cached []*cache.CachedEmail, limit int) EmailsResponse {
	resp := EmailsResponse{
		Emails:  make([]EmailResponse, 0, len(cached)),
		HasMore: limit > 0 && len(cached) >= limit,
	}
	for _, email := range cached {
		resp.Emails = append(resp.Emails, cachedEmailToResponse(email))
	}
	return resp
}

func (s *Server) shouldQueueEmailAction(err error) bool {
	if !s.offlineQueueEnabled() {
		return false
	}
	if !s.IsOnline() {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded)
}

func (s *Server) offlineQueueEnabled() bool {
	return s.offlineQueueConfigured()
}

func (s *Server) enqueueMessageUpdate(grantID, accountEmail, emailID string, updateReq *domain.UpdateMessageRequest) error {
	if accountEmail == "" || !s.offlineQueueEnabled() {
		return errors.New("offline queue unavailable")
	}

	return s.withOfflineQueue(accountEmail, func(queue *cache.OfflineQueue) error {
		return queue.Enqueue(cache.ActionUpdateMessage, emailID, cache.UpdateMessagePayload{
			GrantID: grantID,
			EmailID: emailID,
			Unread:  updateReq.Unread,
			Starred: updateReq.Starred,
			Folders: updateReq.Folders,
		})
	})
}

func (s *Server) enqueueMessageDelete(grantID, accountEmail, emailID string) error {
	if accountEmail == "" || !s.offlineQueueEnabled() {
		return errors.New("offline queue unavailable")
	}

	return s.withOfflineQueue(accountEmail, func(queue *cache.OfflineQueue) error {
		return queue.Enqueue(cache.ActionDelete, emailID, cache.DeleteMessagePayload{
			GrantID: grantID,
			EmailID: emailID,
		})
	})
}

func (s *Server) updateCachedEmail(accountEmail, emailID string, unread, starred *bool, folders []string) {
	if accountEmail == "" || !s.cacheAvailable() {
		return
	}

	_ = s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
		return store.UpdateMessage(emailID, unread, starred, folders)
	})
}

func (s *Server) deleteCachedEmail(accountEmail, emailID string) {
	if accountEmail == "" || !s.cacheAvailable() {
		return
	}

	_ = s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
		return store.Delete(emailID)
	})
}

// emailToResponse converts a domain message to an API response.
func emailToResponse(m domain.Message, includeBody bool) EmailResponse {
	resp := EmailResponse{
		ID:       m.ID,
		ThreadID: m.ThreadID,
		Subject:  m.Subject,
		Snippet:  m.Snippet,
		Date:     m.Date.Unix(),
		Unread:   m.Unread,
		Starred:  m.Starred,
		Folders:  m.Folders,
	}

	if includeBody {
		resp.Body = m.Body
	}

	// Convert participants with pre-allocated slices
	if len(m.From) > 0 {
		resp.From = make([]EmailParticipantResponse, 0, len(m.From))
		for _, p := range m.From {
			resp.From = append(resp.From, EmailParticipantResponse{
				Name:  p.Name,
				Email: p.Email,
			})
		}
	}
	if len(m.To) > 0 {
		resp.To = make([]EmailParticipantResponse, 0, len(m.To))
		for _, p := range m.To {
			resp.To = append(resp.To, EmailParticipantResponse{
				Name:  p.Name,
				Email: p.Email,
			})
		}
	}
	if len(m.Cc) > 0 {
		resp.Cc = make([]EmailParticipantResponse, 0, len(m.Cc))
		for _, p := range m.Cc {
			resp.Cc = append(resp.Cc, EmailParticipantResponse{
				Name:  p.Name,
				Email: p.Email,
			})
		}
	}

	// Convert attachments with pre-allocated slice
	if len(m.Attachments) > 0 {
		resp.Attachments = make([]AttachmentResponse, 0, len(m.Attachments))
		for _, a := range m.Attachments {
			resp.Attachments = append(resp.Attachments, AttachmentResponse{
				ID:          a.ID,
				Filename:    a.Filename,
				ContentType: a.ContentType,
				Size:        a.Size,
			})
		}
	}

	return resp
}

// cachedEmailToResponse converts a cached email to response format.
func cachedEmailToResponse(e *cache.CachedEmail) EmailResponse {
	return EmailResponse{
		ID:       e.ID,
		ThreadID: e.ThreadID,
		Subject:  e.Subject,
		Snippet:  e.Snippet,
		From: []EmailParticipantResponse{
			{Name: e.FromName, Email: e.FromEmail},
		},
		Date:    e.Date.Unix(),
		Unread:  e.Unread,
		Starred: e.Starred,
		Folders: []string{e.FolderID},
	}
}

// demoEmails returns demo email data spread across multiple folders so the
// sidebar (Inbox / Sent / Drafts / Archive / Trash) actually shows different
// content per folder. Includes one calendar-invite (.ics) email so the
// calendar-invite card UI has something to render.
func demoEmails() []EmailResponse {
	now := time.Now()
	return []EmailResponse{
		// Inbox
		{
			ID:      "demo-email-001",
			Subject: "Q4 Product Roadmap Review",
			Snippet: "Hi team, I've attached the updated roadmap for Q4...",
			Body:    "<p>Hi team,</p><p>I've attached the updated roadmap for Q4. Please review the timeline changes and let me know if you have any concerns.</p>",
			From:    []EmailParticipantResponse{{Name: "Sarah Chen", Email: "sarah.chen@company.com"}},
			To:      []EmailParticipantResponse{{Name: "Team", Email: "team@company.com"}},
			Date:    now.Add(-2 * time.Minute).Unix(),
			Unread:  true,
			Starred: true,
			Folders: []string{"inbox"},
			Attachments: []AttachmentResponse{
				{ID: "att-001", Filename: "Q4_Roadmap_v2.pdf", ContentType: "application/pdf", Size: 2516582},
			},
		},
		{
			ID:      "demo-email-002",
			Subject: "[nylas/cli] PR #142: Add focus time feature",
			Snippet: "mergify[bot] merged 1 commit into main...",
			From:    []EmailParticipantResponse{{Name: "GitHub", Email: "notifications@github.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-15 * time.Minute).Unix(),
			Unread:  true,
			Starred: false,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-003",
			Subject: "Re: Meeting Tomorrow",
			Snippet: "That works for me. I'll send a calendar invite...",
			From:    []EmailParticipantResponse{{Name: "Alex Johnson", Email: "demo@example.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-1 * time.Hour).Unix(),
			Unread:  false,
			Starred: false,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-004",
			Subject: "Your December invoice is ready",
			Snippet: "Your invoice for December 2024 is now available...",
			From:    []EmailParticipantResponse{{Name: "Stripe", Email: "billing@stripe.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-3 * time.Hour).Unix(),
			Unread:  false,
			Starred: true,
			Folders: []string{"inbox"},
		},
		{
			ID:      "demo-email-005",
			Subject: "This week in design: AI tools reshaping...",
			Snippet: "The latest trends, tools, and inspiration...",
			From:    []EmailParticipantResponse{{Name: "Design Weekly", Email: "newsletter@designweekly.com"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-5 * time.Hour).Unix(),
			Unread:  false,
			Starred: false,
			Folders: []string{"inbox"},
		},
		// Calendar invite (with .ics attachment)
		{
			ID:      "demo-email-invite-001",
			Subject: "Event Invitation: Quarterly Sync",
			Snippet: "You have received a calendar invitation: Quarterly Sync",
			Body:    "<p>You have received a calendar invitation: <strong>Quarterly Sync</strong></p><p>Please let me know if this time works.</p>",
			From:    []EmailParticipantResponse{{Name: "Priya Patel", Email: "priya@partner.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-30 * time.Minute).Unix(),
			Unread:  true,
			Starred: false,
			Folders: []string{"inbox"},
			Attachments: []AttachmentResponse{
				{
					ID:          "att-invite-001",
					Filename:    "invite.ics",
					ContentType: "text/calendar",
					Size:        1024,
				},
			},
		},
		// Sent — explicitly more than one so we can prove the filter works.
		{
			ID:      "demo-email-sent-001",
			Subject: "Re: Q4 Product Roadmap Review",
			Snippet: "Thanks Sarah, here are my comments on the roadmap...",
			Body:    "<p>Thanks Sarah,</p><p>Here are my comments on the roadmap. Looks good overall — happy to discuss the Q4 priorities live.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Sarah Chen", Email: "sarah.chen@company.com"}},
			Date:    now.Add(-1 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		{
			ID:      "demo-email-sent-002",
			Subject: "Pricing follow-up",
			Snippet: "Hi Mike, sending the updated pricing sheet...",
			Body:    "<p>Hi Mike,</p><p>Sending the updated pricing sheet as discussed. Let me know if you need any changes.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Mike Johnson", Email: "mike@customer.example"}},
			Date:    now.Add(-3 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		{
			ID:      "demo-email-sent-003",
			Subject: "Welcome to the team!",
			Snippet: "Excited to have you joining us next Monday...",
			Body:    "<p>Excited to have you joining us next Monday! Here's the on-boarding checklist.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Jamie Lee", Email: "jamie@newhire.example"}},
			Date:    now.Add(-1 * 24 * time.Hour).Unix(),
			Folders: []string{"sent"},
		},
		// Drafts
		{
			ID:      "demo-email-draft-001",
			Subject: "Draft: Proposal for Acme",
			Snippet: "Hi Acme team, here's the rough proposal...",
			Body:    "<p>Hi Acme team,</p><p>Here's the rough proposal — still working through the timeline section.</p>",
			From:    []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			To:      []EmailParticipantResponse{{Name: "Acme Procurement", Email: "buyers@acme.example"}},
			Date:    now.Add(-4 * time.Hour).Unix(),
			Folders: []string{"drafts"},
		},
		// Archive
		{
			ID:      "demo-email-archive-001",
			Subject: "Confirmation: Subscription renewed",
			Snippet: "Your annual subscription has been renewed...",
			Body:    "<p>Your annual subscription has been renewed for another year.</p>",
			From:    []EmailParticipantResponse{{Name: "Acme Billing", Email: "billing@acme.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-30 * 24 * time.Hour).Unix(),
			Folders: []string{"archive"},
		},
		// Trash
		{
			ID:      "demo-email-trash-001",
			Subject: "URGENT: Winning offer (don't miss out)",
			Snippet: "You've been pre-selected for an exclusive offer...",
			Body:    "<p>You've been pre-selected.</p>",
			From:    []EmailParticipantResponse{{Name: "Promo Bot", Email: "deals@spammy.example"}},
			To:      []EmailParticipantResponse{{Name: "You", Email: "you@example.com"}},
			Date:    now.Add(-7 * 24 * time.Hour).Unix(),
			Folders: []string{"trash"},
		},
	}
}

// filterDemoEmails applies folder/unread/starred filters to a demo email
// list. Folder matching is case-insensitive against email.Folders entries
// and against well-known aliases (e.g., "SENT" → "sent", "Sent Items"
// → "sent"). Empty folder string means "no folder filter."
func filterDemoEmails(emails []EmailResponse, folder string, onlyUnread, onlyStarred bool) []EmailResponse {
	target := normalizeDemoFolder(folder)
	out := make([]EmailResponse, 0, len(emails))
	for _, e := range emails {
		if onlyUnread && !e.Unread {
			continue
		}
		if onlyStarred && !e.Starred {
			continue
		}
		if target != "" && !demoEmailIsInFolder(e, target) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// normalizeDemoFolder turns a UI-supplied folder identifier (e.g., the
// system-folder ID "SENT", the Microsoft display name "Sent Items", or the
// canonical "sent") into the lowercase canonical name used in demoEmails.
func normalizeDemoFolder(folder string) string {
	f := strings.ToLower(strings.TrimSpace(folder))
	switch f {
	case "":
		return ""
	case "inbox":
		return "inbox"
	case "sent", "sent items", "sent mail":
		return "sent"
	case "drafts", "draft":
		return "drafts"
	case "archive", "all", "all mail":
		return "archive"
	case "trash", "deleted items", "deleted":
		return "trash"
	case "spam", "junk", "junk email":
		return "spam"
	case "starred":
		return "starred"
	default:
		return f
	}
}

// demoEmailIsInFolder reports whether the email is in `target` (already
// canonicalised). Special-cases "starred" since starring is a flag, not a
// folder. "all" never filters anything out.
func demoEmailIsInFolder(e EmailResponse, target string) bool {
	if target == "starred" {
		return e.Starred
	}
	if target == "all" {
		return true
	}
	for _, f := range e.Folders {
		if strings.EqualFold(f, target) {
			return true
		}
	}
	return false
}
