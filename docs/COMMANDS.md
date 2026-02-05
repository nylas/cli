# Nylas CLI Command Reference

Quick command reference. For detailed docs, see `docs/commands/<feature>.md`

> **Quick Links:** [README](../README.md) | [Development](DEVELOPMENT.md) | [Architecture](ARCHITECTURE.md)

---

## Global Flags

| Flag | Description | Example |
|------|-------------|---------|
| `--json` | Output as JSON | `nylas email list --json` |
| `--no-color` | Disable color output | `nylas email list --no-color` |
| `--verbose` / `-v` | Enable verbose output | `nylas -v email list` |
| `--config` | Custom config file path | `nylas --config ~/.nylas/alt.yaml email list` |
| `--help` / `-h` | Show help | `nylas email --help` |

**Common per-command flags:**
- `--limit N` - Limit results (most list commands)
- `--yes` / `-y` - Skip confirmations (delete/send commands)

---

## Shell Completion

Enable tab completion for faster command entry. The CLI automatically generates completion scripts for all major shells.

### Bash

```bash
# Generate and install completion script
nylas completion bash > /etc/bash_completion.d/nylas

# Or for user-only install:
mkdir -p ~/.local/share/bash-completion/completions
nylas completion bash > ~/.local/share/bash-completion/completions/nylas
```

### Zsh

```bash
# Generate and install completion script
nylas completion zsh > /usr/local/share/zsh/site-functions/_nylas

# Or add to your .zshrc:
echo "autoload -U compinit; compinit" >> ~/.zshrc
mkdir -p ~/.zsh/completion
nylas completion zsh > ~/.zsh/completion/_nylas
echo "fpath=(~/.zsh/completion $fpath)" >> ~/.zshrc
```

### Fish

```bash
# Generate and install completion script
nylas completion fish > ~/.config/fish/completions/nylas.fish
```

### PowerShell

```powershell
# Add to your PowerShell profile
nylas completion powershell | Out-String | Invoke-Expression

# Or save to profile permanently:
nylas completion powershell >> $PROFILE
```

**After installation, restart your shell or source your profile to activate completion.**

---

## Authentication

```bash
nylas auth config                # Configure API credentials
nylas auth login                 # Authenticate with provider
nylas auth list                  # List connected accounts
nylas auth show [grant-id]       # Show account details
nylas auth status                # Check authentication status
nylas auth whoami                # Show current user info
nylas auth switch <email>        # Switch active account
nylas auth logout                # Logout current account
nylas auth remove <grant-id>     # Remove account completely
nylas auth token                 # Display current API token
nylas auth scopes [grant-id]     # Show granted OAuth scopes
nylas auth providers             # List available providers
nylas auth migrate               # Migrate from v2 to v3
```

---

## Demo Mode (No Account Required)

Explore the CLI with sample data before connecting your accounts:

```bash
nylas demo email list            # Browse sample emails
nylas demo calendar list         # View sample events
nylas demo contacts list         # See sample contacts
nylas demo notetaker list        # Explore AI notetaker
nylas demo tui                   # Interactive demo UI
```

All demo commands mirror real CLI structure: `nylas demo <feature> <command>`

---

## Email

```bash
nylas email list [grant-id]                                    # List emails
nylas email read <message-id>                                  # Read email
nylas email read <message-id> --raw                            # Show raw body without HTML
nylas email read <message-id> --mime                           # Show raw RFC822/MIME format
nylas email read <message-id> --decrypt                        # Decrypt PGP/MIME encrypted email
nylas email read <message-id> --verify                         # Verify GPG signature
nylas email read <message-id> --decrypt --verify               # Decrypt and verify signature
nylas email send --to EMAIL --subject SUBJECT --body BODY      # Send email
nylas email send ... --sign                                    # Send GPG-signed email
nylas email send ... --encrypt                                 # Send GPG-encrypted email
nylas email send ... --sign --encrypt                          # Sign AND encrypt (recommended)
nylas email send --list-gpg-keys                               # List available GPG signing keys
nylas email search --query "QUERY"                             # Search emails
nylas email delete <message-id>                                # Delete email
nylas email mark read <message-id>                             # Mark as read
nylas email mark unread <message-id>                           # Mark as unread
nylas email mark starred <message-id>                          # Star a message
nylas email attachments list <message-id>                      # List attachments
nylas email attachments download <message-id> <attachment-id>  # Download attachment
nylas email metadata show <message-id>                         # Show message metadata
```

