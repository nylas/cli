---
name: generate-tests
description: Generate comprehensive unit and integration tests for Go code
allowed-tools: Read, Write, Edit, Grep, Glob, Bash(go test:*), Bash(go build:*)
---

# Generate Tests

Generate comprehensive unit and integration tests for Go code.

**Patterns:** See `.claude/shared/patterns/` for templates:
- `go-test-patterns.md` - Unit test patterns
- `integration-test-patterns.md` - CLI integration tests
- `playwright-patterns.md` - E2E browser tests

**Agent:** See `.claude/agents/test-writer.md` for autonomous test generation.

## Instructions

1. Ask me for:
   - Which file/function to test
   - Test type: unit, integration, or E2E
   - Specific scenarios to cover (optional)

2. Read the appropriate pattern file for templates.

3. Analyze the code and generate tests following project patterns.

## Test Categories to Cover

| Category | Description | Examples |
|----------|-------------|----------|
| Happy path | Normal inputs, success cases | Valid email, correct credentials |
| Error cases | Invalid inputs, failures | Empty fields, bad format |
| Edge cases | Boundary conditions | Empty slices, nil values, unicode |
| Method guards | Wrong HTTP methods | GET instead of POST |
| JSON handling | Marshaling/unmarshaling | Invalid JSON, missing fields |

## Test Naming Convention

| Type | Pattern | Example |
|------|---------|---------|
| Unit test | `TestFunctionName_Scenario` | `TestParseEmail_ValidInput` |
| CLI integration | `TestCLI_CommandName` | `TestCLI_EmailSend` |
| HTTP handler | `TestHandleFeature_Scenario` | `TestHandleAISummarize_EmptyBody` |

## Verification

After generating tests:
- Tests pass: `go test ./path/to/package/...`
- Linting passes: `golangci-lint run`
- Coverage improved: `make test-coverage`

**Full test commands:** See `/run-tests` for all test targets and environment setup.
