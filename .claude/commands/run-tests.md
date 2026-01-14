---
name: run-tests
description: Run unit and integration tests with proper configuration
allowed-tools: Read, Bash(go test:*), Bash(go build:*), Bash(make test:*)
---

# Run Tests

Run unit tests and/or integration tests for the nylas CLI.

> **This is the authoritative source for test commands.** Other files reference this document.

## Instructions

### Unit Tests

Run all unit tests:
```bash
go test ./...
```

Run tests for specific package:
```bash
go test ./internal/cli/email/...
go test ./internal/cli/webhook/...
go test ./internal/domain/...
go test ./internal/adapters/nylas/...
```

Run with verbose output:
```bash
go test -v ./...
```

Run with coverage:
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

Integration tests require credentials. Set environment variables:
```bash
# Required
export NYLAS_API_KEY="your-api-key"
export NYLAS_GRANT_ID="your-grant-id"
export NYLAS_TEST_BINARY="$(pwd)/bin/nylas"

# For inbound tests
export NYLAS_INBOUND_GRANT_ID="your-inbound-inbox-id"
export NYLAS_INBOUND_EMAIL="your-inbox@nylas.email"

# For email send tests
export NYLAS_TEST_SEND_EMAIL="true"       # Enable send email tests
export NYLAS_TEST_EMAIL="test@example.com" # Recipient for send tests
export NYLAS_TEST_CC_EMAIL="cc@example.com" # CC recipient (optional)

# Optional - for destructive tests
export NYLAS_TEST_DELETE="true"           # Enable delete tests (contacts, folders, webhooks, drafts, calendars)
export NYLAS_TEST_DELETE_MESSAGE="true"   # Enable email message delete tests
export NYLAS_TEST_CREATE_INBOUND="true"   # Enable inbound inbox create tests
export NYLAS_TEST_DELETE_INBOUND="true"   # Enable inbound inbox delete tests
```

Build binary first:
```bash
go build -o bin/nylas ./cmd/nylas
```

Run integration tests:
```bash
go test -tags=integration ./internal/cli/integration/...
```

Run specific integration test:
```bash
go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_EmailList"
```

### Common Test Patterns

If tests fail, check:
1. **Build errors**: Run `go build ./...` first
2. **Missing mocks**: Update `mock.go` if interface changed
3. **API changes**: Update expected values in tests
4. **Credentials**: Ensure env vars are set for integration tests

After fixing, always run full test suite:
```bash
go build ./... && go test ./...
```

### Make Targets (Recommended)

```bash
make ci-full              # Complete CI pipeline with cleanup (RECOMMENDED)
make ci                   # Quick CI (no integration tests)
make test-unit            # Unit tests only
make test-race            # Unit tests with race detector
make test-integration     # CLI integration tests
make test-coverage        # Generate coverage report
make test-cleanup         # Clean up test resources
```

**Note:** Integration tests create real resources. Use `make ci-full` for automatic cleanup.
