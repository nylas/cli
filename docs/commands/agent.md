# Agent

Manage Nylas agent resources from the CLI.

**New to agent accounts?** Start with the [getting started guide](agent-getting-started.md).

Agent accounts are managed email identities backed by provider `nylas`. Unlike OAuth grants, they do not require a third-party mailbox connection. Account operations live under `nylas agent account`, while `nylas agent status` reports connector and account readiness.

## Commands

```bash
nylas agent account list
nylas agent account create <email>
nylas agent account update [agent-id|email] --app-password 'ValidAgentPass123ABC!'
nylas agent account get <agent-id|email>
nylas agent account move <agent-id|email> --workspace <workspace-id>
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
nylas agent rule create --name "Archive outbound mail" --trigger outbound --condition recipient.domain,is,example.com --condition outbound.type,is,compose --action archive
nylas agent rule update <rule-id> --name "Updated Rule" --description "Block example.org"
nylas agent rule delete <rule-id>
nylas agent list list
nylas agent list get <list-id>
nylas agent list create --name "Blocked domains" --type domain --item spam.com
nylas agent list add <list-id> junk.net
nylas agent list remove <list-id> junk.net
nylas agent list delete <list-id> --yes
nylas agent overview
nylas agent studio
nylas agent status
```

Lists hold normalized values (domains, TLDs, or addresses) referenced by rule
`in_list` conditions. **Details:** [Agent list reference](agent-list.md)

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
   Workspace ID: aaaaaaaa-1111-1111-1111-111111111111

2. me@yourapp.nylas.email                 active
   ID: 22222222-2222-2222-2222-222222222222
   Workspace ID: bbbbbbbb-2222-2222-2222-222222222222
```

## Create Agent Account

```bash
nylas agent account create me@yourapp.nylas.email
nylas agent account create me@yourapp.nylas.email --name 'Support Bot'
nylas agent account create me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
nylas agent account create support@yourapp.nylas.email --json
```

Behavior:
- always creates a grant with `provider=nylas`
- automatically creates the `nylas` connector first if it does not exist
- the API auto-creates a default workspace and policy for the account
- stores the created grant locally like other authenticated accounts
- optionally sets a top-level `name` (display name) on the grant
- optionally sets `settings.app_password` on the grant for IMAP/SMTP mail client access
- treats a bare account name such as `agent` as `agent@nylas.email`
- when the requested domain is not registered, points to `https://dashboard-v3.nylas.com/` to create or register the agent domain before retrying

To attach a custom policy after creation:
```bash
nylas workspace update <workspace-id> --policy-id <policy-id>
```

**Example output:**
```bash
$ nylas agent account create me@yourapp.nylas.email

✓ Agent account created successfully!

Email:      me@yourapp.nylas.email
Provider:   nylas
Status:     valid
```

### `--name`

Use `--name` to set a top-level display name on the agent account grant. This is a single name field (not split into first/last) and is independent of `settings`.

```bash
nylas agent account create me@yourapp.nylas.email --name 'Support Bot'
```

Requirements:
- 1 to 256 characters when set
- omitted from the request entirely when not provided

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

## Show Agent Account

```bash
nylas agent account get
nylas agent account get 12345678-1234-1234-1234-123456789012
nylas agent account get me@yourapp.nylas.email
nylas agent account get me@yourapp.nylas.email --json
```

You can look up an agent account by grant ID or by email address. If you omit the identifier, the CLI resolves a local `provider=nylas` grant when one can be identified safely.

## Update Agent Account

```bash
nylas agent account update --app-password 'ValidAgentPass123ABC!'
nylas agent account update 12345678-1234-1234-1234-123456789012 --app-password 'ValidAgentPass123ABC!'
nylas agent account update me@yourapp.nylas.email --name 'Support Bot'
nylas agent account update me@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!' --json
```

Behavior:
- updates the resolved local `provider=nylas` grant when no identifier is passed
- supports rotating or adding `settings.app_password` and/or setting the top-level `name`
- requires at least one of `--app-password` or `--name`
- preserves the existing account email and policy attachment
- preserves the existing display name when `--name` is not passed (the grant update replaces the full record, so the CLI re-sends the current name)

Use this when you want to add mail-client access after creation, rotate an existing IMAP/SMTP app password, or rename the account.

