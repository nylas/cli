# Nylas CLI

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Release](https://img.shields.io/github/v/release/nylas/cli)

A unified command-line tool for Nylas API authentication, email management, calendar, contacts, webhooks, timezone utilities, and OTP extraction.

## Features

- **Time Zone Utilities**: ⚡ **Offline** timezone conversion, DST transitions, and meeting time finder (no API required)
- **Timezone-Aware Calendar**: View events in any timezone with `--timezone` flag, auto-detect local timezone, **DST warnings** ✅, natural language time parsing ✅
- **Smart Meeting Finder** ✅: Multi-timezone meeting scheduling with 100-point scoring algorithm (working hours, time quality, cultural considerations)
- **AI-Powered Scheduling** (Planned): Natural language scheduling, predictive patterns, conflict resolution with privacy-first local AI (Ollama) or cloud AI (Claude, OpenAI)
- **Interactive TUI**: k9s-style terminal interface with vim-style commands, Google Calendar-style views, and email compose/reply
- **Email Management**: List, read, send, search, and organize emails with scheduled sending support
- **Calendar Management**: View calendars, list/create/delete events, check availability
- **Contacts Management**: List, view, create, and delete contacts and contact groups
- **Webhook Management**: Create, update, delete, and test webhooks for event notifications
- **Inbound Email**: Receive emails at managed addresses without OAuth (e.g., support@yourapp.nylas.email)
- **Scheduler Management**: Create and manage meeting configurations, booking pages, sessions, and bookings
- **Admin Operations**: Manage applications, connectors, credentials, and grants across your organization
- **Draft Management**: Create, edit, and send drafts
- **Folder Management**: Create, rename, and delete folders/labels
- **Thread Management**: View and manage email conversations
- **OTP Extraction**: Automatically extract one-time passwords from emails
- **Slack Integration**: List channels, read/send messages, search, and manage users
- **Multi-Account Support**: Manage multiple email accounts with grant switching
- **Secure Credential Storage**: Uses system keyring for credentials

## Installation

**Homebrew (macOS/Linux):**
```bash
brew install nylas/nylas-cli/nylas
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

### Timezone Tools (No API Required!)

```bash
# Convert time between timezones
nylas timezone convert --from PST --to IST

# Check DST transitions
nylas timezone dst --zone America/New_York --year 2026

# Find meeting times across multiple zones
nylas timezone find-meeting --zones "America/New_York,Europe/London,Asia/Tokyo"

# List all timezones
nylas timezone list --filter America

# Get timezone info
nylas timezone info UTC
```

### Timezone-Aware Calendar

```bash
# List events in different timezone
nylas calendar events list --timezone America/Los_Angeles

# Show timezone information for events
nylas calendar events list --show-tz

# View specific event in multiple timezones
nylas calendar events show <event-id> --timezone Europe/London
nylas calendar events show <event-id> --timezone Asia/Tokyo

# Automatic DST warnings for events near DST transitions
# ⚠️ "Daylight Saving Time begins in 2 days (clocks spring forward 1 hour)"
# ⛔ "This time will not exist due to Daylight Saving Time (clocks spring forward)"
```

**Features:**
- ✅ Multi-timezone event display with conversion
- ✅ Automatic DST (Daylight Saving Time) warnings
- ✅ Natural language time parsing ready for integration
- 🔄 Timezone locking (planned - Task 1.5)

**[Full Timezone Documentation](docs/commands/timezone.md)**

### AI-Powered Scheduling (Coming Soon)

```bash
# Natural language scheduling (privacy-first with Ollama)
nylas calendar ai schedule "30-min call with john@example.com tomorrow afternoon"

# Find optimal meeting times across timezones
nylas calendar find-time --participants alice@team.com,bob@team.com --duration 1h

# Analyze scheduling patterns
nylas calendar ai analyze --learn-patterns

# Auto-resolve conflicts
nylas calendar ai reschedule <event-id> --reason "Urgent task"
```

**[Full AI Documentation](docs/commands/ai.md)**

### Email & Calendar (Requires API)

```bash
# Configure with your Nylas credentials
nylas auth config

# Login with your email provider
nylas auth login

# Launch the interactive TUI
nylas tui

# Or use CLI commands directly
nylas email list

# Send an email (immediately)
nylas email send --to "recipient@example.com" --subject "Hello" --body "Hi there!"

# Send an email (scheduled for 2 hours from now)
nylas email send --to "recipient@example.com" --subject "Reminder" --schedule 2h

# List upcoming calendar events
nylas calendar events list

# Check calendar availability
nylas calendar availability check

# Find optimal meeting time across timezones
nylas calendar find-time --participants alice@example.com,bob@example.com --duration 1h

# List contacts
nylas contacts list

# List webhooks
nylas webhook list

# Get the latest OTP code
nylas otp get

# List scheduler configurations
nylas scheduler configurations list

# List all grants (admin)
nylas admin grants list

# List applications (admin)
nylas admin applications list
```

---

## Commands Overview

| Command | Description | API Required |
|---------|-------------|--------------|
| `nylas timezone` | ⚡ Timezone conversion, DST, meeting finder | No |
| `nylas auth` | Authentication and account management | Yes |
| `nylas email` | Email operations (list, read, send, search) | Yes |
| `nylas calendar` | Calendar and event management | Yes |
| `nylas contacts` | Contact management | Yes |
| `nylas webhook` | Webhook configuration | Yes |
| `nylas inbound` | Inbound email inboxes (managed addresses) | Yes |
| `nylas scheduler` | Scheduler configurations, bookings, and pages | Yes |
| `nylas admin` | Administration (applications, connectors, credentials, grants) | Yes |
| `nylas otp` | OTP code extraction | Yes |
| `nylas tui` | Interactive terminal interface | Yes |
| `nylas ui` | Web-based graphical interface | Yes |
| `nylas doctor` | Diagnostic checks | No |

**[Full Command Reference](docs/COMMANDS.md)**

---

## TUI Highlights

![TUI Demo](docs/images/tui-demo.png)

```bash
nylas tui                    # Launch TUI at dashboard
nylas tui --demo             # Demo mode (no credentials needed)
nylas tui --theme amber      # Retro amber CRT theme
```

**Themes:** k9s, amber, green, apple2, vintage, ibm, futuristic, matrix, norton

**Vim-style keys:** `j/k` navigate, `gg/G` first/last, `dd` delete, `:q` quit, `/` search

**[Full TUI Documentation](docs/commands/tui.md)**

---

## Web UI

Launch a browser-based interface for visual CLI management:

```bash
nylas ui                     # Start on http://localhost:7363
nylas ui --port 8080         # Custom port
nylas ui --no-browser        # Don't auto-open browser
```

**Features:** API configuration, account switching, email/calendar/auth commands, ID autocomplete, command history

**Security:** Localhost only, command whitelist, shell injection protection

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
| [Timezone](docs/commands/timezone.md) | Comprehensive timezone utilities guide |
| [Webhooks](docs/commands/webhooks.md) | Webhook testing and development guide |
| [TUI](docs/commands/tui.md) | Terminal UI themes, keys, customization |
| [Architecture](docs/ARCHITECTURE.md) | Hexagonal architecture overview |
| [Development](docs/DEVELOPMENT.md) | Testing, building, and contributing |
| [Security](docs/security/overview.md) | Security practices and credential handling |

---

## Development

### Quick Start

```bash
make build          # Build the CLI binary
make ci             # Quick quality checks (fmt, vet, lint, test-unit, test-race, security, vuln)
make ci-full        # Complete CI pipeline (all checks + integration tests + cleanup)
```

### Available Targets

| Target | Description | Use When |
|--------|-------------|----------|
| `make ci-full` | **Complete validation** (quality + all tests + cleanup) | Before PRs, releases |
| `make ci` | Quality checks only (no integration tests) | Quick pre-commit check |
| `make build` | Build binary to `./bin/nylas` | Development |
| `make test-unit` | Run unit tests | Fast feedback loop |
| `make test-coverage` | Generate coverage report | Check test coverage |
| `make lint` | Run linter only | Fix linting issues |
| `make clean` | Remove build artifacts | Clean workspace |
| `make help` | Show all available targets | See all options |

**Run `make help` for complete list of targets**

**[Development Guide](docs/DEVELOPMENT.md)**

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
