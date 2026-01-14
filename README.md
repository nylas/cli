# Nylas CLI

A unified command-line tool for Nylas API authentication, email management, calendar, contacts, webhooks, and more.

## Features

- **Email Management**: List, read, send, search, and organize emails with scheduled sending support
- **Calendar Management**: View calendars, list/create/delete events, check availability
- **Contacts Management**: List, view, create, and delete contacts and contact groups
- **Webhook Management**: Create, update, delete, and test webhooks for event notifications
- **Inbound Email**: Receive emails at managed addresses without OAuth (e.g., support@yourapp.nylas.email)
- **Scheduler Management**: Create and manage meeting configurations, booking pages, sessions, and bookings
- **Admin Operations**: Manage applications, connectors, credentials, and grants across your organization
- **Notetaker**: AI meeting bot that joins video calls to record and transcribe
- **MCP Integration**: Enable AI assistants (Claude, Cursor, VS Code) to interact with your email and calendar
- **Demo Mode**: Explore CLI features with sample data before connecting accounts
- **Multi-Account Support**: Manage multiple email accounts with grant switching
- **Secure Credential Storage**: Uses system keyring for credentials
- **Self-Update**: Update CLI to latest version with `nylas update`

## Installation

**Homebrew (macOS/Linux):**
```bash
brew tap nylas/tap && brew install nylas
```

**Go Install:**
```bash
go install github.com/nylas/cli/cmd/nylas@latest
```

**Download Binary:**

Download from [Releases](https://github.com/nylas/cli/releases) and add to your PATH.

**Build from Source:**
```bash
make build
```

## Quick Start

```bash
# Configure with your Nylas credentials
nylas auth config

# Login with your email provider
nylas auth login

# Use CLI commands
nylas email list

# Send an email (immediately)
nylas email send --to "recipient@example.com" --subject "Hello" --body "Hi there!"

# Send an email (scheduled for 2 hours from now)
nylas email send --to "recipient@example.com" --subject "Reminder" --schedule 2h

# List upcoming calendar events
nylas calendar events list

# Check calendar availability
nylas calendar availability check

# Find optimal meeting time
nylas calendar find-time --participants alice@example.com,bob@example.com --duration 1h

# List contacts
nylas contacts list

# List webhooks
nylas webhook list

# List scheduler configurations
nylas scheduler configurations list

# List all grants (admin)
nylas admin grants list

# List applications (admin)
nylas admin applications list
```

---

## Commands

Run `nylas --help` to see all available commands, or see the **[Full Command Reference](docs/COMMANDS.md)** for detailed documentation and examples.

---

## Configuration

Credentials are stored securely in your system keyring:
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **macOS**: Keychain
- **Windows**: Windows Credential Manager

Config file location: `~/.config/nylas/config.yaml`

---

## Documentation

| Document | Description |
|----------|-------------|
| [Commands](docs/COMMANDS.md) | CLI command reference with examples |
| [Webhooks](docs/commands/webhooks.md) | Webhook testing and development guide |
| [Architecture](docs/ARCHITECTURE.md) | Hexagonal architecture overview |
| [Development](docs/DEVELOPMENT.md) | Testing, building, and contributing |
| [Security](docs/security/overview.md) | Security practices and credential handling |

---

## Development

```bash
make build          # Build the CLI binary
make ci             # Quick quality checks
make ci-full        # Complete CI pipeline (all checks + integration tests)
make help           # Show all available targets
```

**[Development Guide](docs/DEVELOPMENT.md)** - Full list of make targets, testing, and contribution guidelines.

---

## API Reference

This CLI uses the [Nylas v3 API](https://developer.nylas.com/docs/api/v3/).

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

MIT
