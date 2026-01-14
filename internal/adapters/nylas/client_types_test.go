package nylas_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test mock client implements interface

func TestDomainTypes(t *testing.T) {
	t.Run("Contact String method", func(t *testing.T) {
		tests := []struct {
			contact domain.EmailParticipant
			want    string
		}{
			{domain.EmailParticipant{Name: "John Doe", Email: "john@example.com"}, "John Doe <john@example.com>"},
			{domain.EmailParticipant{Name: "", Email: "john@example.com"}, "john@example.com"},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.want, tt.contact.String())
		}
	})

	t.Run("Provider validation", func(t *testing.T) {
		assert.True(t, domain.ProviderGoogle.IsValid())
		assert.True(t, domain.ProviderMicrosoft.IsValid())
		assert.True(t, domain.ProviderIMAP.IsValid())
		assert.False(t, domain.Provider("invalid").IsValid())
	})

	t.Run("Provider display name", func(t *testing.T) {
		assert.Equal(t, "Google", domain.ProviderGoogle.DisplayName())
		assert.Equal(t, "Microsoft", domain.ProviderMicrosoft.DisplayName())
	})

	t.Run("ParseProvider", func(t *testing.T) {
		provider, err := domain.ParseProvider("google")
		require.NoError(t, err)
		assert.Equal(t, domain.ProviderGoogle, provider)

		_, err = domain.ParseProvider("invalid")
		assert.Error(t, err)
	})

	t.Run("Grant IsValid", func(t *testing.T) {
		validGrant := domain.Grant{GrantStatus: "valid"}
		invalidGrant := domain.Grant{GrantStatus: "invalid"}

		assert.True(t, validGrant.IsValid())
		assert.False(t, invalidGrant.IsValid())
	})
}

func TestMessageQueryParams(t *testing.T) {
	t.Run("creates params with defaults", func(t *testing.T) {
		params := &domain.MessageQueryParams{}
		assert.Equal(t, 0, params.Limit)
		assert.Nil(t, params.Unread)
	})

	t.Run("creates params with values", func(t *testing.T) {
		unread := true
		starred := false
		params := &domain.MessageQueryParams{
			Limit:         20,
			Unread:        &unread,
			Starred:       &starred,
			From:          "sender@example.com",
			SearchQuery:   "important",
			ReceivedAfter: time.Now().Unix(),
		}

		assert.Equal(t, 20, params.Limit)
		assert.True(t, *params.Unread)
		assert.False(t, *params.Starred)
		assert.Equal(t, "sender@example.com", params.From)
	})
}

func TestThreadQueryParams(t *testing.T) {
	t.Run("creates params with defaults", func(t *testing.T) {
		params := &domain.ThreadQueryParams{}
		assert.Equal(t, 0, params.Limit)
	})

	t.Run("creates params with values", func(t *testing.T) {
		unread := true
		params := &domain.ThreadQueryParams{
			Limit:       50,
			Unread:      &unread,
			Subject:     "test subject",
			SearchQuery: "keyword",
		}

		assert.Equal(t, 50, params.Limit)
		assert.True(t, *params.Unread)
	})
}

func TestSendMessageRequest(t *testing.T) {
	req := &domain.SendMessageRequest{
		Subject: "Test Email",
		Body:    "<html><body>Hello World</body></html>",
		To:      []domain.EmailParticipant{{Name: "Recipient", Email: "to@example.com"}},
		Cc:      []domain.EmailParticipant{{Email: "cc@example.com"}},
		Bcc:     []domain.EmailParticipant{{Email: "bcc@example.com"}},
		TrackingOpts: &domain.TrackingOptions{
			Opens: true,
			Links: true,
		},
	}

	assert.Equal(t, "Test Email", req.Subject)
	assert.Len(t, req.To, 1)
	assert.Len(t, req.Cc, 1)
	assert.Len(t, req.Bcc, 1)
	assert.True(t, req.TrackingOpts.Opens)
}

func TestCreateDraftRequest(t *testing.T) {
	req := &domain.CreateDraftRequest{
		Subject:      "Draft Subject",
		Body:         "Draft body content",
		To:           []domain.EmailParticipant{{Email: "to@example.com"}},
		ReplyToMsgID: "original-msg-id",
	}

	assert.Equal(t, "Draft Subject", req.Subject)
	assert.Equal(t, "original-msg-id", req.ReplyToMsgID)
}

func TestCreateFolderRequest(t *testing.T) {
	req := &domain.CreateFolderRequest{
		Name:            "My Folder",
		ParentID:        "parent-folder-id",
		BackgroundColor: "#FF0000",
		TextColor:       "#FFFFFF",
	}

	assert.Equal(t, "My Folder", req.Name)
	assert.Equal(t, "parent-folder-id", req.ParentID)
}

func TestUpdateMessageRequest(t *testing.T) {
	unread := false
	starred := true
	req := &domain.UpdateMessageRequest{
		Unread:  &unread,
		Starred: &starred,
		Folders: []string{"folder-1", "folder-2"},
	}

	assert.False(t, *req.Unread)
	assert.True(t, *req.Starred)
	assert.Len(t, req.Folders, 2)
}

func TestFolderSystemConstants(t *testing.T) {
	assert.Equal(t, "inbox", domain.FolderInbox)
	assert.Equal(t, "sent", domain.FolderSent)
	assert.Equal(t, "drafts", domain.FolderDrafts)
	assert.Equal(t, "trash", domain.FolderTrash)
	assert.Equal(t, "spam", domain.FolderSpam)
	assert.Equal(t, "archive", domain.FolderArchive)
	assert.Equal(t, "all", domain.FolderAll)
}

