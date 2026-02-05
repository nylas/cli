package nylas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// trackAuditRequest extracts request_id from body and calls audit hook.
func trackAuditRequest(body []byte, statusCode int) {
	if ports.AuditRequestHook == nil {
		return
	}
	var meta struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(body, &meta)
	ports.AuditRequestHook(meta.RequestID, statusCode)
}

// trackAuditError calls audit hook for error responses.
func trackAuditError(statusCode int) {
	if ports.AuditRequestHook != nil {
		ports.AuditRequestHook("", statusCode)
	}
}

// ListResponse is a generic paginated response.
type ListResponse[T any] struct {
	Data       []T
	NextCursor string
	HasMore    bool
}

// doGet performs a GET request and decodes the JSON response.
// It handles common patterns: auth headers, error checking, and JSON decoding.
//
// Usage:
//
//	var result struct {
//	    Data messageResponse `json:"data"`
//	}
//	if err := c.doGet(ctx, url, &result); err != nil {
//	    return nil, err
//	}
func (c *HTTPClient) doGet(ctx context.Context, url string, result any) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		trackAuditError(resp.StatusCode)
		return c.parseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	trackAuditRequest(body, resp.StatusCode)

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// doGetWithNotFound performs a GET request with special handling for 404.
// Returns the specified error when the resource is not found.
//
// Usage:
//
//	var result struct {
//	    Data messageResponse `json:"data"`
//	}
//	if err := c.doGetWithNotFound(ctx, url, &result, domain.ErrContactNotFound); err != nil {
//	    return nil, err
//	}
func (c *HTTPClient) doGetWithNotFound(ctx context.Context, url string, result any, notFoundErr error) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		trackAuditError(resp.StatusCode)
		return notFoundErr
	}
	if resp.StatusCode != http.StatusOK {
		trackAuditError(resp.StatusCode)
		return c.parseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	trackAuditRequest(body, resp.StatusCode)

	if err := json.Unmarshal(body, result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// doDelete performs a DELETE request and validates the response.
// Accepts 200 OK or 204 No Content as successful responses.
//
// Usage:
//
//	if err := c.doDelete(ctx, url); err != nil {
//	    return err
//	}
func (c *HTTPClient) doDelete(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}
	c.setAuthHeader(req)

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		trackAuditError(resp.StatusCode)
		return c.parseError(resp)
	}

	// Track successful delete (no request_id in DELETE responses)
	if ports.AuditRequestHook != nil {
		ports.AuditRequestHook("", resp.StatusCode)
	}

	return nil
}

// QueryBuilder helps build URL query parameters from structs.
// Fields are read using the "query" struct tag.
//
// Supported types:
//   - string: added if non-empty
//   - int, int64: added if > 0
//   - bool: added as "true" if true
//   - *bool: added if non-nil
//   - *int: added if non-nil and > 0
//   - []string: each value added with the same key
type QueryBuilder struct {
	values url.Values
}

// NewQueryBuilder creates a new QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{values: url.Values{}}
}

// Add adds a key-value pair to the query.
func (qb *QueryBuilder) Add(key, value string) *QueryBuilder {
	if value != "" {
		qb.values.Set(key, value)
	}
	return qb
}

// AddInt adds an integer value if greater than 0.
func (qb *QueryBuilder) AddInt(key string, value int) *QueryBuilder {
	if value > 0 {
		qb.values.Set(key, strconv.Itoa(value))
	}
	return qb
}

// AddInt64 adds an int64 value if greater than 0.
func (qb *QueryBuilder) AddInt64(key string, value int64) *QueryBuilder {
	if value > 0 {
		qb.values.Set(key, strconv.FormatInt(value, 10))
	}
	return qb
}

// AddBool adds a boolean value if true.
func (qb *QueryBuilder) AddBool(key string, value bool) *QueryBuilder {
	if value {
		qb.values.Set(key, "true")
	}
	return qb
}

// AddBoolPtr adds a boolean pointer value if non-nil.
func (qb *QueryBuilder) AddBoolPtr(key string, value *bool) *QueryBuilder {
	if value != nil {
		qb.values.Set(key, strconv.FormatBool(*value))
	}
	return qb
}

// AddSlice adds each string in the slice with the same key.
func (qb *QueryBuilder) AddSlice(key string, values []string) *QueryBuilder {
	for _, v := range values {
		qb.values.Add(key, v)
	}
	return qb
}

// Encode returns the encoded query string.
func (qb *QueryBuilder) Encode() string {
	return qb.values.Encode()
}

// Values returns the underlying url.Values.
func (qb *QueryBuilder) Values() url.Values {
	return qb.values
}

// BuildURL appends the query string to the base URL if there are parameters.
func (qb *QueryBuilder) BuildURL(baseURL string) string {
	if len(qb.values) == 0 {
		return baseURL
	}
	return baseURL + "?" + qb.values.Encode()
}
