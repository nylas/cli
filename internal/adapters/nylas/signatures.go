package nylas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/util"
)

type signatureResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Body      string `json:"body"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// GetSignatures retrieves all signatures for a grant.
func (c *HTTPClient) GetSignatures(ctx context.Context, grantID string) ([]domain.Signature, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/signatures", c.baseURL, url.PathEscape(grantID))

	var result struct {
		Data []signatureResponse `json:"data"`
	}
	if err := c.doGet(ctx, queryURL, &result); err != nil {
		return nil, err
	}

	return util.Map(result.Data, convertSignature), nil
}

// GetSignature retrieves a specific signature.
func (c *HTTPClient) GetSignature(ctx context.Context, grantID, signatureID string) (*domain.Signature, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/signatures/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(signatureID))

	var result struct {
		Data signatureResponse `json:"data"`
	}
	if err := c.doGetWithNotFound(ctx, queryURL, &result, domain.ErrSignatureNotFound); err != nil {
		return nil, err
	}

	signature := convertSignature(result.Data)
	return &signature, nil
}

// CreateSignature creates a new signature.
func (c *HTTPClient) CreateSignature(ctx context.Context, grantID string, req *domain.CreateSignatureRequest) (*domain.Signature, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/signatures", c.baseURL, url.PathEscape(grantID))

	resp, err := c.doJSONRequestNoRetry(ctx, http.MethodPost, queryURL, req)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data signatureResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	signature := convertSignature(result.Data)
	return &signature, nil
}

// UpdateSignature updates an existing signature.
func (c *HTTPClient) UpdateSignature(ctx context.Context, grantID, signatureID string, req *domain.UpdateSignatureRequest) (*domain.Signature, error) {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/signatures/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(signatureID))

	resp, err := c.doJSONRequestNoRetry(ctx, http.MethodPut, queryURL, req, http.StatusOK)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data signatureResponse `json:"data"`
	}
	if err := c.decodeJSONResponse(resp, &result); err != nil {
		return nil, err
	}

	signature := convertSignature(result.Data)
	return &signature, nil
}

// DeleteSignature deletes a signature.
func (c *HTTPClient) DeleteSignature(ctx context.Context, grantID, signatureID string) error {
	queryURL := fmt.Sprintf("%s/v3/grants/%s/signatures/%s", c.baseURL, url.PathEscape(grantID), url.PathEscape(signatureID))
	return c.doDelete(ctx, queryURL)
}

func convertSignature(s signatureResponse) domain.Signature {
	return domain.Signature{
		ID:        s.ID,
		Name:      s.Name,
		Body:      s.Body,
		CreatedAt: time.Unix(s.CreatedAt, 0),
		UpdatedAt: time.Unix(s.UpdatedAt, 0),
	}
}
