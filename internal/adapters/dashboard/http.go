package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	maxResponseBody = 1 << 20 // 1 MB
)

// doPost sends a JSON POST request and decodes the response into result.
// The server wraps responses in {"request_id","success","data":{...}}.
// This method unwraps the data field before decoding into result.
// If result is nil, the response body is discarded.
func (c *AccountClient) doPost(ctx context.Context, path string, body any, extraHeaders map[string]string, accessToken string, result any) error {
	raw, err := c.doPostRaw(ctx, path, body, extraHeaders, accessToken)
	if err != nil {
		return err
	}

	if result != nil {
		data, unwrapErr := unwrapEnvelope(raw)
		if unwrapErr != nil {
			return unwrapErr
		}
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
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

	// Add DPoP proof
	proof, err := c.dpop.GenerateProof(http.MethodPost, fullURL, accessToken)
	if err != nil {
		return nil, err
	}
	req.Header.Set("DPoP", proof)

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

// DashboardAPIError represents an error from the dashboard API.
// It carries the status code and server message for debugging.
type DashboardAPIError struct {
	StatusCode int
	ServerMsg  string
}

func (e *DashboardAPIError) Error() string {
	if e.ServerMsg != "" {
		return fmt.Sprintf("dashboard API error (HTTP %d): %s", e.StatusCode, e.ServerMsg)
	}
	return fmt.Sprintf("dashboard API error (HTTP %d)", e.StatusCode)
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
	msg := ""
	if json.Unmarshal(body, &errResp) == nil && errResp.Error.Message != "" {
		msg = errResp.Error.Message
		if errResp.Error.Code != "" {
			msg = errResp.Error.Code + ": " + msg
		}
	}
	if msg == "" {
		msg = string(body)
		if len(msg) > 200 {
			msg = msg[:200]
		}
	}
	return &DashboardAPIError{StatusCode: statusCode, ServerMsg: msg}
}
