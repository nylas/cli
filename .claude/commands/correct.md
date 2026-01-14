# Correct Mistake

Mark a mistake and add it to CLAUDE.md learnings.

Correction: $ARGUMENTS

## Instructions

When the user invokes `/correct "explanation"`:

1. **Acknowledge the correction**
   - Understand what was wrong
   - Identify the correct approach

2. **Abstract the learning**
   - Extract the general principle (not the specific instance)
   - Format as actionable directive

3. **Update CLAUDE.md**
   - Read current CLAUDE.md
   - Add to appropriate LEARNINGS subsection:
     - `Project-Specific Gotchas` - Project quirks, conventions
     - `Non-Obvious Workflows` - Unexpected patterns
     - `Time-Wasting Bugs Fixed` - Bugs that cost time
   - Format: `- [Context]: [ALWAYS/NEVER] [action] ([reason])`
   - Keep to ONE line

4. **Confirm update**
   - Show what was added
   - Verify it's in the file

## Formatting Rules

- One line per learning
- Start with context (e.g., "Go tests:", "Integration tests:", "CLI:")
- Use ALWAYS/NEVER for clarity
- Include brief reason in parentheses if helpful

## Examples

**User:** `/correct "we use testify assert not require for non-fatal checks"`

**Action:**
1. Add to CLAUDE.md under `### Project-Specific Gotchas`:
   ```
   - Go tests: Use `assert` for non-fatal checks, `require` only when test cannot continue
   ```
2. Confirm: "Added learning about testify assert vs require to CLAUDE.md"

---

**User:** `/correct "the calendar sync was failing because we forgot to handle DST"`

**Action:**
1. Add to CLAUDE.md under `### Time-Wasting Bugs Fixed`:
   ```
   - Calendar sync: ALWAYS account for DST transitions when comparing times across days
   ```
2. Confirm: "Added learning about DST handling to CLAUDE.md"
