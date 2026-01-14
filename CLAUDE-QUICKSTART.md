# Claude Code Quick Start Guide

Your AI-powered development assistant with self-learning capabilities.

---

## TL;DR - Essential Commands

```bash
# Start of session
/session-start              # Load context from previous sessions

# During development
/generate-tests             # Generate tests for your code
/fix-build                  # Fix build errors
/run-tests                  # Run unit/integration tests
/security-scan              # Security analysis

# When mistakes happen
/correct "description"      # Capture mistake → adds to LEARNINGS

# End of session
/diary                      # Save session learnings
```

---

## Session Workflow

```
┌─────────────────────────────────────────────────────────────┐
│  START SESSION                                              │
│  /session-start                                             │
└─────────────────┬───────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────┐
│  DEVELOPMENT                                                │
│  Write code, use commands, Claude learns from mistakes      │
└─────────────────┬───────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────┐
│  MISTAKE HAPPENS?                                           │
│  /correct "what went wrong"                                 │
└─────────────────┬───────────────────────────────────────────┘
                  ▼
┌─────────────────────────────────────────────────────────────┐
│  END SESSION                                                │
│  /diary                                                     │
└─────────────────────────────────────────────────────────────┘
```

---

## Commands Reference

### Session Management

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `/session-start` | Load context from previous sessions | Start of every session |
| `/diary` | Save learnings to memory | End of session |
| `/reflect` | Review diary, propose CLAUDE.md updates | Weekly review |

### Self-Learning

| Command | Purpose | Example |
|---------|---------|---------|
| `/correct` | Capture a mistake for learning | `/correct "forgot to handle nil pointer"` |

**How it works:**
1. You notice Claude made a mistake
2. Run `/correct "description of what went wrong"`
3. Claude abstracts the pattern and adds it to LEARNINGS in CLAUDE.md
4. Future sessions avoid the same mistake

### Development Commands

| Command | Purpose | Tools Used |
|---------|---------|------------|
| `/add-command` | Create new CLI command | Read, Write, Edit, Glob, Grep |
| `/generate-crud-command` | Generate full CRUD command | Read, Write, Edit, Glob, Grep |
| `/add-flag` | Add flag to existing command | Read, Edit |
| `/add-domain-type` | Add new domain type | Read, Write, Edit |
| `/add-api-method` | Add new API method | Read, Write, Edit |

### Testing Commands

> **Authoritative source:** `.claude/commands/run-tests.md`

| Command | Purpose | Tools Used |
|---------|---------|------------|
| `/run-tests` | Run unit/integration tests | Read, Bash(go test, make test) |
| `/generate-tests` | Generate tests for code | Read, Write, Edit, Bash(go test) |
| `/add-integration-test` | Add integration test | Read, Write, Edit, Bash(go test) |
| `/debug-test-failure` | Analyze and fix test failures | Read, Edit, Bash(go test) |
| `/analyze-coverage` | Analyze test coverage | Read, Bash(go test) |

**Coverage goals:** See `.claude/rules/testing.md` for authoritative targets.

### Quality Commands

> **Make targets:** See `docs/DEVELOPMENT.md` for authoritative list.

| Command | Purpose | Tools Used |
|---------|---------|------------|
| `/fix-build` | Fix Go build errors | Read, Edit, Write, Bash(go build, go vet) |
| `/security-scan` | Security vulnerability scan | Read, Grep, Glob, Bash(make security) |
| `/review-pr` | Review pull request | Read, Grep, Glob, Bash(git) |
| `/update-docs` | Update documentation | Read, Write, Edit |

### Parallel Commands

| Command | Purpose | When to Use |
|---------|---------|-------------|
| `/parallel-explore` | Explore codebase with 4-5 parallel agents | Large codebase search, cross-layer feature discovery |
| `/parallel-review` | Review code with parallel reviewer agents | Large PRs, multi-file reviews |

**Parallel Explore Details:**

```bash
# Basic usage - searches all layers
/parallel-explore "email sending functionality"

# Spawns codebase-explorer agents across:
# - internal/cli/      (commands, flags)
# - internal/adapters/ (API integrations)
# - internal/domain/   (core types)
```

Each agent uses thoroughness levels (quick/medium/thorough) and consolidates results by layer.

---

## Specialized Agents

Claude has specialized agents for different tasks:

### code-writer (Sonnet)
**Best for:** Production-ready Go code

```
Expertise:
- Go: Hexagonal architecture, error wrapping, table-driven tests

Reference: .claude/agents/references/helper-reference.md
```

### codebase-explorer (Haiku)
**Best for:** Fast codebase exploration without coding

```
Thoroughness Levels:
- quick:    1-2 searches, 1-2 files, 50 words  (targeted lookups)
- medium:   3-5 searches, 3-5 files, 150 words (default)
- thorough: Exhaustive, 10+ files, 300 words   (deep dives)

Use for:
- Finding where functionality is implemented
- Understanding code patterns
- Answering "where is X?" questions
```

### test-writer (Sonnet)
**Best for:** Go test generation (unit + integration)

```
Go Tests: Table-driven with t.Run(), testify assertions, rate-limited

Patterns: .claude/shared/patterns/go-test-patterns.md
          .claude/shared/patterns/integration-test-patterns.md
```

### mistake-learner (Sonnet)
**Best for:** Abstracting mistakes into learnings

