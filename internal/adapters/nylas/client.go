// Package nylas provides the Nylas API client implementation.
package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nylas/cli/internal/domain"
	"golang.org/x/time/rate"
)

const (
	baseURLUS = "https://api.us.nylas.com"
	baseURLEU = "https://api.eu.nylas.com"

	// defaultRateLimit is the default rate limit (requests per second)
	// Set to 10 requests per second to avoid API quota exhaustion
	defaultRateLimit = 10
)

// HTTPClient implements the NylasClient interface.
type HTTPClient struct {
	httpClient     *http.Client
	baseURL        string
	clientID       string
	clientSecret   string
	apiKey         string
	rateLimiter    *rate.Limiter
	requestTimeout time.Duration
}

// NewHTTPClient creates a new Nylas HTTP client with rate limiting.
// Rate limiting prevents API quota exhaustion and temporary account suspension.
// Default: 10 requests/second with burst capacity of 20 requests.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		httpClient: &http.Client{
			// Remove global timeout since we use per-request context timeouts
			Timeout: 0,
		},
		baseURL: baseURLUS,
		// Create token bucket rate limiter: 10 requests/second, burst of 20
		rateLimiter:    rate.NewLimiter(rate.Limit(defaultRateLimit), defaultRateLimit*2),
		requestTimeout: domain.TimeoutAPI,
	}
}

// SetRegion sets the API region (us or eu).
func (c *HTTPClient) SetRegion(region string) {
	if region == "eu" {
		c.baseURL = baseURLEU
	} else {
		c.baseURL = baseURLUS
	}
}

// SetCredentials sets the API credentials.
func (c *HTTPClient) SetCredentials(clientID, clientSecret, apiKey string) {
	c.clientID = clientID
	c.clientSecret = clientSecret
	c.apiKey = apiKey
}

// SetBaseURL sets the base URL (for testing purposes).
func (c *HTTPClient) SetBaseURL(url string) {
	c.baseURL = url
}

// setAuthHeader sets the authorization header on the request.
func (c *HTTPClient) setAuthHeader(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// parseError parses an error response from the API.
func (c *HTTPClient) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error.Message != "" {
		return fmt.Errorf("%w: %s", domain.ErrAPIError, errResp.Error.Message)
	}

	return fmt.Errorf("%w: status %d", domain.ErrAPIError, resp.StatusCode)
}

// getRequestID extracts the request ID from response headers.
func getRequestID(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	// Nylas uses X-Request-Id header
	if id := resp.Header.Get("X-Request-Id"); id != "" {
		return id
	}
	return resp.Header.Get("Request-Id")
}

// ensureContext ensures a context has a timeout.
// If the context already has a deadline, it's returned as-is.
// Otherwise, a new context with the default timeout is created.
func (c *HTTPClient) ensureContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		// Context already has timeout, use as-is
		return ctx, func() {}
	}
	// Add default timeout
	return context.WithTimeout(ctx, c.requestTimeout)
}

// doRequest executes an HTTP request with rate limiting and timeout.
// This method applies rate limiting before making the request and ensures
// the context has a timeout to prevent hanging requests.
func (c *HTTPClient) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Apply rate limiting - wait for permission to proceed
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	// Ensure context has timeout
	ctxWithTimeout, cancel := c.ensureContext(ctx)
	defer cancel()

	// Update request context
	req = req.WithContext(ctxWithTimeout)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}

	return resp, nil
}

// doJSONRequest performs a JSON API request with proper error handling.
// This helper consolidates the common pattern of:
//   - Marshaling the request body to JSON (with error handling)
//   - Creating the HTTP request with context
//   - Setting Content-Type and Authorization headers
//   - Executing the request with rate limiting
//   - Validating the response status code
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - method: HTTP method (GET, POST, PUT, PATCH, DELETE)
//   - url: Full URL for the request
//   - body: Request body to marshal to JSON (can be nil for GET/DELETE)
//   - acceptedStatuses: HTTP status codes considered successful (defaults to 200, 201)
//
// Returns the response (caller must close body) or an error.
func (c *HTTPClient) doJSONRequest(
	ctx context.Context,
	method, url string,
	body any,
	acceptedStatuses ...int,
) (*http.Response, error) {
	// Default accepted statuses
	if len(acceptedStatuses) == 0 {
		acceptedStatuses = []int{http.StatusOK, http.StatusCreated}
	}

	// Marshal body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuthHeader(req)

	// Execute request with rate limiting
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Validate status code
	statusOK := false
	for _, status := range acceptedStatuses {
		if resp.StatusCode == status {
			statusOK = true
			break
		}
	}

	if !statusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, c.parseError(resp)
	}

	return resp, nil
}

// decodeJSONResponse decodes a JSON response body into the provided struct.
// It properly closes the response body after reading.
//
// Usage:
//
//	var result struct {
//	    Data MyType `json:"data"`
//	}
//	if err := c.decodeJSONResponse(resp, &result); err != nil {
//	    return nil, err
//	}
func (c *HTTPClient) decodeJSONResponse(resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// doJSONRequestNoAuth performs a JSON API request without authentication.
// This is used for token exchange endpoints that don't require auth headers.
func (c *HTTPClient) doJSONRequestNoAuth(
	ctx context.Context,
	method, url string,
	body any,
	acceptedStatuses ...int,
) (*http.Response, error) {
	// Default accepted statuses
	if len(acceptedStatuses) == 0 {
		acceptedStatuses = []int{http.StatusOK, http.StatusCreated}
	}

	// Marshal body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set Content-Type header (no auth header)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request with rate limiting
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Validate status code
	statusOK := false
	for _, status := range acceptedStatuses {
		if resp.StatusCode == status {
			statusOK = true
			break
		}
	}

	if !statusOK {
		defer func() { _ = resp.Body.Close() }()
		return nil, c.parseError(resp)
	}

	return resp, nil
}

// validateRequired validates that a required field is not empty.
// This is a generic replacement for validateGrantID, validateCalendarID, etc.
//
// Usage:
//
//	if err := validateRequired("grant ID", grantID); err != nil {
//	    return nil, err
//	}
func validateRequired(fieldName, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s is required", domain.ErrInvalidInput, fieldName)
	}
	return nil
}
