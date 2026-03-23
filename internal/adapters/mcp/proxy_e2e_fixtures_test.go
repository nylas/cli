package mcp

// mockToolsListResponse is a realistic upstream tools/list response with a mix of
// grant-requiring tools, utility tools, and edge cases.
const mockToolsListResponse = `{
	"jsonrpc": "2.0",
	"id": 1,
	"result": {
		"tools": [
			{
				"name": "list_messages",
				"description": "List messages for a grant",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"get_all_query_parameters": {"type": "object"}
					}
				}
			},
			{
				"name": "list_calendars",
				"description": "List calendars for a grant",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"}
					}
				}
			},
			{
				"name": "list_contacts",
				"description": "List contacts for a grant",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"query_parameters": {"type": "object"}
					}
				}
			},
			{
				"name": "get_contact",
				"description": "Get a contact by ID",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"contact_id": {"type": "string"}
					},
					"required": ["contact_id"]
				}
			},
			{
				"name": "create_event",
				"description": "Create a calendar event",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"calendar_id": {"type": "string"},
						"event_request": {"type": "object"}
					},
					"required": ["calendar_id", "event_request"]
				}
			},
			{
				"name": "delete_event",
				"description": "Delete a calendar event",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"calendar_id": {"type": "string"},
						"event_id": {"type": "string"}
					},
					"required": ["calendar_id", "event_id"]
				}
			},
			{
				"name": "confirm_send_draft",
				"description": "Confirm and send a draft",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"draft_id": {"type": "string"}
					},
					"required": ["draft_id"]
				}
			},
			{
				"name": "send_message",
				"description": "Send a message",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"message_request": {"type": "object"},
						"confirmation_hash": {"type": "string"}
					},
					"required": ["message_request", "confirmation_hash"]
				}
			},
			{
				"name": "get_grant",
				"description": "Look up grant by email.",
				"inputSchema": {
					"type": "object",
					"properties": {
						"email": {"type": "string"}
					},
					"required": ["email"]
				}
			},
			{
				"name": "current_time",
				"description": "Get current time in a timezone",
				"inputSchema": {
					"type": "object",
					"properties": {
						"timezone": {"type": "string"}
					},
					"required": ["timezone"]
				}
			},
			{
				"name": "epoch_to_datetime",
				"description": "Convert epoch timestamps to human-readable dates",
				"inputSchema": {
					"type": "object",
					"properties": {
						"batch": {"type": "array"}
					},
					"required": ["batch"]
				}
			},
			{
				"name": "availability",
				"description": "Check availability for participants",
				"inputSchema": {
					"type": "object",
					"properties": {
						"availability_request": {"type": "object"}
					},
					"required": ["availability_request"]
				}
			},
			{
				"name": "confirm_send_message",
				"description": "Confirm message content before sending",
				"inputSchema": {
					"type": "object",
					"properties": {
						"message_request": {"type": "object"}
					},
					"required": ["message_request"]
				}
			},
			{
				"name": "brand_new_tool",
				"description": "A hypothetical new tool added upstream",
				"inputSchema": {
					"type": "object",
					"properties": {
						"grant_id": {"type": "string"},
						"payload": {"type": "object"}
					},
					"required": ["payload"]
				}
			}
		]
	}
}`

const mockInitializeResponse = `{
	"jsonrpc": "2.0",
	"id": 1,
	"result": {
		"protocolVersion": "2024-11-05",
		"capabilities": {},
		"serverInfo": {"name": "nylas-mcp", "version": "1.0.0"},
		"instructions": "Nylas MCP server for email and calendar."
	}
}`
