# Claude Code Configuration

This directory contains skills, workflows, rules, agents, and shared patterns for AI-assisted development with Claude Code.

---

## Directory Structure

```
.claude/
├── commands/              # 18 actionable skills (invokable workflows)
├── rules/                 # 6 development rules (auto-applied)
├── agents/                # 6 specialized agents
├── hooks/                 # 4 quality gate hooks
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

## Rules (6 Files)

| Rule | Purpose | Applies To |
|------|---------|-----------|
| `testing.md` | Testing requirements & patterns | All new code |
| `go-quality.md` | Go linting + best practices | All Go code |
| `file-size-limits.md` | 500-line file limit | All files |
| `documentation-maintenance.md` | Doc update requirements | Code + doc changes |
| `git-commits.local.md` | Commit message rules | Git operations |
| `go-cache-cleanup.local.md` | Go cache cleanup | Build issues |

---

## Agents (5 Specialized)

| Agent | Model | Purpose |
|-------|-------|---------|
| `code-writer` | Sonnet | Write Go code |
| `test-writer` | Sonnet | Generate comprehensive tests |
| `code-reviewer` | Sonnet | Independent code review |
| `codebase-explorer` | Sonnet | Fast codebase exploration |
| `mistake-learner` | Sonnet | Abstract mistakes to learnings |

---

## Hooks (4 Quality Gates)

| Hook | Trigger | Purpose |
|------|---------|---------|
| `quality-gate.sh` | Stop | Block on quality failures |
| `subagent-review.sh` | SubagentStop | Block on critical issues |
| `pre-compact.sh` | PreCompact | Warn before compaction |
| `context-injector.sh` | UserPromptSubmit | Inject context reminders |

---

## Shared Patterns (2 Files)

| Pattern | Purpose |
|---------|---------|
| `go-test-patterns.md` | Table-driven tests, mocks, testify |
| `integration-test-patterns.md` | CLI integration tests |

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

## Related Documentation

- **Quick Start:** [`CLAUDE-QUICKSTART.md`](../CLAUDE-QUICKSTART.md)
- **Main Guide:** [`CLAUDE.md`](../CLAUDE.md)
- **Hook Setup:** [`HOOKS-CONFIG.md`](HOOKS-CONFIG.md)
- **Architecture:** [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md)

---

## Metrics

- **Total Skills:** 18
- **Total Rules:** 6
- **Total Agents:** 6
- **Total Hooks:** 4
- **Shared Patterns:** 3
- **Last Updated:** December 30, 2024
