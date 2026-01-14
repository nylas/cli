---
name: debug-test-failure
description: Analyze test failures and suggest fixes
allowed-tools: Read, Edit, Grep, Glob, Bash(go test:*), Bash(go build:*)
---

# Debug Test Failure

Analyze test failures and suggest fixes.

Test/Error: $ARGUMENTS

## Instructions

1. **Reproduce the failure**

   ```bash
   # Run the specific failing test
   go test ./path/to/package -run TestName -v

   # Run with race detection
   go test ./path/to/package -run TestName -race -v

   # Run all tests in package
   go test ./path/to/package/... -v
   ```

2. **Capture detailed output**

   ```bash
   # Verbose output with all test logs
   go test ./path/to/package -run TestName -v 2>&1 | tee test_output.txt

   # With timeout for hanging tests
   go test ./path/to/package -run TestName -v -timeout 30s
   ```

3. **Analyze the failure type**

   | Failure Type | Indicators | Common Causes |
   |--------------|------------|---------------|
   | Assertion failure | "got X, want Y" | Logic error, wrong expected value |
   | Nil pointer | "nil pointer dereference" | Missing initialization, nil return |
   | Timeout | "test timed out" | Infinite loop, deadlock, slow operation |
   | Race condition | "DATA RACE" | Concurrent access without sync |
   | Interface error | "does not implement" | Missing method, signature change |
   | Import cycle | "import cycle" | Circular dependency |

4. **Examine the test code**

   ```bash
   # Read the test file
   cat path/to/package/*_test.go

   # Find the specific test function
   grep -A 50 "func TestName" path/to/package/*_test.go
   ```

5. **Examine the implementation**

   Based on the failure, read relevant source files:
   - For assertion failures: Check the function being tested
   - For nil pointers: Check initialization and return values
   - For interface errors: Check interface definition and implementation

6. **Common fixes by failure type**

   **Assertion Failure:**
   ```go
   // Check if expected value is correct
   // Check if test setup is correct
   // Verify mock returns expected values
   ```

   **Nil Pointer:**
   ```go
   // Add nil check before use
   if obj == nil {
       return nil, errors.New("object is nil")
   }

   // Initialize in test setup
   func TestX(t *testing.T) {
       obj := &MyType{} // Don't forget initialization
   }
   ```

   **Timeout:**
   ```go
   // Add context with timeout
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()

   // Check for infinite loops
   // Add select with done channel
   ```

   **Race Condition:**
   ```go
   // Use mutex for shared state
   var mu sync.Mutex
   mu.Lock()
   defer mu.Unlock()

   // Or use channels for communication
   ```

   **Mock Not Returning Expected Value:**
   ```go
   // Configure mock before test
   mock := &MockClient{
       ListError: nil,
       Items: []domain.Item{{ID: "test-1"}},
   }
   ```

7. **Verify the fix**

   ```bash
   # Run the specific test
   go test ./path/to/package -run TestName -v

   # Run with race detection
   go test ./path/to/package -run TestName -race

   # Run full test suite to check for regressions
   go test ./... -short

   # Run integration tests if applicable
   go test ./... -tags=integration
   ```

## Debugging Techniques

### Add Debug Output

```go
func TestSomething(t *testing.T) {
    t.Logf("Debug: value = %+v", value)
    // t.Log output only shows on failure or with -v flag
}
```

### Use Delve Debugger

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a specific test
dlv test ./path/to/package -- -test.run TestName

# Set breakpoint and continue
(dlv) break path/to/file.go:123
(dlv) continue
(dlv) print variableName
```

### Check Test Isolation

```bash
# Run tests in random order to find order dependencies
go test ./... -shuffle=on

# Run single test multiple times
go test ./path/to/package -run TestName -count=10
```

## Common Test Patterns

**See:** `.claude/shared/patterns/go-test-patterns.md` for table-driven tests, mocks, and assertions.

## Checklist

- [ ] Reproduced the failure locally
- [ ] Identified the failure type
- [ ] Read the test code
- [ ] Read the implementation code
- [ ] Identified root cause
- [ ] Implemented fix
- [ ] Verified fix with `-v` flag
- [ ] Ran with race detection: `go test -race`
- [ ] Ran full test suite: `go test ./... -short`
- [ ] No regressions introduced

## Cleanup

```bash
# Remove debug output files
rm -f test_output.txt
```
