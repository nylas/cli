# Integration Test Patterns

Shared patterns for Go integration tests in the Nylas CLI project.

> **This is the authoritative source for rate limiting patterns.** Other files reference this document.

---

## CLI Integration Tests

### Location & Build Tags

```go
//go:build integration
// +build integration

package integration
```

Location: `internal/cli/integration/{feature}_test.go`

---

## CLI Integration Test Template

```go
func TestCLI_FeatureName(t *testing.T) {
    skipIfMissingCreds(t)
    t.Parallel()

    // Rate limit for API calls
    acquireRateLimit(t)

    // Create test resource
    resource, err := createTestResource(t)
    require.NoError(t, err)

    // Cleanup after test
    t.Cleanup(func() {
        acquireRateLimit(t)
        _ = deleteTestResource(t, resource.ID)
    })

    // Test the CLI command
    stdout, stderr, err := runCLIWithRateLimit(t, "command", "subcommand", "--flag", "value")
    require.NoError(t, err)
    assert.Empty(t, stderr)
    assert.Contains(t, stdout, "expected output")
}
```

---

## Rate Limiting (CRITICAL)

ALWAYS use rate limiting for API calls in parallel tests:

```go
acquireRateLimit(t)  // Call before each API operation
```

**Rate limiting config:**
```bash
export NYLAS_TEST_RATE_LIMIT_RPS="2.0"    # Requests per second
export NYLAS_TEST_RATE_LIMIT_BURST="5"    # Burst size
```

| Command Type | Rate Limit |
|--------------|------------|
| API commands (calendar, email, contacts) | Required |
| Offline commands (version, help) | Not needed |

---

## Parallel Testing

```go
func TestExample(t *testing.T) {
    skipIfMissingCreds(t)
    t.Parallel()  // Enable parallel execution

    // For API calls - use rate-limited wrapper
    stdout, stderr, err := runCLIWithRateLimit(t, "command", "subcommand")

    // For offline commands - no rate limiting
    stdout, stderr, err := runCLI("version")
}
```

---

## Cleanup Pattern

```go
t.Cleanup(func() {
    acquireRateLimit(t)
    _ = client.Delete(ctx, resourceID)
})
```

**Note:** Integration tests create real resources. Always use:
```bash
make ci-full         # RECOMMENDED: Complete CI with automatic cleanup
make test-cleanup    # Manual cleanup if needed
```

---

## Commands

**See:** `.claude/commands/run-tests.md` for full command details.

```bash
make ci-full              # Complete CI pipeline (RECOMMENDED)
make test-integration     # CLI integration tests
```
