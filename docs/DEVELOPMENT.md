# Development

Build and test the Nylas CLI.

> **Quick Links:** [README](../README.md) | [Commands](COMMANDS.md) | [Architecture](ARCHITECTURE.md)

> **This is the authoritative source for make targets.** Other files reference this document.

---

## Prerequisites

- Go 1.26+ (check with `go version`)
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
| `make test-coverage` | Generate coverage report |
| `make test-cleanup` | Clean up test resources |

### Quality Commands

| Target | Description |
|--------|-------------|
| `make lint` | Run golangci-lint |
| `make security` | Run security scan (gosec) |
| `make vuln` | Run vulnerability check (govulncheck) |

**Run `make help` for all available targets.**

### Docker

```bash
docker build -t nylas-cli:dev .
docker run --rm nylas-cli:dev --version
```

See [Docker](development/docker.md) for credential and release-image guidance.

---

## Integration Tests

```bash
export NYLAS_API_KEY="your-api-key"
export NYLAS_GRANT_ID="your-grant-id"

make test-integration
```

**CRITICAL:** Integration tests create real resources. Always use `make ci-full` for automatic cleanup.

### OAuth authorization server tests

`internal/cli/integration/oauth_test.go` drives a real dashboard-account
authorization server instead of the Nylas API, so it needs its own variable and
skips without it:

```bash
NYLAS_OAUTH_AS_URL=http://localhost:3001 \
  go test -tags integration -run TestOAuthAS ./internal/cli/integration/
```

Requirements on the server side:

- dashboard-account running (in a Tilt stack it is on port 3001)
- `/dev` routes enabled — `ENABLE_DEV_ROUTES=true` or `IS_E2E=true`. The tests
  seed their own user, consent grant and authorization code through them, which
  is what lets the token exchange run without a browser.

The tests front the server with a small proxy that rewrites the issuer origin in
the discovery document. dashboard-account builds every advertised endpoint from
`OAUTH_ISSUER`, and in a local stack that is frequently a tunnel hostname that is
stale or unreachable; the client under test is spec-correct and follows whatever
the document says. If you would rather fix it at the source, set
`OAUTH_ISSUER=http://localhost:3001` in `infra/.env.local` and restart the
service — the proxy then rewrites nothing.

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
- **[Docker](development/docker.md)** - Container build, run, and release-image notes

---

**Quick reference:** See `CLAUDE.md` for project overview and AI assistant guidelines.