**Filters:** `--unread`, `--starred`, `--from`, `--to`, `--subject`, `--has-attachment`, `--metadata`

**GPG/PGP security:**
```bash
nylas email send --to EMAIL --subject S --body B --sign        # Sign with your GPG key
nylas email send --to EMAIL --subject S --body B --encrypt     # Encrypt with recipient's key
nylas email send --to EMAIL --subject S --body B --sign --encrypt  # Both (recommended)
nylas email read <message-id> --decrypt                        # Decrypt encrypted email
nylas email read <message-id> --decrypt --verify               # Decrypt + verify signature
```

**AI features:**
```bash
nylas email ai analyze                    # AI-powered inbox summary
nylas email ai analyze --limit 25         # Analyze more emails
nylas email ai analyze --unread           # Only unread emails
nylas email ai analyze --provider claude  # Use specific AI provider
nylas email smart-compose --prompt "..."  # AI-powered email generation
```

**Details:** `docs/commands/email.md`, `docs/commands/email-signing.md`, `docs/commands/encryption.md`, `docs/commands/ai.md`

---

## Email Templates

```bash
nylas email templates list                           # List all templates
nylas email templates create --name NAME --subject SUBJECT --body BODY
nylas email templates show <template-id>             # Show template details
nylas email templates update <template-id> [flags]   # Update template
nylas email templates delete <template-id>           # Delete template
nylas email templates use <template-id> --to EMAIL   # Send using template
```

**Variable syntax:** Use `{{variable}}` in subject/body for placeholders.

**Details:** `docs/commands/templates.md`

---

## Folders & Threads

```bash
# Folders
nylas email folders list                           # List folders
nylas email folders show <folder-id>               # Show folder details
nylas email folders create <name>                  # Create folder
nylas email folders rename <folder-id> <new-name>  # Rename folder
nylas email folders delete <folder-id>             # Delete folder

# Threads
nylas email threads list                           # List threads
nylas email threads show <thread-id>               # Show thread with all messages
nylas email threads search --query "QUERY"         # Search threads
nylas email threads mark <thread-id> --read        # Mark thread as read
nylas email threads delete <thread-id>             # Delete thread
```

---

## Drafts

```bash
nylas email drafts list                           # List drafts
nylas email drafts show <draft-id>                # Show draft details
nylas email drafts create --to EMAIL --subject S  # Create draft
nylas email drafts send <draft-id>                # Send draft
nylas email drafts delete <draft-id>              # Delete draft
```

**Flags:** `--to`, `--cc`, `--bcc`, `--subject`, `--body`, `--reply-to`, `--attach`

---

## Scheduled Emails

```bash
nylas email scheduled list                        # List scheduled emails
nylas email scheduled show <schedule-id>          # Show scheduled email
nylas email scheduled cancel <schedule-id>        # Cancel scheduled email
```

---

## Calendar

```bash
nylas calendar list                                              # List calendars
nylas calendar events list [--days N] [--timezone ZONE]          # List events
nylas calendar events show <event-id>                            # Show event details
nylas calendar events create --title T --start TIME --end TIME   # Create event
nylas calendar events update <event-id> --title "New Title"      # Update event
nylas calendar events delete <event-id>                          # Delete event
nylas calendar events rsvp <event-id> --status yes               # RSVP to event
nylas calendar availability check                                # Check availability
nylas calendar recurring list                                    # List recurring events
nylas calendar virtual list                                      # List virtual meetings
nylas calendar focus-time list                                   # List focus time blocks
```

**Timezone features:**
```bash
nylas calendar events list --timezone America/Los_Angeles --show-tz
```

**AI scheduling:**
```bash
nylas calendar schedule ai "meeting with John next Tuesday afternoon"
nylas calendar analyze                                           # AI-powered analytics
nylas calendar find-time --participants email1,email2 --duration 1h
nylas calendar ai conflicts --days 7                             # Detect conflicts
nylas calendar ai reschedule <event-id> --reason "Conflict"      # AI reschedule
```

**Key features:** DST detection, working hours validation, break protection, AI scheduling

**Details:** `docs/commands/calendar.md`, `docs/commands/timezone.md`, `docs/commands/ai.md`

