# Adding a New API Adapter

Guide for implementing new Nylas API adapters.

---

## Overview

Adapters implement the `ports.NylasClient` interface and handle HTTP communication with the Nylas API.

**Location:** `internal/adapters/nylas/`

---

## Steps

### 1. Update Port Interface

Add methods to `internal/ports/nylas.go`:

```go
type NylasClient interface {
    // ... existing methods
    
    // New feature methods
    GetResource(ctx context.Context, id string) (*domain.Resource, error)
    ListResources(ctx context.Context, params *domain.ResourceParams) ([]*domain.Resource, error)
}
```

### 2. Implement Adapter

Create `internal/adapters/nylas/resource.go`:

```go
package nylas

import (
    "context"
    "fmt"
    "net/http"
)

func (c *Client) GetResource(ctx context.Context, id string) (*domain.Resource, error) {
    url := fmt.Sprintf("%s/v3/resources/%s", c.baseURL, id)
    
    req, err := c.newRequest(ctx, http.MethodGet, url, nil, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    
    var resource domain.Resource
    if err := c.do(req, &resource); err != nil {
        return nil, fmt.Errorf("failed to get resource: %w", err)
    }
    
    return &resource, nil
}
```

### 3. Add to Mock

Update `internal/adapters/nylas/mock.go`:

```go
type MockClient struct {
    GetResourceFunc func(ctx context.Context, id string) (*domain.Resource, error)
}

func (m *MockClient) GetResource(ctx context.Context, id string) (*domain.Resource, error) {
    if m.GetResourceFunc != nil {
        return m.GetResourceFunc(ctx, id)
    }
    return nil, nil
}
```

### 4. Write Tests

```go
func TestGetResource(t *testing.T) {
    mock := &MockClient{
        GetResourceFunc: func(ctx context.Context, id string) (*domain.Resource, error) {
            return &domain.Resource{ID: id}, nil
        },
    }
    
    resource, err := mock.GetResource(context.Background(), "test-id")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if resource.ID != "test-id" {
        t.Errorf("expected ID 'test-id', got '%s'", resource.ID)
    }
}
```

---

## HTTP Client Patterns

### GET Requests

```go
func (c *Client) Get(ctx context.Context, id string) (*domain.Object, error) {
    url := fmt.Sprintf("%s/v3/objects/%s", c.baseURL, id)
    
    req, err := c.newRequest(ctx, http.MethodGet, url, nil, nil)
    if err != nil {
        return nil, err
    }
    
    var obj domain.Object
    return &obj, c.do(req, &obj)
}
```

### POST Requests

```go
func (c *Client) Create(ctx context.Context, obj *domain.Object) (*domain.Object, error) {
    url := fmt.Sprintf("%s/v3/objects", c.baseURL)
    
    req, err := c.newRequest(ctx, http.MethodPost, url, nil, obj)
    if err != nil {
        return nil, err
    }
    
    var created domain.Object
    return &created, c.do(req, &created)
}
```

### Query Parameters

```go
func (c *Client) List(ctx context.Context, params *domain.ListParams) ([]*domain.Object, error) {
    url := fmt.Sprintf("%s/v3/objects", c.baseURL)
    
    query := make(map[string]string)
    if params.Limit > 0 {
        query["limit"] = fmt.Sprintf("%d", params.Limit)
    }
    if params.Offset > 0 {
        query["offset"] = fmt.Sprintf("%d", params.Offset)
    }
    
    req, err := c.newRequest(ctx, http.MethodGet, url, query, nil)
    if err != nil {
        return nil, err
    }
    
    var response struct {
        Data []*domain.Object `json:"data"`
    }
    
    return response.Data, c.do(req, &response)
}
```

---

## Error Handling

```go
// Wrap all errors with context
if err != nil {
    return nil, fmt.Errorf("failed to <operation>: %w", err)
}

// Check HTTP status codes
if resp.StatusCode != http.StatusOK {
    return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
}
```

---

## Testing Adapters

1. **Unit tests** - Test with mocks
2. **Integration tests** - Test with real API
3. **Error cases** - Test failure paths
4. **Edge cases** - Empty responses, large datasets

---

## More Resources

- **Port Interface:** `internal/ports/nylas.go`
- **Existing Adapters:** `internal/adapters/nylas/`
- **Testing Guide:** [testing-guide.md](testing-guide.md)
