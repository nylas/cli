# Agent

Manage Nylas agent resources from the CLI.

Agent accounts are managed email identities backed by provider `nylas`. Unlike OAuth grants, they do not require a third-party mailbox connection. Account operations live under `nylas agent account`, while `nylas agent status` reports connector and account readiness.

## Commands

```bash
nylas agent account list
nylas agent account create <email>
nylas agent account get <agent-id|email>
nylas agent account delete <agent-id|email>
nylas agent policy list
nylas agent policy create --name <name>
nylas agent policy get <policy-id>
nylas agent policy read <policy-id>
nylas agent policy update <policy-id> --name <name>
nylas agent policy delete <policy-id>
nylas agent rule list
nylas agent rule read <rule-id>
nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action mark_as_spam
nylas agent rule update <rule-id> --name "Updated Rule" --description "Block example.org"
nylas agent rule delete <rule-id>
nylas agent status
```

## List Agent Accounts

```bash
nylas agent account list
nylas agent account list --json
```

**Example output:**
```bash
$ nylas agent account list

Agent Accounts (2)

1. support@yourapp.nylas.email            active
   ID: 11111111-1111-1111-1111-111111111111

2. me@yourapp.nylas.email                 active
   ID: 22222222-2222-2222-2222-222222222222
```

## Create Agent Account

```bash
nylas agent account create me@yourapp.nylas.email
nylas agent account create me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
nylas agent account create me@yourapp.nylas.email --policy-id 12345678-1234-1234-1234-123456789012
nylas agent account create support@yourapp.nylas.email --json
```

Behavior:
- always creates a grant with `provider=nylas`
- automatically creates the `nylas` connector first if it does not exist
- stores the created grant locally like other authenticated accounts
- optionally sets `settings.app_password` on the grant for IMAP/SMTP mail client access
- optionally sets `settings.policy_id` on the grant so the new account starts with an attached policy

**Example output:**
```bash
$ nylas agent account create me@yourapp.nylas.email

✓ Agent account created successfully!

Email:      me@yourapp.nylas.email
Provider:   nylas
Status:     valid
```

### `--app-password`

Use `--app-password` when you want the agent account to work with a standard mail client over IMAP/SMTP submission.

```bash
nylas agent account create me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
```

Requirements:
- 18 to 40 characters
- printable ASCII only, with no spaces
- at least one uppercase letter
- at least one lowercase letter
- at least one digit

When set, the agent account email becomes the mail-client username and the app password is used for IMAP/SMTP authentication.

### `--policy-id`

Use `--policy-id` when you want the new agent account to start with a specific policy already attached.

```bash
nylas agent account create me@yourapp.nylas.email --policy-id 12345678-1234-1234-1234-123456789012
```

## Show Agent Account

```bash
nylas agent account get 12345678-1234-1234-1234-123456789012
nylas agent account get me@yourapp.nylas.email
nylas agent account get me@yourapp.nylas.email --json
```

You can look up an agent account by grant ID or by email address.

## Delete Agent Account

```bash
nylas agent account delete 12345678-1234-1234-1234-123456789012
nylas agent account delete me@yourapp.nylas.email --yes
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

## Policies

```bash
nylas agent policy list
nylas agent policy list --all
nylas agent policy create --name "Strict Policy"
nylas agent policy create --data '{"name":"Strict Policy","rules":["rule-123"]}'
nylas agent policy create --data-file policy.json
nylas agent policy get 12345678-1234-1234-1234-123456789012
nylas agent policy read 12345678-1234-1234-1234-123456789012
nylas agent policy update 12345678-1234-1234-1234-123456789012 --name "Updated Policy"
nylas agent policy update 12345678-1234-1234-1234-123456789012 --data-file update.json
nylas agent policy delete 12345678-1234-1234-1234-123456789012 --yes
```

Summary:
- `list` resolves the default `provider=nylas` grant and shows its attached policy
- `list --all` shows only policies that are actually referenced by `provider=nylas` agent accounts
- `get` and `read` are aliases
- `delete` refuses to remove a policy that is still attached to any `provider=nylas` agent account

**Details:** [Agent policy reference](agent-policy.md)

## Rules

```bash
nylas agent rule list
nylas agent rule list --policy-id <policy-id>
nylas agent rule list --all
nylas agent rule read <rule-id>
nylas agent rule get <rule-id>
nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action mark_as_spam
nylas agent rule create --name "VIP sender" --condition from.address,is,ceo@example.com --action mark_as_read --action mark_as_starred
nylas agent rule create --data-file rule.json
nylas agent rule update <rule-id> --name "Updated Rule" --description "Block example.org"
nylas agent rule update <rule-id> --condition from.domain,is,example.org --action mark_as_spam
nylas agent rule delete <rule-id> --yes
```

Summary:
- `list` uses the policy attached to the current default `provider=nylas` grant unless `--policy-id` is passed
- `list --all` shows only rules reachable from policies attached to `provider=nylas` accounts
- `create` supports common-case flags like `--name`, repeatable `--condition`, and repeatable `--action`
- `get` and `read` are aliases
- `update` and `delete` refuse to operate on rules that are outside the current `provider=nylas` agent scope

**Details:** [Agent rule reference](agent-rule.md)

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

- [Agent policies](agent-policy.md)
- [Agent rules](agent-rule.md)
- [Email commands](email.md)
- [Inbound email](inbound.md)