---

## Contacts

```bash
nylas contacts list                                   # List contacts
nylas contacts show <contact-id>                      # Show contact details
nylas contacts create --name "NAME" --email "EMAIL"   # Create contact
nylas contacts update <contact-id> --name "NEW NAME"  # Update contact
nylas contacts delete <contact-id>                    # Delete contact
nylas contacts search --query "QUERY"                 # Search contacts
nylas contacts sync                                   # Sync contacts
```

**Contact groups:**
```bash
nylas contacts groups list                            # List contact groups
nylas contacts groups show <group-id>                 # Show group details
nylas contacts groups create <name>                   # Create group
nylas contacts groups update <group-id> --name "NEW"  # Update group
nylas contacts groups delete <group-id>               # Delete group
```

**Contact photos:**
```bash
nylas contacts photo download <contact-id>            # Download contact photo
nylas contacts photo info                             # Photo info
```

**Details:** `docs/commands/contacts.md`

---

## Webhooks

```bash
nylas webhook list                                    # List webhooks
nylas webhook show <webhook-id>                       # Show webhook details
nylas webhook create --url URL --triggers "event.created,event.updated"
nylas webhook update <webhook-id> --url NEW_URL       # Update webhook
nylas webhook delete <webhook-id>                     # Delete webhook
nylas webhook triggers                                # List available triggers
```

**Testing & development:**
```bash
nylas webhook test send <webhook-url>                 # Send test payload
nylas webhook test payload [trigger-type]             # Generate test payload
nylas webhook server                                  # Start local webhook server
nylas webhook server --port 8080 --tunnel cloudflared # With public tunnel
```

**Details:** `docs/commands/webhooks.md`

---

## Inbound Email

Receive emails at managed addresses without OAuth or third-party mailbox connections.

```bash
nylas inbound list                              # List inbound inboxes
nylas inbound create <email-prefix>             # Create inbox (e.g., support@yourapp.nylas.email)
nylas inbound show <inbox-id>                   # Show inbox details
nylas inbound delete <inbox-id>                 # Delete inbox
nylas inbound messages <inbox-id>               # List messages in inbox
nylas inbound monitor <inbox-id>                # Real-time message monitoring
```

**Real-time monitoring with tunnel:**
```bash
nylas inbound monitor <inbox-id> --tunnel cloudflared
```

**Details:** `docs/commands/inbound.md`

---

## Timezone Utilities

All timezone commands work **100% offline** - no API key required.

```bash
nylas timezone list                                   # List all timezones
nylas timezone list --filter America                  # Filter by region
nylas timezone info <zone>                            # Get timezone info
nylas timezone convert --from PST --to EST            # Convert current time
nylas timezone convert --from UTC --to EST --time "2025-01-01T12:00:00Z"
nylas timezone dst --zone America/New_York            # Check DST transitions
nylas timezone dst --zone PST --year 2026             # Check DST for year
nylas timezone find-meeting --zones "NYC,London,Tokyo"  # Find meeting times
```

**Supported abbreviations:** PST, EST, CST, MST, GMT, IST, JST, AEST

**Details:** `docs/commands/timezone.md`

---

## TUI (Terminal UI)

```bash
nylas tui                        # Launch interactive UI
```

**Navigation:** `↑/↓` navigate, `Enter` select, `q` quit, `?` help

**Details:** `docs/commands/tui.md`

---

## Web UI (`nylas ui`)

Launch a lightweight web interface for CLI configuration and command execution:

```bash
nylas ui                         # Start on default port (7363)
nylas ui --port 8080             # Custom port
nylas ui --no-browser            # Don't auto-open browser
```

**Features:**
- Configure API credentials visually
- View and switch between authenticated accounts
- Execute email, calendar, and auth commands
- ID caching with autocomplete suggestions
- Command output with copy functionality

**Security:**
- Runs on localhost only (not accessible externally)
- Command whitelist prevents arbitrary execution
- Shell injection protection

**URL:** `http://localhost:7363` (default)

> **Note:** For a full email client experience, use `nylas air` instead (see below).

---

## Air (`nylas air`) - Modern Email Client

Launch **Nylas Air** - a full-featured, keyboard-driven email client in your browser:

> **vs `nylas ui`:** Air is a complete email/calendar client. UI is for CLI configuration only.