```
Process:
1. Understand what went wrong
2. Abstract the pattern (not specific instance)
3. Add to CLAUDE.md LEARNINGS section
```

### code-reviewer (Sonnet)
**Best for:** Independent code review for quality

```
Checks: Quality, bugs, security, performance
Can run in parallel with other reviewers
```

### security-auditor (Opus)
**Best for:** Deep security vulnerability analysis

```
Audits: Secrets, command injection, path traversal, dependencies
Checklist: .claude/agents/references/security-checklist.md
```

---

## Quality Hooks

Hooks run automatically to enforce quality:

### quality-gate.sh (Stop Hook)
**Runs:** When Claude tries to complete a task
**Blocks if:** Go code fails fmt, vet, lint, or tests

```bash
# What it checks:
go fmt ./...           # Code formatting
go vet ./...           # Static analysis
golangci-lint run      # Linting
go test -short ./...   # Unit tests
```

### subagent-review.sh (SubagentStop Hook)
**Runs:** When a subagent completes
**Blocks if:** Output contains CRITICAL, FAIL, or BUILD FAILED

### pre-compact.sh (PreCompact Hook)
**Runs:** Before context window compaction
**Action:** Warns to save learnings with `/diary`

### context-injector.sh (UserPromptSubmit Hook)
**Runs:** When you submit a prompt
**Action:** Injects relevant context based on keywords

```
Triggers:
- "test" → Testing patterns reminder
- "security" → Security scan reminder
- "api" → Nylas v3 API reminder
- "commit" → Git rules reminder
```

### file-size-check.sh (PreToolUse Hook for Write)
**Runs:** Before writing Go files
**Blocks if:** File would exceed 600 lines
**Warns if:** File would exceed 500 lines

### auto-format.sh (PostToolUse Hook for Edit)
**Runs:** After editing Go files
**Action:** Auto-runs `gofmt -w` on the edited file

**Hook Configuration:** See `.claude/HOOKS-CONFIG.md` for settings.json setup

---

## Memory & Context

### Session Continuity
```
claude-progress.txt           # What's done, in progress, next up (auto-updated)
~/.claude/memory/diary/       # Session learnings from /diary
```

### Context Management Tips
- Use `/mcp` to disable unused MCP servers
- For large tasks: dump plan to .md, `/clear`, resume
- Press `#` to quickly update CLAUDE.md during sessions
- Agents load reference files on-demand (not auto-loaded)

---

## LEARNINGS Section

CLAUDE.md has a self-updating LEARNINGS section:

### Project-Specific Gotchas
Things unique to this codebase:
- Go tests: ALWAYS use table-driven tests with t.Run()
- Integration tests: ALWAYS use acquireRateLimit(t) before API calls
- Directory permissions: Use 0750, not 0755 (gosec G301)

### Non-Obvious Workflows
Surprising sequences:
- Progressive disclosure: Keep main skill files under 100 lines
- Self-learning: Use "Reflect → Abstract → Generalize → Write"

### Time-Wasting Bugs Fixed
Hard-won knowledge:
- Go build cache corruption: Fix with `sudo rm -rf ~/.cache/go-build ~/go/pkg/mod && go clean -cache`

---

## Best Practices

### Do This

```bash
# Start every session with context
/session-start

# Capture mistakes immediately
/correct "what went wrong"

# Save learnings before ending
/diary

# Use specialized agents for their domain
# - test-writer for tests (Go unit + integration)
# - codebase-explorer for finding code
```

### Avoid This

```bash
# Don't skip session start - you lose context
# Don't ignore mistakes - they'll repeat
# Don't skip /diary - learnings are lost
```

---

## Quick Reference Card

| Task | Command |
|------|---------|
| **Start session** | `/session-start` |
| **Create command** | `/add-command` |
| **Generate tests** | `/generate-tests` |
| **Fix build** | `/fix-build` |
| **Run tests** | `/run-tests` |
| **Security check** | `/security-scan` |
| **Explore codebase** | `/parallel-explore` |
| **Review PR** | `/parallel-review` |
| **Capture mistake** | `/correct "description"` |
| **End session** | `/diary` |
| **Review learnings** | `/reflect` |

---

## Getting Help

- **All commands:** Type `/` and see autocomplete
- **Command details:** Read `.claude/commands/<command>.md`
- **Agent details:** Read `.claude/agents/<agent>.md`
- **Hook setup:** Read `.claude/HOOKS-CONFIG.md`
- **Project rules:** Read `CLAUDE.md`
- **Architecture:** Read `docs/ARCHITECTURE.md`

### Documentation Hierarchy

| Topic | Authoritative Source |
|-------|---------------------|
| Architecture | `docs/ARCHITECTURE.md` |
| Make targets | `docs/DEVELOPMENT.md` |
| Test coverage | `.claude/rules/testing.md` |
| File size limits | `.claude/rules/file-size-limits.md` |
| Go test patterns | `.claude/shared/patterns/go-test-patterns.md` |
| Integration tests | `.claude/shared/patterns/integration-test-patterns.md` |
| CLI helpers | `.claude/agents/references/helper-reference.md` |
| Security checklist | `.claude/agents/references/security-checklist.md` |
| Doc standards | `.claude/agents/references/doc-standards.md` |

---

**Welcome to self-learning Claude Code!**
