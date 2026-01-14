//go:build !integration
// +build !integration

package nylas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConvertNotetaker(t *testing.T) {
	now := time.Now().Unix()

	apiNotetaker := notetakerResponse{
		ID:           "notetaker-123",
		State:        "recording",
		MeetingLink:  "https://zoom.us/j/123456",
		JoinTime:     now - 600,
		MeetingTitle: "Team Standup",
		MediaData: &struct {
			Recording *struct {
				URL         string `json:"url"`
				ContentType string `json:"content_type"`
				Size        int64  `json:"size"`
				ExpiresAt   int64  `json:"expires_at"`
			} `json:"recording"`
			Transcript *struct {
				URL         string `json:"url"`
				ContentType string `json:"content_type"`
				Size        int64  `json:"size"`
				ExpiresAt   int64  `json:"expires_at"`
			} `json:"transcript"`
		}{
			Recording: &struct {
				URL         string `json:"url"`
				ContentType string `json:"content_type"`
				Size        int64  `json:"size"`
				ExpiresAt   int64  `json:"expires_at"`
			}{
				URL:         "https://example.com/recording.mp4",
				ContentType: "video/mp4",
				Size:        1024000,
				ExpiresAt:   now + 86400,
			},
			Transcript: &struct {
				URL         string `json:"url"`
				ContentType string `json:"content_type"`
				Size        int64  `json:"size"`
				ExpiresAt   int64  `json:"expires_at"`
			}{
				URL:         "https://example.com/transcript.txt",
				ContentType: "text/plain",
				Size:        5000,
				ExpiresAt:   now + 86400,
			},
		},
		BotConfig: &struct {
			Name      string `json:"name"`
			AvatarURL string `json:"avatar_url"`
		}{
			Name:      "Meeting Bot",
			AvatarURL: "https://example.com/avatar.png",
		},
		MeetingInfo: &struct {
			Provider    string `json:"provider"`
			MeetingCode string `json:"meeting_code"`
		}{
			Provider:    "zoom",
			MeetingCode: "123456",
		},
		CreatedAt: now - 3600,
		UpdatedAt: now,
		Object:    "notetaker",
	}

	notetaker := convertNotetaker(apiNotetaker)

	assert.Equal(t, "notetaker-123", notetaker.ID)
	assert.Equal(t, "recording", notetaker.State)
	assert.Equal(t, "https://zoom.us/j/123456", notetaker.MeetingLink)
	assert.Equal(t, "Team Standup", notetaker.MeetingTitle)
	assert.Equal(t, time.Unix(now-600, 0), notetaker.JoinTime)
	assert.Equal(t, time.Unix(now-3600, 0), notetaker.CreatedAt)
	assert.Equal(t, time.Unix(now, 0), notetaker.UpdatedAt)
	assert.Equal(t, "notetaker", notetaker.Object)

	// Test BotConfig
	assert.NotNil(t, notetaker.BotConfig)
	assert.Equal(t, "Meeting Bot", notetaker.BotConfig.Name)
	assert.Equal(t, "https://example.com/avatar.png", notetaker.BotConfig.AvatarURL)

	// Test MeetingInfo
	assert.NotNil(t, notetaker.MeetingInfo)
	assert.Equal(t, "zoom", notetaker.MeetingInfo.Provider)
	assert.Equal(t, "123456", notetaker.MeetingInfo.MeetingCode)

	// Test MediaData
	assert.NotNil(t, notetaker.MediaData)
	assert.NotNil(t, notetaker.MediaData.Recording)
	assert.Equal(t, "https://example.com/recording.mp4", notetaker.MediaData.Recording.URL)
	assert.Equal(t, "video/mp4", notetaker.MediaData.Recording.ContentType)
	assert.Equal(t, int64(1024000), notetaker.MediaData.Recording.Size)
	assert.Equal(t, int64(now+86400), notetaker.MediaData.Recording.ExpiresAt)

	assert.NotNil(t, notetaker.MediaData.Transcript)
	assert.Equal(t, "https://example.com/transcript.txt", notetaker.MediaData.Transcript.URL)
	assert.Equal(t, "text/plain", notetaker.MediaData.Transcript.ContentType)
	assert.Equal(t, int64(5000), notetaker.MediaData.Transcript.Size)
	assert.Equal(t, int64(now+86400), notetaker.MediaData.Transcript.ExpiresAt)
}

func TestConvertNotetakers(t *testing.T) {
	now := time.Now().Unix()

	apiNotetakers := []notetakerResponse{
		{
			ID:           "notetaker-1",
			State:        "recording",
			MeetingLink:  "https://zoom.us/j/111",
			MeetingTitle: "Meeting 1",
			JoinTime:     now,
			CreatedAt:    now,
			UpdatedAt:    now,
			Object:       "notetaker",
		},
		{
			ID:           "notetaker-2",
			State:        "completed",
			MeetingLink:  "https://meet.google.com/abc-def",
			MeetingTitle: "Meeting 2",
			JoinTime:     now - 1800,
			CreatedAt:    now - 3600,
			UpdatedAt:    now,
			Object:       "notetaker",
		},
	}

	// Test convertNotetakers uses util.Map
	notetakers := convertNotetakers(apiNotetakers)

	assert.Len(t, notetakers, 2)
	assert.Equal(t, "notetaker-1", notetakers[0].ID)
	assert.Equal(t, "recording", notetakers[0].State)
	assert.Equal(t, "Meeting 1", notetakers[0].MeetingTitle)

	assert.Equal(t, "notetaker-2", notetakers[1].ID)
	assert.Equal(t, "completed", notetakers[1].State)
	assert.Equal(t, "Meeting 2", notetakers[1].MeetingTitle)
}

func TestConvertNotetakers_Empty(t *testing.T) {
	// Test with empty slice
	notetakers := convertNotetakers([]notetakerResponse{})
	assert.NotNil(t, notetakers)
	assert.Len(t, notetakers, 0)
}

func TestConvertNotetaker_MinimalData(t *testing.T) {
	now := time.Now().Unix()

	// Notetaker with minimal required fields
	apiNotetaker := notetakerResponse{
		ID:          "notetaker-min",
		State:       "pending",
		MeetingLink: "https://zoom.us/j/minimal",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	notetaker := convertNotetaker(apiNotetaker)

	assert.Equal(t, "notetaker-min", notetaker.ID)
	assert.Equal(t, "pending", notetaker.State)
	assert.Equal(t, "https://zoom.us/j/minimal", notetaker.MeetingLink)
	assert.Equal(t, "", notetaker.MeetingTitle)
	assert.Nil(t, notetaker.BotConfig)
	assert.Nil(t, notetaker.MeetingInfo)
	assert.Nil(t, notetaker.MediaData)
}

func TestConvertNotetaker_ZeroJoinTime(t *testing.T) {
	now := time.Now().Unix()

	apiNotetaker := notetakerResponse{
		ID:          "notetaker-nojoin",
		State:       "pending",
		MeetingLink: "https://zoom.us/j/test",
		JoinTime:    0, // Not yet joined
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	notetaker := convertNotetaker(apiNotetaker)

	assert.Equal(t, "notetaker-nojoin", notetaker.ID)
	// When JoinTime is 0, it should be the zero time
	assert.True(t, notetaker.JoinTime.IsZero())
}
