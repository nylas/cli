# Go Quality Rules

Go 1.24.2 — auto-applied to all Go code changes.

## Workflow

```bash
# After writing code:
go fmt ./...                              # Format
go vet ./...                              # Static analysis
golangci-lint run --timeout=5m            # Lint all
golangci-lint run --timeout=5m --fix      # Auto-fix
golangci-lint run --new-from-rev=HEAD~1   # Lint changed only
make ci                                   # Full pipeline
```

## Error Handling

```go
// Wrap with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Explicit ignore (with comment)
_ = json.Encode(data)  // Test helper

// Nil check before dereference
if obj == nil {
    return errors.New("object is nil")
}
```

## Quality Gate

Zero linting errors in new code. Fix all errors in files you created/modified. Can ignore pre-existing errors in untouched files.
