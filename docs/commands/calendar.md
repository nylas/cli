## Calendar Management

View calendars, manage events, and check availability.

### List Calendars

```bash
nylas calendar list [grant-id]        # List all calendars
nylas cal list                        # Alias
```

**Example output:**
```bash
$ nylas calendar list

Found 3 calendar(s):

NAME                    ID                      PRIMARY   READ-ONLY
Personal                cal_primary_123         Yes
Work                    cal_work_456
Holidays                cal_holidays_789                  Yes
```

### Calendar Events

```bash
# List events
nylas calendar events list [grant-id]
nylas calendar events list --days 14        # Next 14 days
nylas calendar events list --limit 20       # Limit results
nylas calendar events list --calendar <id>  # Specific calendar
nylas calendar events list --show-cancelled # Include cancelled

# Show event details
nylas calendar events show <event-id>

# Create event
nylas calendar events create --title "Meeting" --start "2024-12-20 14:00" --end "2024-12-20 15:00"
nylas calendar events create --title "Vacation" --start "2024-12-25" --all-day
nylas calendar events create --title "Team Sync" --start "2024-12-20 10:00" \
  --participant "alice@example.com" --participant "bob@example.com"

# Delete event
nylas calendar events delete <event-id>
nylas calendar events delete <event-id> --force
```

**Working Hours Validation:**

The CLI validates event times against configured working hours:
- **Default Hours**: 9:00 AM - 5:00 PM (if not configured)
- **Per-Day Configuration**: Different hours for different days
- **Weekend Support**: Separate weekend hours or disable weekends
- Warns when scheduling outside working hours
- Use `--ignore-working-hours` to skip validation

**Configuration Example:**
```yaml
# ~/.config/nylas/config.yaml
working_hours:
  default:
    enabled: true
    start: "09:00"
    end: "17:00"
  friday:
    enabled: true
    start: "09:00"
    end: "15:00"  # Short Fridays
  weekend:
    enabled: false  # No work on weekends
```

**Example Working Hours Warning:**
```bash
$ nylas calendar events create --title "Late Call" --start "2025-01-15 18:00" --end "2025-01-15 19:00"

⚠️  Working Hours Warning

This event is scheduled outside your working hours:
  • Your hours: 09:00 - 17:00
  • Event time: 6:00 PM Local
  • 1 hour(s) after end

Create anyway? [y/N]: n
Cancelled.

# Or skip validation:
$ nylas calendar events create --title "Late Call" --start "2025-01-15 18:00" --ignore-working-hours
✓ Event created successfully!
```

**Break Time Protection (NEW):**

Protect your lunch breaks and other break periods with hard-block enforcement:

- **Hard Block**: Cannot schedule events during breaks (unlike working hours which allow override)
- **Multiple Breaks**: Configure lunch, coffee breaks, and custom break periods
- **Per-Day Breaks**: Different break times for different days
- Use `--ignore-working-hours` to skip break validation

**Configuration Example:**
```yaml
# ~/.config/nylas/config.yaml
working_hours:
  default:
    enabled: true
    start: "09:00"
    end: "17:00"
    breaks:
      - name: "Lunch"
        start: "12:00"
        end: "13:00"
        type: "lunch"
      - name: "Afternoon Coffee"
        start: "15:00"
        end: "15:15"
        type: "coffee"
  friday:
    enabled: true
    start: "09:00"
    end: "15:00"
    breaks:
      - name: "Lunch"
        start: "11:30"
        end: "12:30"  # Earlier lunch on Fridays
        type: "lunch"
```

**Example Break Conflict:**
```bash
$ nylas calendar events create --title "Quick Sync" --start "2025-01-15 12:30" --end "2025-01-15 13:00"

⛔ Break Time Conflict

Event cannot be scheduled during Lunch (12:00 - 13:00)

Tip: Schedule the event outside of break times, or update your
     break configuration in ~/.nylas/config.yaml
Error: event conflicts with break time

# Break blocks are enforced - you must reschedule:
$ nylas calendar events create --title "Quick Sync" --start "2025-01-15 13:00" --end "2025-01-15 13:30"
✓ Event created successfully!
```

