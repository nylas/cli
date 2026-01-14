# Documentation Maintenance Rule

**CRITICAL**: Always update documentation when making code changes.

---

## Documentation Update Matrix

| Change Type | Update Files | Priority |
|-------------|--------------|----------|
| **New CLI command** | CLAUDE.md, docs/COMMANDS.md, cmd/nylas/main.go | CRITICAL |
| **New integration test** | CLAUDE.md, docs/DEVELOPMENT.md | CRITICAL |
| **New adapter/API method** | CLAUDE.md, docs/ARCHITECTURE.md (if new file) | IF NEEDED |
| **New domain model** | CLAUDE.md, docs/ARCHITECTURE.md (if major) | IF NEEDED |
| **Test structure change** | CLAUDE.md, docs/DEVELOPMENT.md, .claude/rules/testing.md | CRITICAL |
| **New skill/workflow** | CLAUDE.md (if user-facing) | IF NEEDED |
| **Security change** | docs/security/overview.md | CRITICAL |
| **Architecture change** | docs/ARCHITECTURE.md, CLAUDE.md | CRITICAL |
| **Utility feature** | CLAUDE.md, docs/COMMANDS.md | CRITICAL |

---

## Quick Reference Checklist

**Before marking task complete:**

### For New Features:
- [ ] Updated CLAUDE.md file structure table
- [ ] Updated docs/COMMANDS.md with examples
- [ ] Updated README.md (if major feature)

### For New Tests:
- [ ] Updated CLAUDE.md test paths
- [ ] Updated docs/DEVELOPMENT.md test list

### For Structural Changes:
- [ ] Updated ALL affected docs
- [ ] Verified no old references remain
- [ ] Updated .claude/ rules if needed

---

## Golden Rule

**If you changed code -> Update docs**

No exceptions.

---

**Files to Never Reference:**
- `local/*.md` - Temporary/historical docs (excluded from context)
- `local/suggestions.md` - Feature proposals only
- `local/SECURITY_REPORT.md` - Historical report

**Quick verification:**
```bash
# After structural changes, verify no stale references:
grep -r "old-pattern" docs/ .claude/ *.md
```

---

**Last Updated:** January 10, 2026
