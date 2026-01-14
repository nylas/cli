# File Size Limit Rule

**MANDATORY**: Enforce file size limits for all Go code in this repository.

---

## The 500-Line Rule

### File Size Targets

| Target | Lines | Status |
|--------|-------|--------|
| **Ideal** | ≤500 lines | ✅ Preferred |
| **Acceptable** | ≤600 lines | ⚠️ Borderline |
| **Requires Refactoring** | >600 lines | ❌ Must split |

**Line count includes:** Code, comments, blank lines, package declaration, imports

---

## CRITICAL: Always Check File Size

### Before Completing Any Task:

```bash
# Check file size after changes
wc -l <modified-file>.go

# If file exceeds 500 lines:
# → Plan to split it
# → Split by responsibility (types, helpers, handlers)
# → Split before marking task complete
```

### When Writing New Code:

**If adding code would push file over 500 lines:**
1. ✅ **STOP** - Do not add more code to the file
2. ✅ **SPLIT** - Break the file by responsibility (types, helpers, handlers)
3. ✅ **VERIFY** - Run `make build` and `make test-unit`
4. ✅ **DOCUMENT** - Update CLAUDE.md if file structure changes

**Never create files >600 lines**

---

## When to Split Files

### ✅ SPLIT when:
- File exceeds 600 lines (MANDATORY)
- File exceeds 500 lines AND has multiple distinct responsibilities
- Adding new feature would push file over 500 lines
- File has logical groupings (CRUD, helpers, types, tests)

### ⚠️ EVALUATE when:
- File is 500-600 lines but cohesive
- Splitting would create circular dependencies
- File represents a single, tightly-coupled unit

### ❌ DON'T SPLIT when:
- File is under 500 lines
- Functions are tightly coupled
- Split would harm code clarity
- File is a single cohesive algorithm

---

## How to Split Files

**Pattern:** Split by responsibility - types, helpers, handlers, tests

### Quick Split Patterns:

1. **Handler Split by Type** (handlers_types_test.go pattern)
   - Base operations → `<feature>_base.go`
   - Utilities → `<feature>_utilities.go`
   - Specific features → `<feature>_<feature-name>.go`

2. **Command Split by Action** (contacts.go pattern)
   - Main CRUD → `<feature>_main.go`
   - Distinct features → `<feature>_<feature-name>.go`

3. **Test Split by Complexity**
   - Basic tests → `<feature>_test_basic.go`
   - Advanced tests → `<feature>_test_advanced.go`

4. **Feature Split by Module**
   - Core logic → `<feature>_core.go`
   - Helpers → `<feature>_helpers.go`
   - Types → `<feature>_types.go`

---

## Verification Steps

### After Splitting a File:

```bash
# 1. Verify all files ≤600 lines
find . -name "*.go" -exec wc -l {} \; | awk '$1 > 600 {print}'

# 2. Check build
make build

# 3. Run tests
make test-unit

# 4. Run linting
golangci-lint run --timeout=5m

# 5. Verify integration tests (if applicable)
make test-integration
```

**All checks must pass before completing task.**

---

## Exceptions (Rare)

**Only exception allowed:**
- Generated code (marked with `// Code generated`)
- Template files that must be kept together
- Third-party vendored code

**No exceptions for:**
- ❌ "It's almost done, I'll split it later"
- ❌ "It's just a few lines over"
- ❌ "The functions are related"
- ❌ "It would be hard to split"

---

## Integration with Development Workflow

### New Feature Development:

```
Write Code → Check Size → If >500 lines → Split → Verify → Complete
     ↑                                         |
     └─────────── Back if >600 lines ──────────┘
```

### Code Review Checklist:

- [ ] All modified files ≤600 lines
- [ ] All new files ≤500 lines
- [ ] If split was required, followed responsibility-based patterns
- [ ] Build succeeds
- [ ] Tests pass
- [ ] CLAUDE.md updated (if file structure changed)

---

## Enforcement

**This is a MANDATORY rule:**
- ✅ All new code must follow this rule
- ✅ Modified files >600 lines must be split
- ✅ No pull requests with files >600 lines
- ✅ Pre-commit hook could enforce this (future enhancement)

**Validation command:**
```bash
# Check for files exceeding limit
find . -name "*.go" -not -path "*/vendor/*" -exec wc -l {} \; | \
  awk '$1 > 600 {print "❌ EXCEEDS LIMIT:", $2, "(" $1, "lines)"}' | \
  sort -t: -k2 -rn
```
