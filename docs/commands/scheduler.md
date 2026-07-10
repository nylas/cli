## Scheduler Management

Manage Nylas Scheduler for creating booking pages, configurations, sessions, and appointments.

### What is Nylas Scheduler?

Nylas Scheduler enables you to create customizable booking workflows for scheduling meetings. Key features include:
- **Configurations**: Define meeting types with availability rules and settings
- **Sessions**: Generate temporary booking sessions for specific configurations
- **Bookings**: Manage scheduled appointments (view, confirm, reschedule, cancel)
- **Pages**: Create and manage hosted scheduling pages

### Scheduler Configurations

Manage scheduling configurations (meeting types):

> Configurations are grant-scoped (`/v3/grants/{grant_id}/scheduling/configurations`).
> The grant is taken from your default grant, or pass it as an optional trailing
> `[grant-id]` positional. Leading positionals like `<config-id>` are not mistaken
> for a grant.

```bash
# List all scheduler configurations (uses default grant)
nylas scheduler configurations list
nylas scheduler configs list              # Alias
nylas scheduler configurations list --json
nylas scheduler configurations list <grant-id>   # Explicit grant

# Show configuration details
nylas scheduler configurations show <config-id>
nylas scheduler configs show <config-id>

# Create a simple configuration
nylas scheduler configurations create \
  --name "30 Min Meeting" \
  --title "30 Min Meeting" \
  --participants alice@co.com \
  --duration 30

# Create with availability settings
nylas scheduler configurations create \
  --name "Product Demo" \
  --title "Product Demo" \
  --participants alice@co.com \
  --duration 30 \
  --interval 15 \
  --buffer-before 5 \
  --buffer-after 10 \
  --conferencing-provider "Google Meet" \
  --min-booking-notice 120 \
  --available-days-in-future 30

# Create from a JSON file
nylas scheduler configurations create --file config.json

# Create from file with flag overrides
nylas scheduler configurations create --file config.json --duration 60

# Update a configuration
nylas scheduler configurations update <config-id> \
  --name "Updated Name" \
  --duration 60 \
  --buffer-before 10

# Update from a JSON file
nylas scheduler configurations update <config-id> --file update.json

# Delete a configuration
nylas scheduler configurations delete <config-id>
nylas scheduler configs delete <config-id> -y    # Skip confirmation
```

**Configuration Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--name` | string | Configuration name |
| `--participants` | strings | Participant emails (comma-separated, first is organizer) |
| `--duration` | int | Meeting duration in minutes (default: 30) |
| `--title` | string | Event title |
| `--description` | string | Event description |
| `--location` | string | Event location |
| `--interval` | int | Slot interval in minutes |
| `--round-to` | int | Round start times to nearest N minutes |
| `--availability-method` | string | `max-fairness` or `max-availability` |
| `--buffer-before` | int | Buffer minutes before meetings |
| `--buffer-after` | int | Buffer minutes after meetings |
| `--timezone` | string | Event timezone (e.g., `America/New_York`) |
| `--booking-type` | string | `booking` or `organizer-confirmation` |
| `--conferencing-provider` | string | `Google Meet`, `Zoom`, or `Microsoft Teams` |
| `--disable-emails` | bool | Disable email notifications |
| `--reminder-minutes` | ints | Reminder minutes (e.g., `10,60`) |
| `--min-booking-notice` | int | Minimum minutes before booking |
| `--min-cancellation-notice` | int | Minimum minutes before cancellation |
| `--confirmation-method` | string | `automatic` or `manual` |
| `--available-days-in-future` | int | Days in advance bookings are available |
| `--cancellation-policy` | string | Cancellation policy text |
| `--file` | string | JSON config file (flags override file values) |
| `--json` | bool | Output as JSON |

**File Input:**

The `--file` flag accepts a JSON file matching the API request structure. You can export an existing configuration with `--json`, edit it, and re-import:

```bash
# Export → edit → recreate
nylas scheduler configs show abc123 --json > meeting.json
# Edit meeting.json...
nylas scheduler configs create --file meeting.json
```

When both `--file` and flags are provided, flags take precedence over file values.

**Configuration Features:**
- Duration and interval settings
- Availability rules and windows
- Buffer times before/after meetings
- Conferencing auto-creation (Google Meet, Zoom, Teams)
- Booking limits and restrictions
- Reminder notifications
- Custom event settings

### Scheduler Sessions

Create temporary booking sessions for configurations:

```bash
# Create a session for a configuration (TTL is in minutes, max 30)
nylas scheduler sessions create --config-id <config-id>
nylas scheduler sessions create --config-id <config-id> --ttl 10

