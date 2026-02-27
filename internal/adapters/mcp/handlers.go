package mcp

import (
	"context"
	"fmt"
	"time"
)

const toolCallTimeout = 90 * time.Second

// Supported protocol versions (newest first).
var supportedVersions = []string{
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

// latestVersion is the most recent protocol version this server supports.
const latestVersion = "2025-06-18"

// Server info constants.
const (
	serverName    = "nylas-mcp"
	serverVersion = "1.0.0"
)

// negotiateVersion returns the newest version both client and server support.
// If the client's requested version is in our supported list, return it.
// Otherwise, return the latest version we support.
func negotiateVersion(requested string) string {
	for _, v := range supportedVersions {
		if v == requested {
			return v
		}
	}
	return latestVersion
}

// handleInitialize responds to the MCP initialize request with server capabilities.
func (s *Server) handleInitialize(req *Request) []byte {
	params, err := parseInitializeParams(req.Params)
	if err != nil {
		return errorResponse(req.ID, codeInvalidParams, err.Error())
	}
	negotiated := negotiateVersion(params.ProtocolVersion)

	// Detect user's timezone for guidance
	localZone, _ := time.Now().Zone()
	tzName := time.Local.String()
	if tzName == "Local" {
		tzName = localZone
	}

	instructions := fmt.Sprintf(`Nylas MCP Server — native access to email, calendar, and contacts via the Nylas API.

IMPORTANT - Timezone Consistency:
The user's local timezone is: %s (%s)
When displaying ANY timestamps to users (from emails, events, availability, etc.):
1. Always use epoch_to_datetime tool with timezone "%s" to convert Unix timestamps
2. Display ALL times in %s, never in UTC
3. Format times clearly (e.g., "2:00 PM %s")

grant_id: All tools accept an optional grant_id parameter. Omit it to use the default grant.`, tzName, localZone, tzName, localZone, localZone)

	result := map[string]any{
		"protocolVersion": negotiated,
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
		"serverInfo": map[string]any{
			"name":    serverName,
			"version": serverVersion,
		},
		"instructions": instructions,
	}

	return successResponse(req.ID, result)
}

// handleToolsList returns all registered tools.
func (s *Server) handleToolsList(req *Request) []byte {
	tools := registeredTools()
	result := map[string]any{
		"tools": tools,
	}
	return successResponse(req.ID, result)
}

// handleToolCall dispatches a tool call to the appropriate executor.
func (s *Server) handleToolCall(ctx context.Context, req *Request) []byte {
	params, err := parseToolCallParams(req.Params)
	if err != nil {
		return errorResponse(req.ID, codeInvalidParams, err.Error())
	}
	name := params.Name
	if name == "" {
		return errorResponse(req.ID, codeInvalidParams, "tool name is required")
	}
	args := params.Arguments
	if args == nil {
		args = make(map[string]any)
	}

	// Apply per-call timeout to prevent hanging on slow API responses.
	ctx, cancel := context.WithTimeout(ctx, toolCallTimeout)
	defer cancel()

	var result *ToolResponse

	switch name {
	// Email tools
	case "list_messages":
		result = s.executeListMessages(ctx, args)
	case "get_message":
		result = s.executeGetMessage(ctx, args)
	case "send_message":
		result = s.executeSendMessage(ctx, args)
	case "update_message":
		result = s.executeUpdateMessage(ctx, args)
	case "delete_message":
		result = s.executeDeleteMessage(ctx, args)
	case "smart_compose":
		result = s.executeSmartCompose(ctx, args)
	case "smart_compose_reply":
		result = s.executeSmartComposeReply(ctx, args)

	// Draft tools
	case "list_drafts":
		result = s.executeListDrafts(ctx, args)
	case "get_draft":
		result = s.executeGetDraft(ctx, args)
	case "create_draft":
		result = s.executeCreateDraft(ctx, args)
	case "update_draft":
		result = s.executeUpdateDraft(ctx, args)
	case "send_draft":
		result = s.executeSendDraft(ctx, args)
	case "delete_draft":
		result = s.executeDeleteDraft(ctx, args)

	// Thread tools
	case "list_threads":
		result = s.executeListThreads(ctx, args)
	case "get_thread":
		result = s.executeGetThread(ctx, args)
	case "update_thread":
		result = s.executeUpdateThread(ctx, args)
	case "delete_thread":
		result = s.executeDeleteThread(ctx, args)

	// Folder tools
	case "list_folders":
		result = s.executeListFolders(ctx, args)
	case "get_folder":
		result = s.executeGetFolder(ctx, args)
	case "create_folder":
		result = s.executeCreateFolder(ctx, args)
	case "update_folder":
		result = s.executeUpdateFolder(ctx, args)
	case "delete_folder":
		result = s.executeDeleteFolder(ctx, args)

	// Attachment tools
	case "list_attachments":
		result = s.executeListAttachments(ctx, args)
	case "get_attachment":
		result = s.executeGetAttachment(ctx, args)

	// Scheduled message tools
	case "list_scheduled_messages":
		result = s.executeListScheduledMessages(ctx, args)
	case "cancel_scheduled_message":
		result = s.executeCancelScheduledMessage(ctx, args)

	// Calendar tools
	case "list_calendars":
		result = s.executeListCalendars(ctx, args)
	case "get_calendar":
		result = s.executeGetCalendar(ctx, args)
	case "create_calendar":
		result = s.executeCreateCalendar(ctx, args)
	case "update_calendar":
		result = s.executeUpdateCalendar(ctx, args)
	case "delete_calendar":
		result = s.executeDeleteCalendar(ctx, args)

	// Event tools
	case "list_events":
		result = s.executeListEvents(ctx, args)
	case "get_event":
		result = s.executeGetEvent(ctx, args)
	case "create_event":
		result = s.executeCreateEvent(ctx, args)
	case "update_event":
		result = s.executeUpdateEvent(ctx, args)
	case "delete_event":
		result = s.executeDeleteEvent(ctx, args)
	case "send_rsvp":
		result = s.executeSendRSVP(ctx, args)

	// Availability tools
	case "get_free_busy":
		result = s.executeGetFreeBusy(ctx, args)
	case "get_availability":
		result = s.executeGetAvailability(ctx, args)

	// Contact tools
	case "list_contacts":
		result = s.executeListContacts(ctx, args)
	case "get_contact":
		result = s.executeGetContact(ctx, args)
	case "create_contact":
		result = s.executeCreateContact(ctx, args)
	case "update_contact":
		result = s.executeUpdateContact(ctx, args)
	case "delete_contact":
		result = s.executeDeleteContact(ctx, args)

	// Utility tools (no API call)
	case "current_time":
		result = s.executeCurrentTime(args)
	case "epoch_to_datetime":
		result = s.executeEpochToDatetime(args)
	case "datetime_to_epoch":
		result = s.executeDatetimeToEpoch(args)

	default:
		return errorResponse(req.ID, codeInvalidParams, fmt.Sprintf("unknown tool: %s", name))
	}

	return successResponse(req.ID, result)
}
