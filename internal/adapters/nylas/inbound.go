package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

// ListInboundInboxes lists all inbound inboxes (grants with provider=inbox).
func (c *HTTPClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	// Get all grants and filter by provider=inbox
	baseURL := fmt.Sprintf("%s/v3/grants", c.baseURL)
	queryURL := NewQueryBuilder().Add("provider", "inbox").BuildURL(baseURL)

	var result struct {
		Data []struct {
			ID          string          `json:"id"`
			Email       string          `json:"email"`
			Provider    string          `json:"provider"`
			GrantStatus string          `json:"grant_status"`
			CreatedAt   domain.UnixTime `json:"created_at"`
			UpdatedAt   domain.UnixTime `json:"updated_at"`
		} `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	inboxes := make([]domain.InboundInbox, 0, len(result.Data))
	for _, g := range result.Data {
		// Only include inboxes with provider=inbox
		if g.Provider == "inbox" {
			inboxes = append(inboxes, domain.InboundInbox{
				ID:          g.ID,
				Email:       g.Email,
				GrantStatus: g.GrantStatus,
				CreatedAt:   g.CreatedAt,
				UpdatedAt:   g.UpdatedAt,
			})
		}
	}

	return inboxes, nil
}

// GetInboundInbox retrieves a specific inbound inbox by grant ID.
func (c *HTTPClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s", c.baseURL, grantID)

	var result struct {
		Data struct {
			ID          string          `json:"id"`
			Email       string          `json:"email"`
			Provider    string          `json:"provider"`
			GrantStatus string          `json:"grant_status"`
			CreatedAt   domain.UnixTime `json:"created_at"`
			UpdatedAt   domain.UnixTime `json:"updated_at"`
		} `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, fmt.Errorf("%w: inbound inbox not found", domain.ErrAPIError)); err != nil {
		return nil, err
	}

	// Verify it's an inbox provider
	if result.Data.Provider != "inbox" {
		return nil, fmt.Errorf("%w: grant is not an inbound inbox (provider=%s)", domain.ErrAPIError, result.Data.Provider)
	}

	return &domain.InboundInbox{
		ID:          result.Data.ID,
		Email:       result.Data.Email,
		GrantStatus: result.Data.GrantStatus,
		CreatedAt:   result.Data.CreatedAt,
		UpdatedAt:   result.Data.UpdatedAt,
	}, nil
}

// CreateInboundInbox creates a new inbound inbox with the given email address.
// The email parameter is the local part (e.g., "support" for support@app.nylas.email).
func (c *HTTPClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	queryURL := fmt.Sprintf("%s/v3/grants", c.baseURL)

	// Create the request payload for custom auth with inbox provider
	payload := map[string]any{
		"provider": "inbox",
		"settings": map[string]string{
			"email": email,
		},
	}

	resp, err := c.doJSONRequest(ctx, "POST", queryURL, payload)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			ID          string          `json:"id"`
			Email       string          `json:"email"`
			Provider    string          `json:"provider"`
			GrantStatus string          `json:"grant_status"`
			CreatedAt   domain.UnixTime `json:"created_at"`
			UpdatedAt   domain.UnixTime `json:"updated_at"`
		} `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	return &domain.InboundInbox{
		ID:          result.Data.ID,
		Email:       result.Data.Email,
		GrantStatus: result.Data.GrantStatus,
		CreatedAt:   result.Data.CreatedAt,
		UpdatedAt:   result.Data.UpdatedAt,
	}, nil
}

// DeleteInboundInbox deletes an inbound inbox by revoking its grant.
func (c *HTTPClient) DeleteInboundInbox(ctx context.Context, grantID string) error {
	// First verify it's an inbox provider
	inbox, err := c.GetInboundInbox(ctx, grantID)
	if err != nil {
		return err
	}
	if inbox == nil {
		return fmt.Errorf("%w: inbound inbox not found", domain.ErrAPIError)
	}

	// Use RevokeGrant to delete the inbox
	return c.RevokeGrant(ctx, grantID)
}

// GetInboundMessages retrieves messages for an inbound inbox.
func (c *HTTPClient) GetInboundMessages(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.InboundMessage, error) {
	if params == nil {
		params = &domain.MessageQueryParams{Limit: 10}
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}

	baseURL := fmt.Sprintf("%s/v3/grants/%s/messages", c.baseURL, grantID)
	queryURL := NewQueryBuilder().
		AddInt("limit", params.Limit).
		Add("page_token", params.PageToken).
		AddInt("offset", params.Offset).
		Add("subject", params.Subject).
		Add("from", params.From).
		AddBoolPtr("unread", params.Unread).
		AddInt64("received_before", params.ReceivedBefore).
		AddInt64("received_after", params.ReceivedAfter).
		BuildURL(baseURL)

	var result struct {
		Data []messageResponse `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return convertMessages(result.Data), nil
}
