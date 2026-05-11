---
name: review-pr
description: Single-reviewer PR review against hexagonal architecture, CLI standards, security, and test coverage
allowed-tools: Read, Grep, Glob, Bash(git diff:*), Bash(go test:*), Bash(go build:*), Bash(golangci-lint:*), Bash(make security:*)
---

# Review Pull Request

Review code changes following nylas CLI standards and best practices.

## Instructions

1. First, get the diff to review:
```bash
git diff main...HEAD
```

Or for a specific PR:
```bash
gh pr diff <pr-number>
```

2. Review checklist:

### Architecture
- [ ] Changes follow hexagonal architecture (domain → ports → adapters → CLI)
- [ ] No direct dependencies on concrete implementations (use interfaces)
- [ ] New code is in the correct layer/package

### Code Quality
- [ ] Functions are appropriately sized (<50 lines ideal)
- [ ] Error messages are user-friendly with suggestions
- [ ] No hardcoded credentials or secrets
- [ ] Context is passed to all API calls

### CLI Standards
- [ ] Commands follow naming conventions (newXxxCmd)
- [ ] Flags have descriptions and appropriate defaults
- [ ] Help text includes examples
- [ ] Output supports --format flag where appropriate

### Testing
- [ ] Unit tests added/updated for new functionality
- [ ] Mock implementations updated if interface changed
- [ ] Integration tests added for user-facing features
- [ ] Tests pass: `go test ./...`

### Documentation
- [ ] docs/COMMANDS.md or docs/commands/tui.md updated if user-facing changes
- [ ] Code comments for non-obvious logic
- [ ] Examples in command help text

### Security
- [ ] No hardcoded API keys, tokens, or passwords
- [ ] No secrets in logs or error messages
- [ ] Input validation where user data is used
- [ ] No command injection vulnerabilities

3. Run verification:
```bash
# Build
go build ./...

# Lint
golangci-lint run

# Tests
go test ./... -short

# Security scan
make security

# Integration tests (if credentials available)
go test -tags=integration ./internal/cli/integration/...
```

4. Provide feedback in this format:

## Review Output

### Summary
Brief overview of the changes.

### Issues Found
| Severity | File:Line | Issue | Suggestion |
|----------|-----------|-------|------------|
| 🔴 Critical | path/file.go:42 | Description | How to fix |
| 🟡 Warning | path/file.go:100 | Description | How to fix |
| 🔵 Info | path/file.go:15 | Description | How to fix |

### Security Concerns
List any security issues found.

### Verdict
- ✅ **APPROVE** - Ready to merge
- ⚠️ **REQUEST CHANGES** - Must fix issues first
- ❓ **NEEDS DISCUSSION** - Questions to resolve
