package nylas

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

// ListInboundInboxes lists all inbound inboxes (grants with provider=inbox).
func (c *HTTPClient) ListInboundInboxes(ctx context.Context) ([]domain.InboundInbox, error) {
	grants, err := c.listManagedGrants(ctx, domain.ProviderInbox)
	if err != nil {
		return nil, err
	}

	inboxes := make([]domain.InboundInbox, 0, len(grants))
	for _, grant := range grants {
		inboxes = append(inboxes, convertManagedGrantToInboundInbox(grant))
	}

	return inboxes, nil
}

// GetInboundInbox retrieves a specific inbound inbox by grant ID.
func (c *HTTPClient) GetInboundInbox(ctx context.Context, grantID string) (*domain.InboundInbox, error) {
	grant, err := c.getManagedGrant(ctx, grantID)
	if err != nil {
		return nil, err
	}

	if grant.Provider != domain.ProviderInbox {
		return nil, fmt.Errorf("%w: grant is not an inbound inbox (provider=%s)", domain.ErrInvalidGrant, grant.Provider)
	}

	inbox := convertManagedGrantToInboundInbox(*grant)
	return &inbox, nil
}

// CreateInboundInbox creates a new inbound inbox with the given email address.
// The email parameter is the local part (e.g., "support" for support@app.nylas.email).
func (c *HTTPClient) CreateInboundInbox(ctx context.Context, email string) (*domain.InboundInbox, error) {
	grant, err := c.createManagedGrant(ctx, domain.ProviderInbox, email)
	if err != nil {
		return nil, err
	}

	inbox := convertManagedGrantToInboundInbox(*grant)
	return &inbox, nil
}

// DeleteInboundInbox deletes an inbound inbox by revoking its grant.
func (c *HTTPClient) DeleteInboundInbox(ctx context.Context, grantID string) error {
	return c.deleteManagedGrant(ctx, grantID, domain.ProviderInbox)
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
