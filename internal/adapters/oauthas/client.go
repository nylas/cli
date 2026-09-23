// Package oauthas implements a client for the Nylas OAuth 2.1 authorization
// server hosted by dashboard-account.
//
// These endpoints speak plain RFC 6749/7009: no house {"data":...}
// envelope and no DPoP proof. That is why this does not reuse the
// dashboard.AccountClient transport, which adds both.
package oauthas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/version"
)

const (
	maxResponseBody   = 1 << 20 // 1 MB
	discoveryPath     = "/.well-known/oauth-authorization-server"
	defaultHTTPTimout = 30 * time.Second
)

// Client is an HTTP client for the authorization server.
type Client struct {
	baseURL    string
	httpClient *http.Client
	now        func() time.Time

	mu       sync.Mutex
	metadata *domain.OAuthServerMetadata
}

// NewClient creates a client rooted at the authorization server's base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: defaultHTTPTimout},
		now:        time.Now,
	}
}

// Metadata fetches and caches the RFC 8414 metadata document.
func (c *Client) Metadata(ctx context.Context) (*domain.OAuthServerMetadata, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.metadata != nil {
		return c.metadata, nil
	}

	var metadata domain.OAuthServerMetadata
	if err := c.getJSON(ctx, c.baseURL+discoveryPath, "", &metadata); err != nil {
		return nil, fmt.Errorf("failed to discover authorization server at %s: %w", c.baseURL, err)
	}
	if err := metadata.Validate(); err != nil {
		return nil, err
	}

	c.metadata = &metadata
	return c.metadata, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint, accessToken string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return c.do(req, result)
}

func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req, result)
}

func (c *Client) do(req *http.Request, result any) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", version.UserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseOAuthError(resp.StatusCode, body)
	}

	if result == nil {
		return nil
	}
	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// parseOAuthError decodes an RFC 6749 section 5.2 error body. A response that
// is not in that shape (an HTML error page, or the house envelope, whose
// "error" is an object) falls back to the status code and a body snippet.
func parseOAuthError(statusCode int, body []byte) error {
	var payload struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		return &domain.OAuthError{
			Code:        payload.Error,
			Description: payload.Description,
			StatusCode:  statusCode,
		}
	}

	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 200 {
		snippet = snippet[:200]
	}
	return &domain.OAuthError{
		Code:        "http_" + strconv.Itoa(statusCode),
		Description: snippet,
		StatusCode:  statusCode,
	}
}