**Example output (list events):**
```bash
$ nylas calendar events list --days 7

Found 4 event(s):

Team Standup
  When: Mon, Dec 16, 2024, 9:00 AM - 9:30 AM
  Location: Conference Room A
  Status: confirmed
  Guests: 5 participant(s)
  ID: event_abc123

Project Review
  When: Tue, Dec 17, 2024, 2:00 PM - 3:00 PM
  Status: confirmed
  Guests: 3 participant(s)
  ID: event_def456

Holiday Party
  When: Fri, Dec 20, 2024 (all day)
  Location: Main Office
  Status: confirmed
  ID: event_ghi789
```

**Example output (show event):**
```bash
$ nylas calendar events show event_abc123

Team Standup

When
  Mon, Dec 16, 2024, 9:00 AM - 9:30 AM

Location
  Conference Room A

Description
  Daily team standup meeting to discuss progress and blockers.

Organizer
  John Smith <john@company.com>

Participants
  Alice Johnson <alice@company.com> ✓ accepted
  Bob Wilson <bob@company.com> ✓ accepted
  Carol Davis <carol@company.com> ? tentative

Video Conference
  Provider: zoom
  URL: https://zoom.us/j/123456789

Details
  Status: confirmed
  Busy: true
  ID: event_abc123
  Calendar: cal_primary_123
  URL: https://zoom.us/j/987654321

Details
  Status: confirmed
  Busy: true
  ID: event_xyz123
  Calendar: cal_primary_123
```

### Event Update

Update existing calendar events.

```bash
# Update event title
nylas calendar events update <event-id> --title "New Title"

# Update event time
nylas calendar events update <event-id> --start "2024-01-15 14:00" --end "2024-01-15 15:00"

# Update location and description
nylas calendar events update <event-id> --location "Conference Room A" --description "Weekly sync"

# Update participants
nylas calendar events update <event-id> --participant alice@example.com --participant bob@example.com

# Update visibility
nylas calendar events update <event-id> --visibility public
```

**Example output:**
```bash
$ nylas calendar events update event_abc123 --title "Updated Team Standup" --location "Room B"

✓ Event updated successfully

Updated Team Standup
  When: Mon, Dec 16, 2024, 9:00 AM - 9:30 AM
  Location: Room B
  ID: event_abc123
```

### Calendar Availability

```bash
# Check free/busy status
nylas calendar availability check [grant-id]
nylas calendar availability check --emails alice@example.com,bob@example.com
nylas calendar availability check --start "tomorrow 9am" --end "tomorrow 5pm"
nylas calendar availability check --duration 7d
nylas calendar availability check --format json

# Find available meeting times
nylas calendar availability find --participants alice@example.com,bob@example.com
nylas calendar availability find --participants team@example.com --duration 60
nylas calendar availability find --participants alice@example.com \
  --start "tomorrow 9am" --end "tomorrow 5pm" --interval 15
```

**Example output (check):**
```bash
$ nylas calendar availability check --emails alice@example.com,bob@example.com

Free/Busy Status: Mon Dec 16 2:30 PM - Tue Dec 17 2:30 PM
────────────────────────────────────────────────────────────

📧 alice@example.com
   Busy times:
   ● Mon Dec 16 3:00 PM - 4:00 PM
   ● Tue Dec 17 9:00 AM - 10:00 AM

📧 bob@example.com
   ✓ Free during this period
```

**Example output (find):**
```bash
$ nylas calendar availability find --participants alice@example.com,bob@example.com --duration 30

Available 30-minute Meeting Slots
────────────────────────────────────────

📅 Mon, Dec 16
   1. 9:00 AM - 9:30 AM
   2. 9:30 AM - 10:00 AM
   3. 11:00 AM - 11:30 AM
   4. 2:00 PM - 2:30 PM

📅 Tue, Dec 17
   5. 10:30 AM - 11:00 AM
   6. 1:00 PM - 1:30 PM
   7. 3:00 PM - 3:30 PM

Found 7 available slots
```

### Smart Meeting Finder (Multi-Timezone)

**NEW:** Find optimal meeting times across multiple timezones with intelligent scoring.

The smart meeting finder analyzes participant timezones and suggests meeting times using a 100-point scoring algorithm that considers:
- **Working Hours (40 pts)**: All participants within working hours
- **Time Quality (25 pts)**: Quality of time for participants (morning/afternoon preference)
- **Cultural Considerations (15 pts)**: Respects cultural norms (no Friday PM, no lunch hour, no Monday early AM)
- **Weekday Preference (10 pts)**: Prefers mid-week meetings (Tuesday/Wednesday best)
- **Holiday Check (10 pts)**: Avoids holidays

