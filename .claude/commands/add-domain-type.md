# Add Domain Type

Add a new domain type (model, request, response) to the nylas CLI.

## Instructions

1. Ask me for:
   - Type name (e.g., Attachment, Label)
   - Fields with types
   - Whether it needs Request/Response variants
   - Any helper methods needed

2. Create or update file in `internal/domain/`:

### Basic Value Type
File: `internal/domain/{type}.go`

```go
package domain

import "time"

// TypeName represents a resource in Nylas.
type TypeName struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    Description string     `json:"description,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// Helper methods if needed
func (t *TypeName) DisplayName() string {
    if t.Name != "" {
        return t.Name
    }
    return t.ID
}
```

### Request Type (for create/update)
```go
// CreateTypeNameRequest contains fields for creating a TypeName.
type CreateTypeNameRequest struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

// UpdateTypeNameRequest contains fields for updating a TypeName.
// Pointer fields are optional - only non-nil values are updated.
type UpdateTypeNameRequest struct {
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
}
```

### Response Type (for list with pagination)
```go
// TypeNameListResponse wraps a list response with pagination.
type TypeNameListResponse struct {
    Data       []TypeName `json:"data"`
    NextCursor string     `json:"next_cursor,omitempty"`
}
```

### Query Parameters Type
```go
// TypeNameQueryParams contains query parameters for listing.
type TypeNameQueryParams struct {
    Limit      int    `json:"limit,omitempty"`
    PageToken  string `json:"page_token,omitempty"`
    Filter     string `json:"filter,omitempty"`
}
```

### Enum/Constant Type
```go
// TypeStatus represents the status of a TypeName.
type TypeStatus string

const (
    TypeStatusActive   TypeStatus = "active"
    TypeStatusInactive TypeStatus = "inactive"
    TypeStatusPending  TypeStatus = "pending"
)

// IsValid checks if the status is valid.
func (s TypeStatus) IsValid() bool {
    switch s {
    case TypeStatusActive, TypeStatusInactive, TypeStatusPending:
        return true
    }
    return false
}
```

3. Add tests to `internal/domain/domain_test.go`:
```go
func TestTypeName(t *testing.T) {
    t.Run("DisplayName_with_name", func(t *testing.T) {
        tn := &TypeName{ID: "123", Name: "Test"}
        if tn.DisplayName() != "Test" {
            t.Errorf("Expected 'Test', got %q", tn.DisplayName())
        }
    })

    t.Run("DisplayName_without_name", func(t *testing.T) {
        tn := &TypeName{ID: "123"}
        if tn.DisplayName() != "123" {
            t.Errorf("Expected '123', got %q", tn.DisplayName())
        }
    })
}
```

4. Verify:
```bash
go build ./... && go test ./internal/domain/...
```
