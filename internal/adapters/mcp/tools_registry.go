package mcp

// registeredTools returns all MCP tool definitions with annotations and titles.
func registeredTools() []MCPTool {
	tools := []MCPTool{
		// ---- Email tools ----
		{
			Name:        "list_messages",
			Description: "Search and retrieve emails.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"subject":         prop("string", "Filter by subject"),
				"from":            prop("string", "Sender email"),
				"to":              prop("string", "Recipient email"),
				"unread":          prop("boolean", "Filter unread"),
				"starred":         prop("boolean", "Filter starred"),
				"folder_id":       prop("string", "Folder ID"),
				"received_before": prop("number", "Before this time"+epochDesc),
				"received_after":  prop("number", "After this time"+epochDesc),
				"query":           prop("string", "Full-text search query"),
				"limit":           prop("number", "Max results (default 200, max 200 per call)"),
				"has_attachment":  prop("boolean", "Has attachments"),
				"page_token":      prop("string", "Pagination cursor from previous response"),
			}, nil),
		},
		{
			Name:        "get_message",
			Description: "Get full email message with body (truncated at 10k chars).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id": prop("string", "Message ID"),
			}, []string{"message_id"}),
		},
		{
			Name:        "send_message",
			Description: "Send an email.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"to":                  participantArraySchema("Recipients"),
				"cc":                  participantArraySchema("CC recipients"),
				"bcc":                 participantArraySchema("BCC recipients"),
				"subject":             prop("string", "Subject"),
				"body":                prop("string", "Body (HTML or text)"),
				"reply_to_message_id": prop("string", "Message ID to reply to"),
			}, []string{"to", "subject", "body"}),
		},
		{
			Name:        "update_message",
			Description: "Mark read/unread, star/unstar, or move to folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id": prop("string", "Message ID"),
				"unread":     prop("boolean", "Set unread status"),
				"starred":    prop("boolean", "Set starred status"),
				"folders":    {Type: "array", Desc: "Folder IDs", Items: &JSONSchema{Type: "string"}},
			}, []string{"message_id"}),
		},
		{
			Name:        "delete_message",
			Description: "Permanently delete an email.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id": prop("string", "Message ID"),
			}, []string{"message_id"}),
		},
		{
			Name:        "smart_compose",
			Description: "AI-generate an email draft from a prompt.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"prompt": prop("string", "What to compose (max 1000 tokens)"),
			}, []string{"prompt"}),
		},
		{
			Name:        "smart_compose_reply",
			Description: "AI-generate a reply to a message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id": prop("string", "Message to reply to"),
				"prompt":     prop("string", "Reply instructions"),
			}, []string{"message_id", "prompt"}),
		},
		{
			Name:        "list_drafts",
			Description: "List email drafts.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"limit": prop("number", "Max results (default 10)"),
			}, nil),
		},
		{
			Name:        "get_draft",
			Description: "Get a draft by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"draft_id": prop("string", "Draft ID"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "create_draft",
			Description: "Create an email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"subject":             prop("string", "Subject"),
				"body":                prop("string", "Body"),
				"to":                  participantArraySchema("Recipients"),
				"cc":                  participantArraySchema("CC recipients"),
				"bcc":                 participantArraySchema("BCC recipients"),
				"reply_to_message_id": prop("string", "Message ID this replies to"),
			}, nil),
		},
		{
			Name:        "update_draft",
			Description: "Update an email draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"draft_id": prop("string", "Draft ID"),
				"subject":  prop("string", "Subject"),
				"body":     prop("string", "Body"),
				"to":       participantArraySchema("Recipients"),
				"cc":       participantArraySchema("CC recipients"),
				"bcc":      participantArraySchema("BCC recipients"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "send_draft",
			Description: "Send a draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"draft_id": prop("string", "Draft ID"),
			}, []string{"draft_id"}),
		},
		{
			Name:        "delete_draft",
			Description: "Delete a draft.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"draft_id": prop("string", "Draft ID"),
			}, []string{"draft_id"}),
		},

		// ---- Thread tools ----
		{
			Name:        "list_threads",
			Description: "List email threads.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"subject": prop("string", "Filter by subject"),
				"from":    prop("string", "Sender email"),
				"to":      prop("string", "Recipient email"),
				"unread":  prop("boolean", "Filter unread"),
				"limit":   prop("number", "Max results (default 10)"),
			}, nil),
		},
		{
			Name:        "get_thread",
			Description: "Get a thread with its message IDs.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"thread_id": prop("string", "Thread ID"),
			}, []string{"thread_id"}),
		},

		{
			Name:        "update_thread",
			Description: "Mark thread read/unread, star/unstar, or move to folders.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"thread_id": prop("string", "Thread ID"),
				"unread":    prop("boolean", "Set unread status"),
				"starred":   prop("boolean", "Set starred status"),
				"folders":   {Type: "array", Desc: "Folder IDs", Items: &JSONSchema{Type: "string"}},
			}, []string{"thread_id"}),
		},
		{
			Name:        "delete_thread",
			Description: "Delete a thread and all its messages.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"thread_id": prop("string", "Thread ID"),
			}, []string{"thread_id"}),
		},

		// ---- Folder tools ----
		{
			Name:        "list_folders",
			Description: "List email folders.",
			InputSchema: objectSchema(map[string]JSONSchema{}, nil),
		},
		{
			Name:        "get_folder",
			Description: "Get a folder by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"folder_id": prop("string", "Folder ID"),
			}, []string{"folder_id"}),
		},
		{
			Name:        "create_folder",
			Description: "Create an email folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"name":             prop("string", "Folder name"),
				"parent_id":        prop("string", "Parent folder ID"),
				"background_color": prop("string", "Background hex color"),
				"text_color":       prop("string", "Text hex color"),
			}, []string{"name"}),
		},

		{
			Name:        "update_folder",
			Description: "Update a folder (rename, move, recolor).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"folder_id":        prop("string", "Folder ID"),
				"name":             prop("string", "New name"),
				"parent_id":        prop("string", "New parent folder ID"),
				"background_color": prop("string", "Background hex color"),
				"text_color":       prop("string", "Text hex color"),
			}, []string{"folder_id"}),
		},
		{
			Name:        "delete_folder",
			Description: "Delete an email folder.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"folder_id": prop("string", "Folder ID"),
			}, []string{"folder_id"}),
		},

		// ---- Attachment tools ----
		{
			Name:        "list_attachments",
			Description: "List attachments for a message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id": prop("string", "Message ID"),
			}, []string{"message_id"}),
		},
		{
			Name:        "get_attachment",
			Description: "Get attachment metadata (no binary).",
			InputSchema: objectSchema(map[string]JSONSchema{
				"message_id":    prop("string", "Message ID"),
				"attachment_id": prop("string", "Attachment ID"),
			}, []string{"message_id", "attachment_id"}),
		},

		// ---- Scheduled message tools ----
		{
			Name:        "list_scheduled_messages",
			Description: "List scheduled (send-later) messages.",
			InputSchema: objectSchema(map[string]JSONSchema{}, nil),
		},
		{
			Name:        "cancel_scheduled_message",
			Description: "Cancel a scheduled message.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"schedule_id": prop("string", "Schedule ID"),
			}, []string{"schedule_id"}),
		},

		// ---- Calendar tools ----
		{
			Name:        "list_calendars",
			Description: "List calendars.",
			InputSchema: objectSchema(map[string]JSONSchema{}, nil),
		},
		{
			Name:        "get_calendar",
			Description: "Get a calendar by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", "Calendar ID"),
			}, []string{"calendar_id"}),
		},
		{
			Name:        "create_calendar",
			Description: "Create a calendar.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"name":        prop("string", "Calendar name"),
				"description": prop("string", "Description"),
				"location":    prop("string", "Location"),
				"timezone":    prop("string", "IANA timezone (e.g. America/New_York)"),
			}, []string{"name"}),
		},

		{
			Name:        "update_calendar",
			Description: "Update a calendar.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", "Calendar ID"),
				"name":        prop("string", "Name"),
				"description": prop("string", "Description"),
				"location":    prop("string", "Location"),
				"timezone":    prop("string", "IANA timezone"),
			}, []string{"calendar_id"}),
		},
		{
			Name:        "delete_calendar",
			Description: "Delete a calendar.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", "Calendar ID"),
			}, []string{"calendar_id"}),
		},

		// ---- Event tools ----
		{
			Name:        "list_events",
			Description: "List events in a time range.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id":      prop("string", `Calendar ID (default "primary")`),
				"title":            prop("string", "Filter by title"),
				"start":            prop("number", "Start at or after"+epochDesc),
				"end":              prop("number", "Start at or before"+epochDesc),
				"limit":            prop("number", "Max results (default 200, max 200 per call)"),
				"expand_recurring": prop("boolean", "Expand recurring events"),
				"show_cancelled":   prop("boolean", "Include cancelled"),
				"page_token":       prop("string", "Pagination cursor from previous response"),
			}, nil),
		},
		{
			Name:        "get_event",
			Description: "Get an event by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", `Calendar ID (default "primary")`),
				"event_id":    prop("string", "Event ID"),
			}, []string{"event_id"}),
		},
		{
			Name:        "create_event",
			Description: "Create a calendar event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id":      prop("string", `Calendar ID (default "primary")`),
				"title":            prop("string", "Title"),
				"description":      prop("string", "Description"),
				"location":         prop("string", "Location"),
				"start_time":       prop("number", "Start time"+epochDesc),
				"end_time":         prop("number", "End time"+epochDesc),
				"start_date":       prop("string", "All-day start (YYYY-MM-DD)"),
				"end_date":         prop("string", "All-day end (YYYY-MM-DD)"),
				"participants":     participantArraySchema("Participants"),
				"busy":             prop("boolean", "Blocks time as busy"),
				"visibility":       {Type: "string", Desc: "Visibility", Enum: []string{"public", "private"}},
				"conferencing_url": prop("string", "Conferencing URL"),
				"reminders": {
					Type: "array",
					Desc: "Reminders",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"minutes": prop("number", "Minutes before event"),
							"method":  prop("string", "Method (email, popup)"),
						},
					},
				},
			}, []string{"title"}),
		},
		{
			Name:        "update_event",
			Description: "Update a calendar event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id":      prop("string", `Calendar ID (default "primary")`),
				"event_id":         prop("string", "Event ID"),
				"title":            prop("string", "Title"),
				"description":      prop("string", "Description"),
				"location":         prop("string", "Location"),
				"start_time":       prop("number", "Start time"+epochDesc),
				"end_time":         prop("number", "End time"+epochDesc),
				"start_date":       prop("string", "All-day start (YYYY-MM-DD)"),
				"end_date":         prop("string", "All-day end (YYYY-MM-DD)"),
				"participants":     participantArraySchema("Participants"),
				"busy":             prop("boolean", "Blocks time as busy"),
				"visibility":       {Type: "string", Desc: "Visibility", Enum: []string{"public", "private"}},
				"conferencing_url": prop("string", "Conferencing URL"),
				"reminders": {
					Type: "array",
					Desc: "Reminders",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"minutes": prop("number", "Minutes before event"),
							"method":  prop("string", "Method (email, popup)"),
						},
					},
				},
			}, []string{"event_id"}),
		},
		{
			Name:        "delete_event",
			Description: "Delete an event.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", `Calendar ID (default "primary")`),
				"event_id":    prop("string", "Event ID"),
			}, []string{"event_id"}),
		},

		// ---- Availability tools ----
		{
			Name:        "get_free_busy",
			Description: "Check free/busy status for emails in a time range.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"emails":     {Type: "array", Desc: "Emails to check", Items: &JSONSchema{Type: "string"}},
				"start_time": prop("number", "Range start"+epochDesc),
				"end_time":   prop("number", "Range end"+epochDesc),
			}, []string{"emails", "start_time", "end_time"}),
		},
		{
			Name:        "get_availability",
			Description: "Find available meeting slots.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"participants": {
					Type: "array",
					Desc: "Participants with calendars",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"email":        prop("string", "Email"),
							"calendar_ids": {Type: "array", Desc: "Calendar IDs", Items: &JSONSchema{Type: "string"}},
						},
					},
				},
				"start_time":       prop("number", "Window start"+epochDesc),
				"end_time":         prop("number", "Window end"+epochDesc),
				"duration_minutes": prop("number", "Meeting duration (minutes)"),
				"interval_minutes": prop("number", "Slot interval (minutes)"),
				"round_to":         prop("number", "Round to N minutes"),
			}, []string{"participants", "start_time", "end_time", "duration_minutes"}),
		},

		// ---- RSVP tools ----
		{
			Name:        "send_rsvp",
			Description: `RSVP to an event (yes/no/maybe).`,
			InputSchema: objectSchema(map[string]JSONSchema{
				"calendar_id": prop("string", `Calendar ID (default "primary")`),
				"event_id":    prop("string", "Event ID"),
				"status":      {Type: "string", Desc: `"yes", "no", or "maybe"`, Enum: []string{"yes", "no", "maybe"}},
				"comment":     prop("string", "Optional comment"),
			}, []string{"event_id", "status"}),
		},

		// ---- Contact tools ----
		{
			Name:        "list_contacts",
			Description: "List or search contacts.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"email":        prop("string", "Filter by email"),
				"phone_number": prop("string", "Filter by phone"),
				"source":       {Type: "string", Desc: "Source filter", Enum: []string{"address_book", "inbox", "domain"}},
				"limit":        prop("number", "Max results (default 10)"),
				"group":        prop("string", "Group name"),
				"page_token":   prop("string", "Pagination cursor from previous response"),
			}, nil),
		},
		{
			Name:        "get_contact",
			Description: "Get a contact by ID.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"contact_id": prop("string", "Contact ID"),
			}, []string{"contact_id"}),
		},
		{
			Name:        "create_contact",
			Description: "Create a contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"given_name":   prop("string", "First name"),
				"surname":      prop("string", "Last name"),
				"nickname":     prop("string", "Nickname"),
				"company_name": prop("string", "Company"),
				"job_title":    prop("string", "Job title"),
				"emails": {
					Type: "array",
					Desc: "Emails",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"email": prop("string", "Email"),
							"type":  prop("string", "Type (work, home)"),
						},
					},
				},
				"phone_numbers": {
					Type: "array",
					Desc: "Phone numbers",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"number": prop("string", "Number"),
							"type":   prop("string", "Type (work, mobile)"),
						},
					},
				},
				"notes": prop("string", "Notes"),
			}, nil),
		},

		{
			Name:        "update_contact",
			Description: "Update a contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"contact_id":   prop("string", "Contact ID"),
				"given_name":   prop("string", "First name"),
				"surname":      prop("string", "Last name"),
				"nickname":     prop("string", "Nickname"),
				"company_name": prop("string", "Company"),
				"job_title":    prop("string", "Job title"),
				"notes":        prop("string", "Notes"),
				"emails": {
					Type: "array",
					Desc: "Emails",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"email": prop("string", "Email"),
							"type":  prop("string", "Type (work, home)"),
						},
					},
				},
				"phone_numbers": {
					Type: "array",
					Desc: "Phone numbers",
					Items: &JSONSchema{
						Type: "object",
						Properties: map[string]JSONSchema{
							"number": prop("string", "Number"),
							"type":   prop("string", "Type (work, mobile)"),
						},
					},
				},
			}, []string{"contact_id"}),
		},
		{
			Name:        "delete_contact",
			Description: "Delete a contact.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"contact_id": prop("string", "Contact ID"),
			}, []string{"contact_id"}),
		},

		// ---- Utility tools (no API call) ----
		{
			Name:        "current_time",
			Description: "Get current date and time.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"timezone": prop("string", "IANA timezone (default: local)"),
			}, nil),
		},
		{
			Name:        "epoch_to_datetime",
			Description: "Convert unix timestamp to datetime string.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"epoch":    prop("number", "Unix timestamp"),
				"timezone": prop("string", "IANA timezone (default: local)"),
			}, []string{"epoch"}),
		},
		{
			Name:        "datetime_to_epoch",
			Description: "Convert datetime string to unix timestamp.",
			InputSchema: objectSchema(map[string]JSONSchema{
				"datetime": prop("string", `RFC3339 or "2006-01-02 15:04:05" format`),
				"timezone": prop("string", "IANA timezone if no offset in string"),
			}, []string{"datetime"}),
		},
	}

	annotateTools(tools)
	for i := range tools {
		// Output schema left intentionally broad; responses are mirrored in structuredContent.
		tools[i].OutputSchema = &JSONSchema{}
	}
	return tools
}
