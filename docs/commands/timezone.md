# Time Zone Utilities Guide

Complete guide to using Nylas CLI's offline timezone tools for global team coordination, DST management, and meeting scheduling.

> **⚡ Key Feature:** All timezone commands work 100% offline—no API access required. Perfect for remote teams, travel planning, and scheduling across time zones.

---

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [Convert Time Between Zones](#convert-time-between-zones)
  - [Find Meeting Times](#find-meeting-times-across-zones)
  - [Check DST Transitions](#check-dst-transitions)
  - [List Time Zones](#list-available-time-zones)
  - [Get Time Zone Information](#get-time-zone-information)
- [Tips & Tricks](#tips--tricks)
- [Common Use Cases](#common-use-cases)
- [Troubleshooting](#troubleshooting)
- [Performance Notes](#performance-notes)

---

## Overview

Nylas CLI includes powerful timezone utilities that solve common challenges faced by global teams:

- **83% of professionals** struggle with scheduling across time zones
- **DST changes** cause confusion and missed meetings
- **Finding overlapping working hours** is time-consuming and error-prone

### Why Use Timezone Commands?

| Feature | Benefit |
|---------|---------|
| **100% Offline** | Works on planes, trains, anywhere without WiFi |
| **Instant Results** | No network latency, calculations are local |
| **Privacy-First** | No data sent to external servers |
| **No Rate Limits** | Use as frequently as needed |
| **Free Forever** | No API costs or subscription fees |

### Supported Abbreviations

The CLI understands common timezone abbreviations for faster typing:

| Abbreviation | Full IANA Name |
|--------------|----------------|
| PST/PDT | America/Los_Angeles |
| EST/EDT | America/New_York |
| CST/CDT | America/Chicago |
| MST/MDT | America/Denver |
| GMT/BST | Europe/London |
| IST | Asia/Kolkata |
| JST | Asia/Tokyo |
| AEST/AEDT | Australia/Sydney |

---

## Quick Start

### Basic Time Conversion

```bash
# Convert current time from PST to IST
nylas timezone convert --from PST --to IST

# Convert specific time
nylas timezone convert \
  --from UTC \
  --to America/New_York \
  --time "2025-01-01T12:00:00Z"
```

### Check DST Transitions

```bash
# Check DST for New York in 2026
nylas timezone dst --zone America/New_York --year 2026
```

### Find Meeting Times

```bash
# Find overlapping times for 3 zones
nylas timezone find-meeting \
  --zones "America/New_York,Europe/London,Asia/Tokyo"
```

### Quick Zone Lookup

```bash
# Get info about a timezone
nylas timezone info UTC

# List all American timezones
nylas timezone list --filter America
```

---

## Commands

### Convert Time Between Zones

Convert time from one timezone to another with automatic DST handling.

#### Usage

```bash
nylas timezone convert --from <zone> --to <zone>           # Convert current time
nylas timezone convert --from <zone> --to <zone> --time <RFC3339>  # Convert specific time
nylas timezone convert --from <zone> --to <zone> --json    # JSON output
```

#### Flags

- `--from` (required) - Source time zone (IANA name or abbreviation)
- `--to` (required) - Target time zone (IANA name or abbreviation)
- `--time` - Specific time to convert (RFC3339 format: 2025-01-01T12:00:00Z)
- `--json` - Output as JSON

#### Examples

```bash
# Convert current time
nylas timezone convert --from PST --to IST

# Convert specific time
nylas timezone convert --from UTC --to America/New_York --time "2025-01-01T12:00:00Z"

# JSON output
nylas timezone convert --from UTC --to EST --json
```

---

### Find Meeting Times Across Zones

Find overlapping working hours across multiple time zones for scheduling meetings.

#### Usage

```bash
nylas timezone find-meeting --zones <zones>                # Basic meeting finder
nylas timezone find-meeting --zones <zones> --duration <duration>  # Specify duration
nylas timezone find-meeting --zones <zones> --start-hour <HH:MM> --end-hour <HH:MM>  # Custom hours
nylas timezone find-meeting --zones <zones> --exclude-weekends  # Skip weekends
```

#### Flags

- `--zones` (required) - Comma-separated list of time zones
- `--duration` - Meeting duration (default: 1h). Format: 30m, 1h, 1h30m
- `--start-hour` - Working hours start (default: 09:00). Format: HH:MM
- `--end-hour` - Working hours end (default: 17:00). Format: HH:MM
- `--start-date` - Search start date (default: today). Format: YYYY-MM-DD
- `--end-date` - Search end date (default: 7 days from start). Format: YYYY-MM-DD
- `--exclude-weekends` - Skip Saturday and Sunday
- `--json` - Output as JSON

#### Examples

```bash
# Basic meeting finder
nylas timezone find-meeting --zones "America/New_York,Europe/London,Asia/Tokyo"

# Custom working hours
nylas timezone find-meeting --zones "PST,EST,IST" --duration 30m \
  --start-hour 10:00 --end-hour 16:00 --exclude-weekends

# Specific date range
nylas timezone find-meeting --zones "America/Los_Angeles,Europe/Paris" \
  --start-date 2026-01-15 --end-date 2026-01-22
```

> **Note:** Meeting finder algorithm is planned but not yet implemented.

---

### Check DST Transitions

Display Daylight Saving Time transitions for a specific time zone and year.

#### Usage

```bash
nylas timezone dst --zone <zone>                # Check current year
nylas timezone dst --zone <zone> --year <year>  # Check specific year
nylas timezone dst --zone <zone> --json         # JSON output
```

#### Flags

- `--zone` (required) - Time zone to check (IANA name or abbreviation)
- `--year` - Year to check (default: current year)
- `--json` - Output as JSON

#### Examples

```bash
# Check DST for a zone
nylas timezone dst --zone America/New_York --year 2026

# Zone without DST
nylas timezone dst --zone America/Phoenix

# Using abbreviation
nylas timezone dst --zone PST

# JSON output
nylas timezone dst --zone EST --json
```

---

### List Available Time Zones

Display all IANA time zones with current time and offset information.

#### Usage

```bash
nylas timezone list                    # List all zones
nylas timezone list --filter <text>    # Filter by name
nylas timezone list --json             # JSON output
```

#### Flags

- `--filter` - Filter zones by name (case-insensitive)
- `--json` - Output as JSON

#### Examples

```bash
# List all zones (593 total)
nylas timezone list

# Filter by region
nylas timezone list --filter America

# Filter by city
nylas timezone list --filter Tokyo

# JSON output
nylas timezone list --filter UTC --json
```

---

### Get Time Zone Information

Display detailed information about a specific time zone.

#### Usage

```bash
nylas timezone info <zone>                     # Get info for zone
nylas timezone info --zone <zone>              # Alternative syntax
nylas timezone info --zone <zone> --time <RFC3339>  # Info at specific time
nylas timezone info --zone <zone> --json       # JSON output
```

#### Flags

- `--zone` - Time zone to query (IANA name or abbreviation)
- `--time` - Check info at specific time (RFC3339 format)
- `--json` - Output as JSON

> **Note:** Zone can be provided as a positional argument or via `--zone` flag.

#### Examples

```bash
# Get zone information
nylas timezone info America/New_York

# Using abbreviation
nylas timezone info PST

# Zone without DST
nylas timezone info Asia/Tokyo

# Check at specific time
nylas timezone info --zone America/New_York --time "2026-07-01T12:00:00Z"

# JSON output
nylas timezone info UTC --json
```

---

## Tips & Tricks

- **Use abbreviations** - `PST` instead of `America/Los_Angeles`
- **JSON for scripting** - `nylas timezone info UTC --json | jq '.offset_seconds'`
- **Check multiple zones** - Loop through zones in bash scripts
- **Plan for DST** - Check transitions before scheduling recurring meetings
- **Save aliases** - Add common conversions to `~/.bashrc` or `~/.zshrc`
- **Works offline** - No WiFi needed, all calculations are local

---

## Common Use Cases

1. **Remote team coordination** - Convert standup times across timezones
2. **Client calls** - Check if it's business hours before calling
3. **Travel planning** - Convert flight times to local timezone
4. **Meeting scheduling** - Find overlapping working hours globally
5. **DST awareness** - Check transitions before scheduling recurring meetings
6. **Multi-region deployments** - Convert deployment times to all datacenter regions

See [Calendar Integration](#calendar-integration) for timezone support in calendar commands.

---

## Troubleshooting

### Invalid Time Zone Error

```bash
$ nylas timezone info Invalid/Zone
Error: get time zone info: unknown time zone Invalid/Zone

# Solution: Use list to find valid zones
nylas timezone list --filter <search>
```

### Invalid Time Format

```bash
$ nylas timezone convert --from UTC --to EST --time "invalid"
Error: invalid time format (use RFC3339, e.g., 2025-01-01T12:00:00Z)

# Solution: Use RFC3339 format
# YYYY-MM-DDTHH:MM:SSZ (UTC)
# YYYY-MM-DDTHH:MM:SS±HH:MM (with offset)
```

### Missing Required Flag

```bash
$ nylas timezone convert --from PST
Error: required flag(s) "to" not set

# Solution: Both --from and --to are required
nylas timezone convert --from PST --to EST
```

### Abbreviation Not Recognized

```bash
# If abbreviation isn't in the built-in list, use full IANA name
nylas timezone list --filter <region>
# Then use the full name from the list
```

---

## Performance Notes

- **Instant execution** - All operations are local calculations
- **No network calls** - Works 100% offline
- **No rate limits** - Use as frequently as needed
- **Privacy-first** - No data ever sent to external servers
- **Minimal resources** - Uses OS timezone database

---

---

## Calendar Integration

### Viewing Calendar Events in Different Timezones

Convert calendar events to any timezone using `--timezone` and `--show-tz` flags:

```bash
# List events in a specific timezone
nylas calendar events list --timezone America/Los_Angeles

# Show timezone information for events
nylas calendar events list --show-tz

# View specific event in different timezone
nylas calendar events show <event-id> --timezone Europe/London
```

**Auto-detection:** Commands use your system timezone by default (detected from the `TZ` environment variable, then the `/etc/localtime` symlink).

### Creating Events in a Specific Timezone

Event start/end times are parsed in your system timezone by default. Use `--timezone` on `events create` and `events update` to parse them in another IANA zone; the zone is recorded on the event (`start_timezone`/`end_timezone`). On `events update`, `--timezone` only applies while parsing a new time, so it requires `--start`:

```bash
# 2pm New York time, regardless of where you run the command
nylas calendar events create --title "NY Standup" \
  --start "2026-06-15 14:00" --timezone America/New_York
```

**All-day events take a date only** (`YYYY-MM-DD`). Combining `--all-day` with a time component (e.g., `--start "2026-06-15 10:00"`) is an error — remove `--all-day` to create a timed event.

### Timezone Locking

Lock an event to its timezone with `--lock-timezone` on `events create` or `events update` — useful for in-person meetings that should always display in the venue's timezone:

```bash
# Create a locked event (confirmation shows the recorded zone)
nylas calendar events create --title "On-site" \
  --start "2026-06-15 09:00" --timezone Europe/London --lock-timezone

# Lock or unlock an existing event
nylas calendar events update <event-id> --lock-timezone
nylas calendar events update <event-id> --unlock-timezone
```

Locked events keep their recorded timezone in list/show views (shown with a 🔒 indicator) and are never converted to the viewer's timezone.

### DST (Daylight Saving Time) Warnings

Calendar commands automatically warn about events scheduled near or during DST transitions:

- **⛔ Error** - Time doesn't exist (spring forward gap)
- **⚠️ Warning** - DST transition within 7 days
- **ℹ️ Info** - Time occurs twice (fall back)

### Working Hours & Break Management

Configure working hours and break periods in `~/.nylas/config.yaml` to prevent scheduling conflicts:

```yaml
working_hours:
  default:
    enabled: true
    start: "09:00"
    end: "17:00"
    breaks:
      - name: "Lunch"
        start: "12:00"
        end: "13:00"
        type: "lunch"  # lunch, coffee, or custom
```

**Break Types:** `lunch`, `coffee`, `custom`

**Enforcement:**
- **Working Hours** - Soft warning (can override)
- **Breaks** - Hard block (cannot override, protects your break time)

See [Calendar Commands](commands/calendar.md) for detailed configuration examples.

### Event Time Formats

Event creation accepts `YYYY-MM-DD HH:MM`, `YYYY-MM-DDTHH:MM[:SS]`, RFC3339, or `YYYY-MM-DD` (all-day). Natural language scheduling is available via `nylas calendar schedule ai`.

---

## Related Documentation

- **[Command Reference](../COMMANDS.md)** - Quick command reference with examples
- **[AI Features](ai.md)** - AI-powered scheduling and timezone-aware features
- **[Calendar Guide](calendar.md)** - Calendar events with timezone support
- **[TUI Guide](tui.md)** - Interactive terminal interface

---

## FAQ

### Q: Do I need a Nylas API key to use timezone commands?

**A:** No! All timezone commands work 100% offline without any API access.

### Q: How accurate are the DST transition dates?

**A:** DST information comes from your operating system's timezone database, which is regularly updated. For most recent years, it's highly accurate.

### Q: Can I use custom timezone abbreviations?

**A:** The CLI supports common abbreviations (PST, EST, IST, etc.). For other zones, use the full IANA name from `nylas timezone list`.

### Q: Does this work on Windows?

**A:** Yes! Timezone commands work on macOS, Linux, and Windows.

### Q: Can I script timezone operations?

**A:** Absolutely! Use the `--json` flag for machine-readable output that's easy to parse with tools like `jq`.

### Q: Why isn't the meeting finder working?

**A:** The meeting finder algorithm is planned but not yet implemented. The CLI and service interfaces are complete and ready for the algorithm.

---

**Last Updated:** December 21, 2025
**Version:** 1.0
**Maintained By:** Nylas CLI Team
