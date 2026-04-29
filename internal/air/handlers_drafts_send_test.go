package air

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	nylasmock "github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

// newSendTestServer builds a non-demo Server with a mock Nylas client and a
// grant store containing the supplied grants. defaultGrantID identifies which
// grant the server should treat as the active default.
func newSendTestServer(t *testing.T, grants []domain.GrantInfo, defaultGrantID string) (*Server, *nylasmock.MockClient) {
	t.Helper()

	client := nylasmock.NewMockClient()
	store := &testGrantStore{
		grants:       append([]domain.GrantInfo(nil), grants...),
		defaultGrant: defaultGrantID,
	}
	return &Server{
		grantStore:  store,
		nylasClient: client,
	}, client
}

// resetTransactionalMock clears the package-level SendTransactionalMessageFunc
// after each test that touches it.
func resetTransactionalMock(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		nylasmock.SendTransactionalMessageFunc = nil
	})
}

func sendRequest(t *testing.T, body map[string]any) *http.Request {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/send", bytes.NewBuffer(encoded))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleSendMessage_RequestGrantWins_GoogleStaysGoogle(t *testing.T) {
	resetTransactionalMock(t)

	const googleID = "grant-google"
	const nylasID = "grant-nylas"
	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: nylasID, Email: "managed@example.nylas.email", Provider: domain.ProviderNylas},
		{ID: googleID, Email: "qasim.m@nylas.com", Provider: domain.ProviderGoogle},
	}, nylasID) // default points at Nylas to prove request grant wins

	var seenGrantID string
	client.GetGrantFunc = func(_ context.Context, id string) (*domain.Grant, error) {
		seenGrantID = id
		return &domain.Grant{ID: id, Email: "qasim.m@nylas.com", Provider: domain.ProviderGoogle}, nil
	}

	nylasmock.SendTransactionalMessageFunc = func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatalf("transactional endpoint must not be called for a Google-provider grant")
		return nil, nil
	}

	var sentGrantID string
	var sentReq *domain.SendMessageRequest
	client.SendMessageFunc = func(_ context.Context, id string, r *domain.SendMessageRequest) (*domain.Message, error) {
		sentGrantID = id
		sentReq = r
		return &domain.Message{ID: "msg-1"}, nil
	}

	w := httptest.NewRecorder()
	server.handleSendMessage(w, sendRequest(t, map[string]any{
		"grant_id": googleID,
		"to":       []map[string]string{{"email": "to@example.com"}},
		"subject":  "hi",
		"body":     "hello",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if seenGrantID != googleID {
		t.Errorf("GetGrant got %q, want %q", seenGrantID, googleID)
	}
	if sentGrantID != googleID {
		t.Errorf("SendMessage got %q, want %q", sentGrantID, googleID)
	}
	if sentReq == nil || len(sentReq.From) != 0 {
		t.Errorf("standard providers should not have From auto-populated, got %+v", sentReq)
	}
}

func TestHandleSendMessage_NylasGrantArchivesViaPerGrantSend(t *testing.T) {
	// Per /v3/grants/{id}/messages/send + From is what archives a message to
	// the Sent folder for Nylas-managed grants. The transactional endpoint is
	// a non-archiving relay and must not be used here.
	resetTransactionalMock(t)

	const nylasID = "grant-nylas"
	const grantEmail = "support@managed.nylas.email"
	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: nylasID, Email: grantEmail, Provider: domain.ProviderNylas},
	}, nylasID)

	client.GetGrantFunc = func(_ context.Context, id string) (*domain.Grant, error) {
		return &domain.Grant{ID: id, Email: grantEmail, Provider: domain.ProviderNylas}, nil
	}
	nylasmock.SendTransactionalMessageFunc = func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatal("transactional endpoint must not be used; per-grant send is what archives to Sent")
		return nil, nil
	}

	var sentGrantID string
	var sentReq *domain.SendMessageRequest
	client.SendMessageFunc = func(_ context.Context, id string, r *domain.SendMessageRequest) (*domain.Message, error) {
		sentGrantID = id
		sentReq = r
		return &domain.Message{ID: "msg-archived"}, nil
	}

	w := httptest.NewRecorder()
	server.handleSendMessage(w, sendRequest(t, map[string]any{
		"grant_id": nylasID,
		"to":       []map[string]string{{"email": "to@example.com"}},
		"subject":  "hi",
		"body":     "hello",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if sentGrantID != nylasID {
		t.Errorf("SendMessage grantID = %q, want %q", sentGrantID, nylasID)
	}
	if sentReq == nil || len(sentReq.From) != 1 || sentReq.From[0].Email != grantEmail {
		t.Errorf("Nylas grant must have From auto-populated to %q, got %+v", grantEmail, sentReq)
	}
}

func TestHandleSendMessage_NoGrantID_FallsBackToDefault(t *testing.T) {
	resetTransactionalMock(t)

	const googleID = "grant-google"
	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: googleID, Email: "qasim.m@nylas.com", Provider: domain.ProviderGoogle},
	}, googleID)

	client.GetGrantFunc = func(_ context.Context, id string) (*domain.Grant, error) {
		return &domain.Grant{ID: id, Email: "qasim.m@nylas.com", Provider: domain.ProviderGoogle}, nil
	}
	var sentGrantID string
	client.SendMessageFunc = func(_ context.Context, id string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		sentGrantID = id
		return &domain.Message{ID: "msg-default"}, nil
	}

	w := httptest.NewRecorder()
	server.handleSendMessage(w, sendRequest(t, map[string]any{
		// no grant_id — server must use default
		"to":      []map[string]string{{"email": "to@example.com"}},
		"subject": "hi",
		"body":    "hello",
	}))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if sentGrantID != googleID {
		t.Errorf("SendMessage got %q, want default %q", sentGrantID, googleID)
	}
}