### `--name` (update)

`--name` sets the top-level display name (1–256 characters). Omit it to leave the current name unchanged. Clearing an existing name (setting it back to empty) is not currently supported by the grant API.

```bash
nylas agent account update me@yourapp.nylas.email --name 'Support Bot'
```

## Move Agent Account

```bash
nylas agent account move me@yourapp.nylas.email --workspace 12345678-1234-1234-1234-123456789012
nylas agent account move 12345678-1234-1234-1234-123456789012 --workspace <workspace-id>
```

Moves the account to another workspace; the target workspace's policy and
rules govern it immediately. Moves use the workspace manual-assign API
(`POST /v3/workspaces/{id}/manual-assign`), which reassigns the grant even
when it currently belongs to another workspace. Use `nylas workspace list`
to find workspace IDs. The same move is available visually in
[Agent Studio](agent-studio.md) by dragging an account chip onto a
workspace card.

## Delete Agent Account

```bash
nylas agent account delete 12345678-1234-1234-1234-123456789012
nylas agent account delete me@yourapp.nylas.email --yes
```

Deleting an agent account revokes the underlying `provider=nylas` grant.

## Resource Overview

```bash
nylas agent overview
nylas agent overview --json
nylas agent tree          # alias
```

Renders one tree per agent account showing its workspace, attached policy,
attached rules, and the lists those rules reference:

```
support@yourapp.nylas.email  valid
└── Workspace: Support workspace (default, auto-group, shared with 1 other account(s))
    ├── Policy: Default Policy
    └── Rules (2)
        ├── Block listed domains (inbound)
        │   └── List: Blocked domains (domain, 12 items)
        └── Archive newsletters (inbound) [disabled]
```

Workspaces with no policy (or a dangling `policy_id`) note that plan
maximums apply — accounts without a policy run at the billing plan's limits.

The overview also flags problems the API does not prevent:
- ⚠ dangling references — workspace `policy_id`/`rule_ids` or rule `in_list`
  conditions pointing at deleted resources
- auto-group workspaces shared by multiple accounts (changes affect them all)
- unattached policies/rules and lists referenced by no rule

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
- `list` shows all policies from `/v3/policies` with workspace annotations
- `get` and `read` are aliases
- `delete` refuses to remove a policy that is still attached to any `provider=nylas` agent account

**Details:** [Agent policy reference](agent-policy.md)

## Rules

```bash
nylas agent rule list
nylas agent rule read <rule-id>
nylas agent rule get <rule-id>
nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action mark_as_spam
nylas agent rule create --name "VIP sender" --condition from.address,is,ceo@example.com --action mark_as_read --action mark_as_starred
nylas agent rule create --name "Archive outbound mail" --trigger outbound --condition recipient.domain,is,example.com --condition outbound.type,is,compose --action archive
nylas agent rule create --data-file rule.json
nylas agent rule update <rule-id> --name "Updated Rule" --description "Block example.org"
nylas agent rule update <rule-id> --trigger outbound --condition recipient.domain,is,example.org --condition outbound.type,is,reply --action archive
nylas agent rule delete <rule-id> --yes
```

Summary:
- `list` shows all rules from `/v3/rules` with workspace annotations
- `create` supports common-case flags like `--name`, repeatable `--condition`, and repeatable `--action`; attaches the rule to the default grant's workspace
- both inbound and outbound rule triggers are supported
- `get` and `read` are aliases
- `delete` detaches the rule from workspaces before deleting

**Details:** [Agent rule reference](agent-rule.md)

Agent accounts can also expose the mailbox over IMAP and SMTP submission when an `app_password` is configured at creation time.

## Sending Email from Agent Accounts

When the active grant is an agent account (`provider=nylas`):
- `nylas email send` uses per-grant send: `/v3/grants/{grant_id}/messages/send`
- the sender address is taken from the active grant email when one is not supplied
- Agent Account sends do not use the domain transactional relay endpoint
- GPG signing/encryption is not supported for Agent Account sends in the CLI
- stored signatures via `--signature-id` are not supported for Agent Account sends in the CLI

## See Also

- [Agent policies](agent-policy.md)
- [Agent rules](agent-rule.md)
- [Agent lists](agent-list.md)
- [Agent Studio](agent-studio.md)
- [Email commands](email.md)
