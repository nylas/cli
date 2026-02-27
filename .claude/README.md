# Claude Code Configuration

This directory contains skills, workflows, rules, agents, and shared patterns for AI-assisted development with Claude Code.

---

## Directory Structure

```
.claude/
├── commands/              # 18 actionable skills (invokable workflows)
├── rules/                 # 2 development rules (auto-applied)
├── agents/                # 6 specialized agents
├── hooks/                 # 3 quality gate hooks
├── shared/patterns/       # 3 reusable pattern files
├── settings.json          # Security hooks & permissions
├── HOOKS-CONFIG.md        # Hook configuration guide
└── README.md              # This file
```

---

## Skills (18 Total)

### Feature Development (5 skills)

| Skill | Purpose |
|-------|---------|
| `add-command` | New CLI command |
| `add-api-method` | Extend API client |
| `add-domain-type` | New domain models |
| `add-flag` | Add command flags |
| `generate-crud-command` | Auto-generate CRUD operations |

### Testing (5 skills)

| Skill | Purpose |
|-------|---------|
| `run-tests` | Execute test suite |
| `generate-tests` | Generate tests for code |
| `add-integration-test` | Create integration tests |
| `debug-test-failure` | Debug failing tests |
| `analyze-coverage` | Coverage analysis |

### Quality Assurance (3 skills)

| Skill | Purpose |
|-------|---------|
| `fix-build` | Resolve build errors |
| `security-scan` | Security audit |
| `review-pr` | Code review checklist |

### Self-Learning (4 skills)

| Skill | Purpose |
|-------|---------|
| `session-start` | Load context from previous sessions |
| `diary` | Save session learnings to memory |
| `reflect` | Review diary, propose CLAUDE.md updates |
| `correct` | Capture mistake for learning |

### Maintenance (1 skill)

| Skill | Purpose |
|-------|---------|
| `update-docs` | Documentation updates |

---

## Rules (2 Files)

| Rule | Purpose | Applies To |
|------|---------|-----------|
| `testing.md` | Test coverage targets & commands | All new code |
| `go-quality.md` | Go workflow & error handling | All Go code |

---

## Agents (6 Specialized)

| Agent | Model | Purpose |
|-------|-------|---------|
| `code-writer` | Opus | Write Go/JS/CSS code |
| `test-writer` | Opus | Generate comprehensive tests |
| `code-reviewer` | Opus | Independent code review |
| `codebase-explorer` | Sonnet | Fast codebase exploration |
| `frontend-agent` | Sonnet | JS/CSS/Go templates |
| `mistake-learner` | Sonnet | Abstract mistakes to learnings |

---

## Hooks (3 Quality Gates)

| Hook | Trigger | Purpose |
|------|---------|---------|
| `quality-gate.sh` | Stop | Block on quality failures |
| `file-size-check.sh` | PreToolUse (Write) | Block Go files >600 lines |
| `auto-format.sh` | PostToolUse (Edit) | Auto-format Go files |

---

## Shared Patterns (3 Files)

| Pattern | Purpose |
|---------|---------|
| `go-test-patterns.md` | Table-driven tests, mocks, testify |
| `integration-test-patterns.md` | CLI + Air integration tests |
| `playwright-patterns.md` | Selectors, templates, commands |

---

## Security (settings.json)

**Pre-commit Hooks:**
- Check for sensitive files (.env, .pem, .key)
- Scan for secrets (api_key, password, token)

**Permissions:**
- ✅ Allowed: go, golangci-lint, make, git (except push), gh CLI
- ❌ Denied: git push, destructive operations
- 🔐 Protected: .env, .pem/.key, secrets/, credentials

---

## Credential Storage (Keyring)

Credentials from `nylas auth config` are stored in system keyring under service `"nylas"`.

| Key | Description |
|-----|-------------|
| `client_id` | Nylas Application/Client ID |
| `api_key` | Nylas API key (Bearer auth) |
| `client_secret` | Provider OAuth secret (Google/Microsoft) |
| `org_id` | Nylas Organization ID |
| `grants` | JSON array of grant info |
| `default_grant` | Default grant ID |
| `grant_token_<id>` | Per-grant tokens |

**Key Files:**
- `internal/ports/secrets.go` - Key constants
- `internal/adapters/keyring/keyring.go` - Keyring implementation
- `internal/adapters/keyring/grants.go` - Grant storage
- `internal/app/auth/config.go` - `SetupConfig()` saves credentials

**Platforms:** Linux (Secret Service), macOS (Keychain), Windows (Credential Manager)

**Fallback:** Encrypted file in `~/.config/nylas/`

**Testing:** `NYLAS_DISABLE_KEYRING=true` forces file store

---

## Related Documentation

- **Main Guide:** [`CLAUDE.md`](../CLAUDE.md)
- **Hook Setup:** [`HOOKS-CONFIG.md`](HOOKS-CONFIG.md)
- **Architecture:** [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md)

---

## Metrics

- **Total Skills:** 18
- **Total Rules:** 2
- **Total Agents:** 6
- **Total Hooks:** 3
- **Shared Patterns:** 3
- **Last Updated:** February 27, 2026
