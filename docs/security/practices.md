# Security Practices

Comprehensive security guidelines for the Nylas CLI, covering network security, input validation, and OWASP compliance.

> **Quick Links:** [Security Overview](overview.md) | [Development](../DEVELOPMENT.md)

---

## Network Security

### HTTPS Enforcement

All API communications use HTTPS with TLS 1.2+:
- API endpoint: `https://api.us.nylas.com` (US region)
- OAuth endpoints: `https://api.nylas.com/v3/connect/auth`
- No HTTP fallback - connections fail if TLS unavailable

### Certificate Validation

```go
// Certificate pinning not used (relies on system CA store)
// TLS configuration enforces minimum version
tlsConfig := &tls.Config{
    MinVersion: tls.VersionTLS12,
}
```

### Timeout Configuration

All HTTP requests have appropriate timeouts:
- Connection timeout: 30 seconds
- Request timeout: 2 minutes (configurable)
- Idle connection timeout: 90 seconds

---

## Input Validation

### Command Arguments

All user input is validated before processing:

```bash
# Email validation
nylas email send --to "invalid"  # Error: invalid email format

# ID validation
nylas email read ""              # Error: message ID required

# Path validation
nylas config --file "../../../etc/passwd"  # Error: invalid path
```

### Sanitization Rules

| Input Type | Validation |
|------------|------------|
| Email addresses | RFC 5322 format |
| Grant IDs | Alphanumeric + underscores |
| Webhook URLs | HTTPS required, valid URL format |
| File paths | No path traversal, must be in allowed directories |
| API keys | Format validation (nyk_v0 prefix) |

### Shell Injection Prevention

The CLI protects against shell injection in all commands:

```go
// Safe: arguments passed as separate parameters
cmd := exec.Command("program", arg1, arg2)

// Never: string concatenation with user input
// cmd := exec.Command("sh", "-c", "program " + userInput)
```

---

## OWASP Top 10 Compliance

### A01:2021 - Broken Access Control

- **Credential isolation**: Each grant has separate credentials
- **Scope validation**: OAuth scopes enforced by Nylas API
- **Local file permissions**: Config files created with 0600 permissions

### A02:2021 - Cryptographic Failures

- **Credential storage**: System keyring (encrypted at rest)
- **Transport encryption**: TLS 1.2+ for all API calls
- **No plaintext secrets**: API keys never logged or displayed in full

### A03:2021 - Injection

- **SQL injection**: N/A (no database)
- **Command injection**: All shell commands use parameterized execution
- **Template injection**: No user-controlled templates

### A04:2021 - Insecure Design

- **Principle of least privilege**: Commands only access required resources
- **Defense in depth**: Multiple validation layers
- **Secure defaults**: Conservative default settings

### A05:2021 - Security Misconfiguration

- **No default credentials**: Must configure before use
- **Error messages**: Don't expose internal details
- **Hardening**: Security scan catches common misconfigurations

### A06:2021 - Vulnerable Components

- **Dependency scanning**: `make vuln` checks for known vulnerabilities
- **Regular updates**: Dependencies kept current
- **Minimal dependencies**: Only essential packages included

### A07:2021 - Authentication Failures

- **OAuth 2.0**: Industry-standard authentication
- **Token refresh**: Automatic token refresh when possible
- **Session management**: Tokens stored securely in keyring

### A08:2021 - Software and Data Integrity

- **Code signing**: Releases include SHA256 checksums
- **Update verification**: `nylas update` verifies checksums
- **No code execution**: CLI doesn't execute untrusted code

### A09:2021 - Security Logging and Monitoring

- **Audit trail**: Commands can be logged for compliance
- **Error logging**: Failures logged without sensitive data
- **No credential logging**: API keys/tokens never appear in logs

### A10:2021 - Server-Side Request Forgery

- **URL validation**: Webhook URLs validated before registration
- **No arbitrary requests**: CLI only makes requests to Nylas API

---

## Credential Protection

### Storage Locations

| Platform | Storage | Encryption |
|----------|---------|------------|
| macOS | Keychain | AES-256-GCM |
| Linux | Secret Service | Provider-dependent |
| Windows | Credential Manager | DPAPI |

### Credential Lifecycle

```bash
# Store credentials securely
nylas auth config                 # Prompts for API key, stores in keyring

# Credentials never displayed
nylas auth status                 # Shows ******* for sensitive values

# Remove credentials
nylas auth logout                 # Removes from keyring
```

### Environment Variables

For automation, credentials can be set via environment variables:

```bash
export NYLAS_API_KEY="nyk_v0_..."      # API key
export NYLAS_GRANT_ID="grant_..."      # Default grant
export NYLAS_API_URI="https://..."     # Custom API endpoint
```

**Note:** Environment variables override stored credentials.

---

## Security Scanning

### Pre-Commit Checks

```bash
make security                     # Run before commits
```

**Checks performed:**
1. No hardcoded API keys (`nyk_v0` pattern)
2. No credential logging
3. No sensitive files staged
4. No TODO/FIXME security comments

### Vulnerability Scanning

```bash
make vuln                         # Check for known vulnerabilities
```

Uses `govulncheck` to scan dependencies for CVEs.

### Static Analysis

```bash
golangci-lint run                 # Includes security linters
```

Security-focused linters:
- `gosec` - Security rules
- `errcheck` - Unchecked errors
- `ineffassign` - Unused assignments

---

## Reporting Security Issues

If you discover a security vulnerability:

1. **Do not** open a public issue
2. Email security concerns to the maintainers
3. Include steps to reproduce
4. Allow time for fix before disclosure

---

## Compliance Checklist

For enterprise deployments:

- [ ] API keys rotated regularly
- [ ] Minimal OAuth scopes requested
- [ ] Logs do not contain PII
- [ ] Credentials stored in approved vault
- [ ] Network access restricted to required endpoints
- [ ] Regular security scans performed
- [ ] Updates applied promptly

---

**See also:**
- [Security Overview](overview.md)
- [Development Guide](../DEVELOPMENT.md)
- [OWASP Top 10](https://owasp.org/Top10/)
