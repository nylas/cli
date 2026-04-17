package ui

import (
	"embed"
	"encoding/json"
	"html/template"
	"strings"
)

//go:embed templates/*.gohtml templates/**/*.gohtml
var templateFiles embed.FS

// Command represents a CLI command with metadata.
type Command struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Cmd         string `json:"cmd"`
	Desc        string `json:"desc"`
	ParamName   string `json:"paramName,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

// Commands holds categorized commands.
type Commands struct {
	Auth      []Command `json:"auth"`
	Email     []Command `json:"email"`
	Calendar  []Command `json:"calendar"`
	Contacts  []Command `json:"contacts"`
	Scheduler []Command `json:"scheduler"`
	Timezone  []Command `json:"timezone"`
	Webhook   []Command `json:"webhook"`
	OTP       []Command `json:"otp"`
	Admin     []Command `json:"admin"`
	Notetaker []Command `json:"notetaker"`
}

// PageData contains all data needed to render the page.
type PageData struct {
	Configured        bool
	DemoMode          bool
	ClientID          string
	Region            string
	HasAPIKey         bool
	DefaultGrant      string
	DefaultGrantEmail string
	Grants            []Grant
	Commands          Commands
}

// safeJSJSON converts data to JSON safe for embedding in HTML <script> tags.
// Go's json.Marshal already escapes < and > as \u003c and \u003e, which prevents
// XSS attacks like </script> injection. This wrapper adds error handling and
// documents the security properties.
//
// Security: json.Marshal escapes:
//   - < → \u003c (prevents </script>, <!-- injection)
//   - > → \u003e (prevents --> injection)
//   - & → \u0026 (prevents &-based escapes)
//
// This makes the output safe for embedding in HTML script contexts.
func safeJSJSON(v any) template.JS {
	data, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	//nolint:gosec // G203: json.Marshal escapes <, >, & as unicode - safe for script context
	return template.JS(data)
}

// GrantsJSON returns grants as JSON for JavaScript.
func (p PageData) GrantsJSON() template.JS {
	return safeJSJSON(p.Grants)
}

// CommandsJSON returns commands as JSON for JavaScript.
func (p PageData) CommandsJSON() template.JS {
	return safeJSJSON(p.Commands)
}

// Template functions.
var templateFuncs = template.FuncMap{
	"upper": strings.ToUpper,
	"lower": strings.ToLower,
	"slice": func(s string, start, end int) string {
		if start >= len(s) {
			return ""
		}
		if end > len(s) {
			end = len(s)
		}
		return s[start:end]
	},
}

// loadTemplates parses all template files.
func loadTemplates() (*template.Template, error) {
	return template.New("").Funcs(templateFuncs).ParseFS(
		templateFiles,
		"templates/*.gohtml",
		"templates/partials/*.gohtml",
		"templates/pages/*.gohtml",
	)
}

// GetDefaultCommands returns the default command definitions.
func GetDefaultCommands() Commands {
	return Commands{
		Auth: []Command{
			{Key: "login", Title: "Login", Cmd: "auth login", Desc: "Authenticate with Nylas"},
			{Key: "logout", Title: "Logout", Cmd: "auth logout", Desc: "Sign out of current account"},
			{Key: "status", Title: "Status", Cmd: "auth status", Desc: "Check authentication status"},
			{Key: "whoami", Title: "Who Am I", Cmd: "auth whoami", Desc: "Show current user info"},
			{Key: "list", Title: "List", Cmd: "auth list", Desc: "List all authenticated accounts"},
			{Key: "show", Title: "Show", Cmd: "auth show", Desc: "Show account details"},
			{Key: "switch", Title: "Switch", Cmd: "auth switch", Desc: "Switch between accounts"},
			{Key: "config", Title: "Config", Cmd: "auth config", Desc: "View configuration"},
			{Key: "providers", Title: "Providers", Cmd: "auth providers", Desc: "List auth providers"},
			{Key: "scopes", Title: "Scopes", Cmd: "auth scopes", Desc: "List OAuth scopes"},
			{Key: "token", Title: "Token", Cmd: "auth token", Desc: "Show access token"},
			{Key: "migrate", Title: "Migrate", Cmd: "auth migrate", Desc: "Migrate credentials"},
		},
		Email: []Command{
			{Key: "list", Title: "List", Cmd: "email list", Desc: "List recent emails"},
			{Key: "read", Title: "Read", Cmd: "email read", Desc: "Read a specific email", ParamName: "message-id", Placeholder: "Enter message ID..."},
			{Key: "search", Title: "Search", Cmd: "email search", Desc: "Search emails", ParamName: "query", Placeholder: "Enter search query..."},
			{Key: "drafts", Title: "Drafts", Cmd: "email drafts", Desc: "List draft emails"},
			{Key: "folders", Title: "Folders", Cmd: "email folders", Desc: "List email folders"},
			{Key: "threads", Title: "Threads", Cmd: "email threads", Desc: "List email threads"},
			{Key: "scheduled", Title: "Scheduled", Cmd: "email scheduled", Desc: "List scheduled emails"},
			{Key: "attachments", Title: "Attachments", Cmd: "email attachments", Desc: "List attachments"},
			{Key: "metadata", Title: "Metadata", Cmd: "email metadata", Desc: "Show email metadata"},
			{Key: "tracking-info", Title: "Tracking", Cmd: "email tracking-info", Desc: "Email tracking info"},
			{Key: "ai-analyze", Title: "AI Analyze", Cmd: "email ai analyze", Desc: "AI inbox analysis"},
			{Key: "smart-compose", Title: "Smart Compose", Cmd: "email smart-compose", Desc: "AI-assisted compose", ParamName: "prompt", Placeholder: "Enter prompt..."},
		},
		Calendar: []Command{
			{Key: "list", Title: "List", Cmd: "calendar list", Desc: "List calendars"},
			{Key: "events", Title: "Events", Cmd: "calendar events", Desc: "List calendar events"},
			{Key: "show", Title: "Show", Cmd: "calendar show", Desc: "Show event details", ParamName: "event-id", Placeholder: "Enter event ID..."},
			{Key: "availability", Title: "Availability", Cmd: "calendar availability", Desc: "Check availability"},
			{Key: "find-time", Title: "Find Time", Cmd: "calendar find-time", Desc: "Find available time slots"},
			{Key: "recurring-list", Title: "Recurring List", Cmd: "calendar recurring list", Desc: "List recurring event instances", ParamName: "event-id", Placeholder: "Enter event ID..."},
			{Key: "schedule", Title: "Schedule", Cmd: "calendar schedule", Desc: "View schedule"},
			{Key: "virtual-list", Title: "Virtual List", Cmd: "calendar virtual list", Desc: "List virtual calendars"},
			{Key: "virtual-create", Title: "Virtual Create", Cmd: "calendar virtual create", Desc: "Create virtual calendar"},
			{Key: "ai-analyze", Title: "AI Analyze", Cmd: "calendar ai analyze", Desc: "AI meeting analysis"},
			{Key: "ai-focus", Title: "AI Focus Time", Cmd: "calendar ai focus-time", Desc: "AI focus time protection"},
			{Key: "ai-conflicts", Title: "AI Conflicts", Cmd: "calendar ai conflicts", Desc: "AI conflict detection"},
		},
		Contacts: []Command{
			{Key: "list", Title: "List", Cmd: "contacts list", Desc: "List all contacts"},
			{Key: "show", Title: "Show", Cmd: "contacts show", Desc: "Show contact details", ParamName: "contact-id", Placeholder: "Enter contact ID..."},
			{Key: "search", Title: "Search", Cmd: "contacts search", Desc: "Search contacts", ParamName: "query", Placeholder: "Enter search query..."},
			{Key: "create", Title: "Create", Cmd: "contacts create", Desc: "Create a new contact"},
			{Key: "update", Title: "Update", Cmd: "contacts update", Desc: "Update a contact", ParamName: "contact-id", Placeholder: "Enter contact ID..."},
			{Key: "delete", Title: "Delete", Cmd: "contacts delete", Desc: "Delete a contact", ParamName: "contact-id", Placeholder: "Enter contact ID..."},
			{Key: "groups-list", Title: "List Groups", Cmd: "contacts groups list", Desc: "List contact groups"},
			{Key: "groups-show", Title: "Show Group", Cmd: "contacts groups show", Desc: "Show group details", ParamName: "group-id", Placeholder: "Enter group ID..."},
			{Key: "photo-info", Title: "Photo Info", Cmd: "contacts photo info", Desc: "Show photo info", ParamName: "contact-id", Placeholder: "Enter contact ID..."},
			{Key: "sync", Title: "Sync Info", Cmd: "contacts sync", Desc: "Contact sync info"},
		},
		Scheduler: []Command{
			{Key: "config-list", Title: "List Configs", Cmd: "scheduler configurations list", Desc: "List scheduler configurations"},
			{Key: "config-create", Title: "Create Config", Cmd: "scheduler configurations create", Desc: "Create a scheduler configuration"},
			{Key: "config-show", Title: "Show Config", Cmd: "scheduler configurations show", Desc: "Show configuration details", ParamName: "config-id", Placeholder: "Enter configuration ID..."},
			{Key: "session-create", Title: "Create Session", Cmd: "scheduler sessions create", Desc: "Create a scheduling session"},
			{Key: "session-show", Title: "Show Session", Cmd: "scheduler sessions show", Desc: "Show session details", ParamName: "session-id", Placeholder: "Enter session ID..."},
			{Key: "booking-list", Title: "List Bookings", Cmd: "scheduler bookings list", Desc: "List scheduler bookings"},
			{Key: "booking-show", Title: "Show Booking", Cmd: "scheduler bookings show", Desc: "Show booking details", ParamName: "booking-id", Placeholder: "Enter booking ID..."},
			{Key: "page-list", Title: "List Pages", Cmd: "scheduler pages list", Desc: "List scheduler pages"},
			{Key: "page-create", Title: "Create Page", Cmd: "scheduler pages create", Desc: "Create a scheduler page"},
		},
		Timezone: []Command{
			{Key: "list", Title: "List", Cmd: "timezone list", Desc: "List all time zones"},
			{Key: "info", Title: "Info", Cmd: "timezone info", Desc: "Get time zone info", ParamName: "zone", Placeholder: "e.g., America/New_York"},
			{Key: "convert", Title: "Convert", Cmd: "timezone convert", Desc: "Convert time between zones"},
			{Key: "find-meeting", Title: "Find Meeting", Cmd: "timezone find-meeting", Desc: "Find meeting times across zones"},
			{Key: "dst", Title: "DST", Cmd: "timezone dst", Desc: "Check DST transitions"},
		},
		Webhook: []Command{
			{Key: "list", Title: "List", Cmd: "webhook list", Desc: "List all webhooks"},
			{Key: "show", Title: "Show", Cmd: "webhook show", Desc: "Show webhook details", ParamName: "webhook-id", Placeholder: "Enter webhook ID..."},
			{Key: "create", Title: "Create", Cmd: "webhook create", Desc: "Create a new webhook"},
			{Key: "update", Title: "Update", Cmd: "webhook update", Desc: "Update a webhook", ParamName: "webhook-id", Placeholder: "Enter webhook ID..."},
			{Key: "delete", Title: "Delete", Cmd: "webhook delete", Desc: "Delete a webhook", ParamName: "webhook-id", Placeholder: "Enter webhook ID..."},
			{Key: "triggers", Title: "Triggers", Cmd: "webhook triggers", Desc: "List available trigger types"},
			{Key: "test", Title: "Test", Cmd: "webhook test", Desc: "Test webhook functionality"},
			{Key: "server", Title: "Server", Cmd: "webhook server", Desc: "Start local webhook server"},
		},
		OTP: []Command{
			{Key: "get", Title: "Get", Cmd: "otp get", Desc: "Get the latest OTP code"},
			{Key: "watch", Title: "Watch", Cmd: "otp watch", Desc: "Watch for new OTP codes"},
			{Key: "list", Title: "List", Cmd: "otp list", Desc: "List configured accounts"},
			{Key: "messages", Title: "Messages", Cmd: "otp messages", Desc: "Show recent messages"},
		},
		Admin: []Command{
			{Key: "apps-list", Title: "List Apps", Cmd: "admin applications list", Desc: "List applications"},
			{Key: "apps-show", Title: "Show App", Cmd: "admin applications show", Desc: "Show application details", ParamName: "app-id", Placeholder: "Enter application ID..."},
			{Key: "connectors-list", Title: "List Connectors", Cmd: "admin connectors list", Desc: "List connectors"},
			{Key: "connectors-show", Title: "Show Connector", Cmd: "admin connectors show", Desc: "Show connector details", ParamName: "connector-id", Placeholder: "Enter connector ID..."},
			{Key: "credentials-list", Title: "List Credentials", Cmd: "admin credentials list", Desc: "List credentials"},
			{Key: "grants-list", Title: "List Grants", Cmd: "admin grants list", Desc: "List grants"},
			{Key: "grants-stats", Title: "Grant Stats", Cmd: "admin grants stats", Desc: "Show grant statistics"},
		},
		Notetaker: []Command{
			{Key: "list", Title: "List", Cmd: "notetaker list", Desc: "List all notetakers"},
			{Key: "show", Title: "Show", Cmd: "notetaker show", Desc: "Show notetaker details", ParamName: "notetaker-id", Placeholder: "Enter notetaker ID..."},
			{Key: "create", Title: "Create", Cmd: "notetaker create", Desc: "Create a new notetaker"},
			{Key: "delete", Title: "Delete", Cmd: "notetaker delete", Desc: "Delete a notetaker", ParamName: "notetaker-id", Placeholder: "Enter notetaker ID..."},
			{Key: "media", Title: "Media", Cmd: "notetaker media", Desc: "Get recording/transcript", ParamName: "notetaker-id", Placeholder: "Enter notetaker ID..."},
		},
	}
}
