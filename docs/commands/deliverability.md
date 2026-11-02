# Deliverability

Use `nylas deliverability` to see what happened to the email your application sends. The commands follow the "Deliverability & Engagement" section of the Nylas API reference:

| Command group | Data | API routes |
|---|---|---|
| `send-events` | Send activity and send metrics | `/v3/activity/send`, `/v3/metrics/send` |
| `tracking-events` | Opens, link clicks and engagement metrics | `/v3/activity/tracking`, `/v3/metrics/tracking` |
| `send-alerts` | Alert rules on send rates | `/v3/alerts/send` |

All of these routes are application-level. They use your API key (`nylas auth config` or `NYLAS_API_KEY`) and do not take a grant. A grant access token gets `403 api.forbidden`.

---

## Time windows

The two kinds of command take different time formats:

| Commands | `--start` / `--end` | Default | Limit |
|---|---|---|---|
| `list` (activity) | Unix seconds, `YYYY-MM-DD`, `"YYYY-MM-DD HH:MM"` (UTC), RFC3339, or a look-back such as `24h`, `7d`, `2w` | `--end` = now, `--start` = UTC midnight 6 days before `--end` | 90 days. An older `--start` is moved to 90 days ago; an older `--end` is rejected. |
| `summary`, `timeseries`, `breakdown` (metrics) | UTC date, `YYYY-MM-DD`, both days included | `--end` = today, `--start` = 6 days before `--end` | 365 days |

A look-back value counts back from now. For example, `--start 24h` means "24 hours ago". On `list`, a time without a zone is UTC, and a bare date is a whole UTC day: `--start 2026-10-01` starts at 00:00:00Z and `--end 2026-10-01` ends at 23:59:59Z. Words such as `today` or `9am` are not accepted, because they mean a different time on each run.

---

## Send events

```bash
# Activity feed, newest first
nylas deliverability send-events list
nylas deliverability send-events list --event-type bounced --event-type complaint --start 7d
nylas deliverability send-events list --recipient-domain gmail.com --bounce-class hard
nylas deliverability send-events list --message-id "<abc@mail.example.com>" --json

# One event, with the SMTP / bounce / complaint detail
nylas deliverability send-events show <event-id>

# Every recipient's timeline for one message
nylas deliverability send-events message <message-id>

# Metrics
nylas deliverability send-events summary
nylas deliverability send-events summary --start 2026-10-01 --end 2026-10-31 --sending-domain mail.example.com
nylas deliverability send-events timeseries --granularity week --start 2026-08-01
nylas deliverability send-events breakdown --group-by recipient_domain --sort bounce_rate:desc
```

**`list` filters:** `--event-type` (repeatable: `accepted`, `delivered`, `bounced`, `complaint`, `rejected`), `--bounce-class` (`hard`, `soft`, `undetermined`), `--sending-domain`, `--from`, `--grant-id`, `--recipient`, `--recipient-domain`, `--message-id`, `--thread-id`, `--order asc|desc`.

**Paging:** `list` shows one page (`--limit`, default 50, max 200). When there are more rows it prints `Next page token: …`. Pass that token with `--page-token`. Keep the other flags the same, because the token belongs to that exact query, including `--start` and `--end`. A look-back such as `7d` gives a different time on every run, so it cannot be used with `--page-token`. When page 1 used a look-back, the CLI prints `Next page: --page-token … --start <epoch>` instead, with `--end <epoch>` too if you set `--end`. Run the next page with exactly those flags. (Without `--end`, the token itself keeps the end time.) With `--json`, `--format yaml` or `--quiet`, the same line goes to stderr, so stdout stays the plain API response.

**`summary`:** health is always graded on the **last 7 days**, whatever window you ask for. The states are `healthy`, `at_risk`, `critical` and `insufficient_data` (fewer than 100 sends). Totals and rate tiles use your window and compare it with the period of the same length just before it. When the state is `at_risk` or `critical`, `summary` also lists up to 3 findings.

**`timeseries`:** `--granularity auto|day|week|month`. `auto` uses days up to 21 days, weeks up to 92 days, and months after that.

**`breakdown`:** `--group-by sending_domain|agent_account|recipient_domain`, `--sort <key>:<asc|desc>` (keys: `key`, `sent`, `delivery_rate`, `bounce_rate`, `complaint_rate`), `--limit` 1–100 (default 25). Rows after the limit are added together into one `(others)` row. A `recipient_domain` breakdown only covers the last 90 days.

**Rates:** a rate shows `—` when its denominator is 0. That means "no data", not 0%. The bounce rate counts hard bounces only.

---

## Tracking events

```bash
nylas deliverability tracking-events list --event-type link_clicked --start 7d
nylas deliverability tracking-events list --label welcome-campaign --json
nylas deliverability tracking-events show <event-id>
nylas deliverability tracking-events message <message-id>

nylas deliverability tracking-events summary --start 2026-10-01
nylas deliverability tracking-events timeseries --granularity day
nylas deliverability tracking-events breakdown --group-by agent_account --sort open_rate:desc
```

**`list` filters:** `--event-type` (repeatable: `opened`, `link_clicked`), `--sending-domain`, `--from`, `--grant-id`, `--message-id`, `--thread-id`, `--label`.

**`message`:** shows open and click totals, clicks per link, and the newest 100 interactions. The totals count every interaction, not only the 100 shown.

**Rates:** open rate = first opens ÷ messages sent with open tracking. Click rate = first clicks ÷ messages sent with link tracking. `breakdown` sort keys: `key`, `tracked_opens`, `tracked_links`, `open_rate`, `click_rate`.

---

## Send alerts

A send alert rule watches one send rate for each sending domain. When the rate crosses the threshold, Nylas sends the `alert.triggered` webhook to your application's webhook subscriptions. An application can have at most 20 rules.

```bash
nylas deliverability send-alerts list
nylas deliverability send-alerts create --metric bounce_rate --threshold 5 --window-hours 1 --min-volume 50
nylas deliverability send-alerts create --metric delivery_rate --threshold 95 --window-hours 24
nylas deliverability send-alerts show <rule-id>
nylas deliverability send-alerts update <rule-id> --threshold 3
nylas deliverability send-alerts delete <rule-id> --yes
```

| Flag | Meaning |
|---|---|
| `--metric` | `bounce_rate`, `complaint_rate` or `delivery_rate` |
| `--threshold` | Percentage with at most 2 decimals. `bounce_rate` and `complaint_rate` fire **above** it (0 to <100). `delivery_rate` fires **below** it (>0 to 100). |
| `--window-hours` | Look-back window, 1–24 hours |
| `--min-volume` | Sends needed in the window before the rule can fire (default 0) |

`create` needs `--metric`, `--threshold` and `--window-hours`. `update` sends only the flags you give and keeps every other field.

---

## Output

All commands support `--json`, `--format yaml` and `--quiet`. For `list`, the JSON output is the full API response (`data`, `next_cursor`, `request_id`), so a script can page through results. `--quiet` prints one ID per line.
