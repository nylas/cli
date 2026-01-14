---
name: fix-build
description: Diagnose and fix Go build errors
allowed-tools: Read, Edit, Write, Grep, Glob, Bash(go build:*), Bash(go vet:*)
---

# Fix Build Errors

Diagnose and fix build errors in the nylas CLI.

## Instructions

1. First, run the build to see errors:
```bash
go build ./...
```

2. Common error types and fixes:

### Interface Not Implemented
Error: `*HTTPClient does not implement NylasClient (missing method X)`

Fix: Add the method to `internal/adapters/nylas/client.go` or appropriate file:
```go
func (c *HTTPClient) NewMethod(ctx context.Context, ...) error {
    // Implementation
}
```

Also add to mock in `internal/adapters/nylas/mock.go`:
```go
func (m *MockClient) NewMethod(ctx context.Context, ...) error {
    // Mock implementation
}
```

### Undefined Type
Error: `undefined: domain.NewType`

Fix: Add type definition to `internal/domain/` in appropriate file.

### Import Cycle
Error: `import cycle not allowed`

Fix:
- Move shared types to `internal/domain/`
- Use interfaces in `internal/ports/` instead of concrete types
- Extract common code to `internal/cli/common/`

### Unused Import/Variable
Error: `imported and not used` or `declared and not used`

Fix: Remove the unused import/variable, or use `_` for intentionally unused:
```go
_ = unusedVar
```

### Type Mismatch
Error: `cannot use X (type A) as type B`

Fix: Check if you need:
- Type conversion: `B(x)`
- Pointer: `&x` or `*x`
- Type assertion: `x.(B)`

3. After fixing, verify:
```bash
go build ./... && go test ./...
```

4. If tests fail after fixing build:
- Check if mock needs updating
- Check if tests need updating for new signatures
- Run specific failing test with `-v` for details
