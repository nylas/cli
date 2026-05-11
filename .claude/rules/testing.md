---
paths:
  - "**/*_test.go"
  - "**/*.go"
---

# Testing Guidelines

Consolidated testing rules for the Nylas CLI project.

**Detailed Patterns:**
- Go unit tests: `.claude/shared/patterns/go-test-patterns.md`
- Integration tests: `.claude/shared/patterns/integration-test-patterns.md`
- Playwright E2E: `.claude/shared/patterns/playwright-patterns.md`

---

## Test Organization

### Unit Tests
- **Location:** Alongside source (`*_test.go`)
- **Function:** `TestFunctionName_Scenario`
- **Pattern:** Table-driven with `t.Run(tt.name, ...)`

### CLI Integration Tests
- **Location:** `internal/cli/integration/`
- **Build tags:** `//go:build integration` and `// +build integration`
- **Function:** `TestCLI_CommandName`

### Air Integration Tests
- **Location:** `internal/air/integration_*.go`
- **Build tags:** `//go:build integration` and `// +build integration`
- **Function:** `TestIntegration_FeatureName`

---

## Rate Limiting (CRITICAL)

**See:** `.claude/shared/patterns/integration-test-patterns.md` for patterns and config.

---

## Test Coverage

> **This is the authoritative source for coverage goals.** Other files reference this section.

| Package Type | Minimum | Target |
|--------------|---------|--------|
| Core Adapters | 70% | 85%+ |
| Business Logic | 60% | 80%+ |
| CLI Commands | 50% | 70%+ |
| Utilities | 90% | 100% |

```bash
make test-coverage  # Generates coverage.html and opens in browser
```

---

## Quick Reference

**See:** `/run-tests` for full command details.

```bash
make test-unit                   # Unit tests only
make test-integration            # CLI integration tests
make test-cleanup                # Clean up test resources
```

---

## Key Principles

1. Test behavior, not implementation
2. Table-driven tests with `t.Run()` (see `go-test-patterns.md`)
3. Mock external dependencies
4. Clean up with `t.Cleanup()`
5. Use `t.Parallel()` for independent tests
6. Use `acquireRateLimit(t)` for API calls (see `integration-test-patterns.md`)