```bash
# Find optimal meeting time for multiple participants
nylas calendar find-time --participants alice@example.com,bob@example.com --duration 1h

# Custom working hours and date range
nylas calendar find-time \
  --participants alice@example.com,bob@example.com,carol@example.com \
  --duration 1h \
  --working-start 09:00 \
  --working-end 17:00 \
  --days 7

# 30-minute meeting with weekend availability
nylas calendar find-time \
  --participants team@example.com \
  --duration 30m \
  --exclude-weekends=false
```

**Example output:**
```bash
$ nylas calendar find-time --participants alice@example.com,bob@example.com --duration 1h

🌍 Multi-Timezone Meeting Finder

Participants:
  • alice@example.com: America/New_York
  • bob@example.com: Europe/London

Top 3 Suggested Times:

1. 🟢 Tuesday, Jan 7, 10:00 AM PST (Score: 94/100)
   alice: 1:00 PM - 2:00 PM America/New_York (Good)
   bob: 6:00 PM - 7:00 PM Europe/London (Poor ⚠️)

   Score Breakdown:
   • Working Hours: 40/40 (✓)
   • Time Quality: 22/25
   • Cultural: 15/15
   • Weekday: 10/10
   • Holidays: 7/10

2. 🟢 Wednesday, Jan 8, 11:00 AM PST (Score: 92/100)
   alice: 2:00 PM - 3:00 PM America/New_York (Good)
   bob: 7:00 PM - 8:00 PM Europe/London (Bad 🔴)

   Score Breakdown:
   • Working Hours: 40/40 (✓)
   • Time Quality: 20/25
   • Cultural: 15/15
   • Weekday: 10/10
   • Holidays: 7/10

3. 🟡 Thursday, Jan 9, 9:00 AM PST (Score: 75/100)
   alice: 12:00 PM - 1:00 PM America/New_York (Good)
   bob: 5:00 PM - 6:00 PM Europe/London (Poor ⚠️)

   Score Breakdown:
   • Working Hours: 40/40 (✓)
   • Time Quality: 18/25
   • Cultural: 12/15
   • Weekday: 8/10
   • Holidays: 7/10

💡 Recommendation: Book option #1 for best overall experience
```

**Scoring Legend:**
- 🟢 Excellent (85-100): Great time for all participants
- 🟡 Good (70-84): Acceptable with minor compromises
- 🔴 Poor (<70): Significant compromises, consider alternatives

**Time Quality Labels:**
- ✨ Excellent: 9-11 AM
- Good: 11 AM - 2 PM
- Fair: 2-5 PM
- ⚠️ Poor: 8-9 AM or 5-6 PM
- 🔴 Bad: Outside working hours

### Virtual Calendars

Virtual calendars allow scheduling without connecting to a third-party provider. They're perfect for conference rooms, equipment, or external contractors.

**Features:**
- No OAuth required
- Never expire
- Support calendar and event operations only (no email/contacts)
- Maximum 10 calendars per virtual account

```bash
# List all virtual calendar grants
nylas calendar virtual list
nylas calendar virtual list --json

# Create a virtual calendar grant
nylas calendar virtual create --email conference-room-a@company.com
nylas calendar virtual create --email projector-1@company.com

# Show virtual calendar grant details
nylas calendar virtual show <grant-id>
nylas calendar virtual show <grant-id> --json

# Delete a virtual calendar grant
nylas calendar virtual delete <grant-id>
nylas calendar virtual delete <grant-id> -y  # Skip confirmation
```

**Example workflow:**
```bash
# 1. Create a virtual calendar grant for a conference room
$ nylas calendar virtual create --email conference-room-a@company.com
✓ Created virtual calendar grant
  ID:     vcal-grant-123abc
  Email:  conference-room-a@company.com
  Status: valid

# 2. Create a calendar for this virtual grant
$ nylas calendar create vcal-grant-123abc --name "Conference Room A"
✓ Created calendar
  ID:   primary
  Name: Conference Room A

# 3. Create events on the virtual calendar
$ nylas calendar events create vcal-grant-123abc primary \
  --title "Board Meeting" \
  --start "2024-01-15T14:00:00" \
  --end "2024-01-15T16:00:00"
✓ Created event
```

