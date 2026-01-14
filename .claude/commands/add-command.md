# Add New CLI Command

Create a new CLI command following the nylas CLI patterns and hexagonal architecture.

**API Reference:** https://developer.nylas.com/docs/api/v3/

---

## Quick Start

1. Ask for: command name, parent command, operations needed (list, show, create, update, delete)
2. Follow patterns in `references/` directory
3. Run `make ci` to verify

---

## Reference Files

| File | When to Read |
|------|--------------|
| `references/domain-patterns.md` | Creating new domain types |
| `references/adapter-patterns.md` | Implementing API methods |
| `references/cli-patterns.md` | Building CLI commands |

**Read reference files ONLY when working on that specific layer.**

---

## Steps

### 1. Domain Layer (if new types needed)
- Create `internal/domain/{resource}.go`
- Add tests to `internal/domain/domain_test_basic.go`

### 2. Adapter Layer (if new API methods needed)
- Update `internal/ports/nylas.go` with interface methods
- Create `internal/adapters/nylas/{resource}.go`
- Update `internal/adapters/nylas/mock_{resource}.go`

### 3. CLI Layer
- Create `internal/cli/{resource}/` directory
- Add: `{resource}.go`, `list.go`, `show.go`, `create.go`, `helpers.go`
- Add tests: `{resource}_test.go`

### 4. Registration
- Update `cmd/nylas/main.go` to add the command

### 5. Verify
```bash
make ci
```

---

## Common Patterns

- **Context:** 30s timeout for API calls
- **Format flag:** `--format` with table, json, yaml
- **Spinner:** Use pterm spinner for long operations
- **Errors:** Wrap with `fmt.Errorf("context: %w", err)`

---

## For Full CRUD Commands

When creating a complete resource with list, show, create, update, delete:

### Additional Files Needed
- `internal/adapters/nylas/demo_{resource}.go` - Demo data
- `internal/adapters/nylas/mock_{resource}.go` - Mock functions

### Domain Template
```go
type {Resource} struct {
    ID string `json:"id"`
    // Fields from API spec
}
type Create{Resource}Request struct { /* required fields */ }
type Update{Resource}Request struct { /* optional fields */ }
type {Resource}QueryParams struct { Limit int; PageToken string }
```

### Verification
```bash
make ci-full
./bin/nylas {resource} --help
```

---

## Checklist

- [ ] Domain types with JSON tags
- [ ] Port interface updated
- [ ] Adapter with all CRUD methods
- [ ] Mock implementation
- [ ] Demo data (for CRUD commands)
- [ ] CLI with all subcommands
- [ ] Tests passing
- [ ] Registered in main.go
- [ ] `make ci` passes
