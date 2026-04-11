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

```bash
# List all scheduler configurations
nylas scheduler configurations list
nylas scheduler configs list              # Alias
nylas scheduler configurations list --json

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
# Create a session for a configuration
nylas scheduler sessions create <config-id>

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
# List all bookings
nylas scheduler bookings list
nylas scheduler bookings list --json

# Show booking details
nylas scheduler bookings show <booking-id>

# Confirm a booking
nylas scheduler bookings confirm <booking-id>

# Reschedule a booking
nylas scheduler bookings reschedule <booking-id> \\
  --start-time "2024-03-20T10:00:00Z"

# Cancel a booking
nylas scheduler bookings cancel <booking-id>
nylas scheduler bookings cancel <booking-id> \\
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

# 4. View bookings
nylas scheduler bookings list

# 5. Manage bookings
nylas scheduler bookings confirm <booking-id>
nylas scheduler bookings reschedule <booking-id> --start-time "..."
```

**Note:** Some scheduler features may not be available in all Nylas API versions or require specific subscription tiers.

---

