# Agent Lists

Detailed reference for `nylas agent list`.

Lists are backed by `/v3/lists` and hold normalized values referenced by agent
rule `in_list` conditions. Each list has an immutable type — `domain`, `tld`,
or `address` — that determines which rule fields it can match:

| List type | Matches rule fields |
|-----------|---------------------|
| `domain`  | `from.domain`, `recipient.domain` |
| `tld`     | `from.tld`, `recipient.tld` |
| `address` | `from.address`, `recipient.address` |

API reference: https://developer.nylas.com/docs/v3/agent-accounts/policies-rules-lists/

## Commands

```bash
nylas agent list list
nylas agent list get <list-id>
nylas agent list create --name "Blocked domains" --type domain
nylas agent list create --name "VIPs" --type address --item ceo@example.com --item cfo@example.com
nylas agent list update <list-id> --name "New name"
nylas agent list items <list-id>
nylas agent list add <list-id> spam.com junk.net
nylas agent list remove <list-id> spam.com
nylas agent list delete <list-id> --yes
```

## Listing Lists

```bash
nylas agent list list
nylas agent list list --json
```

Lists all lists from `/v3/lists` with their type and item count.

## Showing a List

```bash
nylas agent list get <list-id>
nylas agent list get <list-id> --json
nylas agent list items <list-id>
```

Notes:

- `get` shows the list metadata and its items
- `items` shows only the items
- `--json` returns raw payloads

## Creating Lists

```bash
nylas agent list create --name "Blocked domains" --type domain
nylas agent list create --name "VIPs" --type address --item ceo@example.com
```

Notes:

- `--type` is required and immutable after creation (`domain`, `tld`, or `address`)
- `--item` is repeatable and seeds the list right after creation
- `--description` is optional

## Managing Items

```bash
nylas agent list add <list-id> spam.com junk.net
nylas agent list remove <list-id> spam.com
```

Notes:

- up to 1000 items per request
- values are lowercased, trimmed, and validated against the list's type by the API
- duplicate additions are silently ignored
- rules referencing the list pick up item changes immediately

## Using Lists in Rules

Reference list IDs in `in_list` rule conditions:

```bash
nylas agent rule create \
  --name "Block listed domains" \
  --condition from.domain,in_list,<list-id> \
  --action mark_as_spam
```

For multiple lists, pass additional comma-separated IDs:
`--condition from.domain,in_list,<list-id-1>,<list-id-2>`.

The API rejects rule conditions that reference list IDs that don't exist, so
create the list first and use the returned ID.

## Updating and Deleting

```bash
nylas agent list update <list-id> --name "New name" --description "Updated"
nylas agent list delete <list-id> --yes
```

Notes:

- only name and description can be updated; the type is immutable
- deletion requires `--yes`
- rules referencing a deleted list will no longer match it