func TestUnixTimeUnmarshal(t *testing.T) {
	t.Run("unmarshals unix timestamp", func(t *testing.T) {
		jsonData := `{"created_at": 1703001600}`
		var result struct {
			CreatedAt domain.UnixTime `json:"created_at"`
		}
		err := json.Unmarshal([]byte(jsonData), &result)
		require.NoError(t, err)
		assert.Equal(t, int64(1703001600), result.CreatedAt.Unix())
	})

	t.Run("unmarshals RFC3339 string", func(t *testing.T) {
		jsonData := `{"created_at": "2023-12-19T12:00:00Z"}`
		var result struct {
			CreatedAt domain.UnixTime `json:"created_at"`
		}
		err := json.Unmarshal([]byte(jsonData), &result)
		require.NoError(t, err)
		assert.Equal(t, 2023, result.CreatedAt.Year())
	})
}

func TestAttachmentModel(t *testing.T) {
	attachment := domain.Attachment{
		ID:          "attach-123",
		GrantID:     "grant-456",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        1024000,
		ContentID:   "cid-789",
		IsInline:    false,
	}

	assert.Equal(t, "attach-123", attachment.ID)
	assert.Equal(t, "document.pdf", attachment.Filename)
	assert.Equal(t, int64(1024000), attachment.Size)
	assert.False(t, attachment.IsInline)
}

func TestThreadModel(t *testing.T) {
	now := time.Now()
	thread := domain.Thread{
		ID:                    "thread-123",
		GrantID:               "grant-456",
		Subject:               "Test Thread Subject",
		Snippet:               "This is a preview...",
		HasAttachments:        true,
		HasDrafts:             false,
		Starred:               true,
		Unread:                true,
		EarliestMessageDate:   now.Add(-24 * time.Hour),
		LatestMessageRecvDate: now,
		Participants: []domain.EmailParticipant{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
		MessageIDs: []string{"msg-1", "msg-2"},
		FolderIDs:  []string{"inbox"},
	}

	assert.Equal(t, "thread-123", thread.ID)
	assert.Equal(t, "Test Thread Subject", thread.Subject)
	assert.True(t, thread.HasAttachments)
	assert.Len(t, thread.Participants, 2)
	assert.Len(t, thread.MessageIDs, 2)
}

func TestDraftModel(t *testing.T) {
	draft := domain.Draft{
		ID:           "draft-123",
		GrantID:      "grant-456",
		Subject:      "Draft Email",
		Body:         "Draft content here",
		From:         []domain.EmailParticipant{{Email: "me@example.com"}},
		To:           []domain.EmailParticipant{{Email: "recipient@example.com"}},
		ReplyToMsgID: "original-msg-id",
		ThreadID:     "thread-789",
	}

	assert.Equal(t, "draft-123", draft.ID)
	assert.Equal(t, "Draft Email", draft.Subject)
	assert.Len(t, draft.To, 1)
	assert.Equal(t, "original-msg-id", draft.ReplyToMsgID)
}

func TestFolderModel(t *testing.T) {
	folder := domain.Folder{
		ID:              "folder-123",
		GrantID:         "grant-456",
		Name:            "Important",
		SystemFolder:    "",
		ParentID:        "parent-folder",
		BackgroundColor: "#FF0000",
		TextColor:       "#FFFFFF",
		TotalCount:      100,
		UnreadCount:     25,
		ChildIDs:        []string{"child-1", "child-2"},
		Attributes:      []string{"\\HasNoChildren"},
	}

	assert.Equal(t, "folder-123", folder.ID)
	assert.Equal(t, "Important", folder.Name)
	assert.Equal(t, 100, folder.TotalCount)
	assert.Equal(t, 25, folder.UnreadCount)
	assert.Len(t, folder.ChildIDs, 2)
}

func TestMessageModel(t *testing.T) {
	now := time.Now()
	message := domain.Message{
		ID:       "msg-123",
		GrantID:  "grant-456",
		ThreadID: "thread-789",
		Subject:  "Test Subject",
		From:     []domain.EmailParticipant{{Name: "Sender", Email: "sender@example.com"}},
		To:       []domain.EmailParticipant{{Email: "to@example.com"}},
		Cc:       []domain.EmailParticipant{{Email: "cc@example.com"}},
		Body:     "<html>Hello</html>",
		Snippet:  "Hello...",
		Date:     now,
		Unread:   true,
		Starred:  false,
		Folders:  []string{"INBOX"},
		Attachments: []domain.Attachment{
			{ID: "attach-1", Filename: "file.pdf"},
		},
		Headers: []domain.Header{
			{Name: "X-Custom", Value: "custom-value"},
		},
	}

	assert.Equal(t, "msg-123", message.ID)
	assert.Equal(t, "Test Subject", message.Subject)
	assert.Len(t, message.From, 1)
	assert.Len(t, message.Attachments, 1)
	assert.Len(t, message.Headers, 1)
	assert.True(t, message.Unread)
}

func TestPaginationModel(t *testing.T) {
	pagination := domain.Pagination{
		NextCursor: "cursor-abc123",
		HasMore:    true,
	}

	assert.Equal(t, "cursor-abc123", pagination.NextCursor)
	assert.True(t, pagination.HasMore)
}

func TestTrackingOptions(t *testing.T) {
	opts := domain.TrackingOptions{
		Opens: true,
		Links: true,
		Label: "campaign-2024",
	}

	assert.True(t, opts.Opens)
	assert.True(t, opts.Links)
	assert.Equal(t, "campaign-2024", opts.Label)
}

// TestGetFoldersSystemFolderTypes tests that GetFolders correctly handles
// system_folder field as both boolean (Google) and string (Microsoft) types.
