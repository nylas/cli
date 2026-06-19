package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/version"
)

const (
	maxResponseBody = 1 << 20 // 1 MB
)

// newNonRedirectClient creates an HTTP client that does not follow redirects.
// DPoP proofs are bound to a specific URL (the htu claim), so following a
// redirect would cause the proof to be invalid at the destination.
func newNonRedirectClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func setDashboardUserAgent(req *http.Request) {
	req.Header.Set("User-Agent", version.UserAgent())
}

// doPost sends a JSON POST request and decodes the already-unwrapped payload
// returned by doPostRaw into result. If result is nil, the response body is
// discarded.
func (c *AccountClient) doPost(ctx context.Context, path string, body any, extraHeaders map[string]string, accessToken string, result any) error {
	raw, err := c.doPostRaw(ctx, path, body, extraHeaders, accessToken)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// doPatch sends a JSON PATCH request and decodes the envelope-unwrapped response.
func (c *AccountClient) doPatch(ctx context.Context, path string, body any, extraHeaders map[string]string, accessToken string, result any) error {
	raw, err := c.doPatchRaw(ctx, path, body, extraHeaders, accessToken)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// doDelete sends a DELETE request and decodes the envelope-unwrapped response.
func (c *AccountClient) doDelete(ctx context.Context, path string, extraHeaders map[string]string, accessToken string, result any) error {
	raw, err := c.doDeleteRaw(ctx, path, extraHeaders, accessToken)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

type rawResponse struct {
	Data       []byte
	NextCursor string
}

type dashboardEnvelope struct {
	Data            json.RawMessage `json:"data"`
	NextCursor      string          `json:"nextCursor"`
	NextCursorSnake string          `json:"next_cursor"`
	PageToken       string          `json:"pageToken"`
	Pagination      *cursorEnvelope `json:"pagination"`
	Meta            *cursorEnvelope `json:"meta"`
}

type cursorEnvelope struct {
	NextCursor      string `json:"nextCursor"`
	NextCursorSnake string `json:"next_cursor"`
	PageToken       string `json:"pageToken"`
}

func (e dashboardEnvelope) cursor() string {
	for _, cursor := range []string{e.NextCursor, e.NextCursorSnake, e.PageToken} {
		if cursor != "" {
			return cursor
		}
	}
	for _, nested := range []*cursorEnvelope{e.Pagination, e.Meta} {
		if nested == nil {
			continue
		}
		if cursor := nested.cursor(); cursor != "" {
			return cursor
		}
	}
	return ""
}

func (e cursorEnvelope) cursor() string {
	for _, cursor := range []string{e.NextCursor, e.NextCursorSnake, e.PageToken} {
		if cursor != "" {
			return cursor
		}
	}
	return ""
}

// setDPoPProof generates and sets the DPoP proof header on the request.
func (c *AccountClient) setDPoPProof(req *http.Request, method, fullURL, accessToken string) error {
	proof, err := c.dpop.GenerateProof(method, fullURL, accessToken)
	if err != nil {
		return err
	}
	req.Header.Set("DPoP", proof)
	return nil
}

// doPostRaw sends a JSON POST request and returns the raw response body.
func (c *AccountClient) doPostRaw(ctx context.Context, path string, body any, extraHeaders map[string]string, accessToken string) ([]byte, error) {
	resp, err := c.doRaw(ctx, http.MethodPost, path, body, extraHeaders, accessToken)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// unwrapEnvelope extracts the "data" field from the API response envelope.
// The dashboard-account API wraps all successful responses in:
//
//	{"request_id": "...", "success": true, "data": {...}}
func unwrapEnvelope(body []byte) ([]byte, error) {
	resp, err := unwrapRawResponse(body)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func unwrapRawResponse(body []byte) (rawResponse, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return rawResponse{}, fmt.Errorf("failed to decode response envelope: %w", err)
	}
	if len(envelope.Data) == 0 {
		return rawResponse{Data: body}, nil // no envelope, return as-is
	}

	var full dashboardEnvelope
	if err := json.Unmarshal(body, &full); err != nil {
		return rawResponse{}, fmt.Errorf("failed to decode response envelope: %w", err)
	}
	return rawResponse{
		Data:       full.Data,
		NextCursor: full.cursor(),
	}, nil
}

// doGetRaw sends a GET request and returns the raw (envelope-unwrapped) response body.
func (c *AccountClient) doGetRaw(ctx context.Context, path string, extraHeaders map[string]string, accessToken string) ([]byte, error) {
	resp, err := c.doRaw(ctx, http.MethodGet, path, nil, extraHeaders, accessToken)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// doGet sends a GET request and decodes the envelope-unwrapped response into result.
func (c *AccountClient) doGet(ctx context.Context, path string, extraHeaders map[string]string, accessToken string, result any) error {
	raw, err := c.doGetRaw(ctx, path, extraHeaders, accessToken)
	if err != nil {
		return err
	}

	if result != nil {
		if err := json.Unmarshal(raw, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// doPatchRaw sends a JSON PATCH request and returns the raw response body.
func (c *AccountClient) doPatchRaw(ctx context.Context, path string, body any, extraHeaders map[string]string, accessToken string) ([]byte, error) {
	resp, err := c.doRaw(ctx, http.MethodPatch, path, body, extraHeaders, accessToken)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

// doDeleteRaw sends a DELETE request and returns the raw response body.
func (c *AccountClient) doDeleteRaw(ctx context.Context, path string, extraHeaders map[string]string, accessToken string) ([]byte, error) {
	resp, err := c.doRaw(ctx, http.MethodDelete, path, nil, extraHeaders, accessToken)
	if err != nil {
		return nil, err
	}

	return resp.Data, nil
}

func (c *AccountClient) doGetRawResponse(ctx context.Context, path string, extraHeaders map[string]string, accessToken string) (rawResponse, error) {
	return c.doRaw(ctx, http.MethodGet, path, nil, extraHeaders, accessToken)
}

func (c *AccountClient) doRaw(ctx context.Context, method, path string, body any, extraHeaders map[string]string, accessToken string) (rawResponse, error) {
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return rawResponse{}, fmt.Errorf("failed to encode request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return rawResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	setDashboardUserAgent(req)

	if err := c.setDPoPProof(req, method, fullURL, accessToken); err != nil {
		return rawResponse{}, err
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return rawResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return rawResponse{}, fmt.Errorf("server redirected to %s — the dashboard URL may be incorrect (set NYLAS_DASHBOARD_ACCOUNT_URL)", location)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return rawResponse{}, parseErrorResponse(resp.StatusCode, respBody)
	}

	return unwrapRawResponse(respBody)
}

// parseErrorResponse extracts a user-friendly error from an HTTP error response.
// The dashboard-account error envelope is:
//
//	{"request_id":"...","success":false,"error":{"code":"...","message":"..."}}
func parseErrorResponse(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	code := ""
	msg := ""
	if json.Unmarshal(body, &errResp) == nil {
		code = errResp.Error.Code
		msg = errResp.Error.Message
	}
	if msg == "" && code == "" {
		msg = string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
	}
	return domain.NewDashboardAPIError(statusCode, code, msg)
}