### Recurring Events

Manage recurring calendar events, including viewing all instances and updating or deleting specific occurrences.

**Supported recurrence patterns:**
- Daily: `RRULE:FREQ=DAILY;COUNT=5`
- Weekly: `RRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=10`
- Monthly: `RRULE:FREQ=MONTHLY;COUNT=12`
- Yearly: `RRULE:FREQ=YEARLY;COUNT=3`

```bash
# List all instances of a recurring event
nylas calendar recurring list <master-event-id> --calendar <calendar-id>
nylas calendar recurring list event-123 --calendar cal-456 --limit 100
nylas calendar recurring list event-123 --calendar cal-456 \
  --start 1704067200 --end 1706745600

# Update a single instance
nylas calendar recurring update <instance-id> --calendar <calendar-id> \
  --title "Updated Meeting Title"
nylas calendar recurring update instance-789 --calendar cal-456 \
  --start "2024-01-15T14:00:00" --end "2024-01-15T15:30:00" \
  --location "Conference Room B"

# Delete a single instance (creates an exception)
nylas calendar recurring delete <instance-id> --calendar <calendar-id>
nylas calendar recurring delete instance-789 --calendar cal-456 -y
```

**Example output (list):**
```bash
$ nylas calendar recurring list event-master-123 --calendar primary

INSTANCE ID        TITLE                   START TIME        STATUS
event-inst-1       Weekly Team Meeting     2024-01-08 10:00  confirmed
event-inst-2       Weekly Team Meeting     2024-01-15 10:00  confirmed
event-inst-3       Weekly Team Meeting     2024-01-22 10:00  confirmed
event-inst-4       Weekly Team Meeting     2024-01-29 10:00  confirmed

Total instances: 4
Master Event ID: event-master-123
```

**Understanding recurring events:**
- **Master Event ID**: The parent event that defines the recurrence pattern
- **Instance**: A single occurrence of the recurring series
- **Exception**: An instance that has been modified or deleted
- When you update an instance, it becomes an exception with custom properties
- When you delete an instance, it adds an EXDATE to the recurrence rule

---

## Scheduling Workflows

### Multi-Timezone Meeting Coordination

```bash
#!/bin/bash
# multi-timezone-meeting.sh

# Use the find-time command for optimal meeting scheduling
nylas calendar find-time \
  --participants "alice@example.com,bob@example.com" \
  --duration 60

# View events in different timezones
nylas calendar events list
```

---

### Batch Create Events

```bash
#!/bin/bash
# batch-create-events.sh

# Read from CSV: title,start,duration,participants
while IFS=, read -r title start duration participants; do
  echo "Creating: $title"

  nylas calendar events create \
    --title "$title" \
    --start "$start" \
    --duration "$duration" \
    --participant "$participants" \
    --yes

  sleep 1  # Rate limiting
done < events.csv
```

---

### Interview Scheduling Automation

```bash
#!/bin/bash
# interview-scheduler.sh

CANDIDATE_EMAIL="$1"
CANDIDATE_NAME="$2"
INTERVIEW_DATE="$3"

# Schedule interview panel
PANEL=(
  "hiring-manager@company.com:30:Technical Screen"
  "engineer@company.com:60:Technical Interview"
  "hr@company.com:30:Culture Fit"
)

current_time="$INTERVIEW_DATE 09:00"

for interview in "${PANEL[@]}"; do
  IFS=: read -r interviewer duration title <<< "$interview"

  nylas calendar events create \
    --title "Interview: $CANDIDATE_NAME - $title" \
    --start "$current_time" \
    --duration "$duration" \
    --participant "$CANDIDATE_EMAIL,$interviewer" \
    --yes

  sleep 1
done
```

---

### Scheduling Events

```bash
#!/bin/bash
# schedule-event.sh

# Create a calendar event
nylas calendar events create \
  --title "Important Meeting" \
  --start "2025-03-10 10:00" \
  --duration 60
```

---

### Best Practices

**Rate limiting:**
```bash
for event in "${events[@]}"; do
  nylas calendar events create ...
  sleep 1  # Wait between API calls
done
```

**Timezone best practices:**
1. Always specify timezone explicitly for multi-timezone teams
2. Check DST transitions when scheduling near March/November
3. Use IANA timezone names (not abbreviations like EST/PST)
4. Use UTC as common reference for global teams

---

