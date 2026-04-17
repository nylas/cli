# CLI Integration Tests Index

Quick reference for finding integration tests by feature.

## Test Files

| File | Tests | Command/Feature |
|------|-------|-----------------|
| `admin_test.go` | Admin operations | `nylas admin` (applications, connectors, grants) |
| `ai_test.go` | Core AI operations | `nylas ai suggest`, `nylas ai compose` |
| `ai_break_awareness_test.go` | Break awareness | AI scheduling around breaks |
| `ai_calendar_lifecycle_test.go` | Calendar AI lifecycle | End-to-end AI calendar operations |
| `ai_config_test.go` | AI configuration | AI provider setup, API keys |
| `ai_email_test.go` | AI email features | Smart compose, email analysis |
| `ai_features_test.go` | AI feature integration | Cross-feature AI tests |
| `ai_pattern_learning_test.go` | Pattern learning | AI learning from user behavior |
| `ai_working_hours_test.go` | Working hours AI | AI respecting working hours config |
| `attachments_test.go` | Attachment operations | `nylas email attachment` |
| `auth_test.go` | Authentication | `nylas auth login/logout/status` |
| `auth_enhancements_test.go` | Auth enhancements | Enhanced auth flows |
| `calendar_test.go` | Calendar operations | `nylas calendar` (events, availability) |
| `contacts_test.go` | Contact operations | `nylas contacts` |
| `contact_enhancements_test.go` | Contact enhancements | Group operations, advanced search |
| `drafts_test.go` | Draft operations | `nylas drafts` |
| `email_test.go` | Email operations | `nylas email` (list, send, search) |
| `folders_test.go` | Folder operations | `nylas folders` |
| `inbound_removed_test.go` | Removed inbound command | Unknown-command behavior and help omission |
| `metadata_test.go` | Metadata operations | Email/event metadata |
| `misc_test.go` | Miscellaneous | Version, help, config |
| `notetaker_test.go` | Notetaker operations | `nylas notetaker` |
| `otp_test.go` | OTP operations | One-time password flows |
| `recurring_events_test.go` | Recurring events | `nylas calendar` recurring events |
| `scheduled_messages_test.go` | Scheduled messages | `nylas email schedule` |
| `scheduler_test.go` | Scheduler operations | `nylas scheduler` (pages, bookings) |
| `smart_compose_test.go` | Smart compose | `nylas email compose --smart` |
| `threads_test.go` | Thread operations | `nylas threads` |
| `timezone_test.go` | Timezone utilities | `nylas timezone` (offline) |
| `virtual_calendar_test.go` | Virtual calendars | `nylas calendar virtual` |
| `webhooks_test.go` | Webhook operations | `nylas webhooks` |

## Helper Files

| File | Purpose |
|------|---------|
| `helpers_test.go` | Shared test helpers (`runCLI`, `runCLIWithRateLimit`, etc.) |
| `main_test.go` | Test setup, environment validation |

## Running Tests

```bash
# All integration tests
make test-integration

# Fast mode (skip LLM tests)
make test-integration-fast

# Specific test file
go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_Email"

# Single test
go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_EmailSend"
```

## Environment Variables

Environment variables can be set in a `.env` file at the project root. The file is automatically loaded when tests run.

**Example `.env` file:**
```bash
NYLAS_API_KEY=nyk_v0_xxx
NYLAS_GRANT_ID=abc123
NYLAS_CLIENT_ID=xxx
NYLAS_CLIENT_SECRET=xxx
```

### Required Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `NYLAS_API_KEY` | Yes | Nylas API key |
| `NYLAS_GRANT_ID` | Yes | Grant ID for testing |
| `NYLAS_TEST_BINARY` | No | Path to CLI binary (auto-detected) |
| `NYLAS_CLIENT_ID` | Some | OAuth client ID (auth tests) |
| `NYLAS_CLIENT_SECRET` | Some | OAuth client secret (auth tests) |
| `NYLAS_TEST_RATE_LIMIT_RPS` | No | Rate limit requests per second |
| `NYLAS_TEST_RATE_LIMIT_BURST` | No | Rate limit burst size |

## Test Categories

### Offline Tests (No API Required)
- `timezone_test.go` - All tests run offline
- `misc_test.go` - Version, help commands

### API Tests (Require Credentials)
- All other tests require valid API credentials
- Use `skipIfMissingCreds(t)` helper

### LLM Tests (Slow)
- `ai_*.go` files may invoke LLM providers
- Use `make test-integration-fast` to skip

## Adding New Tests

1. Create file: `internal/cli/integration/<feature>_test.go`
2. Add build tags:
   ```go
   //go:build integration
   // +build integration
   ```
3. Use `package integration`
4. Follow existing test patterns
5. Update this INDEX.md
