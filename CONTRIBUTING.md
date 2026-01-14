# Contributing to Nylas CLI

Thank you for your interest in contributing to the Nylas CLI! This document provides guidelines for contributing.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/cli.git`
3. Create a branch: `git checkout -b feat/your-feature`
4. Make your changes
5. Submit a pull request

## Development Setup

```bash
# Install dependencies
go mod download

# Build
make build

# Run unit tests
make test-unit

# Run linter
make lint

# Full CI check (fmt + vet + lint + test + security + build)
make ci
```

## Code Standards

### Architecture
This project follows **hexagonal architecture** (ports and adapters):
- `internal/domain/` - Business entities
- `internal/ports/` - Interface definitions
- `internal/adapters/` - Implementations
- `internal/cli/` - CLI commands

### Go Best Practices
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Pass `context.Context` to blocking operations

### Testing Requirements

| Change Type | Unit Tests | Integration Tests |
|-------------|------------|-------------------|
| New feature | Required | Required |
| Bug fix | Required | If API-related |
| New command | Required | Required |

```bash
# Run unit tests
make test-unit

# Run integration tests (requires credentials)
NYLAS_API_KEY="..." NYLAS_GRANT_ID="..." make test-integration
```

## Pull Request Process

1. **Create focused PRs** - One feature or fix per PR
2. **Write tests** - All new code must have tests
3. **Update documentation** - Update docs/COMMANDS.md if CLI changes
4. **Pass all checks** - Run `make ci` before submitting
5. **Write clear commit messages** - Use conventional format:
   - `feat: add calendar sync command`
   - `fix: resolve nil pointer in email send`
   - `docs: update webhook examples`

## Security

### Never Commit
- API keys, tokens, or passwords
- `.env` files
- Credential files (`.pem`, `.key`, etc.)
- Personal configuration

### Before Submitting
```bash
# Run security scan
make security

# Check for secrets in your changes
git diff --cached | grep -iE "(api_key|password|secret|token)" || echo "Clean"
```

## AI-Assisted Contributions

We welcome AI-assisted contributions (GitHub Copilot, Claude, ChatGPT, etc.).

### Guidelines
- **Disclose AI usage** - Mention in PR description if AI assisted
- **Review thoroughly** - You are responsible for all submitted code
- **Test everything** - AI-generated code must pass all tests
- **Understand the code** - Don't submit code you don't understand

### Example PR Description
```markdown
## Summary
Added retry logic to email send command.

## AI Assistance
Used Claude Code to help implement exponential backoff logic.
Reviewed and tested all generated code.

## Testing
- Added unit tests for retry mechanism
- Manually tested with rate-limited API responses
```

## Questions?

- Open an issue for bugs or feature requests
- Check existing issues before creating new ones

Thank you for contributing!
