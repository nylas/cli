# Analyze Test Coverage

Analyze test coverage and identify untested code paths that need tests.

Focus area: $ARGUMENTS

## Instructions

1. **Generate coverage report**

   ```bash
   # Full coverage report
   go test ./... -coverprofile=coverage.out -covermode=atomic

   # View coverage summary
   go tool cover -func=coverage.out

   # Generate HTML report (for detailed analysis)
   go tool cover -html=coverage.out -o coverage.html

   # Coverage by package
   go test ./... -cover
   ```

2. **Analyze coverage by component**

   Check coverage for each layer:

   ```bash
   # Domain layer
   go test ./internal/domain/... -cover

   # Adapters
   go test ./internal/adapters/... -cover

   # CLI commands
   go test ./internal/cli/... -cover

   # Specific package
   go test ./internal/cli/{package}/... -coverprofile=pkg.out
   go tool cover -func=pkg.out
   ```

3. **Identify coverage gaps**

   Look for:
   - Functions with 0% coverage
   - Error handling paths not tested
   - Edge cases not covered
   - New code without tests

   ```bash
   # Find functions with low coverage
   go tool cover -func=coverage.out | grep -E "0\.0%|[0-9]\.[0-9]%"

   # Find untested files
   go tool cover -func=coverage.out | grep "0.0%"
   ```

4. **Prioritize test additions**

   High priority (test first):
   - Public API functions
   - Error handling paths
   - Security-sensitive code (auth, credentials)
   - Data validation logic

   Medium priority:
   - Helper functions
   - Format/display logic
   - Edge cases

   Lower priority:
   - Simple getters/setters
   - Demo/mock implementations

5. **Generate test recommendations**

   For each uncovered function, suggest:
   - Test file location
   - Test function name
   - Test cases needed (happy path, error cases, edge cases)

## Coverage Targets

| Component | Target | Rationale |
|-----------|--------|-----------|
| `internal/domain/` | 80%+ | Core business logic |
| `internal/adapters/nylas/` | 70%+ | API integration |
| `internal/cli/*/` | 60%+ | Command handling |
| `internal/cli/common/` | 80%+ | Shared utilities |

## Test Gap Analysis Template

**Test patterns:** See `.claude/shared/patterns/go-test-patterns.md` for table-driven test templates.

For each uncovered function, document:

```markdown
### Function: `{PackageName}.{FunctionName}`

**File:** `{file_path}:{line_number}`
**Current Coverage:** {X}%

**Missing Test Cases:**
1. Happy path - {description}
2. Error case - {description}
3. Edge case - {description}
```

## Common Coverage Issues

| Issue | Solution |
|-------|----------|
| Error paths untested | Add tests with mock returning errors |
| Context cancellation | Add test with cancelled context |
| Nil pointer checks | Add test with nil inputs |
| Pagination logic | Test first page, middle page, last page |
| Empty results | Test with empty slice/nil response |

## Report Format

Generate a coverage report like:

```markdown
# Coverage Report - {date}

## Summary
- Overall: {X}%
- Domain: {X}%
- Adapters: {X}%
- CLI: {X}%

## Critical Gaps (0% coverage)
1. `pkg.Function1` - {file}:{line}
2. `pkg.Function2` - {file}:{line}

## Low Coverage (<50%)
1. `pkg.Function3` - {X}% - Missing: error handling
2. `pkg.Function4` - {X}% - Missing: edge cases

## Recommendations
1. Add tests for {function} - High priority (security)
2. Add tests for {function} - Medium priority (common path)
```

## Cleanup

```bash
# Remove coverage files when done
rm -f coverage.out coverage.html pkg.out
```

## Checklist

- [ ] Generated coverage report
- [ ] Analyzed coverage by layer
- [ ] Identified critical gaps (0% coverage)
- [ ] Identified low coverage areas (<50%)
- [ ] Prioritized test additions
- [ ] Generated test recommendations
- [ ] Created/assigned tasks for missing tests
