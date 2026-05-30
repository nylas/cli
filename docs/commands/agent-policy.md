# Agent Policies

Detailed reference for `nylas agent policy`.

Policies are application-level resources backed by `/v3/policies`. They attach to workspaces via `policy_id`.

## Commands

```bash
nylas agent policy list
nylas agent policy create --name "Strict Policy"
nylas agent policy create --data-file policy.json
nylas agent policy get <policy-id>
nylas agent policy read <policy-id>
nylas agent policy update <policy-id> --name "Updated Policy"
nylas agent policy update <policy-id> --data-file update.json
nylas agent policy delete <policy-id> --yes
```

## Listing Policies

```bash
nylas agent policy list
nylas agent policy list --json
```

Lists all policies from `/v3/policies`. Text output shows which workspace has each policy attached.

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

- delete is rejected if any `provider=nylas` agent workspace still references the policy

## Relationship to Workspaces

Policies attach to workspaces via `policy_id`. To assign a policy to an agent account's workspace:

```bash
nylas workspace update <workspace-id> --policy-id <policy-id>
```

The API auto-creates a default workspace and policy when an agent account is created.

## Troubleshooting

If `nylas agent policy list` returns nothing:

- no policies have been explicitly created via `/v3/policies`
- the API auto-creates a default policy on the workspace, but it does not appear in `/v3/policies`

If `nylas agent policy delete` fails:

- the policy is still attached to one or more agent workspaces
- run `nylas agent policy list` to see the attached workspace mappings

## See Also

- [Agent overview](agent.md)
- [Agent rules](agent-rule.md)
