# Add New Nylas API Method

Add a new method to the Nylas API client following the hexagonal architecture.

**IMPORTANT: This CLI uses Nylas v3 API ONLY. All endpoints must use `/v3/` prefix.**
- v3 API Docs: https://developer.nylas.com/docs/api/v3/
- Base URL: `https://api.us.nylas.com/v3/` or `https://api.eu.nylas.com/v3/`

**Patterns:** See `add-command/references/` for code templates:
- `domain-patterns.md` - Domain type templates
- `adapter-patterns.md` - HTTP client implementation templates

## Instructions

1. First, ask me for:
   - Method name (e.g., GetAttachments, CreateDraft)
   - HTTP method and endpoint (e.g., GET /v3/grants/{grantID}/attachments)
   - Request parameters (path params, query params, body)
   - Response structure

2. Then update these files in order:

### Step 1: Domain Types (if needed)
File: `internal/domain/{resource}.go`
See `add-command/references/domain-patterns.md` for templates.

### Step 2: Port Interface
File: `internal/ports/nylas.go`
Add method signature to NylasClient interface.

### Step 3: HTTP Client Implementation
File: `internal/adapters/nylas/{resource}.go` (new file or existing)
See `add-command/references/adapter-patterns.md` for List/Get/Create/Update/Delete templates.

### Step 4: Mock Implementation
File: `internal/adapters/nylas/mock.go`
Add fields (`{Method}Called`, `{Method}Func`) and implement method.

3. After updating, run:
   - `go build ./...` to verify compilation
   - `go test ./internal/adapters/nylas/...` to run adapter tests