```bash
nylas air                        # Start on default port (7365)
nylas air --port 8080            # Custom port
nylas air --no-browser           # Don't auto-open browser
nylas air --clear-cache          # Clear all cached data before starting
nylas air --encrypted            # Enable encryption for cached data
```

**Features:**
- **Three-pane interface:** Folders, message list, preview
- **Calendar & Contacts:** Full calendar and contact management
- **Keyboard shortcuts:** J/K navigate, C compose, E archive
- **Command palette:** Cmd+K for quick actions
- **Dark mode:** Customizable themes
- **AI-powered:** Email summaries, smart replies
- **Local caching:** Full-text search with offline support
- **Action queuing:** Queue actions when offline
- **Encryption:** Optional encryption for cached data (system keyring)

**Security:**
- Runs on localhost only (not accessible externally)
- All data stored locally on your machine
- Optional encryption for cached data using system keyring

**URL:** `http://localhost:7365` (default)

**Testing:**
```bash
make ci-full                     # Complete CI pipeline (includes Air tests + cleanup)
make test-air-integration        # Run Air integration tests only
```

---

## MCP (Model Context Protocol)

Enable AI assistants (Claude Desktop, Cursor, Windsurf, VS Code) to interact with your email and calendar.

```bash
nylas mcp install                          # Interactive assistant selection
nylas mcp install --assistant claude-code  # Install for Claude Code
nylas mcp install --assistant cursor       # Install for Cursor
nylas mcp install --all                    # Install for all detected assistants
nylas mcp status                           # Check installation status
nylas mcp uninstall --assistant cursor     # Remove configuration
nylas mcp serve                            # Start MCP server (used by assistants)
```

**Supported assistants:**
| Assistant | Config Location |
|-----------|-----------------|
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Claude Code | `~/.claude.json` + permissions in `~/.claude/settings.json` |
| Cursor | `~/.cursor/mcp.json` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` |
| VS Code | `.vscode/mcp.json` (project-level) |

**Features:**
- Auto-detects system timezone for consistent time display
- Auto-configures Claude Code permissions (`mcp__nylas__*`)
- Injects default grant ID for seamless authentication
- Local grant lookup (no email required for `get_grant`)

**Available MCP tools:** `list_messages`, `list_threads`, `list_calendars`, `list_events`, `create_event`, `update_event`, `send_message`, `create_draft`, `availability`, `get_grant`, `epoch_to_datetime`, `current_time`

---

## Slack Integration

Interact with Slack workspaces directly from the CLI.

### Authentication

```bash
nylas slack auth set --token xoxp-...      # Store Slack user token
nylas slack auth status                     # Check authentication status
nylas slack auth remove                     # Remove stored token
```

**Token sources (checked in order):**
1. `SLACK_USER_TOKEN` environment variable
2. System keyring (set via `nylas slack auth set`)

**Get your token:** [api.slack.com/apps](https://api.slack.com/apps) → Your App → OAuth & Permissions → User OAuth Token

### Channels

```bash
# List channels you're a member of
nylas slack channels list                   # List your channels
nylas slack channels list --type public_channel  # List public channels only
nylas slack channels list --type private_channel # List private channels
nylas slack channels list --exclude-archived     # Exclude archived channels
nylas slack channels list --limit 20             # Limit results
nylas slack channels list --id                   # Show channel IDs

# Filter by creation date
nylas slack channels list --created-after 24h    # Channels created in last 24 hours
nylas slack channels list --created-after 7d     # Channels created in last 7 days
nylas slack channels list --created-after 2w     # Channels created in last 2 weeks

# Workspace-wide listing (slower, may hit rate limits)
nylas slack channels list --all-workspace        # List all workspace channels
nylas slack channels list --all                  # Fetch all pages

# Get channel info
nylas slack channels info C01234567890           # Get detailed channel info
```

### Messages

```bash
nylas slack messages list --channel general       # List messages from channel
nylas slack messages list --channel-id C01234567  # Use channel ID
nylas slack messages list --channel general --limit 10  # Limit results
nylas slack messages list --channel general --id  # Show message timestamps
nylas slack messages list --channel general --thread 1234567890.123456  # Show thread replies
```

### Send & Reply

```bash
# Send a message
nylas slack send --channel general --text "Hello team!"
nylas slack send --channel general --text "Message" --yes  # Skip confirmation

