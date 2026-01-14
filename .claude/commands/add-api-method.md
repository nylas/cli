# Add New Nylas API Method

Add a new method to the Nylas API client following the hexagonal architecture.

**IMPORTANT: This CLI uses Nylas v3 API ONLY. All endpoints must use `/v3/` prefix.**
- v3 API Docs: https://developer.nylas.com/docs/api/v3/
- Base URL: `https://api.us.nylas.com/v3/` or `https://api.eu.nylas.com/v3/`

## Instructions

1. First, ask me for:
   - Method name (e.g., GetAttachments, CreateDraft)
   - HTTP method and endpoint (e.g., GET /v3/grants/{grantID}/attachments)
   - Request parameters (path params, query params, body)
   - Response structure

2. Then update these files in order:

### Step 1: Domain Types (if needed)
File: `internal/domain/{resource}.go`
```go
type NewType struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    // ... fields matching API response
}

type NewTypeRequest struct {
    Name string `json:"name"`
    // ... fields for request body
}

type NewTypeQueryParams struct {
    Limit  int
    Offset int
    // ... query parameters
}
```

### Step 2: Port Interface
File: `internal/ports/nylas.go`
Add method signature to NylasClient interface:
```go
GetNewTypes(ctx context.Context, grantID string, params *domain.NewTypeQueryParams) ([]domain.NewType, error)
CreateNewType(ctx context.Context, grantID string, req *domain.NewTypeRequest) (*domain.NewType, error)
```

### Step 3: HTTP Client Implementation
File: `internal/adapters/nylas/{resource}.go` (new file or existing)
```go
func (c *HTTPClient) GetNewTypes(ctx context.Context, grantID string, params *domain.NewTypeQueryParams) ([]domain.NewType, error) {
    url := fmt.Sprintf("%s/v3/grants/%s/newtypes", c.baseURL, grantID)

    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    c.setAuthHeaders(req)

    // Add query params
    q := req.URL.Query()
    if params != nil && params.Limit > 0 {
        q.Set("limit", strconv.Itoa(params.Limit))
    }
    req.URL.RawQuery = q.Encode()

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", domain.ErrNetworkError, err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, c.parseError(resp)
    }

    var result struct {
        Data []domain.NewType `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }

    return result.Data, nil
}
```

### Step 4: Mock Implementation
File: `internal/adapters/nylas/mock.go`
Add to MockClient struct and implement method:
```go
// Add fields
GetNewTypesCalled bool
GetNewTypesFunc   func(ctx context.Context, grantID string, params *domain.NewTypeQueryParams) ([]domain.NewType, error)

// Add method
func (m *MockClient) GetNewTypes(ctx context.Context, grantID string, params *domain.NewTypeQueryParams) ([]domain.NewType, error) {
    m.GetNewTypesCalled = true
    if m.GetNewTypesFunc != nil {
        return m.GetNewTypesFunc(ctx, grantID, params)
    }
    return []domain.NewType{}, nil
}
```

3. After updating, run:
   - `go build ./...` to verify compilation
   - `go test ./internal/adapters/nylas/...` to run adapter tests
