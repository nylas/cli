//go:build !integration
// +build !integration

package nylas

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ut builds a domain.UnixTime from Unix seconds for draftResponse fixtures.
func ut(sec int64) domain.UnixTime {
	return domain.UnixTime{Time: time.Unix(sec, 0)}
}

func TestConvertDraft(t *testing.T) {
	now := time.Now().Unix()

	apiDraft := draftResponse{
		ID:      "draft-123",
		GrantID: "grant-456",
		Subject: "Test Draft",
		Body:    "Draft body content",
		From: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "Sender", Email: "sender@example.com"},
		},
		To: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "Recipient", Email: "recipient@example.com"},
		},
		Cc: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "CC User", Email: "cc@example.com"},
		},
		Bcc: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "BCC User", Email: "bcc@example.com"},
		},
		ReplyTo: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "Reply", Email: "reply@example.com"},
		},
		ReplyToMsgID: "msg-original",
		ThreadID:     "thread-789",
		Attachments: []struct {
			ID          string `json:"id"`
			Filename    string `json:"filename"`
			ContentType string `json:"content_type"`
			Size        int64  `json:"size"`
		}{
			{
				ID:          "attach-1",
				Filename:    "document.pdf",
				ContentType: "application/pdf",
				Size:        50000,
			},
		},
		CreatedAt: ut(now - 3600),
		UpdatedAt: ut(now),
	}

	draft := convertDraft(apiDraft)

	assert.Equal(t, "draft-123", draft.ID)
	assert.Equal(t, "grant-456", draft.GrantID)
	assert.Equal(t, "Test Draft", draft.Subject)
	assert.Equal(t, "Draft body content", draft.Body)
	assert.Equal(t, "msg-original", draft.ReplyToMsgID)
	assert.Equal(t, "thread-789", draft.ThreadID)

	// Test participant conversions
	assert.Len(t, draft.From, 1)
	assert.Equal(t, "Sender", draft.From[0].Name)
	assert.Equal(t, "sender@example.com", draft.From[0].Email)

	assert.Len(t, draft.To, 1)
	assert.Equal(t, "Recipient", draft.To[0].Name)

	assert.Len(t, draft.Cc, 1)
	assert.Equal(t, "CC User", draft.Cc[0].Name)

	assert.Len(t, draft.Bcc, 1)
	assert.Equal(t, "BCC User", draft.Bcc[0].Name)

	assert.Len(t, draft.ReplyTo, 1)
	assert.Equal(t, "Reply", draft.ReplyTo[0].Name)

	// Test attachments
	assert.Len(t, draft.Attachments, 1)
	assert.Equal(t, "attach-1", draft.Attachments[0].ID)
	assert.Equal(t, "document.pdf", draft.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", draft.Attachments[0].ContentType)
	assert.Equal(t, int64(50000), draft.Attachments[0].Size)

	// Test timestamps
	assert.Equal(t, time.Unix(now-3600, 0), draft.CreatedAt)
	assert.Equal(t, time.Unix(now, 0), draft.UpdatedAt)
}

func TestConvertDrafts(t *testing.T) {
	now := time.Now().Unix()

	apiDrafts := []draftResponse{
		{
			ID:      "draft-1",
			GrantID: "grant-1",
			Subject: "Draft One",
			Body:    "Body one",
			To: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User1", Email: "user1@example.com"},
			},
			CreatedAt: ut(now),
			UpdatedAt: ut(now),
		},
		{
			ID:      "draft-2",
			GrantID: "grant-2",
			Subject: "Draft Two",
			Body:    "Body two",
			To: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User2", Email: "user2@example.com"},
			},
			CreatedAt: ut(now),
			UpdatedAt: ut(now),
		},
	}

	// Test convertDrafts uses util.Map
	drafts := convertDrafts(apiDrafts)

	assert.Len(t, drafts, 2)
	assert.Equal(t, "draft-1", drafts[0].ID)
	assert.Equal(t, "Draft One", drafts[0].Subject)
	assert.Equal(t, "draft-2", drafts[1].ID)
	assert.Equal(t, "Draft Two", drafts[1].Subject)
}

func TestConvertDrafts_Empty(t *testing.T) {
	// Test with empty slice
	drafts := convertDrafts([]draftResponse{})
	assert.NotNil(t, drafts)
	assert.Len(t, drafts, 0)
}

// TestConvertDraft_DateFallback verifies that drafts use the `date` field for
// timestamps. The Nylas v3 drafts API carries the real timestamp in `date` and
// leaves created_at/updated_at unset — without this fallback the CLI renders
// every draft at the zero time ("292 years ago") or the Unix epoch.
//
// The two cases are decoded from raw JSON so the test exercises the real
// unmarshal path: created_at/updated_at omitted (decodes to the zero time) and
// sent as null (decodes to the Unix epoch).
func TestConvertDraft_DateFallback(t *testing.T) {
	date := int64(1782059501)

	cases := map[string]string{
		"omitted": `{"id":"d","date":1782059501}`,
		"null":    `{"id":"d","created_at":null,"updated_at":null,"date":1782059501}`,
	}

	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			var d draftResponse
			require.NoError(t, json.Unmarshal([]byte(raw), &d))

			draft := convertDraft(d)

			assert.Equal(t, time.Unix(date, 0), draft.UpdatedAt, "UpdatedAt should fall back to date")
			assert.Equal(t, time.Unix(date, 0), draft.CreatedAt, "CreatedAt should fall back to date")
		})
	}
}

// TestConvertDraft_ExplicitTimestampsWin verifies created_at/updated_at take
// precedence over date when the API does provide them.
func TestConvertDraft_ExplicitTimestampsWin(t *testing.T) {
	date := int64(1782059501)
	created := date - 3600
	updated := date - 60

	draft := convertDraft(draftResponse{
		ID:        "draft-explicit",
		Date:      ut(date),
		CreatedAt: ut(created),
		UpdatedAt: ut(updated),
	})

	assert.Equal(t, time.Unix(created, 0), draft.CreatedAt)
	assert.Equal(t, time.Unix(updated, 0), draft.UpdatedAt)
}
