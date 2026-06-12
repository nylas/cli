package ai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// chunkedReader yields one predefined chunk per Read call to simulate
// SSE events arriving split across network reads.
type chunkedReader struct {
	chunks []string
	i      int
}

func (r *chunkedReader) Read(p []byte) (int, error) {
	if r.i >= len(r.chunks) {
		return 0, io.EOF
	}
	n := copy(p, r.chunks[r.i])
	r.i++
	return n, nil
}

func collectSSEEvents(t *testing.T, r io.Reader) []sseEvent {
	t.Helper()
	scanner := &sseScanner{reader: r}
	var events []sseEvent
	for scanner.Scan() {
		events = append(events, scanner.Event())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	return events
}

func assertEventTypes(t *testing.T, events []sseEvent, want []string) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("got %d events, want %d (events: %+v)", len(events), len(want), events)
	}
	for i, eventType := range want {
		if events[i].Type != eventType {
			t.Errorf("event[%d].Type = %q, want %q", i, events[i].Type, eventType)
		}
	}
}

func TestSSEScanner_MultipleEventsInOneChunk(t *testing.T) {
	stream := "event: message_start\n" +
		"data: {\"type\":\"message_start\"}\n" +
		"\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hel\"}}\n" +
		"\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"lo\"}}\n" +
		"\n" +
		"data: {\"type\":\"message_stop\"}\n" +
		"\n"

	events := collectSSEEvents(t, strings.NewReader(stream))
	assertEventTypes(t, events, []string{
		"message_start",
		"content_block_delta",
		"content_block_delta",
		"message_stop",
	})
}

func TestSSEScanner_EventSplitAcrossReads(t *testing.T) {
	reader := &chunkedReader{chunks: []string{
		"data: {\"type\":\"content_blo",
		"ck_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n",
		"data: {\"type\":\"mess",
		"age_stop\"}\n\n",
	}}

	events := collectSSEEvents(t, reader)
	assertEventTypes(t, events, []string{"content_block_delta", "message_stop"})
}

func TestSSEScanner_SkipsNonDataLines(t *testing.T) {
	// Comments, event: lines, pings, and blank lines must not terminate
	// the stream — even when a whole read contains no data line.
	reader := &chunkedReader{chunks: []string{
		": keepalive comment\n\n",
		"event: ping\ndata: {\"type\":\"ping\"}\n\n",
		"event: content_block_delta\n",
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"after\"}}\n\n",
		"data: {\"type\":\"message_stop\"}\n\n",
	}}

	events := collectSSEEvents(t, reader)
	assertEventTypes(t, events, []string{"ping", "content_block_delta", "message_stop"})
}

func TestSSEScanner_TrailingMalformedEventSurfacesError(t *testing.T) {
	// Stream ends without a trailing blank line and the accumulated data is
	// not valid JSON (e.g. a connection cut mid-payload). Unlike mid-stream
	// parse failures — which are skipped because the next event recovers the
	// stream — there is no next event here, so dropping the data silently
	// would be undetectable data loss. Err() must report it.
	stream := "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n" +
		"\n" +
		"data: {\"type\":\"message_stop\"" // truncated: no closing brace, no blank line

	scanner := &sseScanner{reader: strings.NewReader(stream)}
	var events []sseEvent
	for scanner.Scan() {
		events = append(events, scanner.Event())
	}

	assertEventTypes(t, events, []string{"content_block_delta"})
	if err := scanner.Err(); err == nil {
		t.Fatal("Err() = nil, want error for malformed trailing event")
	}
}

func TestSSEScanner_TrailingWellFormedEventWithoutBlankLine(t *testing.T) {
	// A valid final event that is missing only the terminating blank line
	// must still be delivered, with no error.
	stream := "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n" +
		"\n" +
		"data: {\"type\":\"message_stop\"}"

	events := collectSSEEvents(t, strings.NewReader(stream))
	assertEventTypes(t, events, []string{"content_block_delta", "message_stop"})
}

func TestClaudeClient_StreamChat_StreamsAllChunks(t *testing.T) {
	stream := "event: message_start\n" +
		"data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\"}}\n" +
		"\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n" +
		"\n" +
		": ping\n" +
		"\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n" +
		"\n" +
		"event: message_stop\n" +
		"data: {\"type\":\"message_stop\"}\n" +
		"\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(stream))
	}))
	defer server.Close()

	client := NewClaudeClient(&domain.ClaudeConfig{APIKey: "test-key"})
	client.baseURL = server.URL

	var chunks []string
	err := client.StreamChat(context.Background(), &domain.ChatRequest{
		Messages: []domain.ChatMessage{{Role: "user", Content: "Hello"}},
	}, func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamChat() error = %v", err)
	}

	got := strings.Join(chunks, "")
	if got != "Hello world" {
		t.Errorf("streamed content = %q, want %q (chunks: %v)", got, "Hello world", chunks)
	}
}

func TestClaudeClient_StreamChat_HTTPError(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"unauthorized", http.StatusUnauthorized},
		{"rate limited", http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"request failed"}}`))
			}))
			defer server.Close()

			client := NewClaudeClient(&domain.ClaudeConfig{APIKey: "test-key"})
			client.baseURL = server.URL

			var chunks []string
			err := client.StreamChat(context.Background(), &domain.ChatRequest{
				Messages: []domain.ChatMessage{{Role: "user", Content: "Hello"}},
			}, func(chunk string) error {
				chunks = append(chunks, chunk)
				return nil
			})

			if err == nil {
				t.Fatalf("StreamChat() error = nil, want error for HTTP %d", tt.status)
			}
			if !strings.Contains(err.Error(), strconv.Itoa(tt.status)) {
				t.Errorf("error %q does not mention status %d", err.Error(), tt.status)
			}
			if len(chunks) != 0 {
				t.Errorf("expected no chunks on HTTP error, got %v", chunks)
			}
		})
	}
}
