# Claude Code Hooks Configuration

This document explains how to enable all the custom hooks created for this project.

---

## Available Hooks

| Hook | File | Trigger | Purpose |
|------|------|---------|---------|
| quality-gate.sh | `.claude/hooks/quality-gate.sh` | Stop | Blocks completion if Go code fails checks |
| subagent-review.sh | `.claude/hooks/subagent-review.sh` | SubagentStop | Blocks if subagent finds critical issues |
| pre-compact.sh | `.claude/hooks/pre-compact.sh` | PreCompact | Warns before context compaction |
| context-injector.sh | `.claude/hooks/context-injector.sh` | UserPromptSubmit | Injects contextual reminders |
| file-size-check.sh | `.claude/hooks/file-size-check.sh` | PreToolUse (Write) | Blocks Go files >600 lines, warns >500 |
| auto-format.sh | `.claude/hooks/auto-format.sh` | PostToolUse (Edit) | Auto-runs gofmt on edited Go files |

---

## How to Enable Hooks

### Option 1: Claude Code Settings UI

1. Open Claude Code settings
2. Navigate to Hooks section
3. Add each hook with appropriate trigger

### Option 2: settings.json Configuration

Add to your Claude Code `settings.json`:

```json
{
  "hooks": {
    "Stop": [
      {
        "matcher": "*",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/quality-gate.sh" }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "*",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/subagent-review.sh" }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "*",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/pre-compact.sh" }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "matcher": "*",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/context-injector.sh" }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Write",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/file-size-check.sh" }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "Edit",
        "hooks": [
          { "type": "command", "command": ".claude/hooks/auto-format.sh" }
        ]
      }
    ]
  }
}
```

### Option 3: Project-level .claude/settings.json

Create `.claude/settings.json` in your project root (same as Option 2).

---

## Hook Details

### quality-gate.sh (Stop Hook)

**Purpose:** Ensures code quality before Claude completes a task.

**What it checks:**
- `go fmt ./...` - Code formatting
- `go vet ./...` - Static analysis
- `golangci-lint run` - Linting (2 min timeout)
- `go test -short ./...` - Unit tests (5 min timeout)
- JavaScript syntax check with `node --check`

**When it blocks:**
- Any Go file modified and quality check fails
- Returns exit code 2 with JSON decision

### subagent-review.sh (SubagentStop Hook)

**Purpose:** Validates subagent output for critical issues.

**What it checks:**
- CRITICAL or FATAL keywords
- Test failures (FAIL...Test)
- Build failures (BUILD FAILED)
- Compilation errors

**When it blocks:**
- Subagent output contains critical issues
- Returns exit code 2 with JSON decision

### pre-compact.sh (PreCompact Hook)

**Purpose:** Warns before context window compaction.

**What it does:**
- Prints warning message
- Reminds to use `/diary` to save learnings
- Creates diary file path

**Never blocks:** Always exits 0

### context-injector.sh (UserPromptSubmit Hook)

**Purpose:** Injects relevant contextual reminders based on prompt.

**Triggers on keywords:**
- test, spec, coverage → Testing reminder
- security, auth, credential → Security reminder
- api, endpoint, nylas → API v3 reminder
- commit, push, pr → Git rules reminder
- split, large file → File size reminder

**Never blocks:** Always exits 0

### file-size-check.sh (PreToolUse Hook for Write)

**Purpose:** Enforces file size limits before writing Go files.

**What it checks:**
- Reads `TOOL_INPUT` JSON for `file_path` and `content`
- Only applies to `.go` files
- Counts lines in content being written

**When it blocks:**
- File would exceed 600 lines → Exit 2 (blocked)
- File would exceed 500 lines → Warning only (exit 0)
- Non-Go files → Always passes

### auto-format.sh (PostToolUse Hook for Edit)

**Purpose:** Automatically formats Go files after edits.

**What it does:**
- Reads `TOOL_INPUT` JSON for `file_path`
- Only applies to `.go` files
- Runs `gofmt -w` on the edited file

**Never blocks:** Always exits 0

---

## Testing Hooks

### Test quality-gate.sh

```bash
# Should pass when no Go changes
bash .claude/hooks/quality-gate.sh

# Test with debug output
bash -x .claude/hooks/quality-gate.sh
```

### Test subagent-review.sh

```bash
# Should pass
CLAUDE_TOOL_OUTPUT="Task completed" bash .claude/hooks/subagent-review.sh

# Should block
CLAUDE_TOOL_OUTPUT="CRITICAL: error found" bash .claude/hooks/subagent-review.sh
```

### Test context-injector.sh

```bash
# Test testing context
CLAUDE_USER_PROMPT="write a test for this" bash .claude/hooks/context-injector.sh

# Test security context
CLAUDE_USER_PROMPT="add authentication" bash .claude/hooks/context-injector.sh
```

### Test file-size-check.sh

```bash
# Should pass (small file)
TOOL_INPUT='{"file_path":"test.go","content":"package main\n"}' bash .claude/hooks/file-size-check.sh

# Should warn (>500 lines)
TOOL_INPUT='{"file_path":"test.go","content":"'"$(printf 'line\n%.0s' {1..510})"'"}' bash .claude/hooks/file-size-check.sh

# Should block (>600 lines)
TOOL_INPUT='{"file_path":"test.go","content":"'"$(printf 'line\n%.0s' {1..650})"'"}' bash .claude/hooks/file-size-check.sh
```

### Test auto-format.sh

```bash
# Create test file and format
echo 'package main; func main() {}' > /tmp/test.go
TOOL_INPUT='{"file_path":"/tmp/test.go"}' bash .claude/hooks/auto-format.sh
cat /tmp/test.go  # Should be formatted
```

---

## Troubleshooting

### Hook not running

1. Check file is executable: `chmod +x .claude/hooks/*.sh`
2. Verify settings.json syntax is valid
3. Check hook path is relative to project root

### Hook blocking unexpectedly

1. Run hook manually to see output
2. Check for false positive patterns
3. Review exit codes (0 = pass, 2 = block)

### Hook errors

1. Check `~/.claude/logs/` for hook logs
2. Verify all required tools are installed (go, golangci-lint, node)
3. Test with `bash -x` for debug output

---

## Environment Variables

Hooks receive these environment variables:

| Variable | Description | Available In |
|----------|-------------|--------------|
| `CLAUDE_USER_PROMPT` | User's prompt text | UserPromptSubmit |
| `CLAUDE_TOOL_OUTPUT` | Tool/subagent output | SubagentStop, PostToolUse |
| `CLAUDE_TOOL_INPUT` | Tool input parameters | PreToolUse |

---

## Security Considerations

1. **Never log secrets** - Hooks can see sensitive data
2. **Use timeouts** - Prevent hanging hooks
3. **Fail open** - Exit 0 if unsure (don't block accidentally)
4. **Minimal permissions** - Hooks run with user permissions
