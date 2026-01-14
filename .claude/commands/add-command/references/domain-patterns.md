# Domain Type Patterns

Reference patterns for creating domain types in `internal/domain/`.

---

## Basic Resource Type

```go
package domain

// Resource represents a resource from Nylas API.
type Resource struct {
    ID        string `json:"id"`
    GrantID   string `json:"grant_id"`
    Object    string `json:"object,omitempty"`
    Name      string `json:"name,omitempty"`
    CreatedAt int64  `json:"created_at,omitempty"`
    UpdatedAt int64  `json:"updated_at,omitempty"`
}
```

---

## Resource with Nested Types

```go
// Contact with nested email/phone types
type Contact struct {
    ID           string         `json:"id"`
    GrantID      string         `json:"grant_id"`
    GivenName    string         `json:"given_name,omitempty"`
    Surname      string         `json:"surname,omitempty"`
    Emails       []ContactEmail `json:"emails,omitempty"`
    PhoneNumbers []ContactPhone `json:"phone_numbers,omitempty"`
}

type ContactEmail struct {
    Email string `json:"email"`
    Type  string `json:"type,omitempty"` // home, work, school, other
}

type ContactPhone struct {
    Number string `json:"number"`
    Type   string `json:"type,omitempty"`
}
```

---

## Query Parameters Type

```go
// ResourceQueryParams defines query parameters for listing resources.
type ResourceQueryParams struct {
    Limit     int    `json:"limit,omitempty"`
    PageToken string `json:"page_token,omitempty"`
    // Add resource-specific filters
    Status    string `json:"status,omitempty"`
    Search    string `json:"search,omitempty"`
}
```

---

## List Response Type

```go
// ResourceListResponse represents paginated list response.
type ResourceListResponse struct {
    Data          []Resource `json:"data"`
    NextCursor    string     `json:"next_cursor,omitempty"`
    RequestID     string     `json:"request_id,omitempty"`
}
```

---

## Helper Methods

```go
// DisplayName returns a formatted display name.
func (r Resource) DisplayName() string {
    if r.Name != "" {
        return r.Name
    }
    return r.ID
}

// PrimaryEmail returns the primary email address.
func (c Contact) PrimaryEmail() string {
    for _, e := range c.Emails {
        if e.Type == "primary" || e.Type == "work" {
            return e.Email
        }
    }
    if len(c.Emails) > 0 {
        return c.Emails[0].Email
    }
    return ""
}
```

---

## Create/Update Request Types

```go
// CreateResourceRequest represents a request to create a resource.
type CreateResourceRequest struct {
    Name        string `json:"name"`
    Description string `json:"description,omitempty"`
}

// UpdateResourceRequest represents a request to update a resource.
type UpdateResourceRequest struct {
    Name        *string `json:"name,omitempty"`
    Description *string `json:"description,omitempty"`
}
```

---

## File Location

- Main types: `internal/domain/{resource}.go`
- Tests: `internal/domain/domain_test_basic.go` or `domain_test_advanced.go`

---

## Checklist

- [ ] All fields have JSON tags
- [ ] Optional fields use `omitempty`
- [ ] Pointer types for optional update fields
- [ ] Helper methods for common operations
- [ ] Tests for helper methods
