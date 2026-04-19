# Agent Policies

Detailed reference for `nylas agent policy`.

Agent policies are filtered through `provider=nylas` agent accounts in the CLI, even though the underlying policy objects are application-level resources.

## Commands

```bash
nylas agent policy list
nylas agent policy list --all
nylas agent policy create --name "Strict Policy"
nylas agent policy create --data-file policy.json
nylas agent policy get <policy-id>
nylas agent policy read <policy-id>
nylas agent policy update <policy-id> --name "Updated Policy"
nylas agent policy update <policy-id> --data-file update.json
nylas agent policy delete <policy-id> --yes
```

## Scope Model

The CLI intentionally treats policies as an agent-scoped surface:

- `nylas agent policy list` shows only the policy attached to the current default `provider=nylas` grant
- `nylas agent policy list --all` shows only policies referenced by at least one `provider=nylas` agent account
- text output includes the attached agent email and grant ID so you can see which agent account uses which policy

This means:

- a policy can exist in the application but still not appear under `nylas agent policy`
- a policy with no attached `provider=nylas` account is hidden from the agent policy list

## Listing Policies

### Default Agent Policy

```bash
nylas agent policy list
nylas agent policy list --json
```

Behavior:

- resolves the current default local grant
- requires that default grant to be `provider=nylas`
- returns the single attached policy for that grant

### All Agent Policies

```bash
nylas agent policy list --all
nylas agent policy list --all --json
```

Behavior:

- lists all policies referenced by at least one `provider=nylas` agent account
- text output includes one `Agent:` line per attached agent account

## Reading Policies

```bash
nylas agent policy get <policy-id>
nylas agent policy read <policy-id>
nylas agent policy read <policy-id> --json
```

Notes:

- `get` and `read` are aliases
- text output expands the policy into readable sections for:
  - rules
  - limits
  - options
  - spam detection
- `--json` returns the raw API payload

Use `--json` when you need the exact field names for automation or a follow-up update.

## Creating Policies

### Simple Create

```bash
nylas agent policy create --name "Strict Policy"
nylas agent policy create --name "Strict Policy" --json
```

This is the fastest path when you only need a named policy object and will add rules or settings later.

### Full JSON Create

```bash
nylas agent policy create --data-file policy.json
nylas agent policy create --data '{"name":"Strict Policy","rules":["rule-123"]}'
```

Example payload:

```json
{
  "name": "Strict Policy",
  "rules": ["rule-123"],
  "limits": {
    "limit_attachment_size_limit": 50480000,
    "limit_attachment_count_limit": 10,
    "limit_count_daily_message_per_grant": 500,
    "limit_inbox_retention_period": 30,
    "limit_spam_retention_period": 7
  },
  "options": {
    "additional_folders": [],
    "use_cidr_aliasing": false
  },
  "spam_detection": {
    "use_list_dnsbl": false,
    "use_header_anomaly_detection": false,
    "spam_sensitivity": 1
  }
}
```

## Updating Policies

### Simple Update

```bash
nylas agent policy update <policy-id> --name "Updated Policy"
```

### Partial JSON Update

```bash
nylas agent policy update <policy-id> --data-file update.json
nylas agent policy update <policy-id> --data '{"spam_detection":{"spam_sensitivity":0.8}}'
```

Behavior:

- `--name` updates the policy name directly
- `--data` and `--data-file` send a partial JSON body
- if both are provided, the explicit flags win for overlapping top-level fields

Recommended workflow:

1. `nylas agent policy read <policy-id> --json`
2. edit the payload you need
3. `nylas agent policy update <policy-id> --data-file update.json`

## Deleting Policies

```bash
nylas agent policy delete <policy-id> --yes
```

Safety rule:

- delete is rejected if any `provider=nylas` agent account still references the policy

To remove a policy from active use:

1. create or choose another policy
2. create future agent accounts with `--policy-id <new-policy-id>`
3. remove or rotate away the attached agent accounts that still reference the old policy
4. delete the now-unattached policy

## Relationship to Agent Accounts

Policies are primarily attached at agent account creation time:

```bash
nylas agent account create me@yourapp.nylas.email --policy-id <policy-id>
```

The CLI now has `nylas agent account update`, but it currently manages mutable account settings such as `--app-password`, not `settings.policy_id`. In practice, policy attachment remains a create-time workflow on the agent account surface.

## Troubleshooting

If `nylas agent policy list` returns nothing:

- make sure your default local grant is a `provider=nylas` account
- verify the agent account actually has a `settings.policy_id`
- try `nylas auth list` to confirm which grant is marked default

If `nylas agent policy delete` fails:

- the policy is still attached to one or more `provider=nylas` agent accounts
- run `nylas agent policy list --all` to see the attached agent mappings

## See Also

- [Agent overview](agent.md)
- [Agent rules](agent-rule.md)
