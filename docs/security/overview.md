# Security

Secure credential storage and best practices for the Nylas CLI.

> **Quick Links:** [README](../../README.md) | [Commands](../COMMANDS.md) | [Development](../DEVELOPMENT.md)

---

## Credential Storage

```bash
nylas auth config            # Configure API credentials (stored securely)
```

**Storage locations:**
- **macOS:** Keychain
- **Linux:** Secret Service (GNOME Keyring/KWallet)
- **Windows:** Credential Manager
- **Config:** `~/.config/nylas/config.yaml`

---

## Testing

```bash
# Set credentials for integration tests
export NYLAS_API_KEY="your-api-key"
export NYLAS_GRANT_ID="your-grant-id"

# Run tests
make ci-full   # Complete CI pipeline with tests and cleanup
```

---

## Protected Files

The `.gitignore` blocks these patterns to prevent credential commits:

**Environment & Credentials:**
- `.env`, `.env.*`, `*.env`
- `credentials.json`, `credentials.yaml`, `*credentials*`
- `secrets.json`, `secrets.yaml`, `*secrets*`

**Keys & Tokens:**
- `*.key`, `*.pem`, `*.p12`, `*.pfx`
- `api_key*`, `*token*`, `oauth_token*`
- `id_rsa*`, `id_dsa*`, `*.gpg`

---

## Security Scan

```bash
make security               # Run before commits
```

**Checks:**
- No hardcoded API keys (`nyk_v0` pattern)
- No credential logging
- No sensitive files staged

---

## Best Practices

**Users:**
- Never commit credentials
- Use `--yes` flag carefully (skips confirmations)
- Rotate API keys regularly

**Developers:**
- Run `make security` before commits
- Never log credentials
- Validate all user input

---

**Detailed guide:** See `docs/security/practices.md` for network security, input validation, and OWASP compliance details.
