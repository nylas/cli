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
- **Timezone Support**: Offline utilities + calendar integration ✅
- **Email Signing**: GPG/PGP email signing (RFC 3156 PGP/MIME) ✅
- **AI Chat**: Web-based chat interface using locally installed AI agents ✅
- **Credential Storage**: System keyring (see below)
- **Web UI**: Air - browser-based interface (localhost:7365)

**Details:** See `docs/ARCHITECTURE.md`

---

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `NYLAS_DISABLE_KEYRING` | Disable system keyring, use encrypted file | `false` |
| `NYLAS_API_KEY` | Override API key from keyring (for testing) | - |
| `NYLAS_CLIENT_ID` | Override client ID from keyring (for testing) | - |
| `NYLAS_GRANT_ID` | Override grant ID (for testing) | - |

**Integration test env vars:** See `.claude/commands/run-tests.md` for full list

---

## Credential Storage (Keyring)

Credentials from `nylas auth config` are stored in the system keyring under service name `"nylas"`.

### Keys Stored
| Key | Constant | Description |
|-----|----------|-------------|
| `client_id` | `ports.KeyClientID` | Nylas Application/Client ID |
| `api_key` | `ports.KeyAPIKey` | Nylas API key (Bearer auth) |
| `client_secret` | `ports.KeyClientSecret` | Provider OAuth secret (Google/Microsoft) |
| `org_id` | `ports.KeyOrgID` | Nylas Organization ID |
| `grants` | `grantsKey` | JSON array of grant info |
| `default_grant` | `defaultGrantKey` | Default grant ID |
| `grant_token_<id>` | `ports.GrantTokenKey()` | Per-grant access tokens |

### Key Files
- `internal/ports/secrets.go` - Key constants
- `internal/adapters/keyring/keyring.go` - Keyring implementation (service: `"nylas"`)
- `internal/adapters/keyring/grants.go` - Grant storage
- `internal/app/auth/config.go` - `SetupConfig()` saves to keyring

### Platform Backends
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **macOS**: Keychain
- **Windows**: Windows Credential Manager
- **Fallback**: Encrypted file (`~/.config/nylas/`)

**Disable keyring**: `NYLAS_DISABLE_KEYRING=true`

---

## File Structure

**Hexagonal layers:** CLI (`internal/cli/`) → Port (`internal/ports/`) → Adapter (`internal/adapters/`)

**Core files:** `cmd/nylas/main.go`, `internal/ports/nylas.go`, `internal/adapters/nylas/client.go`

**Quick lookup:** CLI helpers in `internal/cli/common/`, HTTP in `client.go`, Air at `internal/air/`, Chat at `internal/chat/`

**New packages (2024-2026):**
- `internal/ports/output.go` - OutputWriter interface for pluggable formatting
- `internal/adapters/output/` - Table, JSON, YAML, Quiet output adapters
- `internal/httputil/` - HTTP response helpers (WriteJSON, LimitedBody, DecodeJSON)
- `internal/adapters/gpg/` - GPG/PGP email signing service (2026)
- `internal/adapters/mime/` - RFC 3156 PGP/MIME message builder (2026)
- `internal/chat/` - AI chat interface with local agent support (2026)

**Full inventory:** `docs/ARCHITECTURE.md`

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
| `nylas air` | Start Air web UI (localhost:7365) |
| `nylas chat` | Start AI chat interface (localhost:7367) |

**Available targets:** Run `make help` or `make` to see all available commands

**Debugging:** Check `ports/nylas.go` → `adapters/nylas/client.go` → `cli/<feature>/helpers.go`

**API docs:** https://developer.nylas.com/docs/api/v3/

---

## LEARNINGS (Self-Updating)

> **When Claude makes a mistake, use:** "Reflect on this mistake. Abstract and generalize the learning. Write it to CLAUDE.md."

This section captures lessons learned from mistakes. Claude updates this section when errors are caught.

### Project-Specific Gotchas
- Playwright selectors: ALWAYS use semantic selectors (getByRole > getByText > getByLabel > getByTestId), NEVER CSS/XPath
- Go tests: ALWAYS use table-driven tests with t.Run() for multiple scenarios
- Air handlers: ALWAYS return after error responses (prevents writing to closed response)
- Integration tests: ALWAYS use acquireRateLimit(t) before API calls in parallel tests
- Frontend JS: ALWAYS use textContent for user data, NEVER innerHTML (XSS prevention)
- CLI clients: Use `common.GetNylasClient()` directly, NEVER create package-local `getClient()` wrappers
- Grant IDs: Use `common.GetGrantID(args)` directly, NEVER create package-local `getGrantID()` wrappers
- File sizes: Use `common.FormatSize()` for byte formatting, NEVER create duplicate formatBytes/formatFileSize
- Success/Error messages: Use `common.PrintSuccess()`/`common.PrintError()`, delegate from package-local helpers
- HTTP handlers: Use `httputil.WriteJSON()`/`httputil.LimitedBody()` for consistent response handling
- AI clients: Use shared helpers in `adapters/ai/base_client.go` (ConvertMessagesToMaps, ConvertToolsOpenAIFormat, FallbackStreamChat)
- Output formatting: Use `common.GetOutputWriter(cmd)` for JSON/YAML/quiet support, NEVER create custom --format flags
- Client helpers: Use `common.WithClient()` and `WithClientNoGrant()` to reduce boilerplate, NEVER duplicate setup code

### Non-Obvious Workflows
- Progressive disclosure: Keep main skill files under 100 lines, use references/ for details
- Self-learning: Use "Reflect → Abstract → Generalize → Write" when mistakes occur
- Session continuity: Read claude-progress.txt at session start, update at session end
- Hook debugging: Check ~/.claude/logs/ for hook execution errors

### Time-Wasting Bugs Fixed
- Go build cache corruption: Fix with `sudo rm -rf ~/.cache/go-build ~/go/pkg/mod && go clean -cache`
- Playwright MCP not connecting: Run `claude mcp add playwright` to install plugin
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
