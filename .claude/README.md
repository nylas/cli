# Claude Code Configuration

This directory contains skills, workflows, rules, agents, and shared patterns for AI-assisted development with Claude Code.

---

## Directory Structure

```
.claude/
├── commands/              # 19 actionable skills (invokable workflows)
├── rules/                 # 4 development rules (auto-applied)
├── agents/                # 9 specialized agents
├── hooks/                 # 6 hook scripts (2 wired, 4 available)
├── shared/patterns/       # 3 reusable pattern files
├── settings.json          # Security hooks & permissions
├── HOOKS-CONFIG.md        # Hook configuration guide
└── README.md              # This file
```

---

## Skills (19 Total)

### Feature Development (4 skills)

| Skill | Purpose |
|-------|---------|
| `add-command` | New CLI command (includes CRUD generation) |
| `add-api-method` | Extend API client |
| `add-domain-type` | New domain models |
| `add-flag` | Add command flags |

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

### Orchestration (2 skills)

| Skill | Purpose |
|-------|---------|
| `parallel-explore` | Multi-agent codebase exploration |
| `parallel-review` | Multi-agent code review |

### Maintenance (1 skill)

| Skill | Purpose |
|-------|---------|
| `update-docs` | Documentation updates |

---

## Rules (4 Files)

| Rule | Purpose | Applies To |
|------|---------|-----------|
| `testing.md` | Testing requirements & patterns | All new code |
| `go-quality.md` | Go linting + best practices | All Go code |
| `file-size-limits.md` | 500-line file limit | All files |
| `documentation-maintenance.md` | Doc update requirements | Code + doc changes |

---

## Agents (8 Specialized)

| Agent | Purpose |
|-------|---------|
| `code-writer` | Write Go/JS/CSS code |
| `test-writer` | Generate comprehensive tests |
| `code-reviewer` | Independent code review |
| `security-auditor` | Security vulnerability analysis |
| `documentation-writer` | Documentation updates |
| `codebase-explorer` | Fast codebase exploration |
| `frontend-agent` | JS/CSS/Go templates |
| `mistake-learner` | Abstract mistakes to learnings |

**References:** `agents/references/` contains helper-reference, security-checklist, doc-standards.

---

## Hooks (6 Scripts)

| Hook | Trigger | Wired | Purpose |
|------|---------|-------|---------|
| `file-size-check.sh` | PreToolUse (Write) | Yes | Block Go files >600 lines |
| `auto-format.sh` | PostToolUse (Edit) | Yes | Auto-run gofmt |
| `quality-gate.sh` | Stop | No | Block on quality failures |
| `subagent-review.sh` | SubagentStop | No | Block on critical issues |
| `pre-compact.sh` | PreCompact | No | Warn before compaction |
| `context-injector.sh` | UserPromptSubmit | No | Inject context reminders |

**To wire unwired hooks:** See `HOOKS-CONFIG.md` for settings.json config.

---

## Shared Patterns (3 Files)

| Pattern | Purpose |
|---------|---------|
| `go-test-patterns.md` | Table-driven tests, mocks, testify |
| `integration-test-patterns.md` | CLI + Air integration tests |
| `playwright-patterns.md` | Selectors, templates, commands |

---

## Related Documentation

- **Main Guide:** [`CLAUDE.md`](../CLAUDE.md)
- **Hook Setup:** [`HOOKS-CONFIG.md`](HOOKS-CONFIG.md)
- **Architecture:** [`docs/ARCHITECTURE.md`](../docs/ARCHITECTURE.md)
