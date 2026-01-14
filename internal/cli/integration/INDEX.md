# CLI Integration Tests Index

Quick reference for finding integration tests by feature.

## Test Files

| File | Tests | Command/Feature |
|------|-------|-----------------|
| `admin_test.go` | Admin operations | `nylas admin` (applications, connectors, grants) |
| `attachments_test.go` | Attachment operations | `nylas email attachment` |
| `auth_enhancements_test.go` | Auth enhancements | Enhanced auth flows |
| `auth_test_basic.go` | Basic auth | `nylas auth login/logout/status` |
| `auth_test_guarded.go` | Guarded auth | Token validation, grant guarding |
| `auth_test_management.go` | Auth management | Token management operations |
| `calendar_test_availability.go` | Calendar availability | `nylas calendar availability` |
| `calendar_test_commands.go` | Calendar commands | `nylas calendar` command structure |
| `calendar_test_crud.go` | Calendar CRUD | Calendar create/read/update/delete |
| `calendar_test_events.go` | Calendar events | Event operations |
| `contacts_test.go` | Contact operations | `nylas contacts` |
| `contact_enhancements_test.go` | Contact enhancements | Group operations, advanced search |
| `drafts_test.go` | Draft operations | `nylas drafts` |
| `email_delete_metadata_test.go` | Email delete/metadata | Delete and metadata operations |
| `email_list_read_test.go` | Email list/read | `nylas email list`, `nylas email read` |
| `email_send_test.go` | Email send | `nylas email send` |
| `email_threads_test.go` | Email threads | Thread-related email operations |
| `folders_test.go` | Folder operations | `nylas folders` |
| `inbound_test.go` | Inbound email | `nylas inbound` (managed inboxes) |
| `metadata_test.go` | Metadata operations | Email/event metadata |
| `misc_test.go` | Miscellaneous | Version, help, config |
| `notetaker_test.go` | Notetaker operations | `nylas notetaker` |
| `recurring_events_test.go` | Recurring events | `nylas calendar` recurring events |
| `scheduled_messages_test.go` | Scheduled messages | `nylas email schedule` |
| `scheduler_test_advanced.go` | Scheduler advanced | Advanced scheduler operations |
| `scheduler_test_basic.go` | Scheduler basic | `nylas scheduler` (pages, bookings) |
| `threads_test.go` | Thread operations | `nylas threads` |
| `virtual_calendar_test.go` | Virtual calendars | `nylas calendar virtual` |
| `webhooks_test.go` | Webhook operations | `nylas webhooks` |

## Helper Files

| File | Purpose |
|------|---------|
| `test.go` | Shared test helpers (`runCLI`, `runCLIWithRateLimit`, rate limiting, etc.) |

## Running Tests

```bash
# All integration tests
make test-integration

# Specific test file
go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_Email"

# Single test
go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_EmailSend"
```

## Required Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `NYLAS_API_KEY` | Yes | Nylas API key |
| `NYLAS_GRANT_ID` | Yes | Grant ID for testing |
| `NYLAS_TEST_BINARY` | Yes | Path to CLI binary (`./bin/nylas`) |
| `NYLAS_CLIENT_ID` | Some | OAuth client ID (auth tests) |
| `NYLAS_CLIENT_SECRET` | Some | OAuth client secret (auth tests) |
| `NYLAS_TEST_RATE_LIMIT_RPS` | No | Rate limit requests per second |
| `NYLAS_TEST_RATE_LIMIT_BURST` | No | Rate limit burst size |

## Test Categories

### Offline Tests (No API Required)
- `misc_test.go` - Version, help commands

### API Tests (Require Credentials)
- All other tests require valid API credentials
- Use `skipIfMissingCreds(t)` helper

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
