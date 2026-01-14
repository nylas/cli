# Security Audit Checklist

Complete security checklist for auditing the Nylas CLI. Use this during security reviews.

---

## 1. Secrets & Credentials (CRITICAL)

```bash
# Hardcoded API keys (Nylas pattern)
Grep: "nyk_v0[a-zA-Z0-9_]{20,}"

# Credential patterns in code
Grep: "(api_key|password|secret|token|credential)\s*[=:]\s*\"[^\"]+"

# Credentials in logs
Grep: "(Print|Log|Debug|Info|Warn|Error).*([Aa]pi[Kk]ey|[Pp]assword|[Ss]ecret)"

# Environment variable exposure
Grep: "os\.Getenv.*([Kk]ey|[Ss]ecret|[Pp]assword|[Tt]oken)"
```

**Pass criteria:** Zero matches outside test files.

---

## 2. Command Injection (CRITICAL)

```bash
# User input in exec.Command
Grep: "exec\.Command\("

# Shell execution
Grep: "os/exec|exec\.Command|syscall\.Exec"
```

**Check each match:**
- [ ] User input sanitized before use?
- [ ] No string concatenation with user data?
- [ ] Using `exec.Command(name, args...)` not shell expansion?

---

## 3. Path Traversal (HIGH)

```bash
# File operations with user input
Grep: "os\.(Open|Create|Remove|Stat|ReadFile|WriteFile)"

# Filepath operations
Grep: "filepath\.(Join|Clean|Abs)"
```

**Check each match:**
- [ ] Paths validated against base directory?
- [ ] No `../` allowed in user input?
- [ ] Using `filepath.Clean()` before operations?

---

## 4. Injection Vulnerabilities (HIGH)

| Type | Pattern to Search | Risk |
|------|-------------------|------|
| SQL Injection | `fmt.Sprintf.*SELECT\|INSERT\|UPDATE` | Database compromise |
| LDAP Injection | User input in LDAP queries | Auth bypass |
| Template Injection | `template.HTML(userInput)` | XSS |
| Header Injection | `Header().Set.*userInput` | Request smuggling |

---

## 5. Cryptographic Issues (MEDIUM)

```bash
# Weak hashing
Grep: "crypto/md5|crypto/sha1"

# Insecure random
Grep: "math/rand[^/]"  # Should use crypto/rand for security

# Hardcoded IVs/salts
Grep: "iv\s*:?=\s*\[\]byte|salt\s*:?=\s*\[\]byte"
```

---

## 6. Network Security (MEDIUM)

```bash
# Insecure HTTP
Grep: "http://[^\"]*api\.|http://[^\"]*nylas"

# TLS skip verify
Grep: "InsecureSkipVerify:\s*true"

# Missing timeouts
Grep: "&http\.Client\{\}" # Should have Timeout set
```

---

## 7. Error Handling (LOW)

```bash
# Stack traces to users
Grep: "debug\.PrintStack|runtime\.Stack"

# Internal errors exposed
Grep: "fmt\.Errorf.*%v.*err\)" # Check if exposing internals
```

---

## 8. Dependency Audit

```bash
# Run vulnerability check
make vuln

# Check for outdated dependencies
go list -m -u all
```

---

## OWASP Top 10 Mapping

| OWASP Category | CLI Relevance | Check |
|----------------|---------------|-------|
| A01 Broken Access Control | API key validation | Auth before operations |
| A02 Cryptographic Failures | Credential storage | Use system keyring |
| A03 Injection | Command/path injection | Input validation |
| A04 Insecure Design | Threat modeling | This audit |
| A05 Security Misconfiguration | Default settings | Secure defaults |
| A06 Vulnerable Components | Dependencies | `make vuln` |
| A07 Auth Failures | Token handling | No credential logging |
| A08 Data Integrity | Config file tampering | Validate config |
| A09 Logging Failures | Missing audit trail | Sufficient logging |
| A10 SSRF | URL handling | Validate URLs |

---

## Go-Specific Vulnerabilities

| Issue | Pattern | Fix |
|-------|---------|-----|
| Race conditions | Shared state without mutex | Use `sync.Mutex` or channels |
| Unsafe pointer | `unsafe.Pointer` | Avoid or audit carefully |
| Integer overflow | Large user input to int | Validate ranges |
| Nil pointer deref | No nil checks | Check before deref |
| Goroutine leak | Unbounded goroutines | Use context cancellation |
| Resource exhaustion | No limits on input size | Set max limits |

---

## Quick Commands

```bash
# Full security scan
make security && make vuln

# Security-focused lint
golangci-lint run --enable gosec,bodyclose,noctx --timeout=5m

# Check for secrets in git history
git log -p --all -S "nyk_v0" -- "*.go"

# Audit specific file
Read: <file> then apply checklist
```

---

## Passed Checks Template

- [ ] No hardcoded secrets
- [ ] No credential logging
- [ ] No command injection vectors
- [ ] No path traversal vulnerabilities
- [ ] Dependencies free of known CVEs
- [ ] TLS properly configured
- [ ] Input validation in place
