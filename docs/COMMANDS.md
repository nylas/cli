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
nylas auth add                   # Manually add an existing grant
nylas auth remove <grant-id>     # Remove account from local config
nylas auth revoke <grant-id>     # Permanently revoke grant on server
nylas auth token                 # Display current API token
nylas auth scopes [grant-id]     # Show granted OAuth scopes
nylas auth providers             # List available providers
nylas auth detect <email>        # Detect provider from email address
nylas auth migrate               # Migrate credentials to system keyring
```

---

## Demo Mode (No Account Required)

Explore the CLI with sample data before connecting your accounts:

```bash
nylas demo email list            # Browse sample emails
nylas demo calendar list         # View sample events
nylas demo contacts list         # See sample contacts
nylas demo notetaker list        # Explore AI notetaker
```

All demo commands mirror real CLI structure: `nylas demo <feature> <command>`

---

## Email

```bash
nylas email list [grant-id]                                    # List emails
nylas email read <message-id>                                  # Read email
nylas email send --to EMAIL --subject SUBJECT --body BODY      # Send email
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

**Details:** `docs/commands/email.md`

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
# Calendar management
nylas calendar list                                              # List calendars
nylas calendar show <calendar-id>                                # Show calendar details
nylas calendar create --name "NAME"                              # Create a new calendar
nylas calendar update <calendar-id> --name "NEW NAME"            # Update a calendar
nylas calendar delete <calendar-id>                              # Delete a calendar

# Event management
nylas calendar events list [--days N]                            # List events
nylas calendar events show <event-id>                            # Show event details
nylas calendar events create --title T --start TIME --end TIME   # Create event
nylas calendar events update <event-id> --title "New Title"      # Update event
nylas calendar events delete <event-id>                          # Delete event
nylas calendar events rsvp <event-id> --status yes               # RSVP to event

# Availability & scheduling
nylas calendar availability check                                # Check availability
nylas calendar find-time --attendees EMAIL1,EMAIL2               # Find optimal meeting times
nylas calendar recurring list                                    # List recurring events
nylas calendar virtual list                                      # List virtual meetings
```

**Key features:** Working hours validation, break protection

**Details:** `docs/commands/calendar.md`

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
- Auto-configures Claude Code permissions (`mcp__nylas__*`)
- Injects default grant ID for seamless authentication
- Local grant lookup (no email required for `get_grant`)

**Available MCP tools:** `list_messages`, `list_threads`, `list_calendars`, `list_events`, `create_event`, `update_event`, `send_message`, `create_draft`, `availability`, `get_grant`, `epoch_to_datetime`, `current_time`

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
- Calendar: `docs/commands/calendar.md`
- Contacts: `docs/commands/contacts.md`
- Webhooks: `docs/commands/webhooks.md`
- Inbound: `docs/commands/inbound.md`
- Scheduler: `docs/commands/scheduler.md`
- Admin: `docs/commands/admin.md`
- MCP: `docs/commands/mcp.md`
