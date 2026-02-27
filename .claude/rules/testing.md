# Testing Guidelines

Detailed patterns: `.claude/shared/patterns/go-test-patterns.md`, `integration-test-patterns.md`, `playwright-patterns.md`

## Coverage Targets (authoritative source)

| Package Type | Minimum | Target |
|--------------|---------|--------|
| Core Adapters | 70% | 85%+ |
| Business Logic | 60% | 80%+ |
| CLI Commands | 50% | 70%+ |
| Utilities | 90% | 100% |

## Commands

```bash
make ci-full            # Complete CI pipeline (RECOMMENDED)
make test-unit          # Unit tests only
make test-integration   # CLI integration tests
make test-coverage      # Coverage report
make test-cleanup       # Clean up test resources
```
