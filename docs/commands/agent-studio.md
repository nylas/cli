# Agent Studio

Visual management for agent accounts, workspaces, policies, rules, and lists.

```bash
nylas agent studio                 # start on http://localhost:7368 and open the browser
nylas agent studio --port 8080
nylas agent studio --no-browser
```

The server binds to localhost only; mutations are protected by same-origin
checks and a strict Content Security Policy.

## Views

The topbar switches between two views (the choice survives refresh via the
URL hash):

- **Board** — the drag-and-drop workspace canvas (default)
- **Accounts** — a searchable list of every agent account

The resource counts in the topbar are clickable: the accounts count jumps to
the Accounts view, the rest to the Board.

## The board

One card per workspace showing its attached policy, rules (with the lists
their `in_list` conditions reference), and member accounts (with status
dots: green = valid, amber = anything else). Auto-generated workspace names
have their connector UUID stripped for display; the full name shows in the
tooltip and drawer. The left palette holds every policy, rule, and list as
a chip.

- **Drag a policy chip onto a workspace** to attach it (sets `policy_id`)
- **Drag a rule chip onto a workspace** to attach it (adds to `rule_ids`)
- **Drag an account chip onto another workspace** to move the account there
  (the target workspace's policy and rules govern it immediately)
- Every drag action shows an **Undo** toast for ~6 seconds
- Dropping onto an auto-group workspace shared by multiple accounts asks for
  confirmation first, naming the affected accounts

## Accounts view

One row per account: status dot, email, workspace, governing policy ("plan
maximums" when none is attached), rule count, and a shared-workspace badge.
Substring search filters by email, workspace, or policy. Inline quick actions:

- **✈ Test** — send a self-addressed test email
- **⟳ Rotate** — rotate the app password (the new password is shown once)
- **⇄ Move** — move the account to another workspace
- **🗑 Delete** — guarded delete with consequence text

Account moves use the workspace `manual-assign` API; the workspace can also
be chosen up-front when creating the account, or moved from the terminal
with `nylas agent account move <email> --workspace <id>`.

## Plan limits

Your **billing plan** caps every policy limit, enforced by the Nylas API:
omitted limits default to the plan maximum, and values above it are rejected.
A workspace with no policy attached simply runs at plan maximums, so any
policy — including the default workspace's — can be edited, deleted, or
swapped freely.

## Creating resources

The **＋ New** menu creates agent accounts (with an app-password generator and
optional workspace pick), workspaces, policies (blank limits default to plan
maximums), rules, and lists — plus one-click rule recipes.

The **rule builder** is sentence-shaped and constrained by the live API
matrix: inbound rules only offer `from.*` fields; outbound adds `recipient.*`
and `outbound.type`; `in_list` swaps the value input for a picker showing only
lists whose type matches the field. Invalid combinations cannot be expressed.

## Inspector drawer

Click any chip, slot, or account to open the drawer: details, list item
management, policy editing, app-password rotation, a **send test email**
action, and guarded deletes with consequence text. Dangling references
(deleted resources still referenced) render as ⚠ warnings.

## Health surfacing

The status bar flags unattached rules/policies and unused lists; workspace
cards flag shared auto-group workspaces and dangling references — the same
checks as [`nylas agent overview`](agent.md#resource-overview).

## E2E tests

```bash
make test-playwright-studio
```

## See Also

- [Getting started with agent accounts](agent-getting-started.md)
- [Agent command reference](agent.md)
- [Agent lists](agent-list.md) · [Agent rules](agent-rule.md) · [Agent policies](agent-policy.md)
- API reference: https://developer.nylas.com/docs/v3/agent-accounts/
