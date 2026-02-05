// Package nylas provides the Nylas API client implementation.
package nylas

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/nylas/cli/internal/adapters/providers"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/nylas/cli/internal/version"
	"golang.org/x/time/rate"
)

func init() {
	// Register Nylas provider with the global registry
	providers.Register("nylas", func(config providers.ProviderConfig) (ports.NylasClient, error) {
		client := NewHTTPClient()

		if config.BaseURL != "" {
			client.SetBaseURL(config.BaseURL)
		} else if config.Region != "" {
			client.SetRegion(config.Region)
		}

		client.SetCredentials(config.ClientID, config.ClientSecret, config.APIKey)
		return client, nil
	})
}

const (
	baseURLUS = "https://api.us.nylas.com"
	baseURLEU = "https://api.eu.nylas.com"

	// defaultRateLimit is the default rate limit (requests per second)
	// Set to 10 requests per second to avoid API quota exhaustion
	defaultRateLimit = 10

	// defaultMaxRetries is the maximum number of retries for failed requests
	defaultMaxRetries = 3

	// defaultRetryDelay is the base delay between retries
	defaultRetryDelay = time.Second

	// maxRetryDelay is the maximum delay between retries
	maxRetryDelay = 30 * time.Second
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
	maxRetries     int
	retryDelay     time.Duration
}

// NewHTTPClient creates a new Nylas HTTP client with rate limiting and retry logic.
// Rate limiting prevents API quota exhaustion and temporary account suspension.
// Default: 10 requests/second with burst capacity of 20 requests.
// Retry logic handles transient errors with exponential backoff and Retry-After header support.
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
		maxRetries:     defaultMaxRetries,
		retryDelay:     defaultRetryDelay,
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

// SetMaxRetries sets the maximum number of retries (for testing purposes).
func (c *HTTPClient) SetMaxRetries(retries int) {
	c.maxRetries = retries
}

// setAuthHeader sets the authorization header on the request.
func (c *HTTPClient) setAuthHeader(req *http.Request) {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}

// parseError parses an error response from the API.
// Uses streaming decoder with size limit to avoid large allocations.
func (c *HTTPClient) parseError(resp *http.Response) error {
	// Limit error response body to 10KB to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 10*1024)

	var errResp struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	// Use streaming decoder instead of ReadAll + Unmarshal
	if err := json.NewDecoder(limitedReader).Decode(&errResp); err == nil && errResp.Error.Message != "" {
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
	// Set User-Agent header for all requests
	req.Header.Set("User-Agent", version.UserAgent())

	var lastErr error
	var lastResp *http.Response

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Apply rate limiting - wait for permission to proceed
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter: %w", err)
		}

		// Ensure context has timeout
		ctxWithTimeout, cancel := c.ensureContext(ctx)

		// Optimize: on first attempt, use WithContext to avoid allocation.
		// On retries, we must clone since the original request was already used.
		var reqToUse *http.Request
		if attempt == 0 {
			reqToUse = req.WithContext(ctxWithTimeout)
		} else {
			reqToUse = req.Clone(ctxWithTimeout)
		}

		// Execute request
		resp, err := c.httpClient.Do(reqToUse)
		cancel() // Cancel timeout context

		if err != nil {
			// Don't retry if the parent context is done
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			// Don't retry context timeout/cancellation errors - they'll just timeout again
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
			}

			lastErr = fmt.Errorf("%w: %v", domain.ErrNetworkError, err)

			// Only retry transient network errors (connection refused, DNS, etc.)
			if attempt < c.maxRetries {
				delay := c.calculateBackoff(attempt, nil)
				select {
				case <-time.After(delay):
					continue
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, lastErr
		}

		// Check if we should retry based on status code
		if c.shouldRetryStatus(resp.StatusCode) && attempt < c.maxRetries {
			lastResp = resp
			_ = resp.Body.Close() // Close body before retry; error doesn't affect retry logic

			delay := c.calculateBackoff(attempt, resp)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		return resp, nil
	}

	// All retries exhausted
	if lastResp != nil {
		return lastResp, nil
	}
	return nil, lastErr
}

// shouldRetryStatus determines if a status code is retryable
func (c *HTTPClient) shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	default:
		return false
	}
}

// calculateBackoff calculates the delay before the next retry
// It respects the Retry-After header if present, otherwise uses exponential backoff
func (c *HTTPClient) calculateBackoff(attempt int, resp *http.Response) time.Duration {
	// Check Retry-After header if response available
	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try parsing as seconds
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				delay := time.Duration(seconds) * time.Second
				if delay <= maxRetryDelay {
					return delay
				}
				return maxRetryDelay
			}
		}
	}

	// Exponential backoff: 1s, 2s, 4s, 8s, ...
	delay := c.retryDelay * (1 << attempt)
	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}

// doJSONRequestInternal is the shared implementation for JSON API requests.
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - method: HTTP method (GET, POST, PUT, PATCH, DELETE)
//   - url: Full URL for the request
//   - body: Request body to marshal to JSON (can be nil for GET/DELETE)
//   - withAuth: Whether to include the authorization header
//   - acceptedStatuses: HTTP status codes considered successful (defaults to 200, 201)
//
// Returns the response (caller must close body) or an error.
func (c *HTTPClient) doJSONRequestInternal(
	ctx context.Context,
	method, url string,
	body any,
	withAuth bool,
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
	req.Header.Set("User-Agent", version.UserAgent())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if withAuth {
		c.setAuthHeader(req)
	}

	// Execute request with rate limiting
	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// Track request for audit logging
	if ports.AuditRequestHook != nil {
		ports.AuditRequestHook(getRequestID(resp), resp.StatusCode)
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

// doJSONRequest performs a JSON API request with authentication.
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
	return c.doJSONRequestInternal(ctx, method, url, body, true, acceptedStatuses...)
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
	return c.doJSONRequestInternal(ctx, method, url, body, false, acceptedStatuses...)
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
