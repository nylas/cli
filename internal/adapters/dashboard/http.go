package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nylas/cli/internal/domain"
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
	fullURL := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if err := c.setDPoPProof(req, http.MethodPost, fullURL, accessToken); err != nil {
		return nil, err
	}

	// Add extra headers (Authorization, X-Nylas-Org)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return nil, fmt.Errorf("server redirected to %s — the dashboard URL may be incorrect (set NYLAS_DASHBOARD_ACCOUNT_URL)", location)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(resp.StatusCode, respBody)
	}

	// Unwrap the {request_id, success, data} envelope
	data, unwrapErr := unwrapEnvelope(respBody)
	if unwrapErr != nil {
		return nil, unwrapErr
	}

	return data, nil
}

// unwrapEnvelope extracts the "data" field from the API response envelope.
// The dashboard-account API wraps all successful responses in:
//
//	{"request_id": "...", "success": true, "data": {...}}
func unwrapEnvelope(body []byte) ([]byte, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to decode response envelope: %w", err)
	}
	if len(envelope.Data) == 0 {
		return body, nil // no envelope, return as-is
	}
	return envelope.Data, nil
}

// doGetRaw sends a GET request and returns the raw (envelope-unwrapped) response body.
func (c *AccountClient) doGetRaw(ctx context.Context, path string, extraHeaders map[string]string, accessToken string) ([]byte, error) {
	fullURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.setDPoPProof(req, http.MethodGet, fullURL, accessToken); err != nil {
		return nil, err
	}

	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		return nil, fmt.Errorf("server redirected to %s — the dashboard URL may be incorrect (set NYLAS_DASHBOARD_ACCOUNT_URL)", location)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseErrorResponse(resp.StatusCode, respBody)
	}

	data, unwrapErr := unwrapEnvelope(respBody)
	if unwrapErr != nil {
		return nil, unwrapErr
	}

	return data, nil
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
