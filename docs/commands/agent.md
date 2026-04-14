# Agent Accounts

Manage Nylas agent accounts from the CLI.

Agent accounts are managed email identities backed by provider `nylas`. Unlike OAuth grants, they do not require a third-party mailbox connection. The CLI keeps connector setup automatic and always uses `provider=nylas` for this command group.

## Commands

```bash
nylas agent list
nylas agent create <email>
nylas agent create <email> --app-password <password>
nylas agent delete <agent-id|email>
nylas agent status
```

## List Agent Accounts

```bash
nylas agent list
nylas agent list --json
```

**Example output:**
```bash
$ nylas agent list

Agent Accounts (2)

1. support@yourapp.nylas.email            active
   ID: 11111111-1111-1111-1111-111111111111

2. me@yourapp.nylas.email                 active
   ID: 22222222-2222-2222-2222-222222222222
```

## Create Agent Account

```bash
nylas agent create me@yourapp.nylas.email
nylas agent create me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
nylas agent create support@yourapp.nylas.email --json
```

Behavior:
- always creates a grant with `provider=nylas`
- automatically creates the `nylas` connector first if it does not exist
- stores the created grant locally like other authenticated accounts
- optionally sets `settings.app_password` on the grant for IMAP/SMTP mail client access

**Example output:**
```bash
$ nylas agent create me@yourapp.nylas.email

✓ Agent account created successfully!

Email:      me@yourapp.nylas.email
Provider:   nylas
Status:     valid
```

### `--app-password`

Use `--app-password` when you want the agent account to work with a standard mail client over IMAP/SMTP submission.

```bash
nylas agent create me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
```

Requirements:
- 18 to 40 characters
- printable ASCII only, with no spaces
- at least one uppercase letter
- at least one lowercase letter
- at least one digit

When set, the agent account email becomes the mail-client username and the app password is used for IMAP/SMTP authentication.

## Delete Agent Account

```bash
nylas agent delete 12345678-1234-1234-1234-123456789012
nylas agent delete me@yourapp.nylas.email --yes
```

Deleting an agent account revokes the underlying `provider=nylas` grant.

## Connector Status

```bash
nylas agent status
nylas agent status --json
```

This reports:
- whether the `nylas` connector is available
- whether agent accounts already exist
- which managed accounts are currently configured

## Relationship to Inbound

`nylas agent` and `nylas inbound` are different features:
- `nylas agent` creates managed agent accounts backed by provider `nylas`
- `nylas inbound` creates inbound inboxes backed by provider `inbox`

Both can use `@yourapp.nylas.email` addresses, but they are separate command groups and separate provider types.

Agent accounts can also expose the mailbox over IMAP and SMTP submission when an `app_password` is configured at creation time.

## Sending Email from Agent Accounts

When the active grant is an agent account (`provider=nylas`):
- `nylas email send` automatically uses the managed transactional send path
- the sender address is taken from the active grant email
- GPG signing/encryption is not supported on that managed transactional path
- stored signatures via `--signature-id` are not supported on that managed transactional path

## See Also

- [Email commands](email.md)
- [Inbound email](inbound.md)
