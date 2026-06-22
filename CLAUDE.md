# Nylas CLI — AI Assistant Rules

Every line traces to a real mistake. If Claude does it correctly without the instruction, delete it.

---

## Working Principles

1. **Think Before Coding** — State assumptions explicitly. If uncertain, ask. Push back when a simpler approach exists. Stop when confused — name what's unclear.
2. **Goal-Driven Execution** — Define success criteria before starting. Loop until verified. Don't follow steps blindly — iterate toward success.
3. **Simplicity First** — Minimum code that solves the problem. No speculative features. No abstractions for single-use code.
4. **Surgical Changes** — Touch only what you must. Clean up only your own mess. Don't "improve" adjacent code or refactor what isn't broken.
5. **Read Before You Write** — Before adding code, read exports, callers, shared utilities. "Looks orthogonal" is dangerous — this is the root cause of most LEARNINGS below. If unsure why code is structured that way, ask.
6. **Match Conventions** — Conformance > taste. If you disagree with a convention, surface it — don't fork silently.
7. **Surface Conflicts, Don't Blend** — If two patterns contradict, pick one (more recent / more tested). Explain why. Flag the other for cleanup.
8. **Tests Verify Intent** — Tests encode WHY behavior matters, not just WHAT it does. A test that can't fail when business logic changes is wrong. Never modify existing tests to make implementation pass — fix the implementation instead.
9. **Checkpoint After Significant Steps** — Summarize what was done, what's verified, what's left. If you lose track, stop and restate.
10. **Fail Loud** — "Completed" is wrong if anything was skipped. "Tests pass" is wrong if any were skipped. Surface uncertainty, don't hide it.

---

## Critical Rules

**NEVER:** commit secrets (.env, keys, tokens) · skip tests · skip security scans · skip linting · create files >600 lines

**ALWAYS run before commits:**
```bash
make ci-full   # Complete CI: quality + tests + cleanup
make ci        # Quick: fmt → vet → lint → test-unit → test-race → security → vuln → build
```

**Test & doc requirements:**
| Change | Unit Test | Integration Test | Update Docs |
|--------|-----------|------------------|-------------|
| New feature/command | Required | Required | Required |
| Bug fix | Required | If API-related | If behavior changes |
| Flag change | Required | — | Required |

**Do not touch:** `.env*`, `**/secrets/**`, `*.pem`, `*.key`, `go.sum`, `.git/`, `vendor/`

**On compaction:** preserve the list of modified files, established test commands, and any architectural decisions made this session.

---

## Project

Go 1.24+ · Hexagonal architecture (CLI → Port → Adapter) · Cobra CLI · **Nylas v3 API ONLY**

| Resource | Location |
|----------|----------|
| Architecture & file inventory | `docs/ARCHITECTURE.md` |
| Command reference | `docs/COMMANDS.md` |
| All docs index | `docs/INDEX.md` |
| Go quality rules | `.claude/rules/go-quality.md` |
| Testing rules & coverage targets | `.claude/rules/testing.md` |
| Doc maintenance rules | `.claude/rules/documentation-maintenance.md` |
| Hook enforcement | `.claude/HOOKS-CONFIG.md` |
| Agent definitions | `.claude/agents/README.md` |

**Env vars:** `NYLAS_DISABLE_KEYRING`, `NYLAS_API_KEY`, `NYLAS_CLIENT_ID`, `NYLAS_GRANT_ID`, `NYLAS_API_BASE_URL`, `NYLAS_API_TIMEOUT` (e.g. `120s`; or `nylas config set api.timeout`) — see `docs/DEVELOPMENT.md`

**Credentials:** System keyring (service: `"nylas"`, keys: `client_id`, `api_key`, `client_secret`, `org_id`). Grant cache: `os.UserCacheDir()/nylas/grants.json`. Fallback: `~/.config/nylas/` with `NYLAS_DISABLE_KEYRING=true`.

**New feature?** Run `/add-command` for the step-by-step guide.

**Skills:** `/add-command` (commands, flags, methods, types), `/run-tests`, `/generate-tests`, `/security-scan`, `/review-pr`

**Quick commands:**
| Command | Purpose |
|---------|---------|
| `make ci-full` | Complete CI — **run before commits** |
| `make ci` | Quick quality checks (no integration) |
| `make build` | Build binary |
| `make test-coverage` | Coverage report |
| `make help` | List all available targets |
| `nylas init` | First-time setup wizard |
| `nylas air` | Start Air web UI (localhost:7365) |
| `nylas chat` | Start AI chat interface (localhost:7367) |

**Debugging path:** `ports/nylas.go` → `adapters/nylas/client.go` → `cli/<feature>/helpers.go`

---

## LEARNINGS

> On mistakes: "Reflect → Abstract → Generalize → Write to CLAUDE.md"

### Reuse Existing Helpers (Root Cause: Not Reading First)
- Clients: `common.GetNylasClient()`, `common.WithClient()`, `WithClientNoGrant()` — NEVER create package-local wrappers
- Grant IDs: `common.GetGrantID(args)` — NEVER create package-local `getGrantID()`
- Output: `common.GetOutputWriter(cmd)` for JSON/YAML/quiet — NEVER create custom --format flags
- Formatting: `common.FormatSize()`, `common.StatusColor()`/`StatusIcon()`/`ColorSprint()`
- Messages: `common.PrintSuccess()`/`PrintError()` — delegate from package-local helpers
- Pagination: `common.SetupPagination()`, `NormalizePageSize()`, `FetchCursorPages()`
- HTTP: `httputil.WriteJSON()`/`LimitedBody()`, `testutil.WriteJSONResponse()`
- AI: shared helpers in `adapters/ai/base_client.go`

### Framework Gotchas
- Playwright: semantic selectors ONLY (getByRole > getByText > getByLabel > getByTestId)
- Go integration tests: `acquireRateLimit(t)` before API calls in parallel tests — omitting causes flaky 429s
- Air handlers: ALWAYS return after error responses
- Frontend JS: textContent for user data, NEVER innerHTML (XSS)

### Environment Fixes
- Go build cache corruption: `sudo rm -rf ~/.cache/go-build ~/go/pkg/mod && go clean -cache`
- Quality gate timeout: Add `timeout 120` before golangci-lint in hooks
- Hook debugging: Check `~/.claude/logs/` for hook execution errors

### Curation: max 30 items, one line each, prune monthly, remove obsolete when adding new

---

## META

~100 lines of universal rules. Domain knowledge lives in skills and `.claude/rules/`.
Critical rules also enforced by hooks — hooks are deterministic, prose is probabilistic.
