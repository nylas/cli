package mcp

// MCPTool represents an MCP tool definition.
type MCPTool struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	InputSchema JSONSchema `json:"inputSchema"`
}

// JSONSchema represents a JSON Schema object.
type JSONSchema struct {
	Type       string                `json:"type"`
	Properties map[string]JSONSchema `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Items      *JSONSchema           `json:"items,omitempty"`
	Enum       []string              `json:"enum,omitempty"`
	Desc       string                `json:"description,omitempty"`
}

// prop creates a simple typed property with a description.
func prop(typ, desc string) JSONSchema {
	return JSONSchema{Type: typ, Desc: desc}
}

// grantProp is the standard grant_id property description.
const grantDesc = "Nylas grant ID. If omitted, uses the default authenticated grant."

// epochDesc is a suffix for Unix epoch timestamp fields.
const epochDesc = " (Unix epoch timestamp)"

// participantArraySchema returns a JSON Schema for an array of {name, email} objects.
func participantArraySchema(desc string) JSONSchema {
	return JSONSchema{
		Type: "array",
		Desc: desc,
		Items: &JSONSchema{
			Type: "object",
			Properties: map[string]JSONSchema{
				"name":  prop("string", "Display name"),
				"email": prop("string", "Email address"),
			},
		},
	}
}

// objectSchema returns a schema with properties and required fields.
func objectSchema(props map[string]JSONSchema, required []string) JSONSchema {
	return JSONSchema{Type: "object", Properties: props, Required: required}
}

// registeredTools returns all 37 MCP tool definitions.
func registeredTools() []MCPTool {
	return []MCPTool{
		// ---- Email tools ----
		{
			Name:        "list_messages",
			Description: "Search and retrieve emails from the mailbox.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":        prop("string", grantDesc),
				"subject":         prop("string", "Filter by subject"),
				"from":            prop("string", "Filter by sender email"),
				"to":              prop("string", "Filter by recipient email"),
				"unread":          prop("boolean", "Filter by unread status"),
				"starred":         prop("boolean", "Filter by starred status"),
				"folder_id":       prop("string", "Filter by folder ID"),
				"received_before": prop("number", "Return messages received before this time"+epochDesc),
				"received_after":  prop("number", "Return messages received after this time"+epochDesc),
				"query":           prop("string", "Full-text search query"),
				"limit":           prop("number", "Maximum number of messages to return (default 10)"),
				"has_attachment":  prop("boolean", "Filter to messages with attachments"),
			}, nil),
		},
		{
			Name:        "get_message",
			Description: "Get a full email message including body (truncated at 10k characters).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"message_id": prop("string", "ID of the message to retrieve"),
			}, []string{"message_id"}),
		},
		{
			Name:        "send_message",
			Description: "Send an email message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":            prop("string", grantDesc),
				"to":                  participantArraySchema("List of recipient {name, email} objects"),
				"cc":                  participantArraySchema("List of CC recipient {name, email} objects"),
				"bcc":                 participantArraySchema("List of BCC recipient {name, email} objects"),
				"subject":             prop("string", "Email subject line"),
				"body":                prop("string", "Email body content (HTML or plain text)"),
				"reply_to_message_id": prop("string", "Message ID to reply to"),
			}, []string{"to", "subject", "body"}),
		},
		{
			Name:        "update_message",
			Description: "Mark a message as read/unread, star/unstar, or move to a folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"message_id": prop("string", "ID of the message to update"),
				"unread":     prop("boolean", "Set the unread status of the message"),
				"starred":    prop("boolean", "Set the starred status of the message"),
				"folders":    {Type: "array", Desc: "List of folder IDs to assign the message to", Items: &JSONSchema{Type: "string"}},
			}, []string{"message_id"}),
		},
		{
			Name:        "delete_message",
			Description: "Permanently delete an email message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"message_id": prop("string", "ID of the message to delete"),
			}, []string{"message_id"}),
		},
		{
			Name:        "smart_compose",
			Description: "Generate an AI-powered email draft from a plain-language prompt.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"prompt":   prop("string", "Description of the email to compose (max 1000 tokens)"),
			}, []string{"prompt"}),
		},
		{
			Name:        "smart_compose_reply",
			Description: "Generate an AI-powered reply to a specific email message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"message_id": prop("string", "ID of the message to reply to"),
				"prompt":     prop("string", "Instructions for how to reply"),
			}, []string{"message_id", "prompt"}),
		},
		{
			Name:        "list_drafts",
			Description: "List email drafts in the mailbox.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"limit":    prop("number", "Maximum number of drafts to return (default 10)"),
			}, nil),
		},
		{
			Name:        "get_draft",
			Description: "Get a specific email draft by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"draft_id": prop("string", "ID of the draft to retrieve"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "create_draft",
			Description: "Create a new email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":            prop("string", grantDesc),
				"subject":             prop("string", "Draft subject line"),
				"body":                prop("string", "Draft body content"),
				"to":                  participantArraySchema("List of recipient {name, email} objects"),
				"cc":                  participantArraySchema("List of CC recipient {name, email} objects"),
				"bcc":                 participantArraySchema("List of BCC recipient {name, email} objects"),
				"reply_to_message_id": prop("string", "Message ID this draft replies to"),
			}, nil),
		},
		{
			Name:        "update_draft",
			Description: "Update an existing email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"draft_id": prop("string", "ID of the draft to update"),
				"subject":  prop("string", "Updated subject line"),
				"body":     prop("string", "Updated body content"),
				"to":       participantArraySchema("Updated list of recipient {name, email} objects"),
				"cc":       participantArraySchema("Updated list of CC recipient {name, email} objects"),
				"bcc":      participantArraySchema("Updated list of BCC recipient {name, email} objects"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "send_draft",
			Description: "Send an existing email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"draft_id": prop("string", "ID of the draft to send"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "delete_draft",
			Description: "Delete an email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"draft_id": prop("string", "ID of the draft to delete"),
			}, []string{"draft_id"}),
		},

		// ---- Thread tools ----
		{
			Name:        "list_threads",
			Description: "List email threads in the mailbox.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
				"subject":  prop("string", "Filter by subject"),
				"from":     prop("string", "Filter by sender email"),
				"to":       prop("string", "Filter by recipient email"),
				"unread":   prop("boolean", "Filter by unread status"),
				"limit":    prop("number", "Maximum number of threads to return (default 10)"),
			}, nil),
		},
		{
			Name:        "get_thread",
			Description: "Get a specific email thread including its message IDs.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":  prop("string", grantDesc),
				"thread_id": prop("string", "ID of the thread to retrieve"),
			}, []string{"thread_id"}),
		},

		{
			Name:        "update_thread",
			Description: "Update a thread: mark as read/unread, star/unstar, or move to folders.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":  prop("string", grantDesc),
				"thread_id": prop("string", "ID of the thread to update"),
				"unread":    prop("boolean", "Set the unread status of the thread"),
				"starred":   prop("boolean", "Set the starred status of the thread"),
				"folders":   {Type: "array", Desc: "List of folder IDs to assign the thread to", Items: &JSONSchema{Type: "string"}},
			}, []string{"thread_id"}),
		},
		{
			Name:        "delete_thread",
			Description: "Delete an email thread and all its messages.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":  prop("string", grantDesc),
				"thread_id": prop("string", "ID of the thread to delete"),
			}, []string{"thread_id"}),
		},

		// ---- Folder tools ----
		{
			Name:        "list_folders",
			Description: "List all email folders for a grant.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
			}, nil),
		},
		{
			Name:        "get_folder",
			Description: "Get a specific email folder by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":  prop("string", grantDesc),
				"folder_id": prop("string", "ID of the folder to retrieve"),
			}, []string{"folder_id"}),
		},
		{
			Name:        "create_folder",
			Description: "Create a new email folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":         prop("string", grantDesc),
				"name":             prop("string", "Display name for the folder"),
				"parent_id":        prop("string", "Parent folder ID for nested folders"),
				"background_color": prop("string", "Background color (hex, e.g. #FF0000)"),
				"text_color":       prop("string", "Text color (hex, e.g. #FFFFFF)"),
			}, []string{"name"}),
		},

		{
			Name:        "update_folder",
			Description: "Update an email folder (rename, move, change color).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":         prop("string", grantDesc),
				"folder_id":        prop("string", "ID of the folder to update"),
				"name":             prop("string", "New display name for the folder"),
				"parent_id":        prop("string", "New parent folder ID"),
				"background_color": prop("string", "Background color (hex, e.g. #FF0000)"),
				"text_color":       prop("string", "Text color (hex, e.g. #FFFFFF)"),
			}, []string{"folder_id"}),
		},
		{
			Name:        "delete_folder",
			Description: "Delete an email folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":  prop("string", grantDesc),
				"folder_id": prop("string", "ID of the folder to delete"),
			}, []string{"folder_id"}),
		},

		// ---- Attachment tools ----
		{
			Name:        "list_attachments",
			Description: "List all attachments for a specific message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"message_id": prop("string", "ID of the message whose attachments to list"),
			}, []string{"message_id"}),
		},
		{
			Name:        "get_attachment",
			Description: "Get attachment metadata for a specific attachment (no binary download).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":      prop("string", grantDesc),
				"message_id":    prop("string", "ID of the message containing the attachment"),
				"attachment_id": prop("string", "ID of the attachment to retrieve"),
			}, []string{"message_id", "attachment_id"}),
		},

		// ---- Scheduled message tools ----
		{
			Name:        "list_scheduled_messages",
			Description: "List all scheduled (send-later) messages.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
			}, nil),
		},
		{
			Name:        "cancel_scheduled_message",
			Description: "Cancel a scheduled message before it is sent.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"schedule_id": prop("string", "ID of the scheduled message to cancel"),
			}, []string{"schedule_id"}),
		},

		// ---- Calendar tools ----
		{
			Name:        "list_calendars",
			Description: "List all calendars for a grant.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id": prop("string", grantDesc),
			}, nil),
		},
		{
			Name:        "get_calendar",
			Description: "Get a specific calendar by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", "ID of the calendar to retrieve"),
			}, []string{"calendar_id"}),
		},
		{
			Name:        "create_calendar",
			Description: "Create a new calendar.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"name":        prop("string", "Display name for the calendar"),
				"description": prop("string", "Description of the calendar"),
				"location":    prop("string", "Physical or virtual location for the calendar"),
				"timezone":    prop("string", "IANA timezone (e.g. America/New_York)"),
			}, []string{"name"}),
		},

		{
			Name:        "update_calendar",
			Description: "Update a calendar's name, description, timezone, or color.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", "ID of the calendar to update"),
				"name":        prop("string", "New display name"),
				"description": prop("string", "New description"),
				"location":    prop("string", "New location"),
				"timezone":    prop("string", "New IANA timezone"),
			}, []string{"calendar_id"}),
		},
		{
			Name:        "delete_calendar",
			Description: "Delete a calendar.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", "ID of the calendar to delete"),
			}, []string{"calendar_id"}),
		},

		// ---- Event tools ----
		{
			Name:        "list_events",
			Description: "List calendar events within an optional time range.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":         prop("string", grantDesc),
				"calendar_id":      prop("string", `Calendar ID to query (default "primary")`),
				"title":            prop("string", "Filter by event title"),
				"start":            prop("number", "Return events starting at or after this time"+epochDesc),
				"end":              prop("number", "Return events starting at or before this time"+epochDesc),
				"limit":            prop("number", "Maximum number of events to return (default 10)"),
				"expand_recurring": prop("boolean", "Expand recurring events into individual instances"),
				"show_cancelled":   prop("boolean", "Include cancelled events in results"),
			}, nil),
		},
		{
			Name:        "get_event",
			Description: "Get a specific calendar event by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", `Calendar ID containing the event (default "primary")`),
				"event_id":    prop("string", "ID of the event to retrieve"),
			}, []string{"event_id"}),
		},
		{
			Name:        "create_event",
			Description: "Create a new calendar event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":         prop("string", grantDesc),
				"calendar_id":      prop("string", `Calendar to create the event in (default "primary")`),
				"title":            prop("string", "Event title"),
				"description":      prop("string", "Event description or notes"),
				"location":         prop("string", "Physical or virtual location"),
				"start_time":       prop("number", "Event start time"+epochDesc+" (use for timed events)"),
				"end_time":         prop("number", "Event end time"+epochDesc+" (use for timed events)"),
				"start_date":       prop("string", "Start date for all-day events (YYYY-MM-DD)"),
				"end_date":         prop("string", "End date for all-day events (YYYY-MM-DD)"),
				"participants":     participantArraySchema("List of participant {name, email} objects"),
				"busy":             prop("boolean", "Whether the event blocks time as busy"),
				"visibility":       {Type: "string", Desc: "Event visibility", Enum: []string{"public", "private"}},
				"conferencing_url": prop("string", "Video conferencing URL"),
				"reminders": {
					Type: "array",
					Desc: "List of reminder objects with minutes and method",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"minutes": prop("number", "Minutes before event to send reminder"),
							"method":  prop("string", "Reminder delivery method (e.g. email, popup)"),
						},
					},
				},
			}, []string{"title"}),
		},
		{
			Name:        "update_event",
			Description: "Update an existing calendar event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":     prop("string", grantDesc),
				"calendar_id":  prop("string", `Calendar containing the event (default "primary")`),
				"event_id":     prop("string", "ID of the event to update"),
				"title":        prop("string", "Updated event title"),
				"description":  prop("string", "Updated event description"),
				"location":     prop("string", "Updated location"),
				"start_time":   prop("number", "Updated start time"+epochDesc),
				"end_time":     prop("number", "Updated end time"+epochDesc),
				"participants": participantArraySchema("Updated list of participant {name, email} objects"),
				"busy":         prop("boolean", "Whether the event blocks time as busy"),
				"visibility":   {Type: "string", Desc: "Event visibility", Enum: []string{"public", "private"}},
			}, []string{"event_id"}),
		},
		{
			Name:        "delete_event",
			Description: "Delete a calendar event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", `Calendar containing the event (default "primary")`),
				"event_id":    prop("string", "ID of the event to delete"),
			}, []string{"event_id"}),
		},

		// ---- Availability tools ----
		{
			Name:        "get_free_busy",
			Description: "Check the free/busy status for one or more email addresses within a time range.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"emails":     {Type: "array", Desc: "Email addresses to check", Items: &JSONSchema{Type: "string"}},
				"start_time": prop("number", "Start of the time range to check"+epochDesc),
				"end_time":   prop("number", "End of the time range to check"+epochDesc),
			}, []string{"emails", "start_time", "end_time"}),
		},
		{
			Name:        "get_availability",
			Description: "Find available meeting slots for a set of participants.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"participants": {
					Type: "array",
					Desc: "List of participants with their calendars",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"email":        prop("string", "Participant email address"),
							"calendar_ids": {Type: "array", Desc: "Calendar IDs to check for this participant", Items: &JSONSchema{Type: "string"}},
						},
					},
				},
				"start_time":       prop("number", "Start of the availability window"+epochDesc),
				"end_time":         prop("number", "End of the availability window"+epochDesc),
				"duration_minutes": prop("number", "Required meeting duration in minutes"),
				"interval_minutes": prop("number", "Interval between candidate slots in minutes"),
				"round_to":         prop("number", "Round slot start times to this many minutes"),
			}, []string{"participants", "start_time", "end_time", "duration_minutes"}),
		},

		// ---- RSVP tools ----
		{
			Name:        "send_rsvp",
			Description: "Send an RSVP response (accept, decline, maybe) to a calendar event invitation.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":    prop("string", grantDesc),
				"calendar_id": prop("string", `Calendar ID containing the event (default "primary")`),
				"event_id":    prop("string", "ID of the event to RSVP to"),
				"status":      prop("string", `RSVP status: "yes" (accept), "no" (decline), or "maybe" (tentative)`),
				"comment":     prop("string", "Optional comment to include with the RSVP"),
			}, []string{"event_id", "status"}),
		},

		// ---- Contact tools ----
		{
			Name:        "list_contacts",
			Description: "List or search contacts.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":     prop("string", grantDesc),
				"email":        prop("string", "Filter by email address"),
				"phone_number": prop("string", "Filter by phone number"),
				"source":       {Type: "string", Desc: "Contact source filter", Enum: []string{"address_book", "inbox", "domain"}},
				"limit":        prop("number", "Maximum number of contacts to return (default 10)"),
				"group":        prop("string", "Filter by contact group name"),
			}, nil),
		},
		{
			Name:        "get_contact",
			Description: "Get full detail for a specific contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"contact_id": prop("string", "ID of the contact to retrieve"),
			}, []string{"contact_id"}),
		},
		{
			Name:        "create_contact",
			Description: "Create a new contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":     prop("string", grantDesc),
				"given_name":   prop("string", "Contact first name"),
				"surname":      prop("string", "Contact last name"),
				"nickname":     prop("string", "Contact nickname"),
				"company_name": prop("string", "Company or organization name"),
				"job_title":    prop("string", "Job title"),
				"emails": {
					Type: "array",
					Desc: "List of email objects with email and type",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"email": prop("string", "Email address"),
							"type":  prop("string", "Email type (e.g. work, home)"),
						},
					},
				},
				"phone_numbers": {
					Type: "array",
					Desc: "List of phone number objects with number and type",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"number": prop("string", "Phone number"),
							"type":   prop("string", "Phone type (e.g. work, mobile)"),
						},
					},
				},
				"notes": prop("string", "Free-form notes about the contact"),
			}, nil),
		},

		{
			Name:        "update_contact",
			Description: "Update a contact's information.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":     prop("string", grantDesc),
				"contact_id":   prop("string", "ID of the contact to update"),
				"given_name":   prop("string", "Updated first name"),
				"surname":      prop("string", "Updated last name"),
				"nickname":     prop("string", "Updated nickname"),
				"company_name": prop("string", "Updated company name"),
				"job_title":    prop("string", "Updated job title"),
				"notes":        prop("string", "Updated notes"),
			}, []string{"contact_id"}),
		},
		{
			Name:        "delete_contact",
			Description: "Delete a contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"grant_id":   prop("string", grantDesc),
				"contact_id": prop("string", "ID of the contact to delete"),
			}, []string{"contact_id"}),
		},

		// ---- Utility tools (no API call) ----
		{
			Name:        "current_time",
			Description: "Get the current date and time, optionally in a specific timezone.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"timezone": prop("string", `IANA timezone name (e.g. "America/New_York"). Defaults to local system timezone.`),
			}, nil),
		},
		{
			Name:        "epoch_to_datetime",
			Description: "Convert a Unix timestamp to a human-readable date and time string.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"epoch":    prop("number", "Unix epoch timestamp to convert"),
				"timezone": prop("string", `IANA timezone name for output (e.g. "America/New_York"). Defaults to local system timezone.`),
			}, []string{"epoch"}),
		},
		{
			Name:        "datetime_to_epoch",
			Description: "Convert a datetime string to a Unix timestamp.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"datetime": prop("string", `Datetime string to convert. Accepts RFC3339 (e.g. "2024-01-15T14:30:00Z") or "2006-01-02 15:04:05" format.`),
				"timezone": prop("string", `IANA timezone to interpret the datetime in if no timezone offset is present (e.g. "America/New_York").`),
			}, []string{"datetime"}),
		},
	}
}
