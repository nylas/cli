# MCP (Model Context Protocol)

Enable AI assistants to interact with your Nylas email and calendar.

> **Quick Reference:** [COMMANDS.md](../COMMANDS.md#mcp-model-context-protocol) | **Related:** [AI Features](ai.md)

---

## Overview

The Nylas CLI includes a native MCP server that enables AI assistants like Claude Desktop, Cursor, Windsurf, and Codex to access your email and calendar. The server:

1. Calls the Nylas API v3 directly using your local authenticated credentials
2. Injects your default authenticated grant automatically
3. Detects your system timezone for consistent time display
4. Exposes native MCP tools for email, calendar, contacts, and utility functions

---

## Quick Start

```bash
# 1. Authenticate with Nylas
nylas auth login

# 2. Install MCP for your AI assistant
nylas mcp install

# 3. Restart your AI assistant to load the configuration
```

---

## Commands

### Install

Configure MCP for AI assistants:

```bash
nylas mcp install                          # Interactive mode
nylas mcp install --assistant claude-code  # Specific assistant
nylas mcp install --assistant cursor       # Cursor IDE
nylas mcp install --assistant codex        # Codex CLI
nylas mcp install --all                    # All detected assistants
nylas mcp install --binary /path/to/nylas  # Custom binary path
```

### Status

Check installation status:

```bash
nylas mcp status
```

### Uninstall

Remove MCP configuration:

```bash
nylas mcp uninstall --assistant cursor
nylas mcp uninstall --all
```

### Serve

Start the MCP server (called by AI assistants, not directly):

```bash
nylas mcp serve
```

---

## Supported Assistants

| Assistant | ID | Config Location | Notes |
|-----------|-----|-----------------|-------|
| Claude Desktop | `claude-desktop` | `~/Library/Application Support/Claude/claude_desktop_config.json` | macOS |
| Claude Code | `claude-code` | `~/.claude.json` | Auto-configures permissions |
| Cursor | `cursor` | `~/.cursor/mcp.json` | |
| Windsurf | `windsurf` | `~/.codeium/windsurf/mcp_config.json` | |
| VS Code | `vscode` | `.vscode/mcp.json` | Project-level config |
| Codex | `codex` | `~/.codex/config.toml` | Uses `codex mcp add/remove` |

---

## Available MCP Tools

### Email

| Tool | Description |
|------|-------------|
| `list_messages` | Search and retrieve emails |
| `list_threads` | List email threads |
| `create_draft` | Create a new draft |
| `update_draft` | Update an existing draft |
| `send_message` | Send a new email (requires confirmation) |
| `send_draft` | Send a draft (requires confirmation) |
| `get_folder_by_id` | Get folder details |

### Calendar

| Tool | Description |
|------|-------------|
| `list_calendars` | List all calendars |
| `list_events` | List calendar events |
| `create_event` | Create a new event |
| `update_event` | Update an existing event |
| `availability` | Check availability |

### Utilities

| Tool | Description |
|------|-------------|
| `current_time` | Get current time with timezone |
| `epoch_to_datetime` | Convert Unix timestamp to datetime |
| `datetime_to_epoch` | Convert datetime to Unix timestamp |

---

## Features

### Automatic Timezone Detection

The MCP server detects your system timezone and injects it into server instructions. This ensures AI assistants display all timestamps consistently in your local timezone.

**Before:** Mixed timezones (emails in UTC, calendar in Pacific)
**After:** All times in your local timezone (e.g., EST)

### Auto-configured Permissions (Claude Code)

When installing for Claude Code, the CLI automatically adds `mcp__nylas__*` to `~/.claude/settings.json`, granting permission for all Nylas MCP tools without interactive prompts.

### Default Grant Injection

The MCP server automatically injects your authenticated default grant ID into tool calls, so AI assistants don't need to ask for your email address.

---

## Configuration

### Manual Configuration

If you prefer manual setup, add to your assistant's config:

```json
{
  "mcpServers": {
    "nylas": {
      "command": "/path/to/nylas",
      "args": ["mcp", "serve"]
    }
  }
}
```

### Claude Code Permissions

For Claude Code, also add to `~/.claude/settings.json`:

```json
{
  "permissions": {
    "allow": ["mcp__nylas__*"]
  }
}
```

---

## Troubleshooting

### "Permission denied" errors

Run `nylas mcp install --assistant claude-code` to auto-configure permissions.

### Times showing in wrong timezone

Restart your AI assistant after installing MCP. The timezone is detected at startup.

### "No authenticated grants found"

Run `nylas auth login` to authenticate first.

### MCP server not responding

1. Check authentication: `nylas auth list`
2. Test server manually: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | nylas mcp serve`
3. Check binary path: `which nylas`

---

## Architecture

```
AI Assistant (Claude/Cursor/etc.)
        |
        | STDIO (JSON-RPC)
        v
+-------------------------------+
|    Nylas CLI Native MCP       |
|  (nylas mcp serve)            |
|                               |
|  - Calls Nylas API v3 directly|
|  - Injects credentials/grant  |
|  - Detects timezone           |
|  - Serves MCP tools locally   |
+-------------------------------+
        |
| HTTPS
        v
+-------------------------------+
|   Nylas API v3                |
| api.us.nylas.com / api.eu.nylas.com |
+-------------------------------+
        |
        | Provider APIs
        v
+-------------------------------+
|   Email/Calendar Providers    |
|   (Google, Microsoft, etc.)   |
+-------------------------------+
```

---

## See Also

- [Nylas MCP Documentation](https://developer.nylas.com/docs/dev-guide/mcp/)
- [Model Context Protocol Spec](https://modelcontextprotocol.io/)
- [AI Features](ai.md)
