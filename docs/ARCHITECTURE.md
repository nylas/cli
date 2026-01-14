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
    keyring/                  # Secret storage
    config/                   # Configuration validation
    mcp/                      # MCP proxy server
    utilities/                # Scheduling, contacts services
    oauth/                    # OAuth callback server
    browser/                  # Browser automation
    tunnel/                   # Cloudflare tunnel
    webhookserver/            # Webhook server
  cli/                        # CLI commands
    common/                   # Shared helpers (client, context, errors, flags, format, html, timeutil)
    admin/                    # API key management
    auth/                     # Authentication
    calendar/                 # Calendar & events
    contacts/                 # Contact management
    demo/                     # Demo workflows
    email/                    # Email operations
    inbound/                  # Inbound email rules
    integration/              # CLI integration tests
    mcp/                      # MCP server command
    notetaker/                # Meeting notetaker
    scheduler/                # Booking pages
    update/                   # Self-update
    webhook/                  # Webhook management
  app/                        # Shared app logic (auth)
  testutil/                   # Test utilities
  util/                       # General utilities
docs/                         # Documentation
  commands/                   # Command guides
  security/                   # Security docs
  troubleshooting/            # Troubleshooting guides
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
| MCP server | `internal/adapters/mcp/` |
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
| **App Services** | `internal/app/` | Orchestrates adapters for workflows (auth login) |
| **CLI Helpers** | `internal/cli/common/` | Reusable utilities (context, format, colors, pagination) |
| **Adapter Helpers** | `internal/adapters/nylas/client_helpers.go` | HTTP helpers, request building, response handling |

> **Key difference:** App services coordinate multiple adapters. CLI helpers are stateless utilities. Adapter helpers handle API specifics.

#### CLI Common Helpers (`internal/cli/common/`)

| File | Purpose |
|------|---------|
| `client.go` | Nylas client creation and credential access |
| `colors.go` | Shared color definitions |
| `config.go` | Config store access |
| `context.go` | Context creation with timeouts |
| `errors.go` | Error wrapping and formatting |
| `flags.go` | Common CLI flag definitions |
| `format.go` | Display formatting and output helpers |
| `html.go` | HTML-to-text conversion |
| `logger.go` | Logging utilities |
| `pagination.go` | Pagination helpers |
| `path.go` | Safe executable path handling |
| `progress.go` | Progress indicators |
| `retry.go` | Retry logic with backoff |
| `string.go` | String utilities |
| `time.go` | Time formatting and parsing |
| `timeutil.go` | Date parsing and formatting |

#### Adapter Helpers (`internal/adapters/nylas/client_helpers.go`)

| Helper | Purpose |
|--------|---------|
| `doGet`, `doGetWithNotFound`, `doDelete` | HTTP request helpers |
| `ListResponse[T]` | Generic paginated response type |
| `QueryBuilder` | Fluent URL query parameter builder |

---

## Design Principles

### Hexagonal Architecture

**Three layers:**

1. **Domain** (`internal/domain/`)
   - Pure business logic, no external dependencies
   - Core types: Message, Email, Calendar, Event, Contact, Grant, Webhook
   - Feature types: Admin, Scheduler, Notetaker, Inbound
   - Support types: Config, Errors, Provider, Utilities
   - Shared interfaces: `interfaces.go` (Paginated, QueryParams, Resource, Timestamped, Validator)

   **Key type relationships:**
   - `Person` - Base type with Name/Email (in `calendar.go`)
   - `EmailParticipant` - Type alias for `Person` (in `email.go`)
   - `Participant` - Embeds `Person`, adds Status/Comment for calendar events

2. **Ports** (`internal/ports/`) - Interface files
   - `nylas.go` - NylasClient interface (main API operations)
   - `secrets.go` - SecretStore interface (credential storage)
   - `config.go` - Config interface
   - `utilities.go` - Utilities interface
   - `webhook_server.go` - Webhook server interface

3. **Adapters** (`internal/adapters/`) - Adapter directories

   | Adapter | Purpose |
   |---------|---------|
   | `nylas/` | Nylas API client (messages, calendars, contacts, events) |
   | `keyring/` | Credential storage (system keyring, file-based) |
   | `mcp/` | MCP proxy server for AI assistants |
   | `config/` | Configuration validation |
   | `oauth/` | OAuth callback server |
   | `utilities/` | Services (contacts, email, scheduling, webhook) |
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

**For detailed implementation, see `CLAUDE.md` and `docs/DEVELOPMENT.md`**
