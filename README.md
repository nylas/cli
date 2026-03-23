# Nylas CLI

![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)
![Release](https://img.shields.io/github/v/release/nylas/cli)
[![Website](https://img.shields.io/badge/docs-cli.nylas.com-blue)](https://cli.nylas.com/)

**[Documentation](https://cli.nylas.com/)** | Unified CLI for [Nylas API](https://www.nylas.com/) - manage email, calendar, and contacts across providers (Google, Microsoft, IMAP) with a single interface.

## Installation

**Homebrew (macOS/Linux):**
```bash
brew install nylas/nylas-cli/nylas
```

**Go Install:**
```bash
go install github.com/nylas/cli/cmd/nylas@latest
```

**Binary:** Download from [Releases](https://github.com/nylas/cli/releases) and add to PATH.

## Getting Started

**Just want to explore?** Try the demo first - no credentials needed:
```bash
nylas tui --demo
```

**Ready to connect your account?** [Get API credentials](https://dashboard.nylas.com/) (free tier available), then:
```bash
nylas auth config    # Enter your API key
nylas auth login     # Connect your email provider
nylas email list     # You're ready!
```

## Basic Commands

| Command | Example |
|---------|---------|
| Email | `nylas email list`, `nylas email send --to user@example.com` |
| Calendar | `nylas calendar events list` |
| Contacts | `nylas contacts list` |
| Webhooks | `nylas webhook list` |
| TUI | `nylas tui` (interactive terminal, vim keys, [9 themes](docs/commands/tui.md)) |
| Web UI | `nylas air` (browser interface) |

**[Full Command Reference →](docs/COMMANDS.md)** | **[All Documentation →](docs/INDEX.md)**

## Guides

Step-by-step tutorials on [cli.nylas.com](https://cli.nylas.com/guides):

- [Give your AI coding agent an email address](https://cli.nylas.com/guides/give-ai-agent-email-address) — setup for Claude Code, Cursor, Codex CLI, and OpenClaw
- [Send email from the command line](https://cli.nylas.com/guides/send-email-from-terminal) — no SMTP, no sendmail, one command
- [AI agent email access via MCP](https://cli.nylas.com/guides/ai-agent-email-mcp) — connect any MCP-compatible assistant
- [Manage calendar from the terminal](https://cli.nylas.com/guides/manage-calendar-from-terminal) — events, availability, timezone handling

## Features

- **Email**: list, read, send, search, templates, GPG signing/encryption
- **Calendar**: events, availability, timezone conversion, AI scheduling
- **Contacts**: list, create, groups
- **Webhooks**: create, test, manage
- **Timezone**: ⚡ offline conversion, DST info, meeting finder (no API required)
- **Admin**: applications, connectors, credentials, grants
- **Integrations**: [MCP](https://cli.nylas.com/guides/ai-agent-email-mcp) (AI assistants)
- **Interfaces**: CLI, TUI (terminal), Air (web)

## Timezone Tools (No API Required)

```bash
nylas timezone convert --from PST --to IST     # Convert time
nylas timezone dst --zone America/New_York     # Check DST transitions
nylas timezone find-meeting --zones "NYC,LON"  # Find meeting times
```

## Configuration

Credentials stored securely in system keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager).

Config file: `~/.config/nylas/config.yaml`

## Development

```bash
make build      # Build binary
make ci         # Quality checks (fmt, vet, lint, test, security)
make ci-full    # Complete CI (quality + integration tests)
```

**[Development Guide](docs/DEVELOPMENT.md)** | **[Contributing](CONTRIBUTING.md)**

## API Reference

This CLI uses the [Nylas v3 API](https://developer.nylas.com/docs/api/v3/).

## License

MIT
