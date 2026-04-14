# Documentation Index

> **Website:** [cli.nylas.com](https://cli.nylas.com/)

Quick navigation guide to find the right documentation for your needs.

---

## 🎯 I want to...

### Get Started

- **First-time setup** → `nylas init` ([details](COMMANDS.md#getting-started))
- **Learn about Nylas CLI** → [README.md](../README.md)
- **Quick command reference** → [COMMANDS.md](COMMANDS.md)
- **See examples** → [COMMANDS.md](COMMANDS.md) and [commands/](commands/)

### Understand the Project

- **Architecture overview** → [ARCHITECTURE.md](ARCHITECTURE.md)
- **File structure** → [CLAUDE.md](../CLAUDE.md#file-structure)

### Development

- **Set up development environment** → [DEVELOPMENT.md](DEVELOPMENT.md)
- **Testing guidelines** → [.claude/rules/testing.md](../.claude/rules/testing.md)
- **Go quality & linting** → [.claude/rules/go-quality.md](../.claude/rules/go-quality.md)
- **Contributing guidelines** → [CONTRIBUTING.md](../CONTRIBUTING.md)

### Add Features

- **Add a CLI command** → [.claude/commands/add-command.md](../.claude/commands/add-command.md)
- **Add an API method** → [.claude/commands/add-api-method.md](../.claude/commands/add-api-method.md)
- **Add a domain type** → [.claude/commands/add-domain-type.md](../.claude/commands/add-domain-type.md)
- **Add a command flag** → [.claude/commands/add-flag.md](../.claude/commands/add-flag.md)
- **Generate CRUD command** → [.claude/commands/generate-crud-command.md](../.claude/commands/generate-crud-command.md)

### Testing

- **Run tests** → [.claude/commands/run-tests.md](../.claude/commands/run-tests.md)
- **Add integration test** → [.claude/commands/add-integration-test.md](../.claude/commands/add-integration-test.md)
- **Debug test failure** → [.claude/commands/debug-test-failure.md](../.claude/commands/debug-test-failure.md)
- **Analyze coverage** → [.claude/commands/analyze-coverage.md](../.claude/commands/analyze-coverage.md)
- **Testing guidelines** → [.claude/rules/testing.md](../.claude/rules/testing.md)

### Fix Issues

- **Fix build errors** → [.claude/commands/fix-build.md](../.claude/commands/fix-build.md)
- **Debug test failure** → [.claude/commands/debug-test-failure.md](../.claude/commands/debug-test-failure.md)
- **Troubleshooting guide** → [troubleshooting/](troubleshooting/)

### Quality & Security

- **Security scan** → [.claude/commands/security-scan.md](../.claude/commands/security-scan.md)
- **Security guidelines** → [security/overview.md](security/overview.md)
- **Code review** → [.claude/commands/review-pr.md](../.claude/commands/review-pr.md)
- **Go quality & linting** → [.claude/rules/go-quality.md](../.claude/rules/go-quality.md)
- **File size limits** → [.claude/rules/file-size-limits.md](../.claude/rules/file-size-limits.md)

### Maintenance

- **Update documentation** → [.claude/commands/update-docs.md](../.claude/commands/update-docs.md)
- **Documentation rules** → [.claude/rules/documentation-maintenance.md](../.claude/rules/documentation-maintenance.md)
- **Go quality rules** → [.claude/rules/go-quality.md](../.claude/rules/go-quality.md)

### Command Guides

- **Email** → [commands/email.md](commands/email.md)
- **Email signing (GPG)** → [commands/email-signing.md](commands/email-signing.md)
- **Email encryption** → [commands/encryption.md](commands/encryption.md)
- **GPG explained** → [commands/explain-gpg.md](commands/explain-gpg.md)
- **Calendar** → [commands/calendar.md](commands/calendar.md)
- **Contacts** → [commands/contacts.md](commands/contacts.md)
- **Webhooks** → [commands/webhooks.md](commands/webhooks.md)
- **Agent accounts** → [commands/agent.md](commands/agent.md)
- **Inbound email** → [commands/inbound.md](commands/inbound.md)
- **Scheduler** → [commands/scheduler.md](commands/scheduler.md)
- **Admin** → [commands/admin.md](commands/admin.md)
- **Timezone** → [commands/timezone.md](commands/timezone.md)
- **Audit** → [commands/audit.md](commands/audit.md)
- **TUI** → [commands/tui.md](commands/tui.md)
- **Workflows (OTP)** → [commands/workflows.md](commands/workflows.md)
- **Templates** → [commands/templates.md](commands/templates.md)
- **Slack** → [COMMANDS.md#slack-integration](COMMANDS.md#slack-integration)

### AI & MCP

- **AI features** → [commands/ai.md](commands/ai.md)
- **MCP integration** → [commands/mcp.md](commands/mcp.md)
- **AI configuration** → [ai/configuration.md](ai/configuration.md)
- **AI providers** → [ai/providers.md](ai/providers.md)
- **AI privacy** → [ai/privacy-security.md](ai/privacy-security.md)
- **AI best practices** → [ai/best-practices.md](ai/best-practices.md)
- **AI architecture** → [ai/architecture.md](ai/architecture.md)
- **AI features list** → [ai/features.md](ai/features.md)
- **AI FAQ** → [ai/faq.md](ai/faq.md)
- **AI troubleshooting** → [ai/troubleshooting.md](ai/troubleshooting.md)

### Development Guides

- **Adding commands** → [development/adding-command.md](development/adding-command.md)
- **Adding adapters** → [development/adding-adapter.md](development/adding-adapter.md)
- **Testing guide** → [development/testing-guide.md](development/testing-guide.md)
- **Debugging** → [development/debugging.md](development/debugging.md)

### Security & Troubleshooting

- **Security overview** → [security/overview.md](security/overview.md)
- **Security practices** → [security/practices.md](security/practices.md)
- **FAQ** → [troubleshooting/faq.md](troubleshooting/faq.md)
- **Auth issues** → [troubleshooting/auth.md](troubleshooting/auth.md)
- **API issues** → [troubleshooting/api.md](troubleshooting/api.md)
- **Email issues** → [troubleshooting/email.md](troubleshooting/email.md)
- **Timezone issues** → [troubleshooting/timezone.md](troubleshooting/timezone.md)

---

## 📂 Documentation Structure

```
docs/
├── INDEX.md               # This file - start here
├── COMMANDS.md            # CLI quick reference
├── ARCHITECTURE.md        # System design
├── DEVELOPMENT.md         # Development setup
│
├── commands/              # Detailed command guides (17 files)
│   ├── ai.md              # AI features
│   ├── mcp.md             # MCP integration
│   ├── calendar.md        # Calendar events
│   ├── email.md           # Email operations
│   ├── email-signing.md   # GPG/PGP email signing
│   ├── encryption.md      # Email encryption
│   ├── explain-gpg.md     # GPG explained
│   ├── contacts.md        # Contact management
│   ├── webhooks.md        # Webhook setup
│   ├── inbound.md         # Inbound email
│   ├── scheduler.md       # Booking pages
│   ├── admin.md           # API management
│   ├── timezone.md        # Timezone utilities
│   ├── audit.md           # Audit logging & invoker tracking
│   ├── tui.md             # Terminal UI
│   ├── templates.md       # Email templates
│   └── workflows.md       # OTP & automation
│
├── ai/                    # AI configuration (8 files)
│   ├── configuration.md   # Setup guide
│   ├── providers.md       # Provider options
│   ├── privacy-security.md # Privacy controls
│   └── ...
│
├── development/           # Dev guides (4 files)
│   ├── adding-command.md  # Add CLI commands
│   ├── adding-adapter.md  # Add API adapters
│   ├── testing-guide.md   # Testing patterns
│   └── debugging.md       # Debug tips
│
├── security/              # Security (2 files)
│   ├── overview.md        # Quick reference
│   └── practices.md       # Detailed practices
│
├── troubleshooting/       # Debug guides (5 files)
│   ├── faq.md             # Common questions
│   ├── auth.md            # Auth issues
│   ├── api.md             # API errors
│   ├── email.md           # Email issues
│   └── timezone.md        # Timezone issues
│
└── images/                # Screenshots (5 PNGs)
```

---

## 🔍 By Role

### **New Contributors**
1. [README.md](../README.md) - Project overview
2. [CONTRIBUTING.md](../CONTRIBUTING.md) - How to contribute
3. [DEVELOPMENT.md](DEVELOPMENT.md) - Setup instructions
4. [CLAUDE.md](../CLAUDE.md#file-structure) - Code navigation

### **Developers Adding Features**
1. [ARCHITECTURE.md](ARCHITECTURE.md) - Understand the design
2. [.claude/commands/add-command.md](../.claude/commands/add-command.md) - Add CLI commands
3. [.claude/rules/testing.md](../.claude/rules/testing.md) - Testing requirements
4. [.claude/rules/documentation-maintenance.md](../.claude/rules/documentation-maintenance.md) - Doc updates

### **Bug Fixers**
1. [.claude/commands/debug-test-failure.md](../.claude/commands/debug-test-failure.md) - Test debugging
2. [troubleshooting/](troubleshooting/) - Common issues & FAQ
3. [.claude/commands/fix-build.md](../.claude/commands/fix-build.md) - Fix build errors

### **Maintainers**
1. [.claude/commands/security-scan.md](../.claude/commands/security-scan.md) - Security checks
2. [.claude/commands/review-pr.md](../.claude/commands/review-pr.md) - PR review
3. [.claude/rules/go-quality.md](../.claude/rules/go-quality.md) - Go quality & linting

### **Users**
1. [README.md](../README.md) - Getting started
2. [COMMANDS.md](COMMANDS.md) - Command reference
3. [cli.nylas.com/guides](https://cli.nylas.com/guides) - Step-by-step tutorials
4. [troubleshooting/faq.md](troubleshooting/faq.md) - Common questions

---

## 💡 Quick Tips

- **For AI (Claude):** Most docs are in CLAUDE.md and .claude/ directory
- **For humans:** Start with README.md and COMMANDS.md
- **Need help?** Check [troubleshooting/faq.md](troubleshooting/faq.md)
- **Adding code?** Follow workflows in .claude/commands/
- **Security concern?** See [security/overview.md](security/overview.md)

---

**Last Updated:** February 5, 2026
