---
name: security-scan
description: Perform security analysis - scan for secrets, vulnerabilities, and security issues
allowed-tools: Read, Grep, Glob, Bash(grep:*), Bash(make security:*), Bash(golangci-lint:*)
---

# Security Scan

Perform a comprehensive security analysis of the codebase.

Target: $ARGUMENTS (leave empty for full scan)

---

## Threat Model

This CLI:
- Stores API credentials (should use system keyring)
- Makes HTTP requests to Nylas API
- Reads/writes local files
- Executes based on user input

---

## Instructions

### 1. Secret Detection
```bash
# Check for hardcoded API keys
grep -rE "nyk_v0[a-zA-Z0-9_]{20,}" --include="*.go" . | grep -v "_test.go"

# Check for credential patterns
grep -rE "(api_key|password|secret|token)\s*[=:]\s*\"[^\"]+\"" --include="*.go" . | grep -v "_test.go" | grep -v "mock.go"

# Check for potential credential logging
grep -rE "fmt\.(Print|Fprint|Sprint|Log).*([Aa]pi[Kk]ey|[Pp]assword|[Ss]ecret|[Tt]oken)" --include="*.go" .
```

### 2. Sensitive File Check
```bash
# Check for sensitive files in repo
find . -name "*.env*" -o -name "*.pem" -o -name "*.key" -o -name "*credential*" -o -name "*secret*" 2>/dev/null | grep -v ".git"

# Check git history for sensitive files
git log --all --name-only --pretty=format: | grep -E "\.(env|pem|key|p12)$" | sort -u
```

### 3. Dependency Vulnerabilities
```bash
# Check for known vulnerabilities in Go modules
go list -m all
# Review any outdated or suspicious dependencies
```

### 4. Code Security Review

Check for these vulnerabilities:

| Category | What to Look For |
|----------|------------------|
| **Command Injection** | User input in `exec.Command()`, `os/exec` |
| **Path Traversal** | User input in file paths without sanitization |
| **Insecure HTTP** | `http://` instead of `https://` for APIs |
| **Weak Crypto** | MD5, SHA1 for security purposes |
| **Error Exposure** | Stack traces or internal errors exposed to users |

### 5. Authentication & Authorization
- [ ] API keys stored securely (keyring, not plaintext)
- [ ] No credentials in command-line arguments (visible in ps)
- [ ] Tokens not logged or printed
- [ ] Secure credential input (no echo)

### 6. Run Automated Scans
```bash
# Run project security scan
make security

# Run Go security linter (if available)
golangci-lint run --enable gosec
```

## Output Format

### Security Report

**Scan Date:** [current date]
**Scope:** [full repo / specific path]

#### 🔴 Critical Issues
Issues that must be fixed immediately.

#### 🟠 High Severity
Significant security risks.

#### 🟡 Medium Severity
Potential vulnerabilities.

#### 🟢 Low Severity / Informational
Best practice improvements.

#### ✅ Passed Checks
- [ ] No hardcoded secrets
- [ ] No credential logging
- [ ] No sensitive files in repo
- [ ] Dependencies up to date
- [ ] Secure API communication

### Recommendations
Prioritized list of fixes needed.
