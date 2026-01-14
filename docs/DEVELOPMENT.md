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
| `make install` | Install binary to GOPATH/bin | Local installation |
| `make clean` | Remove build artifacts | Clean workspace |
| `make clean-cache` | Clean Go caches (module, test, build) | Fix cache corruption |

### Testing Commands

| Target | Description |
|--------|-------------|
| `make test-unit` | Run unit tests |
| `make test-race` | Run tests with race detector |
| `make test-integration` | Run CLI integration tests (rate limited) |
| `make test-integration-fast` | Fast integration tests (skip LLM-dependent) |
| `make test-coverage` | Generate coverage report |
| `make test-cleanup` | Clean up test resources |
| `make test-pkg PKG=<name>` | Test specific package (e.g., `make test-pkg PKG=email`) |

### Quality Commands

| Target | Description |
|--------|-------------|
| `make fmt` | Format code with go fmt |
| `make vet` | Run go vet static analysis |
| `make lint` | Run golangci-lint |
| `make security` | Scan for hardcoded credentials |
| `make vuln` | Run vulnerability check (govulncheck) |

### Utility Commands

| Target | Description |
|--------|-------------|
| `make deps` | Update and tidy dependencies |
| `make run ARGS="..."` | Build and run with arguments |
| `make check-context` | Check Claude Code context size |
| `make help` | Show all available targets |

**Run `make help` for detailed descriptions.**

---

## Integration Tests

```bash
export NYLAS_API_KEY="your-api-key"
export NYLAS_GRANT_ID="your-grant-id"

make test-integration
```

**Note:** Integration tests create real resources. Use `make ci-full` for automatic cleanup.

---

## Detailed Guides

For contributors, comprehensive guides are available:

- **[Adding Commands](development/adding-command.md)** - Step-by-step guide for new CLI commands
- **[Adding Adapters](development/adding-adapter.md)** - Implementing API adapters
- **[Testing Guide](development/testing-guide.md)** - Unit and integration testing
- **[Debugging](development/debugging.md)** - Debugging tips and techniques

---

**Quick reference:** See `CLAUDE.md` for project overview and AI assistant guidelines.
