package air

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/httputil"
)

// States to filter out from notetaker list
var excludedNotetakerStates = map[string]bool{
	"failed_entry": true,
}

// NotetakerSource represents a source for fetching notetaker emails
type NotetakerSource struct {
	From       string `json:"from"`
	Subject    string `json:"subject"`
	LinkDomain string `json:"linkDomain"`
}

// NotetakerResponse represents a notetaker for the UI
type NotetakerResponse struct {
	ID            string `json:"id"`
	State         string `json:"state"`
	MeetingLink   string `json:"meetingLink"`
	MeetingTitle  string `json:"meetingTitle"`
	JoinTime      string `json:"joinTime,omitempty"`
	Provider      string `json:"provider,omitempty"`
	HasRecording  bool   `json:"hasRecording"`
	HasTranscript bool   `json:"hasTranscript"`
	CreatedAt     string `json:"createdAt,omitempty"`
	IsExternal    bool   `json:"isExternal,omitempty"`
	ExternalURL   string `json:"externalUrl,omitempty"`
	Attendees     string `json:"attendees,omitempty"`
	Summary       string `json:"summary,omitempty"`
}

// CreateNotetakerRequest for creating a notetaker
type CreateNotetakerRequest struct {
	MeetingLink string `json:"meetingLink"`
	JoinTime    int64  `json:"joinTime,omitempty"`
	BotName     string `json:"botName,omitempty"`
}

// MediaResponse for notetaker media
type MediaResponse struct {
	RecordingURL   string `json:"recordingUrl,omitempty"`
	TranscriptURL  string `json:"transcriptUrl,omitempty"`
	RecordingSize  int64  `json:"recordingSize,omitempty"`
	TranscriptSize int64  `json:"transcriptSize,omitempty"`
	ExpiresAt      int64  `json:"expiresAt,omitempty"`
}

