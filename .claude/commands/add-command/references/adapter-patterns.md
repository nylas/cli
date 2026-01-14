# Adapter Implementation Patterns

Reference patterns for implementing API methods in `internal/adapters/nylas/`.

---

## File Structure

```
internal/adapters/nylas/
├── {resource}.go           # Main implementation
├── {resource}_test.go      # Unit tests
└── mock_{resource}.go      # Mock implementation
```

---

## List Method

```go
// GetResources retrieves resources for a grant.
func (c *HTTPClient) GetResources(ctx context.Context, grantID string, params *domain.ResourceQueryParams) ([]domain.Resource, error) {
    result, err := c.GetResourcesWithCursor(ctx, grantID, params)
    if err != nil {
        return nil, err
    }
    return result.Data, nil
}

// GetResourcesWithCursor retrieves resources with pagination cursor.
func (c *HTTPClient) GetResourcesWithCursor(ctx context.Context, grantID string, params *domain.ResourceQueryParams) (*domain.ResourceListResponse, error) {
    queryURL := fmt.Sprintf("%s/v3/grants/%s/resources", c.baseURL, grantID)

    queryParams := url.Values{}
    if params != nil {
        if params.Limit > 0 {
            queryParams.Set("limit", strconv.Itoa(params.Limit))
        }
        if params.PageToken != "" {
            queryParams.Set("page_token", params.PageToken)
        }
        // Add resource-specific params
    }

    if len(queryParams) > 0 {
        queryURL += "?" + queryParams.Encode()
    }

    req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
    if err != nil {
        return nil, err
    }
    c.setAuthHeader(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, c.handleErrorResponse(resp)
    }

    var result domain.ResourceListResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result, nil
}
```

---

## Get Single Method

```go
// GetResource retrieves a single resource by ID.
func (c *HTTPClient) GetResource(ctx context.Context, grantID, resourceID string) (*domain.Resource, error) {
    queryURL := fmt.Sprintf("%s/v3/grants/%s/resources/%s", c.baseURL, grantID, resourceID)

    req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
    if err != nil {
        return nil, err
    }
    c.setAuthHeader(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, c.handleErrorResponse(resp)
    }

    var result struct {
        Data domain.Resource `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result.Data, nil
}
```

---

## Create Method

```go
// CreateResource creates a new resource.
func (c *HTTPClient) CreateResource(ctx context.Context, grantID string, req *domain.CreateResourceRequest) (*domain.Resource, error) {
    queryURL := fmt.Sprintf("%s/v3/grants/%s/resources", c.baseURL, grantID)

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", queryURL, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    c.setAuthHeader(httpReq)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return nil, c.handleErrorResponse(resp)
    }

    var result struct {
        Data domain.Resource `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result.Data, nil
}
```

---

## Update Method

```go
// UpdateResource updates an existing resource.
func (c *HTTPClient) UpdateResource(ctx context.Context, grantID, resourceID string, req *domain.UpdateResourceRequest) (*domain.Resource, error) {
    queryURL := fmt.Sprintf("%s/v3/grants/%s/resources/%s", c.baseURL, grantID, resourceID)

    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "PUT", queryURL, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    c.setAuthHeader(httpReq)
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, c.handleErrorResponse(resp)
    }

    var result struct {
        Data domain.Resource `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result.Data, nil
}
```

---

## Delete Method

```go
// DeleteResource deletes a resource.
func (c *HTTPClient) DeleteResource(ctx context.Context, grantID, resourceID string) error {
    queryURL := fmt.Sprintf("%s/v3/grants/%s/resources/%s", c.baseURL, grantID, resourceID)

    req, err := http.NewRequestWithContext(ctx, "DELETE", queryURL, nil)
    if err != nil {
        return err
    }
    c.setAuthHeader(req)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        return c.handleErrorResponse(resp)
    }
    return nil
}
```

---

## Port Interface Update

Add to `internal/ports/nylas.go`:

```go
// Resource operations
GetResources(ctx context.Context, grantID string, params *domain.ResourceQueryParams) ([]domain.Resource, error)
GetResourcesWithCursor(ctx context.Context, grantID string, params *domain.ResourceQueryParams) (*domain.ResourceListResponse, error)
GetResource(ctx context.Context, grantID, resourceID string) (*domain.Resource, error)
CreateResource(ctx context.Context, grantID string, req *domain.CreateResourceRequest) (*domain.Resource, error)
UpdateResource(ctx context.Context, grantID, resourceID string, req *domain.UpdateResourceRequest) (*domain.Resource, error)
DeleteResource(ctx context.Context, grantID, resourceID string) error
```

---

## Mock Implementation

Add to `internal/adapters/nylas/mock_{resource}.go`:

```go
// Resource mock functions
GetResourcesFunc           func(ctx context.Context, grantID string, params *domain.ResourceQueryParams) ([]domain.Resource, error)
GetResourcesWithCursorFunc func(ctx context.Context, grantID string, params *domain.ResourceQueryParams) (*domain.ResourceListResponse, error)
GetResourceFunc            func(ctx context.Context, grantID, resourceID string) (*domain.Resource, error)
CreateResourceFunc         func(ctx context.Context, grantID string, req *domain.CreateResourceRequest) (*domain.Resource, error)
UpdateResourceFunc         func(ctx context.Context, grantID, resourceID string, req *domain.UpdateResourceRequest) (*domain.Resource, error)
DeleteResourceFunc         func(ctx context.Context, grantID, resourceID string) error

// Implement interface methods
func (m *MockClient) GetResources(ctx context.Context, grantID string, params *domain.ResourceQueryParams) ([]domain.Resource, error) {
    if m.GetResourcesFunc != nil {
        return m.GetResourcesFunc(ctx, grantID, params)
    }
    return nil, nil
}
```

---

## Checklist

- [ ] All methods use context for cancellation
- [ ] Error handling with handleErrorResponse
- [ ] JSON tags match API response
- [ ] Port interface updated
- [ ] Mock implementation added
- [ ] Unit tests written
