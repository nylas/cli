# AI Assistant Guide for Nylas CLI

## CRITICAL RULES

**NEVER:** `git commit`, `git push`, commit secrets, skip tests/security/linting, create files >600 lines

**ALWAYS run:** `make ci-full` (or `make ci` for quick checks without integration tests)

**Test/doc requirements:** New features and commands need unit tests, integration tests, and doc updates. Bug fixes need unit tests. See `.claude/rules/testing.md` for coverage targets.

**Do not touch:** `.env*`, `**/secrets/**`, `*.pem`, `*.key`, `go.sum`, `.git/`, `vendor/`

---

## Project Overview

- **Language**: Go 1.24.2 | **Framework**: Cobra | **Architecture**: Hexagonal (ports/adapters)
- **API**: Nylas v3 ONLY | **Credentials**: System keyring (service `"nylas"`, see `internal/ports/secrets.go`)
- **Web UI**: Air (localhost:7365) | **Chat**: AI chat (localhost:7367)

**Env overrides:** `NYLAS_API_KEY`, `NYLAS_CLIENT_ID`, `NYLAS_GRANT_ID`, `NYLAS_DISABLE_KEYRING`

---

## File Structure

**Layers:** CLI (`internal/cli/`) → Port (`internal/ports/`) → Adapter (`internal/adapters/`)

**Core:** `cmd/nylas/main.go`, `internal/ports/nylas.go`, `internal/adapters/nylas/client.go`

**Helpers:** `internal/cli/common/`, `internal/httputil/`, `internal/adapters/output/`

**Full inventory:** `docs/ARCHITECTURE.md`

---

## Adding a New Feature

1. Domain types → 2. Port interface → 3. Adapter impl → 4. Mock → 5. CLI commands → 6. Register in main.go → 7. Tests → 8. Docs

**Detailed guide:** `/add-command` skill

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make ci-full` | Full CI (quality + tests) |
| `make ci` | Quick checks (no integration) |
| `make build` | Build binary |
| `make help` | All targets |

**Debugging:** `ports/nylas.go` → `adapters/nylas/client.go` → `cli/<feature>/helpers.go`

**API docs:** https://developer.nylas.com/docs/api/v3/

---

## LEARNINGS

### Non-Obvious Gotchas
- Air handlers: ALWAYS return after error responses
- Use shared helpers: `common.GetNylasClient()`, `common.GetGrantID(args)`, `common.WithClient()`, `common.FormatSize()`, `common.PrintSuccess()`/`PrintError()`, `common.GetOutputWriter(cmd)`, `httputil.WriteJSON()`/`LimitedBody()`
- AI clients: Use shared helpers in `adapters/ai/base_client.go`

### Time-Wasting Bugs
- Go build cache corruption: `sudo rm -rf ~/.cache/go-build ~/go/pkg/mod && go clean -cache`
- Quality gate timeout: Add `timeout 120` before golangci-lint in hooks
