# Subagent Documentation

This directory contains specialized agents for the Nylas CLI codebase.

---

## Available Agents

| Agent | Model | Purpose | Parallel Safe |
|-------|-------|---------|---------------|
| `code-writer` | sonnet | Go implementation tasks | Limited |
| `test-writer` | sonnet | Test generation | Limited |
| `code-reviewer` | sonnet | Code review | Safe |
| `security-auditor` | sonnet | Security vulnerability analysis | Safe |
| `documentation-writer` | sonnet | Documentation updates | Limited |
| `codebase-explorer` | haiku | Exploration | Safe |
| `mistake-learner` | sonnet | Learning capture | Serial only |

---

## Parallelization Guide

Use parallel agents to explore or review the 745-file codebase without exhausting context.

### When to Use Parallel Agents

| Task | Agents | Why |
|------|--------|-----|
| Full codebase exploration | 3 | One per major directory |
| Feature search | 3 | Search cli, adapters, domain simultaneously |
| Multi-file PR review | 4 | Review different files in parallel |
| Test coverage analysis | 4 | Analyze different packages |

### Invocation Patterns

```
# Full exploration (3 agents)
"Explore using 3 parallel agents:
 - Agent 1: internal/cli/
 - Agent 2: internal/adapters/
 - Agent 3: internal/domain/ + ports/"

# Feature search (3 agents)
"Find all email-related code using 3 agents across cli, adapters, domain"

# PR review (4 agents)
"Review these 8 files using 4 parallel code-reviewer agents"
```

### Directory Parallelization Value

| Directory | Files | Parallel Value |
|-----------|-------|----------------|
| `internal/cli/` | 268 | HIGH |
| `internal/adapters/` | 158 | HIGH |
| `internal/domain/` | 21 | LOW (shared) |

### Safe vs Unsafe

**Safe to parallelize:**
- Explore, review, search across different directories
- Multiple code-reviewers on different files
- Multiple codebase-explorers

**Avoid parallelizing:**
- Write to same file
- Modify `domain/` or `ports/nylas.go` in parallel
- Run integration tests in parallel (rate limiting)

### Agent Parallel Compatibility

| Agent | Can Run With | Cannot Run With |
|-------|--------------|-----------------|
| `codebase-explorer` | All agents | - |
| `code-reviewer` | All agents | - |
| `security-auditor` | All agents | - |
| `documentation-writer` | Different file writers | Another doc-writer |
| `code-writer` | Different directory writers | Same-file writers |
| `test-writer` | Different package writers | Same-package writers |
| `mistake-learner` | None (serial only) | All write agents |

### Key Benefit

Parallel agents have **isolated context windows** - prevents "dumb Claude mid-session" when exploring large codebases.

---

## Development Pipeline

```
[codebase-explorer] → [code-writer] → [test-writer] → [code-reviewer]
     research          implement         test            review
                              ↓                              ↓
                    [mistake-learner]              [security-auditor]
                       (on errors)                  (security review)
                              ↓
                    [documentation-writer]
                       (update docs)
```

**Handoff signals between agents:**
- Explorer → Writer: Research complete
- Writer → Tester: Implementation complete
- Tester → Reviewer: Tests pass
- Reviewer → Security: Code approved, security check
- Any → Learner: Error detected
- Completion → Doc Writer: Feature complete, update docs

---

## Model Selection Rationale

| Model | Cost | Use For |
|-------|------|---------|
| **sonnet** | $$ | Implementation and review (code-writer, test-writer, documentation-writer, code-reviewer, security-auditor) |
| **haiku** | $ | Exploration (codebase-explorer) |

**Principle:** Use the cheapest model that delivers acceptable quality for the task.

---

## When to Use Each Agent

| Scenario | Agent(s) | Why |
|----------|----------|-----|
| New feature implementation | code-writer → test-writer → code-reviewer | Full dev cycle |
| Security-sensitive change | code-writer → security-auditor | Auth, credentials, input handling |
| Pre-release audit | security-auditor + code-reviewer | Comprehensive review |
| New CLI command | code-writer → documentation-writer | Code + docs together |
| Bug fix | code-writer → test-writer | Fix + regression test |
| Refactoring | code-reviewer → code-writer | Review first, then implement |
| Documentation only | documentation-writer | Standalone doc updates |
| Codebase questions | codebase-explorer | Read-only exploration |
