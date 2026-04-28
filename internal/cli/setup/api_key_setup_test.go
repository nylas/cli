package setup

import (
	"context"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

type testAPIKeySetupClient struct {
	apps              []domain.Application
	listAppsErr       error
	callbackURIs      []domain.CallbackURI
	listCallbackErr   error
	createCallbackErr error
	createdRequests   []*domain.CreateCallbackURIRequest
}

func (m *testAPIKeySetupClient) ListApplications(context.Context) ([]domain.Application, error) {
	return m.apps, m.listAppsErr
}

func (m *testAPIKeySetupClient) ListCallbackURIs(context.Context) ([]domain.CallbackURI, error) {
	return m.callbackURIs, m.listCallbackErr
}

func (m *testAPIKeySetupClient) CreateCallbackURI(_ context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error) {
	m.createdRequests = append(m.createdRequests, req)
	if m.createCallbackErr != nil {
		return nil, m.createCallbackErr
	}
	return &domain.CallbackURI{ID: "cb-new", URL: req.URL, Platform: req.Platform}, nil
}

func TestResolveAPIKeyApplication_AutoSelectsSingleApp(t *testing.T) {
	t.Parallel()

	client := &testAPIKeySetupClient{
		apps: []domain.Application{{
			ID:             "app-1",
			ApplicationID:  "client-123",
			OrganizationID: "org-456",
		}},
	}

	result, err := resolveAPIKeyApplication("nyl_test", "us", "", false, func(region, clientID, apiKey string) apiKeySetupClient {
		return client
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.ClientID != "client-123" {
		t.Fatalf("expected client ID %q, got %q", "client-123", result.ClientID)
	}
	if result.OrgID != "org-456" {
		t.Fatalf("expected org ID %q, got %q", "org-456", result.OrgID)
	}
}

func TestResolveAPIKeyApplication_RequiresClientIDWhenMultipleAppsExist(t *testing.T) {
	t.Parallel()

	client := &testAPIKeySetupClient{
		apps: []domain.Application{
			{ApplicationID: "client-1"},
			{ApplicationID: "client-2"},
		},
	}

	_, err := resolveAPIKeyApplication("nyl_test", "us", "", false, func(region, clientID, apiKey string) apiKeySetupClient {
		return client
	})
	if err == nil {
		t.Fatal("expected error when multiple applications exist without an explicit client ID")
	}
	if !strings.Contains(err.Error(), "--client-id") {
		t.Fatalf("expected client-id guidance in error, got %v", err)
	}
}

func TestResolveAPIKeyApplication_RejectsUnknownExplicitClientID(t *testing.T) {
	t.Parallel()

	client := &testAPIKeySetupClient{
		apps: []domain.Application{
			{ApplicationID: "client-1"},
		},
	}

	_, err := resolveAPIKeyApplication("nyl_test", "us", "missing-client", false, func(region, clientID, apiKey string) apiKeySetupClient {
		return client
	})
	if err == nil {
		t.Fatal("expected error for an explicit client ID that does not belong to the API key")
	}
	if !strings.Contains(err.Error(), "missing-client") {
		t.Fatalf("expected explicit client ID in error, got %v", err)
	}
}

func TestEnsureOAuthCallbackURI_ReturnsAlreadyExists(t *testing.T) {
	t.Parallel()

	client := &testAPIKeySetupClient{
		callbackURIs: []domain.CallbackURI{
			{ID: "cb-1", URL: "http://127.0.0.1:9007/callback", Platform: "web"},
		},
	}

	result, err := ensureOAuthCallbackURI("nyl_test", "client-123", "us", 9007, func(region, clientID, apiKey string) apiKeySetupClient {
		return client
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.AlreadyExists {
		t.Fatal("expected callback URI to be reported as already existing")
	}
	if len(client.createdRequests) != 0 {
		t.Fatalf("expected no callback URI creation request, got %d", len(client.createdRequests))
	}
}

func TestEnsureOAuthCallbackURI_CreatesMissingURI(t *testing.T) {
	t.Parallel()

	client := &testAPIKeySetupClient{}

	result, err := ensureOAuthCallbackURI("nyl_test", "client-123", "us", 9007, func(region, clientID, apiKey string) apiKeySetupClient {
		return client
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Created {
		t.Fatal("expected callback URI to be created")
	}
	if len(client.createdRequests) != 1 {
		t.Fatalf("expected one create callback request, got %d", len(client.createdRequests))
	}
	if got := client.createdRequests[0].URL; got != "http://127.0.0.1:9007/callback" {
		t.Fatalf("expected callback URI %q, got %q", "http://127.0.0.1:9007/callback", got)
	}
}
