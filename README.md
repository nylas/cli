# Nylas CLI

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Release](https://img.shields.io/github/v/release/nylas/cli)
[![Website](https://img.shields.io/badge/docs-cli.nylas.com-blue)](https://cli.nylas.com/)

Email, calendar, and contacts from your terminal. One CLI for Google, Microsoft, and IMAP -- no SMTP config, no provider SDKs, no boilerplate.

**[Documentation](https://cli.nylas.com/)** | **[API Reference](https://developer.nylas.com/docs/api/v3/)**

## Install

**Homebrew (macOS/Linux):**
```bash
brew install nylas/nylas-cli/nylas
```

**Go:**
```bash
go install github.com/nylas/cli/cmd/nylas@latest
```

**Binary:** Download from [Releases](https://github.com/nylas/cli/releases) and add to PATH.

## Get Started

### 1. Run the setup wizard

```bash
nylas init
```

The wizard walks you through four steps:

1. **Account** -- Create a free Nylas account (via Google, Microsoft, or GitHub SSO), or log into an existing one
2. **Application** -- Select an existing app or create your first one automatically
3. **API Key** -- Generate and activate a key (stored securely in your system keyring)
4. **Email Accounts** -- Detect and sync any email accounts already connected to your app

That's it. You're ready to go:

```bash
nylas email list             # See your latest emails
nylas calendar events list   # See upcoming events
nylas contacts list          # See your contacts
```

### Already have an API key?

Skip the interactive wizard entirely:

```bash
nylas init --api-key nyl_abc123
# EU region:
nylas init --api-key nyl_abc123 --region eu
```

### Just want to explore first?

Try the demo mode -- no account or API key needed:

```bash
nylas tui --demo
```

### Connect an email account later

After setup, you can authenticate additional email accounts at any time:

```bash
nylas auth login                      # Google (default)
nylas auth login --provider microsoft # Microsoft / Outlook
```

## What You Can Do

### Email

```bash
nylas email list                          # Recent emails
nylas email read <message-id>             # Read a message
nylas email send --to hi@example.com \
  --subject "Hello" --body "World"        # Send an email
nylas email search "invoice"              # Search
```

### Calendar

```bash
nylas calendar events list                # Upcoming events
nylas calendar availability               # Check availability
nylas scheduler                           # AI-powered scheduling
```

### Contacts

```bash
nylas contacts list                       # All contacts
nylas contacts create --name "Jane"       # Create a contact
```

### Webhooks

```bash
nylas webhook list                        # List webhooks
nylas webhook create                      # Create a webhook
```

### Timezone Tools (works offline, no API key required)

```bash
nylas timezone convert --from PST --to IST     # Convert times
nylas timezone dst --zone America/New_York      # DST transitions
nylas timezone find-meeting --zones "NYC,LON"   # Find meeting times
```

**[Full Command Reference](docs/COMMANDS.md)**

## Interfaces

The CLI has three ways to interact with your data:

| Interface | Launch | Description |
|-----------|--------|-------------|
| **CLI** | `nylas <command>` | Standard command-line interface |
| **TUI** | `nylas tui` | Interactive terminal UI with vim keys and [9 themes](docs/commands/tui.md) |
| **Air** | `nylas air` | Modern web client at localhost:7365 -- email, calendar, contacts in your browser |

## AI & MCP Integration

Give your AI coding agent access to email, calendar, and contacts:

```bash
nylas mcp    # Start the MCP server for AI assistants
nylas ai     # Chat with your data
```

Works with Claude Code, Cursor, Codex CLI, and any MCP-compatible assistant.

## Guides

Step-by-step tutorials on [cli.nylas.com](https://cli.nylas.com/guides):

- [Give your AI coding agent an email address](https://cli.nylas.com/guides/give-ai-agent-email-address) -- Claude Code, Cursor, Codex CLI, and OpenClaw
- [Send email from the command line](https://cli.nylas.com/guides/send-email-from-terminal) -- one command, no SMTP
- [AI agent email access via MCP](https://cli.nylas.com/guides/ai-agent-email-mcp) -- connect any MCP-compatible assistant
- [Manage calendar from the terminal](https://cli.nylas.com/guides/manage-calendar-from-terminal) -- events, availability, timezone handling

## Configuration

Credentials are stored in your system keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager). Non-secret grant metadata, such as account email/provider and the local default grant, is cached separately for fast local lookup.

```bash
nylas auth status    # Check what's configured
nylas config         # View/edit settings
```

Config file: `~/.config/nylas/config.yaml`

## Development

```bash
make build      # Build binary
make ci         # Quality checks (fmt, vet, lint, test, security)
make ci-full    # Complete CI (quality + integration tests)
```

**[Development Guide](docs/DEVELOPMENT.md)** | **[Contributing](CONTRIBUTING.md)**

## License

MIT
