# Getting Started with Agent Accounts

A hands-on walkthrough for setting up and operating Nylas agent accounts from
the CLI — accounts, workspaces, policies, rules, and lists, with examples for
every step.

## What Agent Accounts Are

Agent accounts are **Nylas-managed email identities** (provider `nylas`).
Unlike OAuth grants, they don't connect to Gmail or Outlook — Nylas itself
hosts the mailbox (e.g. `support@yourapp.nylas.email`). They're built for
AI agents and automation: a real inbox your agent can send from, receive to,
and automate, without a human's mailbox behind it.

### How the resources fit together

```
Application
├── Policies              coarse settings bundles (limits, options, spam)
├── Rules                 mail-flow automation (trigger → conditions → actions)
│     └── in_list ──────► Lists (typed value sets: domain / tld / address)
└── Workspaces            the attachment point that makes it all take effect
      ├── policy_id       one policy per workspace
      ├── rule_ids[]      many rules per workspace
      └── Agent accounts  mail through these is governed by the workspace
```

A rule or policy does nothing until a **workspace** references it, and an
agent account is governed by whatever its workspace references.

API reference: https://developer.nylas.com/docs/v3/agent-accounts/

## Prerequisites

```bash
nylas init                      # one-time CLI setup (API key, region)
nylas doctor                    # verify configuration is healthy
```

You need an API key with admin access. Agent account emails use your
application's agent domain (e.g. `yourapp.nylas.email`).

## Step 1 — Create an Agent Account

```bash
nylas agent account create support@yourapp.nylas.email
```

**Example output:**

```
✓ Agent account created successfully!

support@yourapp.nylas.email
  ID:           11111111-1111-1111-1111-111111111111
  Workspace ID: aaaaaaaa-1111-1111-1111-111111111111
  Status:       active
```

Notes:

- the `nylas` connector is created automatically on first use
- the API auto-creates a **default workspace and policy** for the account
- add `--app-password 'ValidAgentPass123ABC!'` to also enable IMAP/SMTP
  mail-client access (see Step 7)
- `--json` prints the raw payload for scripting:

```bash
nylas agent account create support@yourapp.nylas.email --json
```

Check overall readiness at any time:

```bash
nylas agent status
```

## Step 2 — Inspect and Manage Accounts

```bash
nylas agent account list                          # all agent accounts
nylas agent account get support@yourapp.nylas.email
nylas agent account get 11111111-1111-1111-1111-111111111111   # by ID
nylas agent account update support@yourapp.nylas.email --app-password 'NewPass456DEF!'
nylas agent account delete support@yourapp.nylas.email --yes
```

Accounts are grants — switch the CLI's active grant to one to operate as it:

```bash
nylas auth switch 11111111-1111-1111-1111-111111111111
```

## Step 3 — Send and Receive Email

With the agent account as the active grant:

```bash
nylas email send --to user@example.com --subject "Hello" --body "From your agent"
nylas email list --limit 10
nylas email threads list
```

Notes:

- sends go through the per-grant endpoint (`/v3/grants/{id}/messages/send`)
- the sender address defaults to the agent account's email
- GPG signing and stored signatures are not supported for agent sends
- send volume is subject to plan limits:
  https://developer.nylas.com/docs/v3/agent-accounts/send-limits/

## Step 4 — Policies

Policies are settings bundles (limits, options, spam detection) that
workspaces attach via `policy_id`. Your account already has a default one.

```bash
nylas agent policy list
nylas agent policy get <policy-id>
nylas agent policy create --name "Strict Policy"
nylas agent policy create --data '{"name":"Strict Policy","rules":["rule-123"]}'
nylas agent policy create --data-file policy.json
nylas agent policy update <policy-id> --name "Renamed Policy"
nylas agent policy delete <policy-id> --yes        # must be unattached
```

To switch a workspace (and therefore its accounts) to a different policy:

```bash
nylas workspace update <workspace-id> --policy-id <policy-id>
```

**Example output (`policy list`):**

```
Policies (2)

1. Default Policy
   ID: pppppppp-1111-1111-1111-111111111111
   Attached: support@yourapp.nylas.email (workspace aaaaaaaa-...)

2. Strict Policy
   ID: pppppppp-2222-2222-2222-222222222222
   Attached: (none)
```

## Step 5 — Lists

Lists are typed value sets used by rule `in_list` conditions. The type is
**immutable** and determines which rule fields the list can match:

| List type | Matches rule fields |
|-----------|---------------------|
| `domain`  | `from.domain`, `recipient.domain` |
| `tld`     | `from.tld`, `recipient.tld` |
| `address` | `from.address`, `recipient.address` |

```bash
# Create a list (optionally seeding items)
nylas agent list create --name "Blocked domains" --type domain --item spam.com --item junk.net

# Inspect
nylas agent list list
nylas agent list get <list-id>
nylas agent list items <list-id>

# Manage items (up to 1000 per request; values are lowercased, trimmed,
# validated against the type; duplicates silently ignored)
nylas agent list add <list-id> phishing.example
nylas agent list remove <list-id> junk.net

# Update metadata (type cannot change) and delete
nylas agent list update <list-id> --name "Blocklist" --description "Known bad senders"
nylas agent list delete <list-id> --yes
```

