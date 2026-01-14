package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

// threadResponse represents an API thread response.
type threadResponse struct {
	ID                    string `json:"id"`
	GrantID               string `json:"grant_id"`
	HasAttachments        bool   `json:"has_attachments"`
	HasDrafts             bool   `json:"has_drafts"`
	Starred               bool   `json:"starred"`
	Unread                bool   `json:"unread"`
	EarliestMessageDate   int64  `json:"earliest_message_date"`
	LatestMessageRecvDate int64  `json:"latest_message_received_date"`
	LatestMessageSentDate int64  `json:"latest_message_sent_date"`
	Participants          []struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"participants"`
	MessageIDs []string `json:"message_ids"`
	DraftIDs   []string `json:"draft_ids"`
	FolderIDs  []string `json:"folders"`
	Snippet    string   `json:"snippet"`
	Subject    string   `json:"subject"`
}

// GetThreads retrieves threads with query parameters.
func (c *HTTPClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	if params == nil {
		params = &domain.ThreadQueryParams{Limit: 10}
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/threads", c.baseURL, grantID)
	queryURL := NewQueryBuilder().
		AddInt("limit", params.Limit).
		AddInt("offset", params.Offset).
		Add("subject", params.Subject).
		Add("from", params.From).
		Add("to", params.To).
		AddBoolPtr("unread", params.Unread).
		AddBoolPtr("starred", params.Starred).
		Add("q", params.SearchQuery).
		BuildURL(baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data []threadResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return convertThreads(result.Data), nil
}

// GetThread retrieves a single thread by ID.
func (c *HTTPClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/threads/%s", c.baseURL, grantID, threadID)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: thread not found", domain.ErrAPIError)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result struct {
		Data threadResponse `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	thread := convertThread(result.Data)
	return &thread, nil
}

// UpdateThread updates thread properties.
func (c *HTTPClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/threads/%s", c.baseURL, grantID, threadID)

	payload := make(map[string]any)
	if req.Unread != nil {
		payload["unread"] = *req.Unread
	}
	if req.Starred != nil {
		payload["starred"] = *req.Starred
	}
	if len(req.Folders) > 0 {
		payload["folders"] = req.Folders
	}

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data threadResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	thread := convertThread(result.Data)
	return &thread, nil
}

// DeleteThread deletes a thread.
func (c *HTTPClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	if err := validateRequired("thread ID", threadID); err != nil {
		return err
	}
	queryURL := fmt.Sprintf("%s/v3/grants/%s/threads/%s", c.baseURL, grantID, threadID)
	return c.doDelete(ctx, queryURL)
}

// convertThreads converts API thread responses to domain models.
func convertThreads(threads []threadResponse) []domain.Thread {
	return util.Map(threads, convertThread)
}

// convertThread converts an API thread response to domain model.
func convertThread(t threadResponse) domain.Thread {
	participants := util.Map(t.Participants, func(p struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}) domain.EmailParticipant {
		return domain.EmailParticipant{Name: p.Name, Email: p.Email}
	})

	return domain.Thread{
		ID:                    t.ID,
		GrantID:               t.GrantID,
		HasAttachments:        t.HasAttachments,
		HasDrafts:             t.HasDrafts,
		Starred:               t.Starred,
		Unread:                t.Unread,
		EarliestMessageDate:   time.Unix(t.EarliestMessageDate, 0),
		LatestMessageRecvDate: time.Unix(t.LatestMessageRecvDate, 0),
		LatestMessageSentDate: time.Unix(t.LatestMessageSentDate, 0),
		Participants:          participants,
		MessageIDs:            t.MessageIDs,
		DraftIDs:              t.DraftIDs,
		FolderIDs:             t.FolderIDs,
		Snippet:               t.Snippet,
		Subject:               t.Subject,
	}
}
