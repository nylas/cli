# Parallel Explore

Explore the codebase using multiple parallel codebase-explorer agents.

Search scope: $ARGUMENTS

**Agent:** Uses `.claude/agents/codebase-explorer.md` for exploration.

## When to Use

- Large codebase exploration (745+ Go files)
- Feature search across multiple directories
- Understanding cross-layer implementation
- Pre-exploration before writing code

## Directory Structure

| Directory | Best For |
|-----------|----------|
| `internal/cli/` | Commands, flags, user interactions |
| `internal/adapters/` | API integrations, external services |
| `internal/domain/` | Core types, business logic |
| `internal/ports/` | Interface definitions |

## Instructions

### 1. Determine Scope

| Query Type | Directories |
|------------|-------------|
| "Where is X implemented?" | All directories |
| "How does feature Y work?" | cli + adapters + domain |
| "Find all Z handlers" | cli |
| "API integration for W" | adapters + ports |

### 2. Launch 4-5 Parallel Agents

Each codebase-explorer agent searches one directory and reports:
- Key files with `path:line` references
- Patterns found
- Related files

### 3. Consolidate Results

```markdown
## Exploration Results: [query]

### Summary
[Combined answer]

### By Layer
- **CLI:** [findings]
- **Adapters:** [findings]
- **Domain:** [findings]

### Key Files
1. `path/to/file.go:line` - [why]
```

## Examples

```bash
/parallel-explore "email sending functionality"
/parallel-explore "rate limiting implementation"
/parallel-explore "Calendar event types"
```

## Related

- `/parallel-review` - Review code in parallel
- `/analyze-coverage` - Test coverage analysis
