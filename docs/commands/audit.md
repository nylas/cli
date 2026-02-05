# Audit Logging Guide

Complete guide to using Nylas CLI's audit logging for compliance, debugging, and tracking command execution across users and AI agents.

> **Key Feature:** Audit logging captures who ran which commands, from what source (terminal, Claude Code, GitHub Actions, etc.), with full Nylas API traceability via request IDs.

---

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [Initialize Audit Logging](#initialize-audit-logging)
  - [Enable/Disable Logging](#enabledisable-logging)
  - [View Audit Logs](#view-audit-logs)
  - [View Summary Statistics](#view-summary-statistics)
  - [Export Logs](#export-logs)
  - [Configure Settings](#configure-settings)
  - [Clear Logs](#clear-logs)
- [Invoker Identity Detection](#invoker-identity-detection)
- [Filtering and Searching](#filtering-and-searching)
- [Configuration Options](#configuration-options)
- [Storage and Retention](#storage-and-retention)
- [Use Cases](#use-cases)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)

---

## Overview

Audit logging records every CLI command execution with rich metadata for:

- **Compliance** - Track who accessed what data and when
- **Debugging** - Trace issues back to specific commands and API calls
- **Security** - Monitor for unauthorized access patterns
- **AI Agent Tracking** - Know when commands were run by Claude Code, GitHub Copilot, or other AI tools

### What Gets Logged

| Field | Description | Example |
|-------|-------------|---------|
| `timestamp` | When the command was executed | `2026-02-05 10:30:00` |
| `command` | The CLI command run | `email list` |
| `args` | Sanitized command arguments | `--limit 10` |
| `grant_id` | The Nylas grant (account) used | `abc123...` |
| `invoker` | Username who ran the command | `alice`, `dependabot[bot]` |
| `invoker_source` | Where the command originated | `claude-code`, `terminal` |
| `status` | Success or error | `success` |
| `duration` | How long it took | `190ms` |
| `request_id` | Nylas API request ID for tracing | `req_abc123` |
| `http_status` | HTTP response code | `200` |

### Sensitive Data Protection

Arguments containing sensitive data are automatically redacted:
- API keys, tokens, passwords
- Email body and subject content
- Long base64 strings (likely tokens)
- Values for `--api-key`, `--password`, `--token`, `--secret`, etc.

---

## Quick Start

### 1. Initialize Audit Logging

```bash
# Interactive setup (recommended for first time)
nylas audit init

# Non-interactive with immediate enable
nylas audit init --enable

# Custom configuration
nylas audit init --path /custom/path --retention 30 --enable
```

### 2. Enable Logging

```bash
nylas audit logs enable
```

### 3. View Recent Commands

```bash
# Show last 20 commands
nylas audit logs show

# Filter by user
nylas audit logs show --invoker alice

# Filter by source (AI agents, CI/CD, etc.)
nylas audit logs show --source claude-code
```

---

## Commands

### Initialize Audit Logging

Set up audit logging with storage location, retention, and options.

#### Usage

```bash
nylas audit init                    # Interactive setup
nylas audit init --enable           # Non-interactive with defaults, enable immediately
nylas audit init [flags]            # Custom configuration
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--path` | `~/.config/nylas/audit` | Custom log directory |
| `--retention` | `90` | Log retention period in days |
| `--max-size` | `100` | Maximum storage size in MB |
| `--format` | `jsonl` | Log format: `jsonl` or `json` |
| `--enable` | `false` | Enable logging immediately |
| `--no-prompt` | `false` | Skip interactive prompts |

#### Examples

```bash
# Interactive setup with prompts
nylas audit init

# Quick setup with defaults
nylas audit init --enable

# Custom retention and path
nylas audit init --path /var/log/nylas --retention 365 --enable

# CI/CD setup (no prompts)
nylas audit init --no-prompt --enable
```

---

### Enable/Disable Logging

Control whether commands are recorded.

```bash
# Enable audit logging
nylas audit logs enable

# Disable (preserves existing logs)
nylas audit logs disable

# Check current status
nylas audit logs status
```

The `status` command shows:
- Enabled/disabled state
- Configuration settings
- Storage statistics (size, file count, oldest entry)

---

### View Audit Logs

Display recent audit entries with filtering options.

#### Usage

```bash
nylas audit logs show [flags]
```

#### Flags

| Flag | Description |
|------|-------------|
| `-n, --limit` | Number of entries (default: 20) |
| `--since` | Show entries after date (YYYY-MM-DD) |
| `--until` | Show entries before date (YYYY-MM-DD) |
| `--command` | Filter by command prefix |
| `--status` | Filter by status (success/error) |
| `--grant` | Filter by grant ID |
| `--request-id` | Filter by Nylas request ID |
| `--invoker` | Filter by username |
| `--source` | Filter by source platform |

#### Examples

```bash
# Show last 50 entries
nylas audit logs show --limit 50

# Filter by command
nylas audit logs show --command email

# Filter by date range
nylas audit logs show --since 2026-01-01 --until 2026-01-31

# Find commands by a specific user
nylas audit logs show --invoker alice

# Find commands from AI agents
nylas audit logs show --source claude-code

# Find by Nylas request ID (detailed view)
nylas audit logs show --request-id req_abc123

# Filter errors only
nylas audit logs show --status error
```

#### Output

**Table View (default):**
```
TIMESTAMP            COMMAND       GRANT        INVOKER      SOURCE        STATUS   DURATION
2026-02-05 10:30:00  email list    abc123...    alice        claude-code   success  190ms
2026-02-05 10:28:00  auth status   -            alice        terminal      success  50ms
2026-02-05 10:25:00  email send    abc123...    jenkins-svc  github-act... success  1.2s
```

**Detailed View (when filtering by request-id):**
```
Entry Details

  ID:           a1b2c3d4
  Timestamp:    2026-02-05 10:30:00
  Command:      email list
  Arguments:    --limit 10
  Account:      alice@example.com
  Status:       success
  Duration:     190ms

  Invoker Details:
    User:        alice
    Source:      claude-code

  API Details:
    Request ID:  req_abc123
    HTTP Status: 200
```

---

### View Summary Statistics

Display aggregate statistics for audit logs.

#### Usage

```bash
nylas audit logs summary [--days N]
```

#### Examples

```bash
# Summary for last 7 days (default)
nylas audit logs summary

# Summary for last 30 days
nylas audit logs summary --days 30
```

#### Output

```
Audit Log Summary (Last 7 days)

Total Commands:  156
  ✓ Success:     152 (97%)
  ✗ Errors:      4 (3%)

Most Used:
  email list           45
  calendar events list 32
  email send           28
  auth status          15
  contacts list        12

Accounts:
  alice@example.com    89
  bob@company.com      67

Invoker Breakdown:
  alice (terminal)     78
  alice (claude-code)  45
  jenkins (github-actions) 33

API Statistics:
  Total API calls:  142
  Avg response time: 245ms
  Error rate:       2.1%
```

---

### Export Logs

Export audit logs to JSON or CSV files.

#### Usage

```bash
nylas audit export [flags]
```

#### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output` | stdout | Output file path |
| `--format` | auto | Output format: `json` or `csv` |
| `--since` | - | Export entries after date |
| `--until` | - | Export entries before date |
| `-n, --limit` | 10000 | Maximum entries to export |

#### Examples

```bash
# Export to JSON file
nylas audit export --output audit.json

# Export to CSV
nylas audit export --output audit.csv --format csv

# Export with date filter
nylas audit export --since 2026-01-01 --until 2026-01-31 --output january.json

# Export to stdout (for piping)
nylas audit export --format json | jq '.[] | select(.status=="error")'
```

#### CSV Columns

The CSV export includes these columns:
`id`, `timestamp`, `command`, `args`, `grant_id`, `grant_email`, `invoker`, `invoker_source`, `status`, `duration_ms`, `error`, `request_id`, `http_status`

---

### Configure Settings

View and modify audit configuration.

#### View Configuration

```bash
nylas audit config show
```

#### Set Configuration Values

```bash
nylas audit config set <key> <value>
```

**Available Keys:**

| Key | Type | Description |
|-----|------|-------------|
| `retention_days` | integer | Days to keep logs |
| `max_size_mb` | integer | Maximum storage in MB |
| `rotate_daily` | boolean | Create new file each day |
| `compress_old` | boolean | Compress files older than 7 days |
| `log_request_id` | boolean | Log Nylas request IDs |
| `log_api_details` | boolean | Log API endpoint and status |

#### Examples

```bash
# Set retention to 30 days
nylas audit config set retention_days 30

# Enable compression
nylas audit config set compress_old true

# Disable request ID logging
nylas audit config set log_request_id false
```

---

### Clear Logs

Remove all audit log files (configuration is preserved).

```bash
# With confirmation prompt
nylas audit logs clear

# Skip confirmation
nylas audit logs clear --force
```

---

## Invoker Identity Detection

Audit logging automatically detects who or what ran each command.

### Detection Priority

1. **Claude Code** - Detected via `CLAUDE_PROJECT_DIR` or `CLAUDE_CODE_*` environment variables
2. **GitHub Copilot** - Detected via `COPILOT_MODEL` or `GH_COPILOT` environment variables
3. **Custom Override** - Set `NYLAS_INVOKER_SOURCE=<tool>` for any tool
4. **SSH** - Detected via `SSH_CLIENT` environment variable
5. **Script/Automation** - Non-interactive terminal (stdin not a TTY)
6. **Terminal** - Interactive terminal session (default)

### Source Values

| Source | Description | How Detected |
|--------|-------------|--------------|
| `claude-code` | Anthropic's Claude Code | `CLAUDE_PROJECT_DIR` or `CLAUDE_CODE_*` env vars |
| `github-copilot` | GitHub Copilot CLI | `COPILOT_MODEL` or `GH_COPILOT` env vars |
| `ssh` | Remote SSH session | `SSH_CLIENT` env var |
| `script` | Non-interactive script | stdin is not a TTY |
| `terminal` | Interactive terminal | Default for TTY sessions |
| `<custom>` | User-defined | `NYLAS_INVOKER_SOURCE` env var |

### Manual Override

For AI tools or automation not automatically detected, set the source manually:

```bash
# In your script or CI/CD pipeline
export NYLAS_INVOKER_SOURCE=my-automation
nylas email list
```

### Username Detection

The `invoker` field captures the username via:
1. `SUDO_USER` environment variable (if running via sudo)
2. `os/user.Current()` - Current system user

---

## Filtering and Searching

### Filter by User

```bash
# Find all commands by alice
nylas audit logs show --invoker alice

# Find commands by CI service account
nylas audit logs show --invoker jenkins-svc
```

### Filter by Source Platform

```bash
# Commands from Claude Code
nylas audit logs show --source claude-code

# Commands from GitHub Actions
nylas audit logs show --source github-actions

# Commands from interactive terminal
nylas audit logs show --source terminal
```

### Filter by Command

```bash
# All email commands
nylas audit logs show --command email

# Specific subcommand
nylas audit logs show --command "email send"
```

### Filter by Status

```bash
# Only errors
nylas audit logs show --status error

# Only successes
nylas audit logs show --status success
```

### Filter by Date

```bash
# Last week
nylas audit logs show --since 2026-01-29

# Specific range
nylas audit logs show --since 2026-01-01 --until 2026-01-31
```

### Find by Request ID

When you have a Nylas request ID from an error or support ticket:

```bash
nylas audit logs show --request-id req_abc123
```

This shows detailed information about that specific command execution.

---

## Configuration Options

### Default Configuration

```yaml
enabled: false
path: ~/.config/nylas/audit
retention_days: 90
max_size_mb: 100
format: jsonl
rotate_daily: true
compress_old: false
log_request_id: true
log_api_details: true
```

### Configuration File Location

Configuration is stored at `~/.config/nylas/audit/config.json`

---

## Storage and Retention

### Log Format

Logs are stored in JSONL format (one JSON object per line) for efficient appending and streaming.

```json
{"id":"abc123","timestamp":"2026-02-05T10:30:00Z","command":"email list","invoker":"alice","invoker_source":"claude-code","status":"success","duration":190000000}
```

### Rotation

- **Daily rotation** (default): New log file created each day
- **File naming**: `audit-2026-02-05.jsonl`

### Retention

- Logs older than `retention_days` are automatically deleted
- Storage limited to `max_size_mb` (oldest files deleted first)

### Compression

When `compress_old` is enabled:
- Files older than 7 days are gzipped
- Reduces storage by ~90%

---

## Use Cases

### 1. Compliance Auditing

Track all data access for compliance requirements:

```bash
# Export last quarter's access log
nylas audit export \
  --since 2026-01-01 \
  --until 2026-03-31 \
  --output Q1-audit.csv \
  --format csv
```

### 2. Debugging API Issues

Trace an issue back to the specific command and API call:

```bash
# Find the command that caused an error
nylas audit logs show --request-id req_abc123
```

### 3. Monitoring AI Agent Activity

Track what AI assistants are doing:

```bash
# All Claude Code activity
nylas audit logs show --source claude-code --limit 100

# Summary of AI vs human usage
nylas audit logs summary --days 30
```

### 4. Security Monitoring

Detect unusual access patterns:

```bash
# Failed commands (potential security issues)
nylas audit logs show --status error --limit 50

# Commands from unexpected sources
nylas audit logs show --source ssh
```

### 5. CI/CD Pipeline Debugging

Track commands run in automated pipelines:

```bash
# GitHub Actions activity
nylas audit logs show --source github-actions

# Jenkins activity
nylas audit logs show --invoker jenkins-svc
```

---

## Troubleshooting

### Audit logging not initialized

```
Error: audit logging not initialized. Run: nylas audit init
```

**Solution:** Initialize audit logging first:
```bash
nylas audit init --enable
```

### Logs not being recorded

**Check status:**
```bash
nylas audit logs status
```

**Ensure logging is enabled:**
```bash
nylas audit logs enable
```

### Missing invoker information

If `invoker` or `invoker_source` is empty:
- Ensure the detection environment variables are set
- Use `NYLAS_INVOKER_SOURCE` for manual override

### Storage full

```bash
# Check current storage
nylas audit config show

# Reduce retention
nylas audit config set retention_days 30

# Clear old logs
nylas audit logs clear
```

---

## FAQ

### Q: Does audit logging affect performance?

**A:** Minimal impact. Logging is asynchronous and adds ~1-2ms overhead per command.

### Q: Are my credentials logged?

**A:** No. Sensitive arguments (API keys, passwords, tokens, email content) are automatically redacted as `[REDACTED]`.

### Q: Can I disable logging for specific commands?

**A:** Currently, no per-command control. You can disable logging entirely with `nylas audit logs disable`.

### Q: How do I know if a command was run by an AI?

**A:** Check the `invoker_source` field. AI tools like Claude Code are automatically detected and recorded as `claude-code`.

### Q: What if my AI tool isn't detected?

**A:** Set `NYLAS_INVOKER_SOURCE=<tool-name>` in your environment before running commands.

### Q: Can I export logs for SIEM integration?

**A:** Yes. Use `nylas audit export --format json` and pipe to your SIEM ingestion endpoint.

---

## Related Documentation

- **[Command Reference](../COMMANDS.md)** - Quick command reference
- **[Security Overview](../security/overview.md)** - Security best practices
- **[MCP Integration](mcp.md)** - AI assistant integration

---

**Last Updated:** February 5, 2026
**Version:** 1.0
