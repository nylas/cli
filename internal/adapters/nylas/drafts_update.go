package nylas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nylas/cli/internal/domain"
)

// UpdateDraft updates an existing draft.
func (c *HTTPClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	// If there are attachments, use multipart; otherwise use JSON
	if len(req.Attachments) > 0 {
		return c.updateDraftWithMultipart(ctx, grantID, draftID, req)
	}
	return c.updateDraftWithJSON(ctx, grantID, draftID, req)
}

// updateDraftWithJSON updates a draft using JSON encoding (no attachments).
func (c *HTTPClient) updateDraftWithJSON(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(draftID))

	resp, err := c.doJSONRequest(ctx, "PUT", queryURL, buildDraftPayload(req, false), http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data draftResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	draft := convertDraft(result.Data)
	return &draft, nil
}

// updateDraftWithMultipart updates a draft with attachments using multipart/form-data.
func (c *HTTPClient) updateDraftWithMultipart(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/drafts/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(draftID))

	var result struct {
		Data draftResponse `json:"data"`
	}
	if err := c.doMultipartDraft(ctx, "PUT", queryURL, buildDraftPayload(req, false), req.Attachments, &result, http.StatusOK); err != nil {
		return nil, err
	}

	draft := convertDraft(result.Data)
	return &draft, nil
}
