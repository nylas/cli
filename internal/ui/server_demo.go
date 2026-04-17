package ui

import (
	"strings"
)

func demoGrants() []Grant {
	return []Grant{
		{ID: "demo-grant-001", Email: "alice@example.com", Provider: "google"},
		{ID: "demo-grant-002", Email: "bob@work.com", Provider: "microsoft"},
		{ID: "demo-grant-003", Email: "carol@company.org", Provider: "google"},
	}
}

// demoDefaultGrant returns the default grant ID for demo mode.
func demoDefaultGrant() string {
	return "demo-grant-001"
}

// getDemoCommandOutput returns sample output for demo mode commands.
func getDemoCommandOutput(command string) string {
	cmd := strings.TrimSpace(command)
	args := strings.Fields(cmd)
	if len(args) == 0 {
		return "Demo mode - no command specified"
	}

	baseCmd := args[0]
	if len(args) >= 2 {
		baseCmd = args[0] + " " + args[1]
	}

	switch baseCmd {
	case "email list":
		return `Demo Mode - Sample Emails

  ★ ●  alice@example.com       Weekly Team Sync - Agenda        2 min ago
    ●  bob@work.com            Project Update: Q4 Goals         15 min ago
  ★    calendar@google.com     Reminder: Design Review          1 hour ago
    ●  notifications@github    [nylas/cli] New PR opened        2 hours ago
       support@example.com     Welcome!                         1 day ago

Showing 5 of 127 messages`

	case "email threads":
		return `Demo Mode - Sample Threads

  ★ ●  Team Weekly Standup     5 messages    alice, bob, carol    2 min ago
    ●  Project Planning Q1     12 messages   team@company.org     1 hour ago
  ★    Design Review           3 messages    design@example.com   3 hours ago
       Onboarding Docs         2 messages    hr@company.org       1 day ago

Showing 4 threads`

	case "calendar list":
		return `Demo Mode - Sample Calendars

  ID                     NAME                 PRIMARY
  cal-primary-001        Work Calendar        ✓
  cal-personal-002       Personal
  cal-team-003           Team Events

3 calendars found`

	case "calendar events":
		return `Demo Mode - Sample Events

  TODAY
  09:00 - 10:00   Team Standup                  Conference Room A
  14:00 - 15:00   Design Review                 Zoom Meeting

  TOMORROW
  10:00 - 11:00   1:1 with Manager              Office
  15:00 - 16:00   Sprint Planning               Conference Room B

4 upcoming events`

	case "auth status":
		return `Demo Mode - Authentication Status

  Status:     Configured ✓
  Region:     US
  Client ID:  demo-client-id
  API Key:    ********configured

  Default Account: alice@example.com (Google)`

	case "auth list":
		return `Demo Mode - Connected Accounts

  ✓  alice@example.com    Google      demo-grant-001 (default)
     bob@work.com         Microsoft   demo-grant-002
     carol@company.org    Google      demo-grant-003

3 accounts connected`

	case "contacts list", "contacts list --id":
		return `Demo Mode - Sample Contacts

Found 5 contact(s):

ID                                     NAME                   EMAIL                      PHONE
demo-contact-001-alice-johnson-12345   Alice Johnson          alice@example.com          +1-555-0101
demo-contact-002-bob-smith-67890       Bob Smith              bob@work.com               +1-555-0102
demo-contact-003-carol-williams-11111  Carol Williams         carol@company.org          +1-555-0103
demo-contact-004-david-brown-22222     David Brown            david@startup.io           +1-555-0104
demo-contact-005-eve-davis-33333       Eve Davis              eve@consulting.com         +1-555-0105

Showing 5 of 127 contacts`

	case "contacts groups":
		return `Demo Mode - Contact Groups

  ID                     NAME                 MEMBERS
  grp-001                Work                 23
  grp-002                Personal             15
  grp-003                VIP Clients          8
  grp-004                Newsletter           156

4 groups found`

	case "scheduler configurations":
		return `Demo Mode - Scheduler Configurations

  ID                     NAME                 DURATION    AVAILABILITY
  cfg-001                30-min Meeting       30 min      Mon-Fri 9-5
  cfg-002                1-hour Consultation  60 min      Mon-Wed 10-4
  cfg-003                Quick Call           15 min      Daily 8-8

3 configurations`

	case "scheduler bookings":
		return `Demo Mode - Bookings

  UPCOMING
  Dec 27  10:00 AM   30-min Meeting       john@client.com
  Dec 28  02:00 PM   1-hour Consultation  jane@partner.org
  Dec 30  09:00 AM   Quick Call           mike@prospect.io

3 upcoming bookings`

	case "scheduler sessions":
		return `Demo Mode - Scheduling Sessions

  ID                     CONFIGURATION        STATUS      EXPIRES
  sess-001               30-min Meeting       Active      Dec 31, 2024
  sess-002               1-hour Consultation  Active      Jan 15, 2025

2 active sessions`

	case "scheduler pages":
		return `Demo Mode - Scheduling Pages

  ID                     SLUG                 CONFIGURATION        STATUS
  page-001               meet-with-alice      30-min Meeting       Published
  page-002               consultation         1-hour Consultation  Published
  page-003               quick-chat           Quick Call           Draft

3 scheduling pages`

	case "timezone list":
		return `Demo Mode - Time Zones

  REGION          ZONE                    OFFSET    CURRENT TIME
  America         America/New_York        -05:00    10:30 AM EST
  America         America/Los_Angeles     -08:00    7:30 AM PST
  Europe          Europe/London           +00:00    3:30 PM GMT
  Europe          Europe/Paris            +01:00    4:30 PM CET
  Asia            Asia/Tokyo              +09:00    12:30 AM JST

Showing 5 of 594 time zones`

	case "timezone info":
		return `Demo Mode - Time Zone Info

  Zone:      America/New_York
  Offset:    -05:00 (EST)
  DST:       Observed
  Current:   Thu Dec 25, 2024 10:30:00 AM EST

  Next DST Transition:
  Mar 9, 2025 02:00 AM → 03:00 AM (EDT, -04:00)`

	case "timezone convert":
		return `Demo Mode - Time Conversion

  FROM:   Dec 25, 2024 10:30 AM America/New_York (EST)
  TO:     Dec 26, 2024 12:30 AM Asia/Tokyo (JST)

  Time difference: +14:00`

	case "timezone find-meeting":
		return `Demo Mode - Meeting Time Finder

  Zones: America/New_York, Europe/London, Asia/Tokyo

  Best meeting times (next 7 days):
  ┌─────────────────────────────────────────────────────────┐
  │ Dec 26  9:00 AM EST │ 2:00 PM GMT │ 11:00 PM JST        │
  │ Dec 27  9:00 AM EST │ 2:00 PM GMT │ 11:00 PM JST        │
  │ Dec 30  9:00 AM EST │ 2:00 PM GMT │ 11:00 PM JST        │
  └─────────────────────────────────────────────────────────┘

3 available time slots found`

	case "timezone dst":
		return `Demo Mode - DST Transitions

  Zone: America/New_York
  Year: 2025

  Spring Forward:  Mar 9, 2025 02:00 AM → 03:00 AM (EST → EDT)
  Fall Back:       Nov 2, 2025 02:00 AM → 01:00 AM (EDT → EST)

  Current offset: -05:00 (EST)
  Summer offset:  -04:00 (EDT)`

	case "webhook list":
		return `Demo Mode - Webhooks

  ID                     CALLBACK URL                           TRIGGERS        STATUS
  wh-001                 https://example.com/webhook/events     message.*       Active
  wh-002                 https://api.company.io/nylas            calendar.*      Active
  wh-003                 https://hooks.app.com/contacts          contact.*       Paused

3 webhooks configured`

	case "webhook triggers":
		return `Demo Mode - Available Webhook Triggers

  CATEGORY       TRIGGER                    DESCRIPTION
  message        message.created            New message received
  message        message.updated            Message modified
  message        message.opened             Message opened (tracking)
  calendar       calendar.created           New calendar added
  calendar       event.created              New event created
  calendar       event.updated              Event modified
  calendar       event.deleted              Event deleted
  contact        contact.created            New contact added
  contact        contact.updated            Contact modified
  contact        contact.deleted            Contact deleted
  grant          grant.created              New grant connected
  grant          grant.expired              Grant expired
  grant          grant.deleted              Grant removed

14 trigger types available`

	case "webhook test":
		return `Demo Mode - Webhook Test

  Webhook:    wh-001
  URL:        https://example.com/webhook/events

  Test payload sent successfully!

  Response:
  Status:     200 OK
  Latency:    142ms
  Body:       {"received": true}`

	case "webhook server":
		return `Demo Mode - Webhook Server

  Starting local webhook receiver...

  Local URL:      http://localhost:9000/webhook
  Tunnel URL:     https://abc123.ngrok.io/webhook

  Ready to receive webhooks!

  Press Ctrl+C to stop the server.`

	case "otp get":
		return `Demo Mode - OTP Code

  Account:    alice@example.com
  Service:    GitHub
  Code:       847293
  Expires:    28 seconds

  Code copied to clipboard!`

	case "otp watch":
		return `Demo Mode - Watching for OTP codes...

  Monitoring: alice@example.com
  Filter:     All services

  Waiting for new OTP codes...
  (Demo mode - would show real-time codes)

  Press Ctrl+C to stop.`

	case "otp list":
		return `Demo Mode - Configured OTP Accounts

  EMAIL                      DEFAULT    LAST OTP
  alice@example.com          ✓          2 min ago
  bob@work.com                          1 hour ago

2 accounts configured`

	case "otp messages":
		return `Demo Mode - Recent OTP Messages

  TIME          FROM                    SERVICE         CODE
  2 min ago     noreply@github.com      GitHub          847293
  15 min ago    verify@google.com       Google          531842
  1 hour ago    security@amazon.com     Amazon          729461
  2 hours ago   auth@microsoft.com      Microsoft       184629

4 recent OTP messages`

	case "admin applications":
		return `Demo Mode - Applications

  ID                     NAME                     CREATED
  app-001                Production App           Jan 15, 2024
  app-002                Development App          Mar 22, 2024

2 applications`

	case "admin connectors":
		return `Demo Mode - Connectors

  ID                     PROVIDER       NAME                 STATUS
  conn-001               google         Google Workspace     Active
  conn-002               microsoft      Microsoft 365        Active
  conn-003               imap           Custom IMAP          Inactive

3 connectors configured`

	case "admin credentials":
		return `Demo Mode - Credentials

  ID                     NAME                     TYPE           CREATED
  cred-001               Google OAuth             oauth2         Jan 15, 2024
  cred-002               MS Graph API             oauth2         Jan 20, 2024
  cred-003               IMAP Server              password       Feb 10, 2024

3 credentials stored`

	case "admin grants":
		return `Demo Mode - Grants

  ID                     EMAIL                      PROVIDER       STATUS
  grant-001              alice@example.com          Google         Active
  grant-002              bob@work.com               Microsoft      Active
  grant-003              carol@company.org          Google         Expired

3 grants`

	case "notetaker list":
		return `Demo Mode - Notetakers

  ID                     MEETING                          STATUS        CREATED
  nt-001                 Team Standup                     Completed     Dec 24
  nt-002                 Client Presentation              Recording     Now
  nt-003                 Sprint Planning                  Scheduled     Dec 27

3 notetakers`

	case "notetaker create":
		return `Demo Mode - Create Notetaker

  Created notetaker successfully!

  ID:           nt-004
  Meeting:      https://zoom.us/j/123456789
  Status:       Scheduled
  Join Time:    Immediately when meeting starts

  The notetaker bot will join and record the meeting.`

	case "notetaker media":
		return `Demo Mode - Notetaker Media

  Notetaker:    nt-001
  Meeting:      Team Standup
  Duration:     32 minutes

  Available Media:
  ✓  Video Recording    128 MB    video.mp4
  ✓  Audio Recording    24 MB     audio.mp3
  ✓  Transcript         42 KB     transcript.txt
  ✓  Summary            8 KB      summary.md

  Use --download to save files locally.`

	case "version":
		return `nylas version dev (demo mode)
Built: 2024-01-01T00:00:00Z
Go: go1.24`

	default:
		return "Demo Mode - Command: " + cmd + "\n\n(This is sample output. Connect your account with 'nylas auth login' to see real data.)"
	}
}

// Start starts the HTTP server.
