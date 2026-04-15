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

## Getting Started

```bash
nylas init                       # Guided first-time setup
nylas init --api-key <key>       # Quick setup with existing API key
nylas init --api-key <key> --region eu  # Setup with EU region
nylas init --google              # Setup with Google SSO shortcut
```

The `init` command walks you through:
1. Creating or logging into your Nylas account (SSO)
2. Selecting or creating an application
3. Generating and activating an API key
4. Syncing existing email accounts

Run `nylas init` again after partial setup — it skips completed steps.

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

## Dashboard

Manage your Nylas Dashboard account, applications, and API keys directly from the CLI.

### Account

```bash
nylas dashboard register             # Create a new account (SSO)
nylas dashboard register --google    # Register with Google SSO
nylas dashboard register --microsoft # Register with Microsoft SSO
nylas dashboard register --github    # Register with GitHub SSO

nylas dashboard login                # Log in (interactive)
nylas dashboard login --google       # Log in with Google SSO
nylas dashboard login --email --user user@example.com  # Email/password

nylas dashboard logout               # Log out
nylas dashboard status               # Show current auth status
nylas dashboard refresh              # Refresh session tokens
```

### SSO (Direct)

```bash
nylas dashboard sso login --provider google      # SSO login
nylas dashboard sso register --provider github   # SSO registration
```

### Applications

```bash
nylas dashboard apps list                        # List all applications
nylas dashboard apps list --region us            # Filter by region
nylas dashboard apps create --name "My App" --region us  # Create app
nylas dashboard apps use <app-id> --region us    # Set active app
```

### API Keys

```bash
nylas dashboard apps apikeys list                # List keys (active app)
nylas dashboard apps apikeys list --app <id> --region us  # Explicit app
nylas dashboard apps apikeys create              # Create key (active app)
nylas dashboard apps apikeys create --name "CI"  # Custom name
nylas dashboard apps apikeys create --expires 30 # Expire in 30 days
```

After creating a key, you choose: activate in CLI (recommended), copy to clipboard, or save to file.

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
nylas email send --to EMAIL --subject SUBJECT --body BODY --yes  # Skip confirmation
nylas email send ... --sign                                    # Send GPG-signed email
nylas email send ... --encrypt                                 # Send GPG-encrypted email
nylas email send ... --sign --encrypt                          # Sign AND encrypt (recommended)
nylas email send --list-gpg-keys                               # List available GPG signing keys
nylas email send --to EMAIL --template-id TPL --template-data '{}'  # Send using a hosted template
nylas email send --template-id TPL --template-data-file data.json --render-only
nylas email send --to EMAIL --subject SUBJECT --body BODY --signature-id SIG  # Send with stored signature
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

**Managed send behavior:**
- Grants with provider `inbox` and `nylas` use the managed transactional send path automatically.
- The sender address comes from the active grant email for those managed providers.
- GPG signing/encryption and `--signature-id` are not supported for managed transactional sends.

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

## Local Email Templates

```bash
nylas email templates list                           # List all templates
nylas email templates create --name NAME --subject SUBJECT --body BODY
nylas email templates show <template-id>             # Show template details
nylas email templates update <template-id> [flags]   # Update template
nylas email templates delete <template-id>           # Delete template
nylas email templates use <template-id> --to EMAIL   # Send using template
nylas email templates use <template-id> --to EMAIL --signature-id SIG  # Send using template + stored signature
```

**Variable syntax:** Use `{{variable}}` in subject/body for placeholders.

**Details:** `docs/commands/templates.md`

---

## Hosted Templates

```bash
nylas template list
nylas template create --name NAME --subject SUBJECT --body BODY
nylas template show <template-id>
nylas template update <template-id> [flags]
nylas template delete <template-id> --yes
nylas template render <template-id> --data '{}'
nylas template render-html --body "<p>{{x}}</p>" --engine mustache --data '{}'
```

**Scopes:** `--scope app` for application templates, `--scope grant --grant-id <id>` for grant-level templates.

**Details:** `docs/commands/templates.md`

---

## Hosted Workflows

```bash
nylas workflow list
nylas workflow create --name NAME --template-id TPL --trigger-event booking.created
nylas workflow show <workflow-id>
nylas workflow update <workflow-id> [flags]
nylas workflow delete <workflow-id> --yes
```

**Scopes:** `--scope app` for application workflows, `--scope grant --grant-id <id>` for grant-level workflows.

**Details:** `docs/commands/workflows.md`

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
nylas email drafts create --to EMAIL --subject S --signature-id SIG  # Create draft with stored signature
nylas email drafts send <draft-id>                # Send draft
nylas email drafts send <draft-id> --signature-id SIG  # Send draft with stored signature
nylas email drafts delete <draft-id>              # Delete draft
```

**Flags:** `--to`, `--cc`, `--bcc`, `--subject`, `--body`, `--reply-to`, `--attach`, `--signature-id`

---

## Signatures

```bash
nylas email signatures list [grant-id]                         # List stored signatures
nylas email signatures show <signature-id> [grant-id]          # Show signature details
nylas email signatures create [grant-id] --name NAME --body BODY  # Create signature
nylas email signatures create [grant-id] --name NAME --body-file FILE
nylas email signatures update <signature-id> [grant-id] [flags]    # Update signature
nylas email signatures delete <signature-id> [grant-id] --yes      # Delete signature
```

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
nylas webhook rotate-secret <webhook-id> --yes        # Rotate webhook signing secret
nylas webhook verify --payload-file body.json --signature SIG --secret SECRET
nylas webhook triggers                                # List available triggers
```