func TestHandleSendMessage_UnknownGrantID_Rejected(t *testing.T) {
	resetTransactionalMock(t)

	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: "grant-known", Email: "x@y.com", Provider: domain.ProviderGoogle},
	}, "grant-known")

	client.SendMessageFunc = func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatal("send must not run when grant_id does not belong to user")
		return nil, nil
	}
	nylasmock.SendTransactionalMessageFunc = func(_ context.Context, _ string, _ *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatal("transactional send must not run when grant_id is unknown")
		return nil, nil
	}

	w := httptest.NewRecorder()
	server.handleSendMessage(w, sendRequest(t, map[string]any{
		"grant_id": "grant-evil",
		"to":       []map[string]string{{"email": "to@example.com"}},
		"subject":  "hi",
		"body":     "hello",
	}))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown grant, got %d: %s", w.Code, w.Body.String())
	}
}

func TestResolveSendGrantID(t *testing.T) {
	server, _ := newSendTestServer(t, []domain.GrantInfo{
		{ID: "g1", Email: "a@x.com", Provider: domain.ProviderGoogle},
		{ID: "g2", Email: "b@y.com", Provider: domain.ProviderNylas},
	}, "g1")

	tests := []struct {
		name      string
		requested string
		want      string
		wantErr   error
	}{
		{name: "empty falls back to default", requested: "", want: "g1"},
		{name: "valid request grant", requested: "g2", want: "g2"},
		{name: "unknown grant rejected", requested: "g-bogus", wantErr: errSendGrantNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := server.resolveSendGrantID(tt.requested, "g1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSendMessageForGrant_GetGrantError(t *testing.T) {
	resetTransactionalMock(t)
	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: "g1", Email: "a@x.com", Provider: domain.ProviderGoogle},
	}, "g1")
	client.GetGrantFunc = func(_ context.Context, _ string) (*domain.Grant, error) {
		return nil, errors.New("boom")
	}

	_, err := server.sendMessageForGrant(context.Background(), "g1", &domain.SendMessageRequest{})
	if err == nil || !strings.Contains(err.Error(), "fetch grant") {
		t.Fatalf("expected fetch-grant wrapped error, got %v", err)
	}
}

func TestSendMessageForGrant_PreservesCallerFrom(t *testing.T) {
	// If the caller already supplied a From, sendMessageForGrant must respect
	// it (no auto-populate, no GetGrant call).
	resetTransactionalMock(t)

	server, client := newSendTestServer(t, []domain.GrantInfo{
		{ID: "g1", Email: "real@x.com", Provider: domain.ProviderNylas},
	}, "g1")
	client.GetGrantFunc = func(_ context.Context, _ string) (*domain.Grant, error) {
		t.Fatal("GetGrant must not be called when From is already set")
		return nil, nil
	}

	var sentReq *domain.SendMessageRequest
	client.SendMessageFunc = func(_ context.Context, _ string, r *domain.SendMessageRequest) (*domain.Message, error) {
		sentReq = r
		return &domain.Message{ID: "msg"}, nil
	}

	original := []domain.EmailParticipant{{Email: "explicit@example.com", Name: "Caller"}}
	_, err := server.sendMessageForGrant(context.Background(), "g1", &domain.SendMessageRequest{
		From: original,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sentReq == nil || len(sentReq.From) != 1 || sentReq.From[0].Email != "explicit@example.com" {
		t.Errorf("From was rewritten; want explicit@example.com, got %+v", sentReq)
	}
}
