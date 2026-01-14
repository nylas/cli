# Reflect on Diary Entries

Analyze recent diary entries and propose CLAUDE.md updates.

## Purpose

Review accumulated session learnings and suggest permanent additions to CLAUDE.md.

## Instructions

1. **Read recent diary entries**
   - Location: `~/.claude/memory/diary/`
   - Focus on entries from the last 7-14 days
   - If directory doesn't exist, inform user to use `/diary` first

2. **Read current CLAUDE.md**
   - Check existing LEARNINGS section
   - Identify what's already captured

3. **Identify patterns across sessions**
   - Recurring mistakes → Add to LEARNINGS
   - Common workflows → Add to Quick Reference
   - Rule violations → Strengthen existing rules
   - Outdated advice → Flag for removal

4. **Propose updates**
   - Format as ready-to-add bullets
   - Group by LEARNINGS subsection
   - Flag duplicates or conflicts

5. **Apply only after user approval**

## Output Format

```markdown
## Diary Analysis

**Entries reviewed:** N entries from [date range]

---

## Proposed CLAUDE.md Updates

### New LEARNINGS entries:

**Project-Specific Gotchas:**
- [New entry 1]
- [New entry 2]

**Non-Obvious Workflows:**
- [New entry]

**Time-Wasting Bugs Fixed:**
- [New entry]

### Rule modifications:
- [Existing rule] → [Strengthened version]

### Remove (obsolete):
- [Outdated entry from LEARNINGS]

---

## Confidence Levels

| Entry | Confidence | Reason |
|-------|------------|--------|
| Entry 1 | High | Appeared in 3 sessions |
| Entry 2 | Medium | Single occurrence but significant |

---

**Apply these updates?** Use `/correct` for individual entries or confirm to apply all.
```

## Deduplication Rules

Before proposing an entry, check if CLAUDE.md already contains:
- The exact same advice
- A more general version of the advice
- A conflicting rule

If duplicate found, skip or suggest merging.

## Example

After reviewing 5 diary entries that all mention rate limiting issues:

```markdown
## Diary Analysis

**Entries reviewed:** 5 entries from 2025-12-23 to 2025-12-30

---

## Proposed CLAUDE.md Updates

### New LEARNINGS entries:

**Project-Specific Gotchas:**
- Integration tests: ALWAYS use `acquireRateLimit(t)` before API calls in parallel tests

**Time-Wasting Bugs Fixed:**
- Calendar sync: Handle DST by using `time.In(loc)` not raw hour arithmetic

### Rule modifications:
- None

### Remove (obsolete):
- None

---

## Confidence Levels

| Entry | Confidence | Reason |
|-------|------------|--------|
| Rate limiting | High | Mentioned in 4/5 sessions |
| Table-driven tests | High | Mentioned in 3/5 sessions |
| DST handling | Medium | Single occurrence but 2-hour debug session |

---

**Apply these updates?**
```
