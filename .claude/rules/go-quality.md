# Go Quality Rules

Auto-applied to all Go code changes. Combines Go best practices and linting requirements.

---

## Mandatory Workflow

### Before Writing Code:
1. **Check Go version:** Currently **Go 1.24.2**
2. **Research official docs:** `go.dev/ref/spec`, `pkg.go.dev`
3. **Find existing patterns:** Use Grep/Glob to match project style

### After Writing Code:
```bash
go fmt ./...                    # Format
go vet ./...                    # Static analysis
golangci-lint run --timeout=5m  # Lint
make ci                         # Full quality pipeline
```

---

## Modern Go Patterns (Go 1.24+)

| Instead of | Use | Since |
|------------|-----|-------|
| `io/ioutil.ReadFile` | `os.ReadFile` | Go 1.16+ |
| `interface{}` | `any` | Go 1.18+ |
| Manual slice helpers | `slices` package | Go 1.21+ |
| Manual map helpers | `maps` package | Go 1.21+ |
| Custom min/max | `min()`, `max()` | Go 1.21+ |
| `sort.Slice` | `slices.SortFunc` | Go 1.21+ |

---

## Error Handling

```go
// ✅ Wrap with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// ✅ Explicit ignore (with comment)
_ = json.Encode(data)  // Test helper

// ✅ Nil check before dereference
if obj == nil {
    return errors.New("object is nil")
}
```

---

## Common Linting Fixes

| Error | Fix |
|-------|-----|
| **errcheck** | Add `_ =` or handle error |
| **unused** | Delete unused code |
| **SA5011** | Add nil check before deref |
| **SA9003** | Implement or remove empty branch |
| **SA1019** | Use non-deprecated function |
| **ineffassign** | Remove/use variable |
| **G301** | Use 0750 for directories, not 0755 |

---

## Forbidden Patterns

- ❌ `io/ioutil` - Deprecated
- ❌ `interface{}` - Use `any`
- ❌ Custom slice/map helpers - Use stdlib
- ❌ `math/rand` - Use `math/rand/v2`

---

## Quality Gate

**Zero linting errors in new code.**

```
Write Code → Format → Lint → Fix → Test → Complete
     ↑                         |
     └──── Back if errors ─────┘
```

**What to fix:**
- ✅ All errors in files you created/modified
- ⚠️ Can ignore pre-existing errors in untouched files

---

## Commands

```bash
go fmt ./...                              # Format
golangci-lint run --timeout=5m            # Lint all
golangci-lint run --timeout=5m --fix      # Auto-fix
golangci-lint run --new-from-rev=HEAD~1   # Lint changed only
make ci                                   # Full pipeline
make ci-full                              # With integration tests
```

---

## Resources

- **Go Spec:** https://go.dev/ref/spec
- **Effective Go:** https://go.dev/doc/effective_go
- **Uber Guide:** https://github.com/uber-go/guide/blob/master/style.md