# Reply to a thread
nylas slack reply --channel general --thread 1234567890.123456 --text "Reply"
```

### Users

```bash
nylas slack users list                      # List all users
nylas slack users list --limit 50           # Limit results
nylas slack users list --id                 # Show user IDs
```

### Search

```bash
nylas slack search --query "project update" # Search messages
nylas slack search --query "from:@john"     # Search with Slack modifiers
nylas slack search --query "in:#general"    # Search in specific channel
nylas slack search --query "meeting" --limit 20
```

**Required OAuth Scopes:**
- `channels:read`, `groups:read`, `im:read`, `mpim:read` - List channels
- `channels:history`, `groups:history`, `im:history`, `mpim:history` - Read messages
- `chat:write` - Send messages
- `users:read` - List users
- `search:read` - Search messages

---

## Notetaker (AI Meeting Bot)

Manage Nylas Notetaker bots that join video meetings to record and transcribe.

```bash
# List all notetakers
nylas notetaker list                              # List all notetakers
nylas notetaker list --state scheduled            # Filter by state
nylas notetaker list --limit 10                   # Limit results

# Create a notetaker to join a meeting
nylas notetaker create --meeting-link "https://zoom.us/j/123456789"
nylas notetaker create --meeting-link "https://meet.google.com/abc-defg" --join-time "tomorrow 2pm"
nylas notetaker create --meeting-link "https://zoom.us/j/123" --bot-name "Meeting Recorder"

# Show notetaker details
nylas notetaker show <notetaker-id>

# Get recording and transcript URLs
nylas notetaker media <notetaker-id>

# Delete/cancel a notetaker
nylas notetaker delete <notetaker-id>             # With confirmation
nylas notetaker delete <notetaker-id> --yes       # Skip confirmation
```

**Aliases:** `nylas nt`, `nylas bot`

**Supported Providers:** Zoom, Google Meet, Microsoft Teams

**States:** `scheduled`, `connecting`, `waiting`, `attending`, `processing`, `complete`, `cancelled`, `failed`

**Join Time Formats:**
- ISO: `2024-01-15 14:00`
- Relative: `30m`, `2h`, `1d`
- Natural: `tomorrow 9am`, `tomorrow 2:30pm`

---

## OTP (One-Time Password)

Retrieve OTP/verification codes from email automatically.

```bash
# Get the latest OTP code (auto-copies to clipboard)
nylas otp get                                     # From default account
nylas otp get user@example.com                    # From specific account
nylas otp get --no-copy                           # Don't copy to clipboard
nylas otp get --raw                               # Output only the code

# Watch for new OTP codes (continuous polling)
nylas otp watch                                   # Poll every 10 seconds
nylas otp watch --interval 5                      # Poll every 5 seconds
nylas otp watch user@example.com                  # Watch specific account

# List configured accounts
nylas otp list

# Debug: Show recent messages with OTP detection
nylas otp messages                                # Show last 10 messages
nylas otp messages --limit 20                     # Show more messages
```

**Features:**
- Auto-copies OTP to clipboard
- Supports 4-8 digit codes
- Detects OTPs from common providers (Google, Microsoft, banks, etc.)
- Pretty-printed display with sender info

---

## Scheduler (Booking Pages)

Manage scheduling pages for appointment booking.

```bash
# Booking pages
nylas scheduler pages list                            # List scheduling pages
nylas scheduler pages show <page-id>                  # Show page details
nylas scheduler pages create                          # Create booking page
nylas scheduler pages update <page-id>                # Update page
nylas scheduler pages delete <page-id>                # Delete page

# Bookings
nylas scheduler bookings list                         # List bookings
nylas scheduler bookings show <booking-id>            # Show booking details
nylas scheduler bookings confirm <booking-id>         # Confirm booking
nylas scheduler bookings reschedule <booking-id>      # Reschedule booking
nylas scheduler bookings cancel <booking-id>          # Cancel booking

# Configurations
nylas scheduler configurations list                   # List configurations
nylas scheduler configurations show <config-id>       # Show configuration
nylas scheduler configurations create                 # Create configuration
nylas scheduler configurations update <config-id>     # Update configuration
nylas scheduler configurations delete <config-id>     # Delete configuration

