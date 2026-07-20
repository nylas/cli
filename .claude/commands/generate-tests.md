---
name: generate-tests
description: Generate unit or integration tests for Go code following project patterns and coverage targets
allowed-tools: Read, Write, Edit, Grep, Glob, Bash(go test:*), Bash(go build:*)
---

# Generate Tests

Generate unit or integration tests for Go code.

**Patterns:** See `.claude/shared/patterns/` for templates:
- `go-test-patterns.md` - Unit test patterns
- `integration-test-patterns.md` - CLI integration tests + rate limiting

**Agent:** See `.claude/agents/test-writer.md` for autonomous test generation.

## Instructions

1. Ask me for:
   - Which file/function to test
   - Test type: unit or integration
   - Specific scenarios to cover (optional)

2. Read the appropriate pattern file for templates.

3. Analyze the code and generate tests following project patterns.

### Integration Tests

See `.claude/rules/testing.md` for location, build tags, and naming. Key additions:
- Always use `skipIfMissingCreds(t)` and `acquireRateLimit(t)`
- Run: `go test -tags=integration ./internal/cli/integration/... -run "TestCLI_Name"`

## Test Categories to Cover

| Category | Description | Examples |
|----------|-------------|----------|
| Happy path | Normal inputs, success cases | Valid email, correct credentials |
| Error cases | Invalid inputs, failures | Empty fields, bad format |
| Edge cases | Boundary conditions | Empty slices, nil values, unicode |
| Method guards | Wrong HTTP methods | GET instead of POST |
| JSON handling | Marshaling/unmarshaling | Invalid JSON, missing fields |

## Verification

After generating tests:
- Tests pass: `go test ./path/to/package/...`
- Linting passes: `golangci-lint run`
- Coverage improved: `make test-coverage`

**Full test commands:** See `/run-tests` for all test targets and environment setup.
