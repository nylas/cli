# MCP (Model Context Protocol)

Enable AI assistants to interact with your Nylas email and calendar.

> **Quick Reference:** [COMMANDS.md](../COMMANDS.md#mcp-model-context-protocol) | **Related:** [AI Features](ai.md)

---

## Overview

The Nylas CLI includes an MCP proxy server that enables AI assistants like Claude Desktop, Cursor, and Windsurf to access your email and calendar. The proxy:

1. Forwards requests to the official Nylas MCP server (region-specific)
2. Injects your authenticated credentials automatically
3. Detects your system timezone for consistent time display
4. Handles local grant lookups without requiring email input

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
| `get_grant` | Get grant information (email optional) |
| `current_time` | Get current time with timezone |
| `epoch_to_datetime` | Convert Unix timestamp to datetime |
| `datetime_to_epoch` | Convert datetime to Unix timestamp |

---

## Features

### Automatic Timezone Detection

The MCP proxy detects your system timezone and injects it into server instructions. This ensures AI assistants display all timestamps consistently in your local timezone.

**Before:** Mixed timezones (emails in UTC, calendar in Pacific)
**After:** All times in your local timezone (e.g., EST)

### Auto-configured Permissions (Claude Code)

When installing for Claude Code, the CLI automatically adds `mcp__nylas__*` to `~/.claude/settings.json`, granting permission for all Nylas MCP tools without interactive prompts.

### Default Grant Injection

The proxy automatically injects your authenticated grant ID into tool calls, so AI assistants don't need to ask for your email address.

### Local Grant Lookup

The `get_grant` tool can be called without an email parameter. The proxy returns your default authenticated grant from local storage.

---

## Regional Endpoints

The Nylas MCP server operates in two regions. The CLI automatically selects the correct endpoint based on your configured region:

| Region | Endpoint |
|--------|----------|
| `us` (default) | `https://mcp.us.nylas.com` |
| `eu` | `https://mcp.eu.nylas.com` |

Configure your region in `~/.config/nylas/config.yaml`:

```yaml
region: eu  # or "us" (default)
```

The MCP proxy reads this setting and routes requests to the appropriate regional endpoint.

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
|     Nylas CLI MCP Proxy       |
|  (nylas mcp serve)            |
|                               |
|  - Reads region from config   |
|  - Injects credentials        |
|  - Detects timezone           |
|  - Handles local get_grant    |
|  - Modifies tool schemas      |
+-------------------------------+
        |
        | HTTPS (region-based)
        v
+-------------------------------+
|   Nylas MCP Server            |
|   mcp.us.nylas.com (US)       |
|   mcp.eu.nylas.com (EU)       |
+-------------------------------+
        |
        | Nylas API v3
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
