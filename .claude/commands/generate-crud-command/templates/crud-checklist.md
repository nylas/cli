# CRUD Command Generation Checklist

Complete checklist for generating a full CRUD command package.

---

## Pre-Generation

- [ ] Resource name (singular, e.g., "widget")
- [ ] Parent command if nested (e.g., "email" for "email drafts")
- [ ] Operations needed: list, show, create, update, delete
- [ ] API endpoint path (e.g., "/grants/{grantID}/widgets")
- [ ] Key fields from API spec

---

## Files to Generate

### Domain Layer
- [ ] `internal/domain/{resource}.go`
  - Main type with JSON tags
  - CreateRequest type
  - UpdateRequest type
  - QueryParams type
  - Helper methods (DisplayName, etc.)

### Port Layer
- [ ] `internal/ports/nylas.go` - Add interface methods:
  - List{Resource}s
  - Get{Resource}
  - Create{Resource}
  - Update{Resource}
  - Delete{Resource}

### Adapter Layer
- [ ] `internal/adapters/nylas/{resource}s.go` - Implementation
- [ ] `internal/adapters/nylas/mock_{resource}.go` - Mock functions
- [ ] `internal/adapters/nylas/demo_{resource}.go` - Demo data

### CLI Layer
- [ ] `internal/cli/{resource}/{resource}.go` - Root command
- [ ] `internal/cli/{resource}/list.go` - List subcommand
- [ ] `internal/cli/{resource}/show.go` - Show subcommand
- [ ] `internal/cli/{resource}/create.go` - Create subcommand
- [ ] `internal/cli/{resource}/update.go` - Update subcommand
- [ ] `internal/cli/{resource}/delete.go` - Delete subcommand
- [ ] `internal/cli/{resource}/helpers.go` - Standard helpers
- [ ] `internal/cli/{resource}/{resource}_test.go` - Unit tests

### Registration
- [ ] `cmd/nylas/main.go` - Add command import and registration

---

## Verification

```bash
# Build
go build ./...

# Test
go test ./internal/cli/{resource}/... -v
go test ./... -short

# Verify command
./bin/nylas {resource} --help
./bin/nylas {resource} list --help

# Full CI
make ci-full
```

---

## Common Flags per Subcommand

| Subcommand | Required Flags | Optional Flags |
|------------|----------------|----------------|
| list | - | --limit, --format |
| show | <id> | --format |
| create | resource-specific | --format |
| update | <id> | resource-specific |
| delete | <id> | --force |

---

## Standard Features

- [ ] --format flag (table, json, yaml)
- [ ] Spinner for long operations
- [ ] Context with 30s timeout
- [ ] Helpful error messages
- [ ] Pagination support for list
