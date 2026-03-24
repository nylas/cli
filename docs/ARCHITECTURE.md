# Architecture

Hexagonal (ports and adapters) architecture for clean separation of concerns.

> **Quick Links:** [README](../README.md) | [Commands](COMMANDS.md) | [Development](DEVELOPMENT.md)

---

## Project Structure

```
cmd/nylas/                    # Entry point (main.go)
internal/
  domain/                     # Business entities (28 files)
  ports/                      # Interface contracts (7 files)
  adapters/                   # Implementations
    nylas/                    # Nylas API client (94 files)
    ai/                       # AI providers (Claude, OpenAI, Groq, Ollama)
    analytics/                # Focus optimizer, meeting scorer
    keyring/                  # Secret storage
    config/                   # Configuration validation
    mcp/                      # MCP proxy server
    slack/                    # Slack API client
    utilities/                # Timezone, scheduling, contacts services
    oauth/                    # OAuth callback server
    browser/                  # Browser automation
    tunnel/                   # Cloudflare tunnel
    webhookserver/            # Webhook server
  cli/                        # CLI commands
    common/                   # Shared helpers (client, context, errors, flags, format, html, timeutil)
    admin/                    # API key management
    ai/                       # AI commands
    auth/                     # Authentication
    calendar/                 # Calendar & events
    contacts/                 # Contact management
    email/                    # Email operations
    inbound/                  # Inbound email rules
    integration/              # CLI integration tests
    mcp/                      # MCP server command
    notetaker/                # Meeting notetaker
    otp/                      # OTP extraction
    scheduler/                # Booking pages
    setup/                    # First-time setup wizard (nylas init)
    slack/                    # Slack integration
    timezone/                 # Timezone utilities
    update/                   # Self-update
    webhook/                  # Webhook management
  ui/                         # Web UI (port 7363)
  air/                        # Web email client (port 7365)
  tui/                        # Terminal UI
  app/                        # Shared app logic (auth, otp)
  testutil/                   # Test utilities
  util/                       # General utilities
docs/                         # Documentation
  commands/                   # Command guides (12 files)
  ai/                         # AI docs (8 files)
  security/                   # Security docs (2 files)
  troubleshooting/            # Troubleshooting (5 files)
  development/                # Dev guides (4 files)
.claude/                      # Claude configuration
  commands/                   # Skills/commands
  agents/                     # Agent definitions
  rules/                      # Project rules
  shared/patterns/            # Code patterns
```

### Quick Lookup

| Looking for | Location |
|-------------|----------|
| **CLI** | |
| CLI entry & registration | `cmd/nylas/main.go` |
| CLI helpers (context, config) | `internal/cli/common/` |
| CLI integration tests | `internal/cli/integration/` |
| **Adapters** | |
| Nylas HTTP client | `internal/adapters/nylas/client.go` |
| AI providers | `internal/adapters/ai/` |
| MCP server | `internal/adapters/mcp/` |
| Slack adapter | `internal/adapters/slack/` |
| Timezone service | `internal/adapters/utilities/timezone/` |
| **User Interfaces** | |
| Air web client (port 7365) | `internal/air/` |
| UI config tool (port 7363) | `internal/ui/` |
| TUI terminal client | `internal/tui/` |
| **Tests** | |
| CLI integration tests | `internal/cli/integration/*_test.go` |
| Air integration tests | `internal/air/integration_*_test.go` |
| Test utilities | `internal/testutil/` |
| **Docs** | |
| Documentation index | `docs/INDEX.md` |
| Command reference | `docs/COMMANDS.md` |
| Claude rules | `.claude/rules/` |

### Helper Layers (Avoid Duplicates)

