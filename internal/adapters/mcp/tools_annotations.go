package mcp

import "strings"

// toolTitles maps tool names to human-readable titles.
var toolTitles = map[string]string{
	// Email tools
	"list_messages":       "List Email Messages",
	"get_message":         "Get Email Message",
	"send_message":        "Send Email Message",
	"update_message":      "Update Email Message",
	"delete_message":      "Delete Email Message",
	"smart_compose":       "Smart Compose Email",
	"smart_compose_reply": "Smart Compose Reply",

	// Draft tools
	"list_drafts":  "List Email Drafts",
	"get_draft":    "Get Email Draft",
	"create_draft": "Create Email Draft",
	"update_draft": "Update Email Draft",
	"send_draft":   "Send Email Draft",
	"delete_draft": "Delete Email Draft",

	// Thread tools
	"list_threads":  "List Email Threads",
	"get_thread":    "Get Email Thread",
	"update_thread": "Update Email Thread",
	"delete_thread": "Delete Email Thread",

	// Folder tools
	"list_folders":  "List Email Folders",
	"get_folder":    "Get Email Folder",
	"create_folder": "Create Email Folder",
	"update_folder": "Update Email Folder",
	"delete_folder": "Delete Email Folder",

	// Attachment tools
	"list_attachments": "List Message Attachments",
	"get_attachment":   "Get Attachment Metadata",

	// Scheduled message tools
	"list_scheduled_messages":  "List Scheduled Messages",
	"cancel_scheduled_message": "Cancel Scheduled Message",

	// Calendar tools
	"list_calendars":  "List Calendars",
	"get_calendar":    "Get Calendar",
	"create_calendar": "Create Calendar",
	"update_calendar": "Update Calendar",
	"delete_calendar": "Delete Calendar",

	// Event tools
	"list_events":  "List Calendar Events",
	"get_event":    "Get Calendar Event",
	"create_event": "Create Calendar Event",
	"update_event": "Update Calendar Event",
	"delete_event": "Delete Calendar Event",

	// Availability tools
	"get_free_busy":    "Check Free/Busy Status",
	"get_availability": "Find Available Meeting Slots",

	// RSVP tools
	"send_rsvp": "Send Event RSVP",

	// Contact tools
	"list_contacts":  "List Contacts",
	"get_contact":    "Get Contact",
	"create_contact": "Create Contact",
	"update_contact": "Update Contact",
	"delete_contact": "Delete Contact",

	// Utility tools
	"current_time":      "Get Current Time",
	"epoch_to_datetime": "Convert Epoch to Datetime",
	"datetime_to_epoch": "Convert Datetime to Epoch",
}

// localUtilityTools are tools that don't make API calls.
var localUtilityTools = map[string]bool{
	"current_time":      true,
	"epoch_to_datetime": true,
	"datetime_to_epoch": true,
}

// annotateTools applies titles and behavior annotations to all tools in-place.
func annotateTools(tools []MCPTool) {
	for i := range tools {
		name := tools[i].Name

		// Apply title.
		if title, ok := toolTitles[name]; ok {
			tools[i].Title = title
		}

		// Apply annotations based on tool name prefix / category.
		tools[i].Annotations = buildAnnotations(name)
	}
}

// buildAnnotations returns the appropriate ToolAnnotations for a tool based on its name.
func buildAnnotations(name string) *ToolAnnotations {
	// Local utility tools: read-only, idempotent, no open-world (no API call).
	if localUtilityTools[name] {
		return &ToolAnnotations{
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
		}
	}

	// Read-only tools: list_*, get_*, get_free_busy, get_availability.
	if strings.HasPrefix(name, "list_") || strings.HasPrefix(name, "get_") {
		return &ToolAnnotations{
			ReadOnlyHint:   boolPtr(true),
			IdempotentHint: boolPtr(true),
			OpenWorldHint:  boolPtr(true),
		}
	}

	// Destructive tools: delete_*, cancel_scheduled_message.
	if strings.HasPrefix(name, "delete_") || name == "cancel_scheduled_message" {
		return &ToolAnnotations{
			DestructiveHint: boolPtr(true),
			OpenWorldHint:   boolPtr(true),
		}
	}

	// Mutating tools: send_*, create_*, update_*, smart_compose*.
	return &ToolAnnotations{
		OpenWorldHint: boolPtr(true),
	}
}
