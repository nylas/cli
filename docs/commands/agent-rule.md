# Agent Rules

Detailed reference for `nylas agent rule`.

Rules are backed by `/v3/rules` and attach to workspaces via `rule_ids[]`.

## Commands

```bash
nylas agent rule list
nylas agent rule get <rule-id>
nylas agent rule read <rule-id>
nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action mark_as_spam
nylas agent rule create --name "Archive outbound mail" --trigger outbound --condition recipient.domain,is,example.com --condition outbound.type,is,compose --action archive
nylas agent rule create --data-file rule.json
nylas agent rule update <rule-id> --name "Updated Rule"
nylas agent rule delete <rule-id> --yes
```

## Listing Rules

```bash
nylas agent rule list
nylas agent rule list --json
```

Lists all rules from `/v3/rules`. Text output shows which workspace has each rule attached.

## Reading Rules

```bash
nylas agent rule get <rule-id>
nylas agent rule read <rule-id>
nylas agent rule read <rule-id> --json
```

Notes:

- `get` and `read` are aliases
- text output expands the rule into readable sections for trigger, match logic, actions, and workspace references
- `--json` returns the raw rule payload

## Creating Rules

You can create rules either from common-case flags or from raw JSON.

### Common-Case Flags

```bash
nylas agent rule create \
  --name "Block Example" \
  --condition from.domain,is,example.com \
  --action mark_as_spam
```

```bash
nylas agent rule create \
  --name "VIP sender" \
  --condition from.address,is,ceo@example.com \
  --action mark_as_read \
  --action mark_as_starred
```

```bash
nylas agent rule create \
  --name "Archive outbound mail" \
  --trigger outbound \
  --condition recipient.domain,is,example.com \
  --condition outbound.type,is,compose \
  --action archive
```

Available common flags:

- `--name`
- `--description`
- `--priority`
- `--enabled`
- `--disabled`
- `--trigger`
- `--match-operator all|any`
- repeatable `--condition`
- repeatable `--action`

### Raw JSON

```bash
nylas agent rule create --data-file rule.json
nylas agent rule create --data '{"name":"Block Example","trigger":"inbound","match":{"operator":"any","conditions":[{"field":"from.domain","operator":"is","value":"example.com"}]},"actions":[{"type":"mark_as_spam"}]}'
```

The rule is created via `/v3/rules` then attached to the default grant's workspace `rule_ids[]`.

## Updating Rules

```bash
nylas agent rule update <rule-id> --name "Updated Rule"
nylas agent rule update <rule-id> --condition from.domain,is,example.org --action mark_as_starred
nylas agent rule update <rule-id> --data-file update.json --json
```

Updates the rule directly via `/v3/rules/{id}`.

## Deleting Rules

```bash
nylas agent rule delete <rule-id> --yes
```

The `--yes` flag is required to confirm deletion.

Behavior:
- detaches the rule from all agent workspaces that reference it
- deletes the rule via `/v3/rules/{id}`
- rolls back workspace changes if the delete fails

## Relationship to Workspaces

Rules attach to workspaces via `rule_ids[]`. The practical flow:

1. create a workspace: `nylas workspace create --name "My Workspace"`
2. create a policy: `nylas agent policy create --name "Strict Policy"`
3. attach policy to workspace: `nylas workspace update <ws-id> --policy-id <policy-id>`
4. create agent account (auto-assigns to default workspace)
5. create rules: `nylas agent rule create --name "Block" --condition ... --action ...`

## Troubleshooting

If `nylas agent rule list` returns nothing:

- confirm rules have been created via `/v3/rules`
- check if your default grant is `provider=nylas`

If `nylas agent rule delete` fails:

- verify the rule ID exists
- check if the rule is attached to workspaces

## See Also

- [Agent overview](agent.md)
- [Agent policies](agent-policy.md)