// handleNotetakersRoute dispatches notetaker requests by method
func (s *Server) handleNotetakersRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListNotetakers(w, r)
	case http.MethodPost:
		s.handleCreateNotetaker(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNotetakerByID dispatches requests for individual notetakers
func (s *Server) handleNotetakerByID(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetNotetaker(w, r)
	case http.MethodDelete:
		s.handleDeleteNotetaker(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListNotetakers returns all notetakers from the Nylas API
func (s *Server) handleListNotetakers(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	// Parse sources from query params (JSON array)
	sourcesJSON := r.URL.Query().Get("sources")
	var sources []NotetakerSource
	if sourcesJSON != "" {
		if err := json.Unmarshal([]byte(sourcesJSON), &sources); err != nil {
			// Fall back to empty sources if parsing fails
			sources = []NotetakerSource{}
		}
	} else {
		// Empty sources if none provided - user must configure in Settings
		sources = []NotetakerSource{}
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	response := make([]*NotetakerResponse, 0)

	// Fetch notetakers from Nylas API
	notetakers, err := s.nylasClient.ListNotetakers(ctx, grantID, nil)
	if err == nil {
		// Convert to UI response format, filtering out excluded states
		for _, nt := range notetakers {
			if !excludedNotetakerStates[nt.State] {
				response = append(response, domainToNotetakerResponse(&nt))
			}
		}
	}

	// Fetch external links from each configured source
	for _, source := range sources {
		if source.From != "" && source.LinkDomain != "" {
			externalLinks := s.fetchNotetakerSummaryEmails(ctx, grantID, source)
			response = append(response, externalLinks...)
		}
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// domainToNotetakerResponse converts a domain.Notetaker to NotetakerResponse
func domainToNotetakerResponse(nt *domain.Notetaker) *NotetakerResponse {
	resp := &NotetakerResponse{
		ID:            nt.ID,
		State:         nt.State,
		MeetingLink:   nt.MeetingLink,
		MeetingTitle:  nt.MeetingTitle,
		HasRecording:  nt.MediaData != nil && nt.MediaData.Recording != nil,
		HasTranscript: nt.MediaData != nil && nt.MediaData.Transcript != nil,
	}

	if !nt.JoinTime.IsZero() {
		resp.JoinTime = nt.JoinTime.Format(time.RFC3339)
	}
	if !nt.CreatedAt.IsZero() {
		resp.CreatedAt = nt.CreatedAt.Format(time.RFC3339)
	}

	// Get provider from meeting info or detect from link
	if nt.MeetingInfo != nil && nt.MeetingInfo.Provider != "" {
		resp.Provider = nt.MeetingInfo.Provider
	} else {
		resp.Provider = detectMeetingProvider(nt.MeetingLink)
	}

	// Set default title if empty
	if resp.MeetingTitle == "" {
		resp.MeetingTitle = "Meeting Recording"
	}

	return resp
}

// handleCreateNotetaker creates a new notetaker via the Nylas API
func (s *Server) handleCreateNotetaker(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	var req CreateNotetakerRequest
	if !parseJSONBody(w, r, &req) {
		return
	}

	if req.MeetingLink == "" {
		writeError(w, http.StatusBadRequest, "meetingLink is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	// Build API request
	apiReq := &domain.CreateNotetakerRequest{
		MeetingLink: req.MeetingLink,
		JoinTime:    req.JoinTime,
	}

	// Add bot config if name provided
	if req.BotName != "" {
		apiReq.BotConfig = &domain.BotConfig{
			Name: req.BotName,
		}
	}

	// Create notetaker via Nylas API
	nt, err := s.nylasClient.CreateNotetaker(ctx, grantID, apiReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to create notetaker: " + err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, domainToNotetakerResponse(nt))
}

// handleGetNotetaker returns a single notetaker from the Nylas API
func (s *Server) handleGetNotetaker(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	nt, err := s.nylasClient.GetNotetaker(ctx, grantID, id)
	if err != nil {
		http.Error(w, "Notetaker not found", http.StatusNotFound)
		return
	}

	httputil.WriteJSON(w, http.StatusOK, domainToNotetakerResponse(nt))
}

// handleGetNotetakerMedia returns media for a notetaker from the Nylas API
func (s *Server) handleGetNotetakerMedia(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	// Get media from Nylas API
	mediaData, err := s.nylasClient.GetNotetakerMedia(ctx, grantID, id)
	if err != nil {
		http.Error(w, "Recording not yet available", http.StatusNotFound)
		return
	}

	media := MediaResponse{}
	if mediaData.Recording != nil {
		media.RecordingURL = mediaData.Recording.URL
		media.RecordingSize = mediaData.Recording.Size
		media.ExpiresAt = mediaData.Recording.ExpiresAt
	}
	if mediaData.Transcript != nil {
		media.TranscriptURL = mediaData.Transcript.URL
		media.TranscriptSize = mediaData.Transcript.Size
	}

	httputil.WriteJSON(w, http.StatusOK, media)
}

// handleDeleteNotetaker cancels a notetaker via the Nylas API
func (s *Server) handleDeleteNotetaker(w http.ResponseWriter, r *http.Request) {
	grantID := s.withAuthGrant(w, nil)
	if grantID == "" {
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	ctx, cancel := s.withTimeout(r)
	defer cancel()

	err := s.nylasClient.DeleteNotetaker(ctx, grantID, id)
	if err != nil {
		http.Error(w, "Failed to cancel notetaker: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// detectMeetingProvider detects the meeting provider from URL
func detectMeetingProvider(link string) string {
	link = strings.ToLower(link)
	switch {
	case strings.Contains(link, "zoom.us"):
		return "zoom"
	case strings.Contains(link, "meet.google.com"):
		return "google_meet"
	case strings.Contains(link, "teams.microsoft.com"):
		return "teams"
	default:
		return "unknown"
	}
}

// fetchNotetakerSummaryEmails fetches emails containing recording links from a source
func (s *Server) fetchNotetakerSummaryEmails(ctx context.Context, grantID string, source NotetakerSource) []*NotetakerResponse {
	result := make([]*NotetakerResponse, 0)

	// Search for emails from the configured sender
	query := &domain.MessageQueryParams{
		From:  source.From,
		Limit: 20,
	}

	// Add subject filter if specified
	if source.Subject != "" {
		query.Subject = source.Subject
	}

	messages, err := s.nylasClient.GetMessagesWithParams(ctx, grantID, query)
	if err != nil {
		return result
	}

	// Build regex to extract URLs with recording IDs from email body
	// Escape dots in the domain for regex
	escapedDomain := strings.ReplaceAll(source.LinkDomain, ".", `\.`)
	urlPattern := `https://` + escapedDomain + `/app/library/[a-zA-Z0-9]+`
	linkRegex := regexp.MustCompile(urlPattern)

	for _, msg := range messages {
		// Only include emails that have a matching link in the body
		urls := linkRegex.FindAllString(msg.Body, 1)
		if len(urls) == 0 {
			continue // Skip emails without a valid recording link
		}

		externalURL := urls[0]

		// Use subject as meeting title directly
		title := msg.Subject
		if title == "" {
			title = "Meeting Recording"
		}

		resp := &NotetakerResponse{
			ID:            "email-" + msg.ID,
			State:         "completed",
			MeetingTitle:  title,
			Provider:      "nylas_notebook",
			HasRecording:  true,
			HasTranscript: true,
			IsExternal:    true,
			ExternalURL:   externalURL,
			Summary:       msg.Body,
		}

		// Parse date from message
		if !msg.Date.IsZero() {
			resp.CreatedAt = msg.Date.Format(time.RFC3339)
		}

		result = append(result, resp)
	}

	return result
}
