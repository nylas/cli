# Security

Secure credential storage and best practices for the Nylas CLI.

> **Quick Links:** [README](../../README.md) | [Commands](../COMMANDS.md) | [Development](../DEVELOPMENT.md)

---

## Credential Storage

```bash
nylas auth config            # Configure API credentials (stored securely)
```

### Keyring Storage

Secrets are stored in the system keyring under service name `"nylas"`:

| Key | Constant | Description |
|-----|----------|-------------|
| `client_id` | `ports.KeyClientID` | Nylas Application/Client ID |
| `api_key` | `ports.KeyAPIKey` | Nylas API key (Bearer auth) |
| `client_secret` | `ports.KeyClientSecret` | Provider OAuth secret (Google/Microsoft) |
| `org_id` | `ports.KeyOrgID` | Nylas Organization ID |

Grant IDs, emails, providers, and the local default grant are non-secret metadata.
They are stored in the grant cache at `filepath.Join(os.UserCacheDir(), "nylas", "grants.json")`.
Keyring remains secrets-only.

### Implementation Files

| File | Purpose |
|------|---------|
| `internal/ports/secrets.go` | Key constants (`KeyClientID`, `KeyAPIKey`, etc.) |
| `internal/adapters/keyring/keyring.go` | System keyring implementation |
| `internal/adapters/grantcache/cache.go` | File-backed non-secret grant metadata/default cache |
| `internal/app/auth/config.go` | `SetupConfig()` saves credentials to keyring |

### Platform Backends

- **macOS:** Keychain
- **Linux:** Secret Service (GNOME Keyring/KWallet)
- **Windows:** Credential Manager
- **Fallback:** Encrypted file store (`~/.config/nylas/`)

### Environment Override

```bash
NYLAS_DISABLE_KEYRING=true   # Force encrypted file store (useful for testing/CI)
```

### Config File

Non-sensitive settings stored in `~/.config/nylas/config.yaml`:
- Region (us/eu)
- Callback port
- Local default grant mirror

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
