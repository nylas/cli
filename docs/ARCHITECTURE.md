# Architecture

Hexagonal (ports and adapters) architecture for clean separation of concerns.

> **Quick Links:** [README](../README.md) | [Commands](COMMANDS.md) | [Development](DEVELOPMENT.md)

---

## Project Structure

```
cmd/nylas/                    # Entry point (main.go)
internal/
  domain/                     # Business entities
  ports/                      # Interface contracts
  adapters/                   # Implementations
    nylas/                    # Nylas API client
    ai/                       # AI providers (Claude, OpenAI, Groq, Ollama)
    analytics/                # Focus optimizer, meeting scorer
    keyring/                  # Secret storage
    grantcache/               # Non-secret local grant metadata/default cache
    config/                   # Configuration validation
    mcp/                      # MCP proxy server
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
    integration/              # CLI integration tests
    mcp/                      # MCP server command
    notetaker/                # Meeting notetaker
    otp/                      # OTP extraction
    scheduler/                # Booking pages
    setup/                    # First-time setup wizard (nylas init)
    timezone/                 # Timezone utilities
    update/                   # Self-update
    webhook/                  # Webhook management
  tui/                        # Terminal UI
  app/                        # Shared app logic (auth, otp)
  testutil/                   # Test utilities
  util/                       # General utilities
docs/                         # Documentation
  commands/                   # Command guides
  ai/                         # AI docs
  security/                   # Security docs
  troubleshooting/            # Troubleshooting
  development/                # Dev guides
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
| Timezone service | `internal/adapters/utilities/timezone/` |
| **User Interfaces** | |
| TUI terminal client | `internal/tui/` |
| **Tests** | |
| CLI integration tests | `internal/cli/integration/*_test.go` |
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

1. **Domain** (`internal/domain/`)
   - Pure business logic, no external dependencies
   - Core types: Message, Email, Calendar, Event, Contact, Grant, Webhook
   - Feature types: AI, Analytics, Admin, Scheduler, Notetaker, Agent
   - Support types: Config, Errors, Provider, Utilities
   - Shared interfaces: `interfaces.go` (Paginated, QueryParams, Resource, Timestamped, Validator)

   **Key type relationships:**
   - `Person` - Base type with Name/Email (in `calendar.go`)
   - `EmailParticipant` - Type alias for `Person` (in `email.go`)
   - `Participant` - Embeds `Person`, adds Status/Comment for calendar events

2. **Ports** (`internal/ports/`)

   One file per interface; the main ones:
   - `nylas.go` - NylasClient interface (main API operations)
   - `secrets.go` - SecretStore interface (credential storage)
   - `llm.go` - LLM interface (AI providers)
   - `config.go` - Config interface
   - `webhook_server.go` - Webhook server interface

3. **Adapters** (`internal/adapters/`)

   | Adapter | Purpose |
   |---------|---------|
   | `nylas/` | Nylas API client (messages, calendars, contacts, events) |
   | `ai/` | AI clients (Claude, OpenAI, Groq, Ollama), email analyzer |
   | `analytics/` | Focus optimizer, conflict resolver, meeting scorer |
   | `keyring/` | Secret storage (system keyring, encrypted file fallback) |
   | `grantcache/` | Non-secret local grant metadata/default cache |
   | `mcp/` | MCP proxy server for AI assistants |
   | `config/` | Configuration validation |
   | `oauth/` | OAuth callback server |
   | `utilities/` | Services (contacts, email, scheduling, timezone, webhook) |
   | `browser/` | Browser automation |
   | `tunnel/` | Cloudflare tunnel |
   | `webhookserver/` | Webhook server |

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

The CLI provides a terminal-based interface:

| Interface | Command | Port | Purpose | Location |
|-----------|---------|------|---------|----------|
| **TUI** | `nylas tui` | N/A | Terminal-based email/calendar client | `internal/tui/` |

---

**For detailed implementation, see `CLAUDE.md` and `docs/DEVELOPMENT.md`**
