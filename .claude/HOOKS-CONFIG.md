# Claude Code Hooks Configuration

Hook scripts for quality enforcement. Some are wired in `settings.json`, others are available to enable.

---

## Hook Status

| Hook | File | Trigger | Status |
|------|------|---------|--------|
| file-size-check.sh | `.claude/hooks/file-size-check.sh` | PreToolUse (Write) | **Active** |
| auto-format.sh | `.claude/hooks/auto-format.sh` | PostToolUse (Edit) | **Active** |
| quality-gate.sh | `.claude/hooks/quality-gate.sh` | Stop | Available |
| subagent-review.sh | `.claude/hooks/subagent-review.sh` | SubagentStop | Available |
| pre-compact.sh | `.claude/hooks/pre-compact.sh` | PreCompact | Available |
| context-injector.sh | `.claude/hooks/context-injector.sh` | UserPromptSubmit | Available |

---

## Enabling Available Hooks

Add to the `"hooks"` section in `.claude/settings.json`:

```json
"Stop": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/quality-gate.sh" }
    ]
  }
],
"SubagentStop": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/subagent-review.sh" }
    ]
  }
],
"PreCompact": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/pre-compact.sh" }
    ]
  }
],
"UserPromptSubmit": [
  {
    "matcher": "",
    "hooks": [
      { "type": "command", "command": ".claude/hooks/context-injector.sh" }
    ]
  }
]
```

---

## Hook Behavior

### Active Hooks

**file-size-check.sh** (PreToolUse/Write)
- Blocks Go files >600 lines (exit 2), warns >500 lines
- Skips non-Go files

**auto-format.sh** (PostToolUse/Edit)
- Runs `gofmt -w` on edited Go files
- Never blocks (exit 0)

### Available Hooks

**quality-gate.sh** (Stop)
- Runs `go fmt`, `go vet`, `golangci-lint` on modified Go files
- Runs `node --check` on modified JS files
- Blocks if any check fails

**subagent-review.sh** (SubagentStop)
- Scans subagent output for CRITICAL/FATAL/FAIL patterns
- Blocks if critical issues found

**pre-compact.sh** (PreCompact)
- Prints reminder to save progress before context compaction
- Never blocks

**context-injector.sh** (UserPromptSubmit)
- Injects reminders based on prompt keywords (test, security, api, playwright, commit)
- Never blocks

---

## Troubleshooting

1. **Hook not running:** Check `chmod +x .claude/hooks/*.sh`
2. **Hook blocking unexpectedly:** Run manually: `bash -x .claude/hooks/<hook>.sh`
3. **Hook errors:** Check `~/.claude/logs/`
4. **Exit codes:** 0 = pass, 2 = block

## Environment Variables

| Variable | Available In |
|----------|-------------|
| `CLAUDE_USER_PROMPT` | UserPromptSubmit |
| `CLAUDE_TOOL_OUTPUT` | SubagentStop, PostToolUse |
| `TOOL_INPUT` | PreToolUse |
