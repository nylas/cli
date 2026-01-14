# Session Diary Entry

Capture learnings from the current session for persistent memory.

## Purpose

Create a diary entry to preserve session insights before they're lost to context limits or session end.

## Instructions

1. **Review the session**
   - What was accomplished?
   - What problems were encountered?
   - What decisions were made and why?

2. **Create diary file**
   - Location: `~/.claude/memory/diary/`
   - Filename: `YYYY-MM-DD-session-N.md` (increment N for multiple sessions per day)
   - Create directory if it doesn't exist

3. **Write diary entry**

```markdown
# Session Diary - [DATE]

## Project
nylas-cli

## Accomplishments
- [What was completed]

## Design Choices Made
- [Key decisions and rationale]

## Obstacles Encountered
- [Problems hit and how resolved]

## User Preferences Discovered
- [Implicit or explicit preferences noted]

## Learnings to Persist
- [One-line items that should go to CLAUDE.md LEARNINGS section]

## Follow-up Needed
- [Tasks that weren't completed]
```

4. **Suggest CLAUDE.md updates**
   - If any learnings are significant, suggest adding them via `/correct`

## When to Use

- Before ending a long session
- When context is getting full (pre-compaction)
- After completing a major feature
- After debugging a tricky issue
- When the user asks to save session state

## Example Output

```markdown
# Session Diary - 2025-12-30

## Project
nylas-cli

## Accomplishments
- Implemented webhook retry logic
- Fixed rate limiting in integration tests
- Added DST handling to calendar sync

## Design Choices Made
- Used exponential backoff with jitter for webhook retries (prevents thundering herd)
- Chose SQLite over Redis for local cache (simpler deployment)

## Obstacles Encountered
- Integration tests were flaky due to rate limits → Added acquireRateLimit() helper
- Calendar events off by 1 hour → DST transition not handled → Added timezone-aware comparison

## User Preferences Discovered
- Prefers testify assert over require for non-fatal checks
- Wants all handlers to explicitly return after error responses

## Learnings to Persist
- Calendar: ALWAYS handle DST when comparing times across days
- Integration tests: Use acquireRateLimit(t) before any API call

## Follow-up Needed
- [ ] Add retry tests for webhook failures
- [ ] Document rate limiting strategy in DEVELOPMENT.md
```

Saved to: `~/.claude/memory/diary/2025-12-30-session-1.md`
