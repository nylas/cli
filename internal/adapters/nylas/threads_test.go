//go:build !integration
// +build !integration

package nylas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConvertThread(t *testing.T) {
	now := time.Now().Unix()

	apiThread := threadResponse{
		ID:                    "thread-123",
		GrantID:               "grant-456",
		HasAttachments:        true,
		HasDrafts:             false,
		Starred:               true,
		Unread:                false,
		EarliestMessageDate:   now - 3600,
		LatestMessageRecvDate: now - 1800,
		LatestMessageSentDate: now,
		Participants: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
		MessageIDs: []string{"msg-1", "msg-2", "msg-3"},
		DraftIDs:   []string{},
		FolderIDs:  []string{"folder-1"},
		Snippet:    "This is a preview...",
		Subject:    "Important Discussion",
	}

	thread := convertThread(apiThread)

	assert.Equal(t, "thread-123", thread.ID)
	assert.Equal(t, "grant-456", thread.GrantID)
	assert.True(t, thread.HasAttachments)
	assert.False(t, thread.HasDrafts)
	assert.True(t, thread.Starred)
	assert.False(t, thread.Unread)
	assert.Equal(t, time.Unix(now-3600, 0), thread.EarliestMessageDate)
	assert.Equal(t, time.Unix(now-1800, 0), thread.LatestMessageRecvDate)
	assert.Equal(t, time.Unix(now, 0), thread.LatestMessageSentDate)

	// Test participants conversion using util.Map
	assert.Len(t, thread.Participants, 2)
	assert.Equal(t, "Alice", thread.Participants[0].Name)
	assert.Equal(t, "alice@example.com", thread.Participants[0].Email)
	assert.Equal(t, "Bob", thread.Participants[1].Name)
	assert.Equal(t, "bob@example.com", thread.Participants[1].Email)

	assert.Equal(t, []string{"msg-1", "msg-2", "msg-3"}, thread.MessageIDs)
	assert.Equal(t, []string{}, thread.DraftIDs)
	assert.Equal(t, []string{"folder-1"}, thread.FolderIDs)
	assert.Equal(t, "This is a preview...", thread.Snippet)
	assert.Equal(t, "Important Discussion", thread.Subject)
}

func TestConvertThreads(t *testing.T) {
	now := time.Now().Unix()

	apiThreads := []threadResponse{
		{
			ID:      "thread-1",
			GrantID: "grant-1",
			Subject: "Thread One",
			Participants: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User1", Email: "user1@example.com"},
			},
			EarliestMessageDate:   now,
			LatestMessageRecvDate: now,
			LatestMessageSentDate: now,
		},
		{
			ID:      "thread-2",
			GrantID: "grant-2",
			Subject: "Thread Two",
			Participants: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User2", Email: "user2@example.com"},
			},
			EarliestMessageDate:   now,
			LatestMessageRecvDate: now,
			LatestMessageSentDate: now,
		},
	}

	// Test convertThreads uses util.Map
	threads := convertThreads(apiThreads)

	assert.Len(t, threads, 2)
	assert.Equal(t, "thread-1", threads[0].ID)
	assert.Equal(t, "Thread One", threads[0].Subject)
	assert.Equal(t, "thread-2", threads[1].ID)
	assert.Equal(t, "Thread Two", threads[1].Subject)

	// Verify participants were converted correctly
	assert.Len(t, threads[0].Participants, 1)
	assert.Equal(t, "User1", threads[0].Participants[0].Name)
	assert.Len(t, threads[1].Participants, 1)
	assert.Equal(t, "User2", threads[1].Participants[0].Name)
}

func TestConvertThreads_Empty(t *testing.T) {
	// Test with empty slice
	threads := convertThreads([]threadResponse{})
	assert.NotNil(t, threads)
	assert.Len(t, threads, 0)
}

func TestConvertThread_EmptyParticipants(t *testing.T) {
	now := time.Now().Unix()

	apiThread := threadResponse{
		ID:      "thread-empty",
		GrantID: "grant-123",
		Subject: "No Participants",
		Participants: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{},
		EarliestMessageDate:   now,
		LatestMessageRecvDate: now,
		LatestMessageSentDate: now,
	}

	thread := convertThread(apiThread)

	assert.Equal(t, "thread-empty", thread.ID)
	assert.Equal(t, "No Participants", thread.Subject)
	assert.NotNil(t, thread.Participants)
	assert.Len(t, thread.Participants, 0)
}
