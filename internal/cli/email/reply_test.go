package email

import (
	"context"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplySubject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		original string
		want     string
	}{
		{name: "adds prefix", original: "Hello", want: "Re: Hello"},
		{name: "keeps existing Re prefix", original: "Re: Hello", want: "Re: Hello"},
		{name: "keeps existing prefix case-insensitively", original: "re: hello", want: "re: hello"},
		{name: "keeps uppercase prefix", original: "RE: hello", want: "RE: hello"},
		{name: "ignores surrounding whitespace when detecting prefix", original: "  Re: hi", want: "  Re: hi"},
		{name: "empty subject", original: "", want: "Re:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, replySubject(tt.original))
		})
	}
}

func emails(participants []domain.EmailParticipant) []string {
	out := make([]string, len(participants))
	for i, p := range participants {
		out[i] = p.Email
	}
	return out
}

func TestBuildReplyRecipients(t *testing.T) {
	t.Parallel()

	t.Run("sender only reply targets the original sender", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "alice@example.com"}},
			To:   []domain.EmailParticipant{{Email: "me@example.com"}, {Email: "bob@example.com"}},
			Cc:   []domain.EmailParticipant{{Email: "carol@example.com"}},
		}
		to, cc, err := buildReplyRecipients(orig, "me@example.com", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com"}, emails(to))
		assert.Empty(t, cc)
	})

	t.Run("reply-all adds other recipients and excludes self", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "alice@example.com"}},
			To:   []domain.EmailParticipant{{Email: "me@example.com"}, {Email: "bob@example.com"}},
			Cc:   []domain.EmailParticipant{{Email: "carol@example.com"}},
		}
		to, cc, err := buildReplyRecipients(orig, "me@example.com", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com"}, emails(to))
		assert.Equal(t, []string{"bob@example.com", "carol@example.com"}, emails(cc))
	})

	t.Run("prefers Reply-To header over From", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From:    []domain.EmailParticipant{{Email: "alice@example.com"}},
			ReplyTo: []domain.EmailParticipant{{Email: "list@example.com"}},
		}
		to, _, err := buildReplyRecipients(orig, "me@example.com", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"list@example.com"}, emails(to))
	})

	t.Run("ignores a blank Reply-To and falls back to From", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From:    []domain.EmailParticipant{{Email: "alice@example.com"}},
			ReplyTo: []domain.EmailParticipant{{Email: "   "}},
		}
		to, _, err := buildReplyRecipients(orig, "me@example.com", false)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com"}, emails(to))
	})

	t.Run("excludes self case-insensitively and dedupes cc", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "alice@example.com"}},
			To:   []domain.EmailParticipant{{Email: "ME@example.com"}, {Email: "bob@example.com"}},
			Cc:   []domain.EmailParticipant{{Email: "Bob@example.com"}, {Email: "carol@example.com"}},
		}
		_, cc, err := buildReplyRecipients(orig, "me@example.com", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"bob@example.com", "carol@example.com"}, emails(cc))
	})

	t.Run("reply-all does not duplicate the reply target in cc", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "alice@example.com"}},
			To:   []domain.EmailParticipant{{Email: "alice@example.com"}, {Email: "bob@example.com"}},
		}
		to, cc, err := buildReplyRecipients(orig, "me@example.com", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"alice@example.com"}, emails(to))
		assert.Equal(t, []string{"bob@example.com"}, emails(cc))
	})

	t.Run("errors when nothing to reply to", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{Subject: "orphan"}
		_, _, err := buildReplyRecipients(orig, "me@example.com", false)
		require.Error(t, err)
	})

	t.Run("reply-all to your own message targets the original recipients", func(t *testing.T) {
		t.Parallel()
		// You sent this message; replying-all must go to the people you sent it
		// to, not back to yourself.
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "me@example.com"}},
			To:   []domain.EmailParticipant{{Email: "bob@example.com"}},
			Cc:   []domain.EmailParticipant{{Email: "carol@example.com"}},
		}
		to, cc, err := buildReplyRecipients(orig, "me@example.com", true)
		require.NoError(t, err)
		assert.Equal(t, []string{"bob@example.com", "carol@example.com"}, emails(to))
		assert.Empty(t, cc, "self should never appear and there is no separate target")
	})

	t.Run("errors when replying to your own message without --all", func(t *testing.T) {
		t.Parallel()
		orig := &domain.Message{
			From: []domain.EmailParticipant{{Email: "me@example.com"}},
			To:   []domain.EmailParticipant{{Email: "bob@example.com"}},
		}
		_, _, err := buildReplyRecipients(orig, "me@example.com", false)
		require.Error(t, err, "only recipient would be yourself; should guide toward --all")
	})
}

func TestReadReplyBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "terminated by dot", input: "line one\nline two\n.\nignored", want: "line one\nline two"},
		{name: "handles CRLF line endings", input: "line one\r\nline two\r\n.\r\n", want: "line one\nline two"},
		{name: "preserves blank lines before dot", input: "a\n\nb\n.\n", want: "a\n\nb"},
		{name: "eof without dot does not loop", input: "no terminator\nstill no terminator", want: "no terminator\nstill no terminator"},
		{name: "empty input", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, readReplyBody(strings.NewReader(tt.input)))
		})
	}
}

func TestReplyCmd_ThreadsViaReplyToMsgID(t *testing.T) {
	t.Parallel()

	client := nylas.NewMockClient()
	client.GetMessageFunc = func(_ context.Context, _, messageID string) (*domain.Message, error) {
		return &domain.Message{
			ID:      messageID,
			Subject: "Project update",
			From:    []domain.EmailParticipant{{Email: "alice@example.com"}},
		}, nil
	}
	client.GetGrantFunc = func(_ context.Context, grantID string) (*domain.Grant, error) {
		return &domain.Grant{ID: grantID, Provider: domain.ProviderGoogle, Email: "me@example.com"}, nil
	}

	var gotReq *domain.SendMessageRequest
	client.SendMessageFunc = func(_ context.Context, _ string, req *domain.SendMessageRequest) (*domain.Message, error) {
		gotReq = req
		return &domain.Message{ID: "sent-id", Subject: req.Subject}, nil
	}

	grant := &domain.Grant{ID: "grant-1", Provider: domain.ProviderGoogle, Email: "me@example.com"}
	req, err := buildReplyRequest(context.Background(), client, "grant-1", grant, "msg-original", "Sounds good", false)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "msg-original", req.ReplyToMsgID, "reply must thread via reply_to_message_id")
	assert.Equal(t, "Re: Project update", req.Subject)
	assert.Equal(t, []string{"alice@example.com"}, emails(req.To))
	assert.Equal(t, "Sounds good", req.Body)

	// Sanity: the request actually sends through the per-grant path used by send.
	sent, err := sendMessageForGrant(context.Background(), client, "grant-1", grant, req)
	require.NoError(t, err)
	assert.Equal(t, "sent-id", sent.ID)
	require.NotNil(t, gotReq)
	assert.Equal(t, "msg-original", gotReq.ReplyToMsgID)
}
