# Development

Build and test the Nylas CLI.

> **Quick Links:** [README](../README.md) | [Commands](COMMANDS.md) | [Architecture](ARCHITECTURE.md)

> **This is the authoritative source for make targets.** Other files reference this document.

---

## Prerequisites

- Go 1.24+ (check with `go version`)
- Make

---

## Make Targets

### Essential Commands

| Target | Description | When to Use |
|--------|-------------|-------------|
| `make ci-full` | **Complete CI pipeline** (quality + tests + cleanup) | Before PRs, releases |
| `make ci` | Quality checks only (no integration tests) | Quick pre-commit |
| `make build` | Build binary to `./bin/nylas` | Development |
| `make clean` | Remove build artifacts | Clean workspace |

### Testing Commands

| Target | Description |
|--------|-------------|
| `make test-unit` | Run unit tests |
| `make test-race` | Run tests with race detector |
| `make test-integration` | Run CLI integration tests |
| `make test-air-integration` | Run Air web UI integration tests |
| `make test-coverage` | Generate coverage report |
| `make test-cleanup` | Clean up test resources |

### Quality Commands

| Target | Description |
|--------|-------------|
| `make lint` | Run golangci-lint |
| `make security` | Run security scan (gosec) |
| `make vuln` | Run vulnerability check (govulncheck) |

**Run `make help` for all available targets.**

---

## Integration Tests

```bash
export NYLAS_API_KEY="your-api-key"
export NYLAS_GRANT_ID="your-grant-id"

make test-integration
```

**CRITICAL:** Air tests create real resources. Always use `make ci-full` for automatic cleanup.

---

## Project Structure

```
cmd/nylas/main.go           # Entry point
internal/
  ├── domain/               # Domain models
  ├── ports/                # Interfaces
  ├── adapters/             # Implementations
  ├── cli/                  # Commands (incl. setup/ for nylas init)
  └── ...
```

---

## Detailed Guides

For contributors, comprehensive guides are available:

- **[Adding Commands](development/adding-command.md)** - Step-by-step guide for new CLI commands
- **[Adding Adapters](development/adding-adapter.md)** - Implementing API adapters
- **[Testing Guide](development/testing-guide.md)** - Unit and integration testing
- **[Debugging](development/debugging.md)** - Debugging tips and techniques

---

**Quick reference:** See `CLAUDE.md` for project overview and AI assistant guidelines.
