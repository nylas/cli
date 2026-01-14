# Nylas API Adapter

This package implements the `ports.NylasClient` interface for the Nylas v3 API.

## File Organization

| File | Purpose |
|------|---------|
| `client.go` | Core HTTP client, authentication, base URL configuration |
| `messages.go` | Email message CRUD operations (list, get, send, update, delete) |
| `drafts.go` | Draft CRUD operations |
| `threads.go` | Email thread operations |
| `folders.go` | Folder/label management |
| `attachments.go` | File attachment operations |
| `calendars.go` | Calendar CRUD and availability |
| `contacts.go` | Contact management |
| `webhooks.go` | Webhook configuration |
| `auth.go` | OAuth and authentication flows |
| `scheduler.go` | Scheduling pages and bookings |
| `notetakers.go` | Meeting notetaker (Nylas Notetaker API) |
| `admin.go` | Admin operations (applications, grants) |
| `inbound.go` | Inbound email parsing |

## Special Files

| File | Purpose |
|------|---------|
| `mock.go` | Mock implementation for unit testing |
| `demo.go` | Demo data generation for screenshots/demos |

## Usage Pattern

```go
// Create client with API key
client := nylas.NewClient(apiKey, "us")

// All methods accept context for cancellation/timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use domain types for requests and responses
messages, err := client.ListMessages(ctx, grantID, &domain.MessageQueryParams{
    Limit: 50,
})
```

## API Version

This adapter supports **Nylas v3 API only**. Do not use v1 or v2 endpoints.

- US Region: `https://api.us.nylas.com/v3/`
- EU Region: `https://api.eu.nylas.com/v3/`

## Adding New Endpoints

1. Add method signature to `internal/ports/nylas.go`
2. Implement method in appropriate file (or create new file for new resource)
3. Add mock implementation in `mock.go`
4. Add tests in `*_test.go`
