# AI Assistant Guide for Nylas CLI

Quick reference for AI assistants working on this codebase.

---

## ⛔ CRITICAL RULES - YOU MUST FOLLOW THESE

### NEVER DO (IMPORTANT - violations will break the workflow):
- **NEVER run `git commit`** - User commits manually
- **NEVER run `git push`** - User pushes manually
- **NEVER commit secrets** - No API keys, tokens, passwords, .env files
- **NEVER skip tests** - All changes require passing tests
- **NEVER skip security scans** - Run `make security` before commits
- **NEVER create files >600 lines** - Split by responsibility (see `.claude/rules/file-size-limits.md`)

### ALWAYS DO (every code change):

```bash
make ci-full   # Complete CI: quality checks → tests → cleanup
# OR for quick checks without integration tests:
make ci        # Runs: fmt → vet → lint → test-unit → test-race → security → vuln → build
```

**⚠️ CRITICAL: Never skip linting. Fix ALL linting errors in code you wrote.**

**⚠️ CRITICAL: Enforce file size limits. Files must be ≤500 lines (ideal) or ≤600 lines (max).**

**Details:** See `.claude/rules/go-quality.md`, `.claude/rules/file-size-limits.md`

### Test & Doc Requirements:
| Change | Unit Test | Integration Test | Update Docs |
|--------|-----------|------------------|-------------|
| New feature | ✅ REQUIRED | ✅ REQUIRED | ✅ REQUIRED |
| Bug fix | ✅ REQUIRED | ⚠️ If API-related | ⚠️ If behavior changes |
| New command | ✅ REQUIRED | ✅ REQUIRED | ✅ REQUIRED |
| Flag change | ✅ REQUIRED | ❌ Not needed | ✅ REQUIRED |

### Test Coverage Goals:

**See:** `.claude/rules/testing.md` for coverage targets by package type.

**Check coverage:** `make test-coverage`

### Documentation:
**📚 See `docs/INDEX.md` for all documentation.**

**Rules:** `.claude/rules/documentation-maintenance.md`

### Do Not Touch:
| Path | Reason |
|------|--------|
| `.env*`, `**/secrets/**` | Contains secrets |
| `*.pem`, `*.key` | Certificates/keys |
| `go.sum` | Auto-generated |
| `.git/`, `vendor/` | Managed externally |

---

## Project Overview

- **Language**: Go 1.24.2 (use latest features!)
- **Architecture**: Hexagonal (ports and adapters)
- **CLI Framework**: Cobra
- **API**: Nylas v3 ONLY (never use v1/v2)

**Details:** See `docs/ARCHITECTURE.md`

---

## File Structure

**Hexagonal layers:** CLI (`internal/cli/`) → Port (`internal/ports/`) → Adapter (`internal/adapters/`)

**Core files:** `cmd/nylas/main.go`, `internal/ports/nylas.go`, `internal/ports/client_factory.go`, `internal/adapters/nylas/client.go`

**Quick lookup:** CLI helpers in `internal/cli/common/`, HTTP helpers in `internal/adapters/nylas/client_helpers.go`

**Full inventory:** `docs/ARCHITECTURE.md`

### ⚠️ Use Existing Helpers - Don't Duplicate

**Before writing utility code, check for existing helpers:**

| Need | Check First |
|------|-------------|
| Client creation | `internal/adapters/client/factory.go` (ClientFactory) |
| Error wrapping | `internal/cli/common/errors.go` |
| Output formatting, tables, colors | `internal/cli/common/format.go`, `colors.go` |
| Time parsing/formatting | `internal/cli/common/time.go`, `timeutil.go` |
| Progress indicators | `internal/cli/common/progress.go` |
| Pagination | `internal/cli/common/pagination.go` |
| HTTP requests | `internal/adapters/nylas/client_helpers.go` |
| Query building | `internal/adapters/nylas/client_helpers.go` (`QueryBuilder`) |
| Tunnel creation | `internal/adapters/tunnel/provider.go` (TunnelProvider) |

**Rule:** Search existing helpers before writing new utility functions. Duplicate code = rejected PR.

---

## Adding a New Feature

**Quick pattern:**
1. Domain: `internal/domain/<feature>.go` - Define types
2. Port: `internal/ports/nylas.go` - Add interface methods
3. Adapter: `internal/adapters/nylas/<feature>.go` - Implement methods
4. Mock: `internal/adapters/nylas/mock.go` - Add mock methods
5. CLI: `internal/cli/<feature>/` - Add commands
6. Register: `cmd/nylas/main.go` - Add command
7. Tests: `internal/cli/integration/<feature>_test.go`
8. Docs: `docs/COMMANDS.md` - Add examples

**Detailed guide:** Use `/add-command` skill

---

## Go Modernization

**See:** `.claude/rules/go-quality.md` for modern Go patterns (1.21+), error handling, and linting fixes.

---

## Testing

**Command:** `make ci-full` (complete CI: quality + tests + cleanup)

**Quick checks:** `make ci` (no integration tests)

**Details:** `.claude/rules/testing.md`

---

## Hooks & Commands

**Hooks:** Auto-enforce quality (blocks bad code, auto-formats). See `.claude/HOOKS-CONFIG.md`

**Skills:** `/session-start`, `/run-tests`, `/add-command`, `/generate-tests`, `/security-scan`

**Agents:** See `.claude/agents/README.md` for parallelization guide

---

## Context & Session

**Token tips:** Use `/compact` mid-session, `/clear` for new tasks, `/mcp` to disable unused servers

**On-demand docs:** `docs/COMMANDS.md`, `docs/ARCHITECTURE.md`, `.claude/shared/patterns/*.md`

**Session handoff:** Update `claude-progress.txt` after major tasks (Branch → Summary → Next Steps)

---

## Quick Reference

| Command | Purpose |
|---------|---------|
| `make ci-full` | Complete CI (quality + tests) - **run before commits** |
| `make ci` | Quick quality checks (no integration) |
| `make build` | Build binary |

**Debugging:** Check `ports/nylas.go` → `adapters/nylas/client.go` → `cli/<feature>/helpers.go`

**API docs:** https://developer.nylas.com/docs/api/v3/

---

## LEARNINGS (Self-Updating)

> **When Claude makes a mistake, use:** "Reflect on this mistake. Abstract and generalize the learning. Write it to CLAUDE.md."

This section captures lessons learned from mistakes. Claude updates this section when errors are caught.

### Project-Specific Gotchas
- Go tests: ALWAYS use table-driven tests with t.Run() for multiple scenarios
- Integration tests: ALWAYS use acquireRateLimit(t) before API calls in parallel tests

### Non-Obvious Workflows
- Progressive disclosure: Keep main skill files under 100 lines, use references/ for details
- Self-learning: Use "Reflect → Abstract → Generalize → Write" when mistakes occur
- Session continuity: Read claude-progress.txt at session start, update at session end
- Hook debugging: Check ~/.claude/logs/ for hook execution errors

### Time-Wasting Bugs Fixed
- Go build cache corruption: Fix with `sudo rm -rf ~/.cache/go-build ~/go/pkg/mod && go clean -cache`
- Quality gate timeout: Add `timeout 120` before golangci-lint in hooks

### Curation Rules
- Maximum 30 items per category
- Remove obsolete entries when adding new
- One imperative line per item
- Monthly review to prune stale advice

---

## META

**Quick update:** Press `#` key to add instructions during sessions

**Maintain:** ALWAYS/NEVER for critical rules, max 30 LEARNINGS items, prune monthly
