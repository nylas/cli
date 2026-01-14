---
name: add-integration-test
description: Add a new integration test to the CLI test suite
allowed-tools: Read, Write, Edit, Grep, Glob, Bash(go test:*), Bash(go build:*)
---

# Add Integration Test

Add a new integration test to the CLI test suite.

**See also:**
- `.claude/rules/testing.md` - Test patterns and conventions
- `.claude/shared/patterns/integration-test-patterns.md` - Rate limiting (CRITICAL)

## Instructions

1. Ask me for:
   - Which command to test
   - What scenario to test
   - Expected output/behavior

2. Add test to `internal/cli/integration/<feature>_test.go`:
   - For email tests: `internal/cli/integration/email_test.go`
   - For auth tests: `internal/cli/integration/auth_test.go`
   - For calendar tests: `internal/cli/integration/calendar_test.go`
   - etc.

### Test Template

```go
func TestCLI_NewFeature(t *testing.T) {
    skipIfMissingCreds(t)  // Skip if no API credentials

    stdout, stderr, err := runCLI("command", "subcommand", "--flag", "value")
    skipIfProviderNotSupported(t, stderr)  // Skip if provider doesn't support feature

    if err != nil {
        t.Fatalf("command failed: %v\nstderr: %s", err, stderr)
    }

    // Verify expected output
    if !strings.Contains(stdout, "expected text") {
        t.Errorf("Expected 'expected text' in output, got: %s", stdout)
    }

    t.Logf("command output:\n%s", stdout)  // Log for debugging
}
```

### Test with API Client

```go
func TestCLI_FeatureWithData(t *testing.T) {
    skipIfMissingCreds(t)

    // Get data using API client
    client := getTestClient()
    ctx := context.Background()

    items, err := client.GetItems(ctx, testGrantID, 1)
    if err != nil {
        t.Fatalf("Failed to get items: %v", err)
    }
    if len(items) == 0 {
        t.Skip("No items available for test")
    }

    itemID := items[0].ID

    // Test CLI command with the ID
    stdout, stderr, err := runCLI("command", "show", itemID, testGrantID)

    if err != nil {
        t.Fatalf("command show failed: %v\nstderr: %s", err, stderr)
    }

    // Verify output contains expected data
    if !strings.Contains(stdout, "ID:") {
        t.Errorf("Expected 'ID:' in output, got: %s", stdout)
    }

    t.Logf("command show output:\n%s", stdout)
}
```

### Test with Flags

```go
func TestCLI_CommandWithFlags(t *testing.T) {
    skipIfMissingCreds(t)

    // Test with specific flag
    stdout, stderr, err := runCLI("command", "list", "--limit", "5", "--format", "json")

    if err != nil {
        t.Fatalf("command list failed: %v\nstderr: %s", err, stderr)
    }

    // Verify JSON output
    var result []map[string]interface{}
    if err := json.Unmarshal([]byte(stdout), &result); err != nil {
        t.Errorf("Expected valid JSON output, got: %s", stdout)
    }

    if len(result) > 5 {
        t.Errorf("Expected at most 5 items, got %d", len(result))
    }
}
```

### Test Categories (add to appropriate section in file)

- **Email tests**: After `// EMAIL COMMAND TESTS`
- **Webhook tests**: After `// WEBHOOK COMMAND TESTS`
- **Calendar tests**: After `// CALENDAR COMMAND TESTS`
- **Contact tests**: After `// CONTACTS COMMAND TESTS`
- **Auth tests**: After `// AUTH COMMAND TESTS`

3. Run the new test:
```bash
NYLAS_API_KEY="key" NYLAS_GRANT_ID="id" NYLAS_TEST_BINARY="./bin/nylas" \
  go test -tags=integration -v ./internal/cli/integration/... -run "TestCLI_NewFeature"
```