**Pub/Sub channels:**
```bash
nylas webhook pubsub list
nylas webhook pubsub show <channel-id>
nylas webhook pubsub create --topic projects/PROJ/topics/TOPIC --triggers "message.created"
nylas webhook pubsub update <channel-id> --status inactive
nylas webhook pubsub delete <channel-id> --yes
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

## Agent Accounts

Create and manage Nylas-managed agent accounts backed by provider `nylas`.

```bash
nylas agent account list                       # List agent accounts
nylas agent account create <email>             # Create agent account
nylas agent account create <email> --app-password PW  # Create account with IMAP/SMTP app password
nylas agent account create <email> --policy-id <policy-id>  # Create account attached to a policy
nylas agent account get <agent-id|email>       # Show one agent account
nylas agent account delete <agent-id|email>    # Delete/revoke agent account
nylas agent account delete <agent-id|email> --yes  # Skip confirmation
nylas agent policy list                        # List policy for default agent account
nylas agent policy list --all                  # List all policies attached to agent accounts
nylas agent policy create --name NAME          # Create a policy
nylas agent policy get <policy-id>             # Show one policy
nylas agent policy read <policy-id>            # Read one policy
nylas agent policy update <policy-id> --name NAME  # Update a policy
nylas agent policy delete <policy-id> --yes    # Delete an unattached policy
nylas agent rule list                          # List rules for default agent policy
nylas agent rule list --all                    # List all rules attached to agent policies
nylas agent rule read <rule-id>                # Read one rule
nylas agent rule get <rule-id>                 # Show one rule
nylas agent rule create --name NAME --condition from.domain,is,example.com --action mark_as_spam  # Create a rule from common flags
nylas agent rule create --data-file rule.json  # Create a rule from full JSON
nylas agent rule update <rule-id> --name NAME --description TEXT  # Update a rule
nylas agent rule delete <rule-id> --yes        # Delete a rule
nylas agent status                             # Check connector + account status
```

**Details:** `docs/commands/agent.md`, `docs/commands/agent-policy.md`, `docs/commands/agent-rule.md`

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

## Chat (`nylas chat`) - AI Chat Interface

Launch **Nylas Chat** - a web-based AI chat interface that can access your email, calendar, and contacts:

```bash
nylas chat                       # Start with auto-detected agent (default port 7367)
nylas chat --agent claude        # Use specific agent (claude, codex, ollama)
nylas chat --agent ollama --model llama2  # Use Ollama with specific model
nylas chat --port 8080           # Custom port
nylas chat --no-browser          # Don't auto-open browser
```

**Features:**
- **Local AI agents:** Uses Claude, Codex, or Ollama installed on your system
- **Email & Calendar access:** AI can read emails, check calendar, manage contacts
- **Conversation history:** Persistent chat sessions stored locally
- **Agent switching:** Change agents without restarting
- **Web interface:** Clean, modern chat UI

**Supported Agents:**
| Agent | Description | Auto-detected |
|-------|-------------|---------------|
| Claude | Anthropic's Claude (via `claude` CLI) | ✅ |
| Codex | OpenAI Codex | ✅ |
| Ollama | Local LLM runner (customizable models) | ✅ |

**Agent Detection:**
The CLI automatically detects installed agents on your system. Use `--agent` to override the default selection.

**Conversation Storage:**
- Location: `~/.config/nylas/chat/conversations/`
- Format: JSON files per conversation
- Persistent across sessions

**Security:**
- Runs on localhost only (not accessible externally)
- All data stored locally on your machine
- Agent communication happens through local processes

**URL:** `http://localhost:7367` (default)

**Examples:**
```bash
# Quick start with best available agent
nylas chat

# Force use of Claude
nylas chat --agent claude

# Use Ollama with Mistral model
nylas chat --agent ollama --model mistral

# Run on different port
nylas chat --port 9000 --no-browser
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

Manage Nylas applications, callback URIs, connectors, credentials, and grants.

```bash
# Applications
nylas admin applications list                         # List applications
nylas admin applications show <app-id>                # Show app details
nylas admin applications create                       # Create application
nylas admin applications update <app-id>              # Update application
nylas admin applications delete <app-id>              # Delete application

# Callback URIs (OAuth redirect endpoints)
nylas admin callback-uris list                        # List callback URIs
nylas admin callback-uris show <uri-id>               # Show callback URI
nylas admin callback-uris create --url <url>          # Create callback URI
nylas admin callback-uris update <uri-id> --url <url> # Update callback URI
nylas admin callback-uris delete <uri-id>             # Delete callback URI

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