| Layer | Location | Purpose |
|-------|----------|---------|
| **App Services** | `internal/app/` | Orchestrates adapters for workflows (auth login, OTP extraction) |
| **CLI Helpers** | `internal/cli/common/` | Reusable utilities (context, format, colors, pagination) |
| **Adapter Helpers** | `internal/adapters/nylas/client_helpers.go` | HTTP helpers, request building, response handling |
| **Air Helpers** | `internal/air/handlers_helpers.go` | Handler utilities (config checks, JSON parsing, demo mode) |

> **Key difference:** App services coordinate multiple adapters. CLI helpers are stateless utilities. Adapter helpers handle API specifics.

#### CLI Common Helpers (`internal/cli/common/`)

| File | Helpers | Purpose |
|------|---------|---------|
| `client.go` | `GetNylasClient()`, `GetCachedNylasClient()`, `ResetCachedClient()`, `GetAPIKey()`, `GetGrantID()` | Nylas client creation and credential access |
| `colors.go` | `Bold`, `BoldWhite`, `Dim`, `Cyan`, `Green`, `Yellow`, `Red`, `Blue` | Shared color definitions |
| `config.go` | `GetConfigStore()`, `GetConfigPath()` | Config store access from commands |
| `context.go` | `CreateContext()`, `CreateContextWithTimeout()` | Context creation with timeouts |
| `errors.go` | `WrapError()`, `FormatError()`, `PrintFormattedError()`, `NewUserError()`, `NewInputError()`, `WrapGetError()`, `WrapFetchError()`, `WrapCreateError()`, `WrapUpdateError()`, `WrapDeleteError()`, `WrapSendError()` | Error wrapping and formatting |
| `flags.go` | `AddLimitFlag()`, `AddFormatFlag()`, `AddIDFlag()`, `AddPageTokenFlag()`, `AddForceFlag()`, `AddYesFlag()`, `AddVerboseFlag()` | Common CLI flag definitions |
| `format.go` | `ParseFormat()`, `NewFormatter()`, `NewTable()`, `FormatParticipant()`, `FormatParticipants()`, `FormatSize()`, `PrintEmptyState()`, `PrintEmptyStateWithHint()`, `PrintListHeader()`, `PrintSuccess()`, `PrintError()`, `PrintWarning()`, `PrintInfo()`, `Confirm()` | Display formatting and output helpers |
| `html.go` | `StripHTML()`, `RemoveTagWithContent()` | HTML-to-text conversion |
| `logger.go` | `InitLogger()`, `GetLogger()`, `IsDebug()`, `IsQuiet()`, `Debug()`, `Info()`, `Warn()`, `Error()`, `DebugHTTP()`, `DebugAPI()` | Logging utilities |
| `pagination.go` | `FetchAllPages()`, `FetchAllWithProgress()`, `NewPaginatedDisplay()`, `PageResult[T]` | Pagination helpers |
| `path.go` | `ValidateExecutablePath()`, `FindExecutableInPath()`, `SafeCommand()` | Safe executable path handling |
| `progress.go` | `NewSpinner()`, `NewProgressBar()`, `NewCounter()` | Progress indicators |
| `retry.go` | `WithRetry()`, `DefaultRetryConfig()`, `NoRetryConfig()`, `IsRetryable()`, `IsRetryableStatusCode()` | Retry logic with backoff |
| `string.go` | `Truncate()` | String utilities |
| `time.go` | `FormatTimeAgo()`, `ParseTimeOfDay()`, `ParseTimeOfDayInLocation()`, `ParseDuration()` | Time formatting and parsing |
| `timeutil.go` | `ParseDate()`, `ParseTime()`, `FormatDate()`, `FormatDisplayDate()` + constants | Date parsing and formatting |

#### Adapter Helpers (`internal/adapters/nylas/client_helpers.go`)

| Helper | Purpose |
|--------|---------|
| `doGet(ctx, url, &result)` | GET request with JSON decoding |
| `doGetWithNotFound(ctx, url, &result, notFoundErr)` | GET with 404 handling |
| `doDelete(ctx, url)` | DELETE request (accepts 200/204) |
| `ListResponse[T]` | Generic paginated response type |
| `QueryBuilder` | Fluent URL query parameter builder |