# Show session details
nylas scheduler sessions show <session-id>
```

**Session Features:**
- Temporary booking URLs with expiration
- Configuration-specific availability
- Session-based booking tracking

### Scheduler Bookings

Manage scheduled appointments:

```bash
# Booking commands are authorized by a Scheduler session token that the CLI
# mints from the booking's configuration, so --configuration-id is required on
# every booking command (the API key is not accepted on booking endpoints).

# Show booking details
nylas scheduler bookings show <booking-id> --configuration-id <config-id>

# Confirm a booking. --salt is required and comes from the booking reference
# (in the organizer confirmation link, the cancel/reschedule URL, or a Scheduler
# webhook). It cannot be looked up from the booking ID.
nylas scheduler bookings confirm <booking-id> --configuration-id <config-id> --salt <salt>

# Reschedule a booking
nylas scheduler bookings reschedule <booking-id> \\
  --configuration-id <config-id> \\
  --start-time 1710930600 --end-time 1710934200
# If the reschedule is applied but the booking cannot be read back afterwards,
# the command still succeeds and prints a warning on stderr (in --json mode the
# output additionally carries a "warning" field); re-run `bookings show` to
# verify the booking's current server-side record.

# Cancel a booking
nylas scheduler bookings cancel <booking-id> --configuration-id <config-id>
nylas scheduler bookings cancel <booking-id> \\
  --configuration-id <config-id> \\
  --reason "Meeting no longer needed"
```

**Booking Information Includes:**
- Event ID and configuration ID
- Start and end times
- Participant details
- Status (pending, confirmed, cancelled)

### Scheduler Pages

Create and manage hosted booking pages:

```bash
# List all scheduler pages
nylas scheduler pages list
nylas scheduler pages list --json

# Show page details
nylas scheduler pages show <page-id>

# Create a new page
nylas scheduler pages create \\
  --config-id <config-id> \\
  --slug "meet-me"

# Update a page
nylas scheduler pages update <page-id> \\
  --slug "new-slug" \\
  --name "Updated Page"

# Delete a page
nylas scheduler pages delete <page-id>
```

**Page Features:**
- Custom slugs for friendly URLs
- Configuration-based availability
- Optional custom domain support
- Appearance customization

**Example Workflow:**

```bash
# 1. Create a meeting type
nylas scheduler configs create \\
  --name "Product Demo" \\
  --duration 30

# 2. Create a booking page
nylas scheduler pages create \\
  --config-id <config-id> \\
  --slug "product-demo"

# 3. Share the booking URL with prospects
# URL format: https://schedule.nylas.com/product-demo

# 4. Show a booking (booking IDs arrive via Scheduler webhooks or confirmation links)
nylas scheduler bookings show <booking-id> --configuration-id <config-id>

# 5. Manage bookings (--configuration-id is required on booking commands)
nylas scheduler bookings confirm <booking-id> --configuration-id <config-id> --salt <salt>
nylas scheduler bookings reschedule <booking-id> --configuration-id <config-id> --start-time <unix> --end-time <unix>
```

**Note:** Some scheduler features may not be available in all Nylas API versions or require specific subscription tiers.

---

