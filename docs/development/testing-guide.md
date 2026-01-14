# Testing Guide

Comprehensive testing guidelines for contributors.

---

## Test Types

### Unit Tests
- **Location:** Alongside source (`*_test.go`)
- **Purpose:** Test individual functions/methods
- **No external dependencies:** Use mocks

### Integration Tests
- **Location:** `internal/cli/integration/*_test.go`
- **Purpose:** Test full command execution
- **Requires:** API credentials

---

## Writing Unit Tests

### Table-Driven Pattern

```go
func TestFormatEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid email",
            email: "user@example.com",
            want:  "user@example.com",
            wantErr: false,
        },
        {
            name:  "invalid email",
            email: "not-an-email",
            want:  "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FormatEmail(tt.email)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("wantErr %v, got error %v", tt.wantErr, err)
            }
            
            if got != tt.want {
                t.Errorf("want %q, got %q", tt.want, got)
            }
        })
    }
}
```

### Using Mocks

```go
func TestListEmails(t *testing.T) {
    mock := &nylas.MockClient{
        ListMessagesFunc: func(ctx context.Context, grantID string, params *domain.MessageListParams) ([]*domain.Message, error) {
            return []*domain.Message{
                {ID: "msg_1", Subject: "Test"},
            }, nil
        },
    }
    
    messages, err := mock.ListMessages(context.Background(), "grant_123", nil)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if len(messages) != 1 {
        t.Errorf("expected 1 message, got %d", len(messages))
    }
}
```

---

## Writing Integration Tests

### Template

```go
//go:build integration
// +build integration

package integration

import (
    "testing"
)

func TestFeature(t *testing.T) {
    skipIfMissingCreds(t)
    t.Parallel()
    
    t.Run("SubTest", func(t *testing.T) {
        acquireRateLimit(t)
        
        stdout, stderr, err := runCLIWithRateLimit(t, "command", "subcommand")
        
        if err != nil {
            t.Fatalf("command failed: %v\nStderr: %s", err, stderr)
        }
        
        if !contains(stdout, "expected output") {
            t.Error("expected output not found")
        }
    })
}
```

### With Cleanup

```go
func TestCreateResource(t *testing.T) {
    skip IfMissingCreds(t)
    t.Parallel()
    
    acquireRateLimit(t)
    
    // Create resource
    stdout, _, err := runCLIWithRateLimit(t, "resource", "create", "--name", "test")
    
    if err != nil {
        t.Fatalf("create failed: %v", err)
    }
    
    // Extract ID from output
    resourceID := extractID(stdout)
    
    // Cleanup
    t.Cleanup(func() {
        acquireRateLimit(t)
        _, _, _ = runCLIWithRateLimit(t, "resource", "delete", resourceID)
    })
    
    // Verify resource exists
    acquireRateLimit(t)
    stdout, _, err = runCLIWithRateLimit(t, "resource", "show", resourceID)
    
    if err != nil {
        t.Fatalf("show failed: %v", err)
    }
}
```

---

## Running Tests

```bash
# Unit tests only
go test ./... -short

# All tests
go test ./...

# Integration tests only
go test -tags=integration ./internal/cli/integration/...

# With coverage
go test ./... -short -coverprofile=coverage.out
go tool cover -html=coverage.out

# Specific package
go test ./internal/cli/email/...

# Specific test
go test ./internal/cli/email/... -run TestSendEmail

# Verbose output
go test -v ./...
```

---

## Best Practices

### Test Naming

```go
// Function: TestFunctionName_Scenario
func TestFormatEmail_InvalidInput(t *testing.T) {}

// Integration: TestCLI_CommandName
func TestCLI_EmailSend(t *testing.T) {}
```

### Assertions

```go
// ✅ Good - clear error messages
if got != want {
    t.Errorf("FormatEmail() = %q, want %q", got, want)
}

// ❌ Bad - unclear
if got != want {
    t.Error("wrong result")
}
```

### Test Organization

```go
// ✅ Good - Arrange, Act, Assert
func TestFunction(t *testing.T) {
    // Arrange
    input := "test"
    want := "expected"
    
    // Act
    got := Function(input)
    
    // Assert
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

### Parallel Tests

```go
// Enable parallel execution
func TestParallel(t *testing.T) {
    t.Parallel()  // Runs in parallel with other tests
    
    // Test logic
}
```

### Rate Limiting (Integration)

```go
// Always rate limit API calls in parallel tests
acquireRateLimit(t)
stdout, stderr, err := runCLIWithRateLimit(t, "email", "list")
```

---

## Coverage Goals

**See:** `.claude/rules/testing.md` for authoritative coverage targets.

| Package Type | Minimum | Target |
|--------------|---------|--------|
| Core Adapters | 70% | 85%+ |
| Business Logic | 60% | 80%+ |
| CLI Commands | 50% | 70%+ |
| Utilities | 90% | 100% |

---

## Troubleshooting Tests

### Tests Failing Randomly
- Check for race conditions: `go test -race`
- Ensure tests are independent
- Use `t.Parallel()` correctly

### Integration Tests Timeout
- Increase timeout: `go test -timeout 10m`
- Check rate limiting configuration
- Verify API credentials

### Flaky Tests
- Add retries for external dependencies
- Use deterministic inputs
- Avoid time-dependent logic

---

## More Resources

- **Testing Package:** https://pkg.go.dev/testing
- **Table-Driven Tests:** https://go.dev/wiki/TableDrivenTests
- **Test Rules:** `../../.claude/rules/testing.md`