# Sessions
nylas scheduler sessions create                       # Create booking session
nylas scheduler sessions show <session-id>            # Show session details
```

**Details:** `docs/commands/scheduler.md`

---

## Admin (API Management)

Manage Nylas applications, connectors, credentials, and grants.

```bash
# Applications
nylas admin applications list                         # List applications
nylas admin applications show <app-id>                # Show app details
nylas admin applications create                       # Create application
nylas admin applications update <app-id>              # Update application
nylas admin applications delete <app-id>              # Delete application

# Connectors (provider integrations)
nylas admin connectors list                           # List connectors
nylas admin connectors show <connector-id>            # Show connector
nylas admin connectors create                         # Create connector
nylas admin connectors update <connector-id>          # Update connector
nylas admin connectors delete <connector-id>          # Delete connector

# Credentials
nylas admin credentials list <connector-id>           # List credentials
nylas admin credentials show <credential-id>          # Show credential
nylas admin credentials create                        # Create credential
nylas admin credentials update <credential-id>        # Update credential
nylas admin credentials delete <credential-id>        # Delete credential

# Grants
nylas admin grants list                               # List all grants
nylas admin grants stats                              # Grant statistics
```

**Details:** `docs/commands/admin.md`

---

## Audit Logging

Track CLI command execution for compliance, debugging, and AI agent monitoring.

```bash
# Setup
nylas audit init                              # Interactive setup
nylas audit init --enable                     # Quick setup with defaults
nylas audit logs enable                       # Enable logging
nylas audit logs disable                      # Disable logging
nylas audit logs status                       # Check status

# View logs
nylas audit logs show                         # Show last 20 entries
nylas audit logs show --limit 50              # More entries
nylas audit logs show --command email         # Filter by command
nylas audit logs show --invoker alice         # Filter by username
nylas audit logs show --source claude-code    # Filter by source (AI agents, etc.)
nylas audit logs show --status error          # Filter by status
nylas audit logs show --request-id req_abc123 # Find by Nylas request ID

# Statistics and export
nylas audit logs summary                      # Summary for last 7 days
nylas audit logs summary --days 30            # Summary for 30 days
nylas audit export --output audit.json        # Export to JSON
nylas audit export --output audit.csv         # Export to CSV

# Configuration
nylas audit config show                       # Show configuration
nylas audit config set retention_days 30      # Set retention
nylas audit logs clear                        # Clear all logs
```

**Invoker detection:** Automatically tracks who ran commands:
- `terminal` - Interactive terminal session
- `claude-code` - Anthropic's Claude Code
- `github-copilot` - GitHub Copilot CLI
- `ssh` - Remote SSH session
- `script` - Non-interactive automation
- Custom via `NYLAS_INVOKER_SOURCE` env var

**Details:** `docs/commands/audit.md`

---

## Utility Commands

```bash
nylas version                    # Show version
nylas doctor                     # System diagnostics
nylas update                     # Update CLI to latest version
nylas update --check             # Check for updates without installing
nylas update --force             # Force update even if on latest
nylas update --yes               # Skip confirmation prompt
```

**Update command features:**
- Downloads from GitHub releases
- SHA256 checksum verification
- Automatic backup and restore on failure
- Detects Homebrew installs (redirects to `brew upgrade`)

---

## Command Pattern

All commands follow consistent pattern:
- `nylas <resource> list` - List resources
- `nylas <resource> show <id>` - Show details
- `nylas <resource> create` - Create resource
- `nylas <resource> update <id>` - Update resource
- `nylas <resource> delete <id>` - Delete resource

---

**For detailed documentation on any feature, see:**
- Email: `docs/commands/email.md`
- Email Signing: `docs/commands/email-signing.md`
- Email Encryption: `docs/commands/encryption.md`
- Calendar: `docs/commands/calendar.md`
- Contacts: `docs/commands/contacts.md`
- Webhooks: `docs/commands/webhooks.md`
- Inbound: `docs/commands/inbound.md`
- Scheduler: `docs/commands/scheduler.md`
- Admin: `docs/commands/admin.md`
- Workflows: `docs/commands/workflows.md` (OTP, automation)
- Timezone: `docs/commands/timezone.md`
- Audit: `docs/commands/audit.md`
- AI: `docs/commands/ai.md`
- MCP: `docs/commands/mcp.md`
- TUI: `docs/commands/tui.md`