**QueryBuilder methods:**
- `NewQueryBuilder()` - Create new builder
- `Add(key, value)` - Add string value (if non-empty)
- `AddInt(key, value)` - Add int value (if > 0)
- `AddInt64(key, value)` - Add int64 value (if > 0)
- `AddBool(key, value)` - Add bool value (if true)
- `AddBoolPtr(key, value)` - Add bool pointer (if non-nil)
- `AddSlice(key, values)` - Add multiple values with same key
- `Encode()` - Get encoded query string
- `Values()` - Get underlying url.Values
- `BuildURL(baseURL)` - Append query to URL

**QueryBuilder usage:**
```go
qb := NewQueryBuilder().
    Add("limit", "50").
    AddInt("offset", params.Offset).
    AddBoolPtr("unread", params.Unread)
url := qb.BuildURL(baseURL)
```

---

## Design Principles

### Hexagonal Architecture

**Three layers:**

1. **Domain** (`internal/domain/`) - 29 files
   - Pure business logic, no external dependencies
   - Core types: Message, Email, Calendar, Event, Contact, Grant, Webhook
   - Feature types: AI, Analytics, Admin, Scheduler, Notetaker, Slack, Inbound
   - Support types: Config, Errors, Provider, Utilities
   - Shared interfaces: `interfaces.go` (Paginated, QueryParams, Resource, Timestamped, Validator)

   **Key type relationships:**
   - `Person` - Base type with Name/Email (in `calendar.go`)
   - `EmailParticipant` - Type alias for `Person` (in `email.go`)
   - `Participant` - Embeds `Person`, adds Status/Comment for calendar events

2. **Ports** (`internal/ports/`) - 7 interface files
   - `nylas.go` - NylasClient interface (main API operations)
   - `secrets.go` - SecretStore interface (credential storage)
   - `llm.go` - LLM interface (AI providers)
   - `slack.go` - Slack interface
   - `config.go` - Config interface
   - `utilities.go` - Utilities interface
   - `webhook_server.go` - Webhook server interface

3. **Adapters** (`internal/adapters/`) - 12 adapter directories

   | Adapter | Files | Purpose |
   |---------|-------|---------|
   | `nylas/` | 94 | Nylas API client (messages, calendars, contacts, events) |
   | `ai/` | 24 | AI clients (Claude, OpenAI, Groq, Ollama), email analyzer |
   | `analytics/` | 14 | Focus optimizer, conflict resolver, meeting scorer |
   | `keyring/` | 6 | Credential storage (system keyring, file-based) |
   | `mcp/` | 8 | MCP proxy server for AI assistants |
   | `slack/` | 21 | Slack API client (channels, messages, users) |
   | `config/` | 5 | Configuration validation |
   | `oauth/` | 3 | OAuth callback server |
   | `utilities/` | 12 | Services (contacts, email, scheduling, timezone, webhook) |
   | `browser/` | 2 | Browser automation |
   | `tunnel/` | 2 | Cloudflare tunnel |
   | `webhookserver/` | 2 | Webhook server |

**Benefits:**
- Testability (mock adapters)
- Flexibility (swap implementations)
- Clean separation of concerns

---

## Working Hours and Breaks

Calendar enforces working hours (soft warnings) and break blocks (hard constraints).

**Domain models:**
- `WorkingHoursConfig` - Per-day working hours with break periods
- `DaySchedule` - Working hours for specific weekday
- `BreakBlock` - Break periods (lunch, coffee) with hard constraints

**Configuration:** `~/.nylas/config.yaml`
**Implementation:** `internal/cli/calendar/helpers.go` (`checkBreakViolation()`)
**Tests:** `internal/cli/calendar/helpers_test.go`

