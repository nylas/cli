# Claude Rules

Rules that auto-apply to all Claude sessions in this project.

---

## Rule Files

| File | Purpose |
|------|---------|
| `go-quality.md` | Go best practices, linting, modern patterns |
| `testing.md` | Test organization, coverage targets, rate limiting |
| `file-size-limits.md` | 500-line limit enforcement |
| `documentation-maintenance.md` | When to update docs |

---

## Local Overrides

Files ending in `.local.md` are **gitignored project-specific overrides**.

They take precedence over standard rules and system defaults.

| Local Rule | Purpose |
|------------|---------|
| `commit-message.local.md` | Project-specific commit format (no Claude attribution) |

### Creating Local Rules

```bash
# Create a local rule (automatically gitignored)
touch .claude/rules/my-rule.local.md
```

Local rules are useful for:
- Team-specific conventions
- Project overrides
- Personal preferences not shared with team

---

## Rule Principles

1. **Single concern** - Each rule focuses on one topic
2. **Actionable** - Clear commands and examples
3. **Concise** - Under 100 lines ideal, max 150
4. **No duplication** - Reference other rules, don't repeat

---

## Adding New Rules

1. Create `rule-name.md` in this directory
2. Keep under 100 lines (reference patterns for details)
3. Include examples
4. Update this README

For project-specific rules, use `.local.md` suffix.
