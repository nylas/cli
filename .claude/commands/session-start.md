# Session Start Protocol

Initialize a new Claude Code session with full context.

## Purpose

At the start of each session, quickly understand:
- What work was done previously
- What's currently in progress
- What needs to be done next

## Instructions

1. **Confirm working directory**
   ```bash
   pwd
   ```

2. **Read progress file**
   - Read `claude-progress.txt` in project root
   - Understand completed work, in-progress tasks, and next steps

3. **Check recent git history**
   ```bash
   git log --oneline -10
   ```

4. **Check for uncommitted changes**
   ```bash
   git status
   ```

5. **Read any recent diary entries**
   - Check `~/.claude/memory/diary/` for entries from last 3 days
   - Note any learnings that should be applied

6. **Report session context**

## Output Format

```markdown
## Session Context

**Project:** nylas-cli
**Working Directory:** [path]
**Branch:** [branch name]

### Previous Session Summary
[From claude-progress.txt]

### Uncommitted Changes
[From git status, or "Clean working tree"]

### Recent Commits
[Last 5 commits]

### Pending Work
1. [In progress item]
2. [Next up item]

### Relevant Learnings
[Any diary entries from last 3 days]

---

Ready to continue. What would you like to work on?
```

## When to Use

- At the start of every new Claude Code session
- After a long break from the project
- When context feels stale or incomplete
