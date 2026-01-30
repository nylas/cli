# Comprehensive FAQ

Detailed answers to 50+ frequently asked questions.

---

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [Email](#email)
- [Calendar](#calendar)
- [Timezone](#timezone)
- [TUI](#tui)
- [Webhooks](#webhooks)
- [Technical](#technical)

---

## Getting Started

### Q: How do I install the Nylas CLI?

**A: Multiple installation options:**

```bash
# Homebrew (recommended for macOS/Linux)
brew install nylas/nylas-cli/nylas

# Go install
go install github.com/nylas/cli/cmd/nylas@latest

# Download binary from releases
# https://github.com/nylas/cli/releases
```

---

### Q: What do I need to get started?

**A: You need:**
1. **Nylas account** - Sign up at https://dashboardv3.nylas.com
2. **API Key** - Create app in dashboard, get API key
3. **Grant ID** - Connect your email account, get grant ID

**For timezone utilities only:** No credentials needed! Works offline.

---

### Q: How do I get my API credentials?

**A: Step-by-step:**

1. **Go to:** https://dashboardv3.nylas.com
2. **Create app** (or use existing)
3. **Get API Key:**
   - Apps → Your App → API Keys
   - Copy key (starts with `nyk_`)
4. **Get Grant ID:**
   - Grants → Create Grant → Connect account
   - Copy grant ID (starts with `grant_`)
5. **Configure CLI:**
```bash
nylas auth config
# Paste API key and Grant ID
```

---

### Q: Can I use this without an API key?

**A: Yes, for timezone utilities!**

Timezone commands work completely **offline**:
```bash
nylas timezone convert --from PST --to EST
nylas timezone dst --zone America/New_York
nylas timezone find-meeting --zones "NY,London,Tokyo"
```

**For email/calendar:** API key required.

---

### Q: How do I configure multiple email accounts?

**A: Use different grants:**

```bash
# Get grant IDs for each account
nylas admin grants list

# Use specific grant per command
nylas email list grant_account1
nylas email list grant_account2

# Switch default grant
nylas auth config  # Update default grant ID
```

---

## Authentication

### Q: Where are my credentials stored?

**A: Securely in system keyring:**
- **macOS:** Keychain
- **Linux:** Secret Service (GNOME Keyring, KWallet)
- **Windows:** Windows Credential Manager

**Fallback:** `~/.config/nylas/config.yaml` if keyring unavailable

---

### Q: How do I reset my credentials?

**A: Reconfigure:**

```bash
# Option 1: Reconfigure
nylas auth config

# Option 2: Remove and reconfigure
rm ~/.config/nylas/config.yaml
nylas auth config

# Option 3: Clear keyring (macOS)
security delete-generic-password -s nylas-api-key
nylas auth config
```

---

### Q: What's the difference between API key and OAuth?

**A:**
- **API Key + Grant ID:** Simpler, recommended for CLI use
- **OAuth:** Browser-based login, more steps, better for apps

**For CLI, use API Key method:**
```bash
nylas auth config
```

---

### Q: "401 Unauthorized" - What does this mean?

**A: Invalid credentials.**

**Fix:**
1. Verify API key in dashboard
2. Check grant ID exists
3. Reconfigure:
```bash
nylas auth config
```

See: [Auth Troubleshooting](auth.md)

---

## Email

### Q: Why don't I see any emails?

**A: Common causes:**

1. **Wrong grant ID** - Check: `nylas auth status`
2. **No email scope** - Grant needs `email.read` permission
3. **Provider not connected** - Check Nylas Dashboard
4. **Filters too restrictive** - Try: `nylas email list --limit 10`

See: [Email Troubleshooting](email.md#no-emails-showing)

---

### Q: How do I send an email?

**A: Basic send:**

```bash
nylas email send \
  --to "recipient@example.com" \
  --subject "Hello" \
  --body "Your message here"

# With CC/BCC
nylas email send \
  --to "to@example.com" \
  --cc "cc@example.com" \
  --bcc "bcc@example.com" \
  --subject "Meeting" \
  --body "Details..."

# With attachments
nylas email send \
  --to "user@example.com" \
  --subject "Document" \
  --body "See attached" \
  --attach "/path/to/file.pdf"
```

See: [Email Commands](../commands/email.md)

---

### Q: Can I schedule emails to send later?

**A: Yes!**

```bash
# Send in 2 hours
nylas email send --to "..." --schedule 2h

# Send tomorrow at 9am
nylas email send --to "..." --schedule "tomorrow 9am"

# Send at specific date/time
nylas email send --to "..." --schedule "2024-12-25 10:00"
```

**Note:** Not all email providers support scheduled sending.

---

### Q: How do I search emails?

**A: Use filters:**

```bash
# By sender
nylas email list --from "sender@example.com"

# By subject
nylas email list --subject "Meeting"

# Unread only
nylas email list --unread

# Starred only
nylas email list --starred

# Combine filters
nylas email list --from "boss@company.com" --unread

# Increase results
nylas email list --limit 100
```

---

### Q: Can I use the CLI in scripts?

**A: Yes! Perfect for automation:**

```bash
#!/bin/bash

# Check for urgent emails
urgent=$(nylas email list --unread | grep -i "urgent" | wc -l)

if [ $urgent -gt 0 ]; then
  echo "You have $urgent urgent emails!"
fi

# Send daily report
nylas email send \
  --to "team@example.com" \
  --subject "Daily Report $(date +%Y-%m-%d)" \
  --body "Report content here"
```

---

## Calendar

### Q: How do I view my calendar?

**A: List events:**

```bash
# Default: next 7 days
nylas calendar events list

# Next 14 days
nylas calendar events list --days 14

# In different timezone
nylas calendar events list --timezone America/Los_Angeles

# Show timezone info
nylas calendar events list --show-tz
```

---

### Q: How do I create a calendar event?

**A: Basic create:**

```bash
# Simple event
nylas calendar events create \
  --title "Meeting" \
  --start "2024-12-25 14:00" \
  --end "2024-12-25 15:00"

# All-day event
nylas calendar events create \
  --title "Vacation" \
  --start "2024-12-25" \
  --all-day

# With participants
nylas calendar events create \
  --title "Team Sync" \
  --start "2024-12-25 10:00" \
  --participant "alice@example.com" \
  --participant "bob@example.com"
```

---

### Q: Can I see events in different timezones?

**A: Yes! Multiple ways:**

```bash
# List events in specific timezone
nylas calendar events list --timezone Europe/London

# Show timezone abbreviations
nylas calendar events list --show-tz

# View specific event in multiple timezones
nylas calendar events show <event-id> --timezone Asia/Tokyo
nylas calendar events show <event-id> --timezone America/New_York
```

See: [Timezone Guide](../commands/timezone.md)

---

### Q: How do I find the best meeting time across timezones?

**A: Use the meeting finder:**

```bash
# Basic usage
nylas timezone find-meeting \
  --zones "America/New_York,Europe/London,Asia/Tokyo"

# With preferences
nylas timezone find-meeting \
  --zones "America/New_York,Europe/London,Asia/Tokyo" \
  --duration 60 \
  --earliest 9 \
  --latest 17

# Shows top-scored meeting times
# Considers working hours, time quality, cultural factors
```

---

### Q: What are DST warnings?

**A: Automatic Daylight Saving Time alerts:**

When viewing/creating events near DST transitions, CLI shows:

```
⚠️ Daylight Saving Time begins in 2 days (clocks spring forward 1 hour)
⚠️ This event is scheduled during DST transition
⛔ This time will not exist due to Daylight Saving Time
```

**Helps avoid:** Scheduling meetings at times that don't exist!

---

## Timezone

### Q: Do timezone utilities require an API key?

**A: No! Completely offline.**

```bash
# Works without any credentials
nylas timezone convert --from PST --to EST
nylas timezone dst --zone America/New_York
nylas timezone list
```

---

### Q: How do I convert times between timezones?

**A: Use convert command:**

```bash
# Basic conversion (uses current time)
nylas timezone convert \
  --from America/New_York \
  --to Europe/London

# Specific time
nylas timezone convert \
  --from America/New_York \
  --to Europe/London \
  --time "2025-03-15 14:00"

# Multiple target zones
nylas timezone convert \
  --from America/Los_Angeles \
  --to "Europe/London,Asia/Tokyo"
```

---

### Q: How do I check Daylight Saving Time transitions?

**A: Use DST command:**

```bash
# Check DST for specific year
nylas timezone dst --zone America/New_York --year 2025

# Shows:
# - Spring forward date/time
# - Fall back date/time
# - Time changes
```

---

### Q: Why can't I use "EST" or "PST"?

**A: Use full IANA timezone names:**

```bash
# ❌ Wrong - abbreviations
nylas timezone info EST
nylas timezone info PST

# ✅ Correct - full names
nylas timezone info America/New_York  # For EST/EDT
nylas timezone info America/Los_Angeles  # For PST/PDT

# Find timezone
nylas timezone list --filter America
```

**Why?** Abbreviations are ambiguous (EST = Eastern Standard Time, but when does DST apply?). IANA names handle DST automatically.

---

### Q: What timezones are available?

**A: All IANA timezones (600+):**

```bash
# List all
nylas timezone list

# Filter by region
nylas timezone list --filter America
nylas timezone list --filter Europe
nylas timezone list --filter Asia

# Search for specific city
nylas timezone list --filter Tokyo
nylas timezone list --filter London
```

---

## TUI

### Q: What is the TUI?

**A: Terminal User Interface - Interactive visual mode:**

```bash
# Launch TUI
nylas tui

# Demo mode (no credentials)
nylas tui --demo

# With theme
nylas tui --theme amber
```

**Features:**
- Email inbox view
- Calendar view
- Contacts management
- Vim-style keyboard shortcuts
- Multiple color themes

---

### Q: What keyboard shortcuts are available?

**A: Vim-style navigation:**

```
j/k         Navigate up/down
gg/G        First/last item
Enter       Select/open
dd          Delete
q or :q     Quit
/           Search
?           Help
Tab         Switch panels
```

See: [TUI Documentation](../commands/tui.md)

---

### Q: Can I customize the TUI theme?

**A: Yes! 9 themes available:**

```bash
# Available themes:
nylas tui --theme k9s         # Default
nylas tui --theme amber       # Retro amber CRT
nylas tui --theme green       # Classic green terminal
nylas tui --theme apple2      # Apple II style
nylas tui --theme vintage     # Warm vintage
nylas tui --theme ibm         # IBM blue screen
nylas tui --theme futuristic  # Neon blue
nylas tui --theme matrix      # Matrix green
nylas tui --theme norton      # Norton Commander
```

---

### Q: Can I try TUI without credentials?

**A: Yes! Use demo mode:**

```bash
nylas tui --demo

# Shows sample data
# No API key required
# Perfect for testing themes and navigation
```

---

## Webhooks

### Q: How do I set up webhooks?

**A: Create webhook:**

```bash
# Basic webhook
nylas webhook create \
  --url "https://myapp.com/webhook" \
  --triggers "message.created,message.updated"

# Specific events
nylas webhook create \
  --url "https://myapp.com/webhook" \
  --triggers "calendar.created,calendar.updated"

# List webhooks
nylas webhook list

# Test webhook
nylas webhook test <webhook-id>
```

See: [Webhook Guide](../commands/webhooks.md)

---

### Q: What webhook events are available?

**A: Many event types:**

- `message.created` - New email received
- `message.updated` - Email modified
- `calendar.created` - New calendar event
- `calendar.updated` - Event modified
- `contact.created` - New contact
- `contact.updated` - Contact modified

See: [Webhook Events](https://developer.nylas.com/docs/api/v3/webhooks/)

---

### Q: How do I test webhooks locally?

**A: Use ngrok or similar:**

```bash
# 1. Start ngrok tunnel
ngrok http 8080

# 2. Use ngrok URL for webhook
nylas webhook create \
  --url "https://abc123.ngrok.io/webhook" \
  --triggers "message.created"

# 3. Test
nylas webhook test <webhook-id>
```

---

## Technical

### Q: What API version does this CLI use?

**A: Nylas v3 API only.**

This CLI **only supports v3**. Not compatible with v1 or v2.

**API endpoint:** `https://api.nylas.com/v3/`

---

### Q: Where is configuration stored?

**A: Two locations:**

1. **Credentials:** System keyring (secure)
2. **Config file:** `~/.config/nylas/config.yaml`

**Config file format:**
```yaml
api_key: nyk_xxx...
grant_id: grant_xxx...
api_url: https://api.nylas.com
```

---

### Q: How do I enable debug mode?

**A: Use debug flag:**

```bash
# Run command with debug output
nylas --debug email list

# Check logs (if available)
tail -f ~/.config/nylas/nylas.log
```

---

### Q: What are the rate limits?

**A: Depends on your plan:**

- **Free tier:** 100 requests/minute
- **Paid tier:** Higher (varies)

**Handle rate limits:**
- Add delays between requests
- Use webhooks instead of polling
- Implement exponential backoff

See: [API Troubleshooting](api.md#rate-limiting)

---

### Q: Can I use environment variables?

**A: Yes:**

```bash
# Set credentials via environment
export NYLAS_API_KEY="nyk_xxx..."
export NYLAS_GRANT_ID="grant_xxx..."

# Commands will use these automatically
nylas email list
```

**Precedence:**
1. Command-line arguments
2. Environment variables
3. Config file

---

### Q: Is this CLI open source?

**A: Yes!**

- **Repository:** https://github.com/nylas/cli
- **License:** MIT
- **Contributions:** Welcome!

See: [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

### Q: What languages/tools are used?

**A:**
- **Language:** Go 1.21+
- **Architecture:** Hexagonal (ports and adapters)
- **CLI framework:** Cobra
- **TUI framework:** tview
- **API:** Nylas v3

---

### Q: How do I report bugs or request features?

**A: GitHub Issues:**

1. **Search existing:** https://github.com/nylas/cli/issues
2. **Create new issue** if not found
3. **Include:**
   - CLI version: `nylas version`
   - Operating system
   - Command that failed
   - Error message
   - Steps to reproduce

---

### Q: How do I update the CLI?

**A: Depends on installation method:**

```bash
# Homebrew
brew upgrade nylas/nylas-cli/nylas

# Go install
go install github.com/nylas/cli/cmd/nylas@latest

# Manual download
# Download latest from releases page
# https://github.com/nylas/cli/releases
```

---

### Q: What's the difference between `email read` and `email show`?

**A: They're aliases - same command:**

```bash
nylas email read <message-id>
# Same as:
nylas email show <message-id>
```

---

### Q: Can I pipe CLI output to other commands?

**A: Yes! CLI is pipe-friendly:**

```bash
# Find urgent emails
nylas email list --unread | grep -i "urgent"

# Count unread emails
nylas email list --unread | grep -c "From:"

# Extract email IDs
nylas email list | grep "ID:" | awk '{print $2}'

# Save to file
nylas email list > emails.txt
```

---

### Q: How do I get help for a specific command?

**A: Use --help flag:**

```bash
# General help
nylas --help

# Command help
nylas email --help
nylas email send --help
nylas timezone convert --help

# Shows all options and examples
```

---

### Q: Does this work on Windows?

**A: Yes!**

Windows support includes:
- PowerShell and CMD
- Windows Credential Manager for secure storage
- Same commands as macOS/Linux

**Note:** Some shell-specific features may differ.

---

### Q: How do I contribute?

**A: We welcome contributions!**

1. **Fork repository**
2. **Create branch:** `git checkout -b feat/your-feature`
3. **Make changes**
4. **Run tests:** `make test`
5. **Submit PR**

See: [CONTRIBUTING.md](../../CONTRIBUTING.md)

---

## Still Have Questions?

- **Troubleshooting:** See guides in this folder
- **Commands:** [COMMANDS.md](../COMMANDS.md)
- **GitHub Issues:** https://github.com/nylas/cli/issues
