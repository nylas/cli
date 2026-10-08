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
| `dashboard_user_token`, `dashboard_org_token` | `ports.KeyDashboardUserToken`, `ports.KeyDashboardOrgToken` | Dashboard session tokens |
| `dashboard_user_public_id`, `dashboard_org_public_id` | `ports.KeyDashboardUserPublicID`, `ports.KeyDashboardOrgPublicID` | Signed-in user and active organization |
| `dashboard_app_id`, `dashboard_app_region` | `ports.KeyDashboardAppID`, `ports.KeyDashboardAppRegion` | Active application (`nylas dashboard apps use`) |
| `dashboard_dpop_key` | `ports.KeyDashboardDPoPKey` | Ed25519 seed the dashboard session is DPoP-bound to |
| `dashboard_session_origin`, `dashboard_session_expires_at` | `ports.KeyDashboardSessionOrigin`, `ports.KeyDashboardSessionExpiresAt` | Set on a session exchanged from `nylas oauth login` |
| `dashboard_session_server` | `ports.KeyDashboardSessionServer` | Servers the dashboard session was issued for; it is never sent elsewhere |
| `oauth_access_token`, `oauth_refresh_token` | `ports.KeyOAuthAccessToken`, `ports.KeyOAuthRefreshToken` | `nylas oauth login` session |
| `oauth_expires_at`, `oauth_scope`, `oauth_resource` | `ports.KeyOAuthExpiresAt`, `ports.KeyOAuthScope`, `ports.KeyOAuthResource` | Session expiry, granted scopes, RFC 8707 resource |
| `oauth_issuer`, `oauth_server_url` | `ports.KeyOAuthIssuer`, `ports.KeyOAuthServerURL` | Authorization server the session came from; it is never sent elsewhere |

A value larger than one system keychain item (2560 bytes on Windows) is split
into chunks named `<key>.chunk.<id>.<n>`, and the key itself holds a
`nylas:chunked:v1:` header. A CLI release from before chunking reads that
header as the value, so downgrading after storing a large token means logging
in again. Chunks a crashed write left behind are not cleaned up, and nothing
reads them.

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

### OAuth Login Callback

`nylas oauth login` and `nylas auth login` receive the redirect on a loopback
callback server (`127.0.0.1`, plus `::1` for `localhost`):

- The expected `state` is set before the browser opens, and is compared in
  constant time.
- Only a request carrying that state can end the login, with a code or an
  `error`. Any other request to the port is refused with 400 and the login
  keeps waiting, so a stray or hostile request cannot abort it.
- The `error` value is shown only if it matches `^[a-z_]{1,64}$`, so control
  characters from the URL never reach the terminal.

### Session Locks

Session writes are serialised across CLI processes by advisory file locks,
always taken in this order:

| Lock | Guards |
|------|--------|
| `dashboard-session.lock` | Dashboard session keys: every login, logout, refresh, renewal from OAuth, org switch, app selection and reset, and creating the DPoP key |
| `oauth-session.lock` | OAuth session keys (login, refresh, logout, reset) |
| `.secrets.lock` | Each read and write of the encrypted file store |

With the encrypted file store the first two sit in the config directory; with
the system keyring they sit in `~/.config/nylas/` under the account's home
directory, whatever `XDG_CONFIG_HOME` is.

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