**Example output (`list create`):**

```
✓ List created successfully!

Blocked domains
  ID:    dddddddd-1111-1111-1111-111111111111
  Type:  domain
  Items: 2
```

Rules referencing a list pick up item changes **immediately** — no rule
update needed. Deleting a list doesn't break rules; they just stop matching.

## Step 6 — Rules

Rules automate mail flow: a `trigger` (`inbound`/`outbound`), conditions, and
actions. The CLI creates the rule and attaches it to your default agent
account's workspace in one step.

```bash
# Simple: spam-flag a domain
nylas agent rule create \
  --name "Block Example" \
  --condition from.domain,is,example.com \
  --action mark_as_spam

# Multiple conditions and actions, any-match
nylas agent rule create \
  --name "Tidy newsletters" \
  --match-operator any \
  --condition from.domain,contains,newsletter \
  --condition from.address,is,digest@example.com \
  --action archive --action mark_as_read

# Outbound trigger
nylas agent rule create \
  --name "Archive outbound mail" \
  --trigger outbound \
  --condition recipient.domain,is,example.com \
  --condition outbound.type,is,compose \
  --action archive

# Using a list (create the list first — the API validates list IDs exist)
nylas agent rule create \
  --name "Block listed domains" \
  --condition from.domain,in_list,<list-id> \
  --action mark_as_spam

# Multiple lists in one condition
nylas agent rule create \
  --name "Block all listed" \
  --condition "from.domain,in_list,<list-id-1>,<list-id-2>" \
  --action block

# Priority and state
nylas agent rule create --name "Low priority" --priority 5 --disabled \
  --condition from.tld,is,xyz --action trash

# From raw JSON
nylas agent rule create --data-file rule.json

# Manage
nylas agent rule list
nylas agent rule read <rule-id>
nylas agent rule update <rule-id> --name "Updated Rule" --enabled
nylas agent rule delete <rule-id> --yes      # detaches from workspaces first
```

Supported condition operators: `is`, `is_not`, `contains`, `in_list`.
Common actions: `archive`, `mark_as_read`, `mark_as_starred`, `mark_as_spam`,
`block`, `trash`.

## Step 7 — Mail-Client Access (IMAP/SMTP)

With an app password set (at create time or via `account update`), agent
mailboxes work in any mail client:

```bash
nylas agent account update support@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
```

Guide: https://developer.nylas.com/docs/v3/agent-accounts/mail-clients/

## Step 8 — Workspaces Directly

Most flows don't need direct workspace surgery (the CLI attaches rules and
the API creates a default workspace per account), but for full control:

```bash
nylas workspace list
nylas workspace get <workspace-id>
nylas workspace create --name "Support workspace"
nylas workspace update <workspace-id> --policy-id <policy-id>
nylas workspace delete <workspace-id> --yes
```

## Worked Example: OTP-Reading Agent

A common pattern — an agent account that receives one-time passcodes and
your automation extracts them:

```bash
nylas agent account create otp-bot@yourapp.nylas.email
nylas auth switch <agent-grant-id>
nylas otp get                      # latest OTP code from the inbox
nylas otp watch                    # stream new codes as they arrive
```

Guide: https://developer.nylas.com/docs/cookbook/agent-accounts/extract-otp-code/

## Limits and Troubleshooting

**Plan limits.** Applications are capped per plan — for example 5 rules and
10 lists on the free plan. Hitting a cap returns a 403 mentioning the plan;
the CLI surfaces it as a plan-limit error rather than a permission error.

**`invalid list ID ... (invalid_field)`** when creating a rule: the
`in_list` condition references a list that doesn't exist. Create the list
first (`nylas agent list create`) and use the returned ID — fabricated or
deleted IDs are rejected.

**`rule_ids entry not found or does not belong to this application`** when
creating/attaching rules: the workspace's `rule_ids` contains a stale entry
(a rule that was deleted without being detached — e.g. an interrupted
cleanup). Inspect and repair:

```bash
nylas workspace get <workspace-id>             # shows rule_ids
nylas agent rule list                          # rules that actually exist
# detach stale entries by updating the workspace with only valid rule IDs
```

**`default grant is not a nylas agent account`**: rule/policy commands that
resolve the default account need the active grant to be an agent account —
run `nylas auth switch <agent-grant-id>` first.

**Connector readiness**: `nylas agent status` verifies the `nylas` connector
exists and shows managed accounts.

**Visual inspection**: `nylas air` exposes a Policy & Rules page (Email →
Policy & Rules) showing the active agent account's workspace, policy, rules,
and lists in one view.

## See Also

- [Agent command reference](agent.md)
- [Agent policies](agent-policy.md)
- [Agent rules](agent-rule.md)
- [Agent lists](agent-list.md)
- Provisioning guide: https://developer.nylas.com/docs/v3/agent-accounts/provisioning/
- Mailboxes: https://developer.nylas.com/docs/v3/agent-accounts/mailboxes/
- Policies, rules & lists: https://developer.nylas.com/docs/v3/agent-accounts/policies-rules-lists/
