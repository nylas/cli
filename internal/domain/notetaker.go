package domain

import "time"

// Notetaker represents a Nylas Notetaker bot instance.
type Notetaker struct {
	ID           string       `json:"id"`
	State        string       `json:"state"` // scheduled, connecting, waiting_for_entry, attending, media_processing, complete, cancelled, failed
	MeetingLink  string       `json:"meeting_link,omitempty"`
	JoinTime     time.Time    `json:"join_time,omitempty"`
	MeetingTitle string       `json:"meeting_title,omitempty"`
	MediaData    *MediaData   `json:"media_data,omitempty"`
	BotConfig    *BotConfig   `json:"bot_config,omitempty"`
	MeetingInfo  *MeetingInfo `json:"meeting_info,omitempty"`
	CreatedAt    time.Time    `json:"created_at,omitempty"`
	UpdatedAt    time.Time    `json:"updated_at,omitempty"`
	Object       string       `json:"object,omitempty"`
}

// NotetakerState constants for notetaker states.
const (
	NotetakerStateScheduled       = "scheduled"
	NotetakerStateConnecting      = "connecting"
	NotetakerStateWaitingForEntry = "waiting_for_entry"
	NotetakerStateAttending       = "attending"
	NotetakerStateMediaProcessing = "media_processing"
	NotetakerStateComplete        = "complete"
	NotetakerStateCancelled       = "cancelled"
	NotetakerStateFailed          = "failed"
)

// MediaData represents the media output from a notetaker session.
type MediaData struct {
	Recording  *MediaFile `json:"recording,omitempty"`
	Transcript *MediaFile `json:"transcript,omitempty"`
}

// MediaFile represents a media file from a notetaker session.
type MediaFile struct {
	URL         string `json:"url,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

// BotConfig represents the configuration for a notetaker bot.
type BotConfig struct {
	Name      string `json:"name,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// MeetingInfo represents information about the meeting.
type MeetingInfo struct {
	Provider    string `json:"provider,omitempty"` // zoom, google_meet, teams
	MeetingCode string `json:"meeting_code,omitempty"`
}

// CreateNotetakerRequest for creating a new notetaker.
type CreateNotetakerRequest struct {
	MeetingLink string     `json:"meeting_link"`
	JoinTime    int64      `json:"join_time,omitempty"` // Unix timestamp for when to join
	BotConfig   *BotConfig `json:"bot_config,omitempty"`
}

// NotetakerListResponse represents a list of notetakers.
type NotetakerListResponse struct {
	Data       []Notetaker `json:"data"`
	Pagination Pagination  `json:"pagination,omitempty"`
}

// NotetakerQueryParams for filtering notetakers.
type NotetakerQueryParams struct {
	Limit     int    `json:"limit,omitempty"`
	PageToken string `json:"page_token,omitempty"`
	State     string `json:"state,omitempty"` // Filter by state
}