**Details:** See [commands/timezone.md](commands/timezone.md#working-hours--break-management)

---

## Domain Interfaces

Shared interfaces in `internal/domain/interfaces.go` enable generic programming:

| Interface | Purpose | Implemented By |
|-----------|---------|----------------|
| `Paginated` | Resources with pagination info | `MessageListResponse`, `EventListResponse`, etc. |
| `QueryParams` | Query parameter types | `MessageQueryParams`, `EventQueryParams`, etc. |
| `Resource` | Resources with ID and GrantID | `Message`, `Event`, `Contact`, etc. |
| `Timestamped` | Resources with timestamps | `Message`, `Event`, `Draft`, etc. |
| `Validator` | Self-validating types | `EventWhen`, `SendMessageRequest`, `BreakBlock` |

**Type embedding example:**
```go
// Person is the base type for email/calendar participants
type Person struct {
    Name  string `json:"name,omitempty"`
    Email string `json:"email"`
}

// Participant embeds Person and adds calendar-specific fields
type Participant struct {
    Person
    Status  string `json:"status,omitempty"`
    Comment string `json:"comment,omitempty"`
}

// EmailParticipant is an alias for Person (backward compatibility)
type EmailParticipant = Person
```

---

## CLI Pattern

Each feature follows consistent structure:

```
internal/cli/<feature>/
  ├── <feature>.go    # Main command
  ├── list.go         # List subcommand
  ├── create.go       # Create subcommand
  ├── update.go       # Update subcommand
  ├── delete.go       # Delete subcommand
  └── helpers.go      # Shared helpers
```

---

## User Interfaces

The CLI provides three different interfaces:

| Interface | Command | Port | Purpose | Location |
|-----------|---------|------|---------|----------|
| **TUI** | `nylas tui` | N/A | Terminal-based email/calendar client | `internal/tui/` |
| **UI** | `nylas ui` | 7363 | Web-based CLI configuration tool | `internal/ui/` |
| **Air** | `nylas air` | 7365 | Full web-based email/calendar client | `internal/air/` |

> **Which to use?** TUI for terminal lovers, Air for browser-based email client, UI for API credential setup.

---

## Air (Web Email Client)

**Air** is a full-featured web-based email client for Nylas CLI, providing browser interface for email, calendar, and productivity features.

### Architecture

- **Location:** `internal/air/`
- **Server:** HTTP server with middleware stack (CORS, compression, security, caching)
- **Handlers:** Feature-specific HTTP handlers (email, calendar, contacts, AI)
- **Templates:** Go templates with Tailwind CSS
- **Port:** Default `:7365` (configurable)

### File Organization

**All files are ≤500 lines for maintainability.** Large files have been refactored into focused modules:

**Server Core** (refactored from server.go):
- `server.go` - Server struct definition
- `server_lifecycle.go` - Initialization, routing, lifecycle
- `server_stores.go` - Cache store accessors
- `server_sync.go` - Background sync logic
- `server_offline.go` - Offline queue processing
- `server_converters.go` - Domain to cache conversions
- `server_template.go` - Template handling
- `server_modules_test.go` - Unit tests

**Handler Helpers:**
- `handlers_helpers.go` - Common handler utilities (see pattern below)

**Handlers** (organized by feature):
- Email: `handlers_email.go`, `handlers_drafts.go`, `handlers_bundles.go`
- Calendar: `handlers_calendars.go`, `handlers_events.go`, `handlers_calendar_helpers.go`
- Contacts: `handlers_contacts.go`, `handlers_contacts_crud.go`, `handlers_contacts_search.go`, `handlers_contacts_helpers.go`
- AI: `handlers_ai_types.go`, `handlers_ai_summarize.go`, `handlers_ai_smart.go`, `handlers_ai_thread.go`, `handlers_ai_complete.go`, `handlers_ai_config.go`
- Productivity: `handlers_scheduled_send.go`, `handlers_undo_send.go`, `handlers_templates.go`, `handlers_snooze_*.go`, `handlers_splitinbox_*.go`

**Other:**
- `middleware.go` - Middleware stack
- `data.go` - Data models
- `templates/` - HTML templates
- `integration_*.go` - Integration tests (organized by feature)

### Handler Helper Pattern

All HTTP handlers use common helpers for consistency and reduced boilerplate:

| Helper | Location | Purpose |
|--------|----------|---------|
| `withTimeout(r)` | `handlers_helpers.go` | Creates context with 30s default timeout |
| `requireConfig(w)` | `handlers_helpers.go` | Checks Nylas client is configured, writes error if not |
| `parseJSONBody[T](w, r, &dest)` | `handlers_helpers.go` | Generic JSON body parsing with error handling |
| `handleDemoMode(w, data)` | `handlers_helpers.go` | Returns demo response if in demo mode |
| `requireMethod(w, r, method)` | `handlers_helpers.go` | Validates HTTP method |
| `writeError(w, status, msg)` | `handlers_helpers.go` | Writes JSON error response |
| `requireDefaultGrant(w)` | `server_stores.go` | Gets default grant ID, writes error if not set |
| `getEmailStore(email)` | `server_stores.go` | Gets email cache store for account |
| `getEventStore(email)` | `server_stores.go` | Gets event cache store for account |
| `getContactStore(email)` | `server_stores.go` | Gets contact cache store for account |
| `getFolderStore(email)` | `server_stores.go` | Gets folder cache store for account |
| `getSyncStore(email)` | `server_stores.go` | Gets sync cache store for account |

**Standard handler pattern:**
```go
func (s *Server) handleX(w http.ResponseWriter, r *http.Request) {
    if s.handleDemoMode(w, demoData) { return }
    if !s.requireConfig(w) { return }
    grantID, ok := s.requireDefaultGrant(w)
    if !ok { return }
    ctx, cancel := s.withTimeout(r)
    defer cancel()
    // ... handler logic
}
```

**Complete file listing:** See `CLAUDE.md` for detailed file structure with line counts

### Integration Tests

Air integration tests are **split by feature** for better maintainability:

| File | Tests | Purpose |
|------|-------|---------|
| `integration_base_test.go` | 0 | Shared `testServer()` helper, utilities, rate limiting |
| `integration_core_test.go` | 5 | Config, Grants, Folders, Index page |
| `integration_email_test.go` | 4 | Email listing, filtering, drafts |
| `integration_calendar_test.go` | 11 | Calendars, events, availability, conflicts |
| `integration_contacts_test.go` | 4 | Contact CRUD operations |
| `integration_cache_test.go` | 4 | Cache store operations, invalidation |
| `integration_ai_test.go` | 15 | AI summarization, smart compose, thread analysis, config |
| `integration_middleware_test.go` | 6 | Compression, security headers, CORS |
| `integration_bundles_test.go` | 8 | Email bundles, categorization, bundle operations |
| `integration_productivity_test.go` | 8 | Scheduled send, undo send, snooze, reply later |

**Total:** 65 integration tests across 10 organized files

**Running tests:**
```bash
make ci-full                     # RECOMMENDED: Complete CI with automatic cleanup
make test-air-integration        # Run Air integration tests only
make test-cleanup                # Manual cleanup if needed
```

**Why cleanup?** Air tests create real resources (drafts, events, contacts) in the connected Nylas account. The `make ci-full` target automatically runs cleanup after all tests.

**Pattern:** Air tests use `httptest` to test HTTP handlers directly:
```go
func TestIntegration_Feature(t *testing.T) {
    server := testServer(t)  // Shared helper
    req := httptest.NewRequest(http.MethodGet, "/api/endpoint", nil)
    w := httptest.NewRecorder()
    server.handleEndpoint(w, req)
    // Assertions...
}
```

---

**For detailed implementation, see `CLAUDE.md` and `docs/DEVELOPMENT.md`**
