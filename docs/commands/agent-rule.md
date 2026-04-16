# Agent Rules

Detailed reference for `nylas agent rule`.

Agent rules are filtered through policies that are attached to `provider=nylas` agent accounts. The CLI hides rules that are outside that agent scope.

## Commands

```bash
nylas agent rule list
nylas agent rule list --policy-id <policy-id>
nylas agent rule list --all
nylas agent rule get <rule-id>
nylas agent rule read <rule-id>
nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action mark_as_spam
nylas agent rule create --name "Block Replies" --trigger outbound --condition outbound.type,is,reply --action block
nylas agent rule create --data-file rule.json
nylas agent rule update <rule-id> --name "Updated Rule"
nylas agent rule update <rule-id> --trigger outbound --condition recipient.domain,is,example.org --action archive
nylas agent rule delete <rule-id> --yes
```

## Scope Model

The CLI resolves rules through agent policy attachment:

- `nylas agent rule list` uses the policy attached to the current default `provider=nylas` grant
- `nylas agent rule list --policy-id <policy-id>` uses that specific policy within the agent scope
- `nylas agent rule list --all` shows rules reachable from any policy attached to any `provider=nylas` agent account
- `get`, `read`, `update`, and `delete` validate that the rule is reachable from the selected agent scope before operating on it

This prevents the agent command surface from mutating rules that are only in non-agent policy usage.

Runtime note:

- inbound rule evaluation is policy-linked
- outbound rule evaluation is application-scoped in the API
- the CLI still attaches created outbound rules to the selected policy so they remain visible in the agent-scoped surface

## Listing Rules

### Rules for the Default Agent Policy

```bash
nylas agent rule list
nylas agent rule list --json
```

Behavior:

- resolves the default local `provider=nylas` grant
- finds the policy attached to that grant
- returns the rules attached to that policy

### Rules for a Specific Agent Policy

```bash
nylas agent rule list --policy-id <policy-id>
```

Use this when you want to inspect one policy without changing your default grant.

### All Agent Rules

```bash
nylas agent rule list --all
nylas agent rule list --all --json
```

Behavior:

- shows only rules referenced by policies attached to `provider=nylas` accounts
- text output includes policy and agent account references

## Reading Rules

```bash
nylas agent rule get <rule-id>
nylas agent rule read <rule-id>
nylas agent rule read <rule-id> --json
```

Notes:

- `get` and `read` are aliases
- text output expands the rule into readable sections for:
  - trigger
  - match logic
  - actions
  - policy references
  - agent account references
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
  --name "Block Replies" \
  --trigger outbound \
  --condition outbound.type,is,reply \
  --action block
```

Available common flags:

- `--name`
- `--description`
- `--priority`
- `--enabled`
- `--disabled`
- `--trigger`
- `--policy-id`
- `--match-operator all|any`
- repeatable `--condition`
- repeatable `--action`

Defaults when creating from flags:

- `trigger=inbound`
- `enabled=true`
- `match.operator=all`

### `--condition`

Format:

```bash
--condition <field>,<operator>,<value>
```

Examples:

```bash
--condition from.domain,is,example.com
--condition from.address,is,ceo@example.com
--condition recipient.domain,is,example.com
--condition outbound.type,is,reply
```

Important:

- condition values are treated as strings by default
- values like `true` and `123` stay strings
- there is no implicit JSON coercion for condition values
- inbound rules support `from.address`, `from.domain`, and `from.tld`
- outbound rules also support `recipient.address`, `recipient.domain`, `recipient.tld`, and `outbound.type`
- `outbound.type` only supports `is` and `is_not`, with values `compose` or `reply`
- `in_list` requires a JSON array value, so use `--data` or `--data-file` instead of the scalar `--condition` flag syntax

### `--action`

Formats:

```bash
--action <type>
--action <type>=<value>
```

Examples:

```bash
--action block
--action mark_as_spam
--action mark_as_read
--action assign_to_folder=vip
--action archive
```

Action values are also treated as strings by default.

Important:

- supported actions are `block`, `mark_as_spam`, `assign_to_folder`, `mark_as_read`, `mark_as_starred`, `archive`, and `trash`
- `assign_to_folder` requires a value
- `block` cannot be combined with other actions

### Full JSON Create

```bash
nylas agent rule create --data-file rule.json
nylas agent rule create --data '{"name":"Block Example","enabled":true,"trigger":"inbound","match":{"operator":"all","conditions":[{"field":"from.domain","operator":"is","value":"example.com"}]},"actions":[{"type":"mark_as_spam"}]}'
```

Use JSON when the rule structure is more complex than the common flags make comfortable.

## Updating Rules

### Simple Top-Level Updates

```bash
nylas agent rule update <rule-id> --name "Updated Rule"
nylas agent rule update <rule-id> --description "Block example.org"
nylas agent rule update <rule-id> --priority 20 --enabled
```

### Replacing Conditions and Actions with Flags

```bash
nylas agent rule update <rule-id> \
  --match-operator any \
  --condition from.domain,is,example.org \
  --condition from.tld,is,org \
  --action mark_as_spam
```

```bash
nylas agent rule update <rule-id> \
  --trigger outbound \
  --condition recipient.domain,is,example.org \
  --action archive
```

Behavior:

- `--condition` replaces the rule's condition set
- `--action` replaces the rule's action set
- existing `match.operator` is preserved unless you explicitly pass `--match-operator`

### Partial JSON Update

```bash
nylas agent rule update <rule-id> --data-file update.json
nylas agent rule update <rule-id> --data '{"description":"Updated via JSON"}'
```

Recommended workflow:

1. `nylas agent rule read <rule-id> --json`
2. edit the payload you need
3. `nylas agent rule update <rule-id> --data-file update.json`

## Deleting Rules

```bash
nylas agent rule delete <rule-id> --yes
```

Safety rules:

- delete is rejected if the rule is referenced outside the current `provider=nylas` agent scope
- delete is rejected if removing an inbound rule would leave an attached agent policy with zero attached rules
- outbound rules can still be deleted when they are the only rule attached to a policy, because outbound evaluation is not policy-selected at send time

These checks are there to prevent accidental breakage of active agent policy configuration.

## Relationship to Policies

Rules are attached to policies, and policies are attached to agent accounts.
Inbound evaluation follows that chain directly. Outbound evaluation does not:

- inbound rules run only when attached to the selected policy
- outbound rules are evaluated by the application at send time
- the CLI keeps outbound rules attached to the selected policy so they stay discoverable through the agent-scoped command surface

Practical flow:

1. create or choose a policy
2. create a rule and attach it to that policy in the same command
3. create an agent account with that policy using `--policy-id`

The CLI scope always follows that chain:

- agent account
- policy
- rules reachable from that policy

## Troubleshooting

If `nylas agent rule list` returns nothing:

- make sure your default grant is `provider=nylas`
- confirm that default agent account has a policy attached
- confirm the policy actually has rules attached

If `nylas agent rule read` or `update` says the rule is not found:

- the rule may exist in the application but outside the current agent scope
- try `nylas agent rule list --all` to see what is reachable from agent accounts

If `nylas agent rule delete` is rejected:

- the rule is shared outside the current agent scope, or
- deleting it would leave an attached policy with no remaining rules

## See Also

- [Agent overview](agent.md)
- [Agent policies](agent-policy.md)
