# Go Test Patterns

Shared patterns for Go unit tests in the Nylas CLI project.

---

## Table-Driven Tests (REQUIRED)

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {
            name:  "valid input returns expected output",
            input: InputType{Field: "value"},
            want:  OutputType{Result: "expected"},
        },
        {
            name:    "empty input returns error",
            input:   InputType{},
            wantErr: true,
        },
        {
            name:  "handles special characters",
            input: InputType{Field: "日本語 & émojis 🎉"},
            want:  OutputType{Result: "processed"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("got = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Mock Pattern

Use mocks from `internal/adapters/nylas/mock.go`:

```go
type MockClient struct {
    ListFunc   func(ctx context.Context) ([]domain.Item, error)
    CreateFunc func(ctx context.Context, item *domain.Item) error
}

func (m *MockClient) List(ctx context.Context) ([]domain.Item, error) {
    if m.ListFunc != nil {
        return m.ListFunc(ctx)
    }
    return nil, nil
}

// Usage in tests:
mock := &MockClient{
    ListFunc: func(ctx context.Context) ([]domain.Item, error) {
        return []domain.Item{{ID: "123"}}, nil
    },
}
```

---

## Testify Assertions

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Use require for fatal checks (test cannot continue)
require.NoError(t, err)
require.NotNil(t, result)

// Use assert for non-fatal checks
assert.Equal(t, expected, actual)
assert.Contains(t, output, "expected substring")
assert.Len(t, items, 3)
```

---

## Parallel Tests

```go
func TestFeature(t *testing.T) {
    t.Parallel()  // Enable for independent tests

    tests := []struct{ /* ... */ }{}
    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Run subtests in parallel too
            // Test logic
        })
    }
}
```

---

## Cleanup Pattern

```go
func TestWithResources(t *testing.T) {
    tmpDir := t.TempDir()  // Auto-cleaned

    t.Cleanup(func() {
        // Custom cleanup if needed
    })
}
```

---

## Test Categories

1. **Happy Path** - Normal inputs, expected outputs
2. **Error Cases** - Invalid inputs, missing fields, API errors
3. **Edge Cases** - Empty, nil, max/min values, unicode
4. **Boundary Conditions** - First/last item, pagination, limits

---

## Test Naming

| Type | Pattern | Example |
|------|---------|---------|
| Unit test | `TestFunctionName_Scenario` | `TestParseEmail_ValidInput` |
| HTTP handler | `TestHandleFeature_Scenario` | `TestHandleAISummarize_EmptyBody` |

---

## Commands

**See:** `.claude/commands/run-tests.md` for full command details.

```bash
go test -v ./path/to/...           # Run specific tests
go test -run TestName ./...        # Run by name
make test-unit                     # All unit tests
```
