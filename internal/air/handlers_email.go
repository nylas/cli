package air

import (
	"log/slog"
	"net/http"
	"strings"

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
		writeUpstreamError(w, http.StatusInternalServerError,
			"Failed to fetch emails — please try again", err,
			"account", redactEmail(accountEmail))
		return
	}

	// Cache the results. Cache write failures must not fail the request
	// (the user already has the data), but a silently-wedged cache will
	// drift further from server state on every refresh, so we log the
	// first put error per request to keep the failure debuggable.
	if s.cacheAvailable() {
		if cacheErr := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			var firstErr error
			for i := range result.Data {
				if putErr := store.Put(domainMessageToCached(&result.Data[i])); putErr != nil && firstErr == nil {
					firstErr = putErr
				}
			}
			return firstErr
		}); cacheErr != nil {
			slog.Warn("email list cache fill failed", "account", redactEmail(accountEmail), "err", cacheErr)
		}
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
		writeError(w, http.StatusBadRequest, "Email ID required")
		return
	}
	emailID := parts[0]

	// Sub-resource: /api/emails/{id}/invite returns parsed iCalendar
	// invite data so the email preview can show a Gmail-style RSVP card.
	if len(parts) > 1 && parts[1] == "invite" {
		s.handleEmailInvite(w, r, emailID)
		return
	}
	// Sub-resource: /api/emails/{id}/rsvp accepts {status: yes|no|maybe}
	// and forwards to the Nylas send-rsvp endpoint after resolving the
	// invite's iCalendar UID to a Nylas event.
	if len(parts) > 1 && parts[1] == "rsvp" {
		s.handleEmailRSVP(w, r, emailID)
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
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
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
		writeUpstreamError(w, http.StatusInternalServerError,
			"Failed to fetch email — please try again", err,
			"emailID", emailID, "account", redactEmail(accountEmail))
		return
	}

	// Cache the result. A wedged single-message cache silently drifts
	// from server state on every fetch otherwise; mirror the
	// handleListEmails:Cache fill failed log so support can diagnose
	// from production logs without changing the user-facing 200.
	if s.cacheAvailable() {
		if err := s.withEmailStore(accountEmail, func(store *cache.EmailStore) error {
			return store.Put(domainMessageToCached(msg))
		}); err != nil {
			slog.Warn("get-email cache fill failed",
				"emailID", emailID,
				"account", redactEmail(accountEmail),
				"err", err)
		}
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
		} else {
			// Offline AND the queue is broken. Falling through to the live
			// API call is a best-effort retry (IsOnline() can be stale),
			// but the queue failure itself must not be invisible — a
			// silently-wedged queue will drop user actions repeatedly.
			slog.Warn("offline enqueue failed, attempting live API call",
				"emailID", emailID,
				"account", redactEmail(accountEmail),
				"err", err,
			)
		}
	}

	_, err := s.nylasClient.UpdateMessage(ctx, grantID, emailID, updateReq)
	if err != nil {
		if s.shouldQueueEmailAction(err) {
			queueErr := s.enqueueMessageUpdate(grantID, accountEmail, emailID, updateReq)
			if queueErr == nil {
				s.SetOnline(false)
				s.updateCachedEmail(accountEmail, emailID, req.Unread, req.Starred, req.Folders)
				writeJSON(w, http.StatusOK, UpdateEmailResponse{
					Success: true,
					Message: "Email update queued until connection is restored",
				})
				return
			}
			// Queue write failed under a known-transient upstream error —
			// the user is about to see a 500, but they also lost the
			// fallback path that would have stashed their action. Log so
			// the queue health regression is debuggable.
			slog.Error("queue enqueue after transient API error failed",
				"emailID", emailID,
				"account", redactEmail(accountEmail),
				"apiErr", err,
				"queueErr", queueErr,
			)
		}
		// UpdateEmailResponse envelope — frontend reads `error`. Raw err
		// is logged, generic message goes to the user.
		slog.Error("Failed to update email",
			"err", err,
			"emailID", emailID,
			"account", redactEmail(accountEmail),
		)
		writeJSON(w, http.StatusInternalServerError, UpdateEmailResponse{
			Success: false,
			Error:   "Failed to update email — please try again",
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
		} else {
			// Offline AND the queue is broken. Falling through to the live
			// API call is a best-effort retry (IsOnline() can be stale),
			// but the queue failure itself must not be invisible — a
			// silently-wedged queue will drop user actions repeatedly.
			// Mirrors handleUpdateEmail's offline-enqueue log.
			slog.Warn("offline enqueue failed, attempting live API call",
				"emailID", emailID,
				"account", redactEmail(accountEmail),
				"err", err,
			)
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
			} else {
				// Queue write failed under a known-transient upstream
				// error. The user is about to see a 500 AND lost the
				// fallback path. Co-log apiErr + queueErr so the
				// double-failure is debuggable. Mirrors handleUpdateEmail.
				slog.Error("queue enqueue after transient API error failed",
					"emailID", emailID,
					"account", redactEmail(accountEmail),
					"apiErr", err,
					"queueErr", queueErr,
				)
			}
		}
		slog.Error("Failed to delete email",
			"err", err,
			"emailID", emailID,
			"account", redactEmail(accountEmail),
		)
		writeJSON(w, http.StatusInternalServerError, UpdateEmailResponse{
			Success: false,
			Error:   "Failed to delete email — please try again",
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
