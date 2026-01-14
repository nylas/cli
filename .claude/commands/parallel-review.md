# Parallel Review

Review code changes using multiple parallel code-reviewer agents.

Files to review: $ARGUMENTS

**Agent:** Uses `.claude/agents/code-reviewer.md` for review criteria.

## When to Use

- Large PRs with many changed files
- Changes spanning multiple directories
- Pre-commit review of staged changes

## Instructions

### 1. Get Files to Review

```bash
git diff --staged --name-only  # Staged changes
git diff --name-only           # All uncommitted
git diff main...HEAD --name-only  # PR changes
```

### 2. Group Files by Layer

| Group | Pattern | Focus |
|-------|---------|-------|
| CLI | `internal/cli/**/*.go` | Commands, flags, UX |
| Adapters | `internal/adapters/**/*.go` | API calls, retries |
| Domain | `internal/domain/*.go` | Types, validation |
| Tests | `*_test.go` | Coverage, mocks |

### 3. Launch Parallel Reviewers

| Changed Files | Reviewers |
|---------------|-----------|
| 1-3 | 1 reviewer |
| 4-8 | 2 reviewers |
| 9-15 | 3 reviewers |
| 16+ | 4 reviewers |

Launch code-reviewer agents with file groups. Each reviewer uses the checklist from code-reviewer.md.

### 4. Consolidate Results

```markdown
## Parallel Review Results

**Files reviewed:** N | **Reviewers:** N | **Issues:** N critical, N warnings

### Critical (Must Fix)
| Location | Issue | Fix |
|----------|-------|-----|

### Warnings (Should Fix)
| Location | Issue | Fix |
|----------|-------|-----|

### Verdict
[ ] APPROVE | [ ] CHANGES NEEDED | [ ] DISCUSS
```

## Examples

```bash
/parallel-review staged                    # Review staged changes
/parallel-review pr                        # Review PR changes
/parallel-review internal/cli/email/*.go   # Review specific files
```

## Related

- `/review-pr` - Single-reviewer PR review
- `/parallel-explore` - Explore codebase in parallel
