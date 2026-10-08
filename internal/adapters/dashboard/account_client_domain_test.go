package dashboard

import (
	"context"
	"net/http"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountClientDomainOperations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any)
		run     func(t *testing.T, client *AccountClient)
	}{
		{
			name: "list domains",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, _ map[string]any) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/orgs/inbox/domains", r.URL.Path)
				assert.Equal(t, "25", r.URL.Query().Get("limit"))
				assert.Equal(t, "cursor-1", r.URL.Query().Get("pageToken"))
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
				assert.NotEmpty(t, r.Header.Get("DPoP"))

				writeDashboardEnvelopeWithCursor(t, w, []map[string]any{
					{
						"id":                "dom_1",
						"name":              "Example",
						"domainAddress":     "example.com",
						"organizationId":    "org_1",
						"region":            "us",
						"branded":           true,
						"verifiedOwnership": true,
					},
				}, "cursor-2")
			},
			run: func(t *testing.T, client *AccountClient) {
				page, err := client.ListDomains(context.Background(), 25, "cursor-1", "user-token", "org-token")
				require.NoError(t, err)
				require.Len(t, page.Domains, 1)
				assert.Equal(t, "dom_1", page.Domains[0].ID)
				assert.Equal(t, "example.com", page.Domains[0].DomainAddress)
				assert.True(t, page.Domains[0].VerifiedOwnership)
				assert.Equal(t, "cursor-2", page.NextCursor)
			},
		},
		{
			name: "check availability",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, _ map[string]any) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/orgs/inbox/domains/availability", r.URL.Path)
				assert.Equal(t, "example.com", r.URL.Query().Get("domainAddress"))

				writeDashboardEnvelope(t, w, map[string]any{
					"domainAddress": "example.com",
					"available":     true,
					"conflictsWith": nil,
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				result, err := client.CheckDomainAvailability(context.Background(), "example.com", "user-token", "org-token")
				require.NoError(t, err)
				assert.True(t, result.Available)
				assert.Nil(t, result.ConflictsWith)
			},
		},
		{
			name: "create domain",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/orgs/inbox/domains", r.URL.Path)
				assert.Equal(t, "Example", body["name"])
				assert.Equal(t, "example.com", body["domainAddress"])
				assert.Equal(t, "eu", body["region"])

				writeDashboardEnvelope(t, w, map[string]any{
					"id":            "dom_new",
					"name":          "Example",
					"domainAddress": "example.com",
					"region":        "eu",
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				created, err := client.CreateDomain(context.Background(), domain.DashboardCreateInboxDomainInput{
					Name:          "Example",
					DomainAddress: "example.com",
					Region:        "eu",
				}, "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "dom_new", created.ID)
			},
		},
		{
			name: "update domain",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/orgs/inbox/domains/dom_1", r.URL.Path)
				assert.Equal(t, "us", r.URL.Query().Get("region"))
				assert.Equal(t, "Renamed", body["name"])

				writeDashboardEnvelope(t, w, map[string]any{
					"id":            "dom_1",
					"name":          "Renamed",
					"domainAddress": "example.com",
					"region":        "us",
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				updated, err := client.UpdateDomain(context.Background(), "dom_1", "us", domain.DashboardUpdateInboxDomainInput{Name: "Renamed"}, "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "Renamed", updated.Name)
			},
		},
		{
			name: "get domain info",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, _ map[string]any) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/orgs/inbox/domains/dom_1/info", r.URL.Path)
				assert.Equal(t, "us", r.URL.Query().Get("region"))
				assert.Equal(t, "mx", r.URL.Query().Get("type"))

				writeDashboardEnvelope(t, w, map[string]any{
					"domainId": "dom_1",
					"status":   "pending",
					"message":  "Configure MX",
					"attempt": map[string]any{
						"type": "mx",
						"options": map[string]any{
							"host":  "example.com",
							"type":  "MX",
							"value": "10 inbound.nylas.com",
						},
					},
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				info, err := client.GetDomainInfo(context.Background(), "dom_1", "us", "mx", "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "pending", info.Status)
				require.NotNil(t, info.Attempt)
				assert.Equal(t, "MX", info.Attempt.Options.Type)
			},
		},
		{
			name: "verify domain",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/orgs/inbox/domains/dom_1/verify", r.URL.Path)
				assert.Equal(t, "eu", r.URL.Query().Get("region"))
				assert.Equal(t, "ownership", body["type"])

				writeDashboardEnvelope(t, w, map[string]any{
					"domainId": "dom_1",
					"status":   "done",
					"message":  "Verified",
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				result, err := client.VerifyDomain(context.Background(), "dom_1", "eu", domain.DashboardVerifyInboxDomainInput{Type: "ownership"}, "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "done", result.Status)
			},
		},
		{
			name: "delete domain",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, _ map[string]any) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/orgs/inbox/domains/dom_1", r.URL.Path)
				assert.Equal(t, "us", r.URL.Query().Get("region"))

				writeDashboardEnvelope(t, w, map[string]any{"success": true})
			},
			run: func(t *testing.T, client *AccountClient) {
				deleted, err := client.DeleteDomain(context.Background(), "dom_1", "us", "user-token", "org-token")
				require.NoError(t, err)
				assert.True(t, deleted)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := newAccountClientTestServer(t, tt.handler)
			defer server.Close()

			client := &AccountClient{
				baseURL:    server.URL,
				httpClient: server.Client(),
				dpop:       &mockDPoP{proof: "test-proof"},
			}

			tt.run(t, client)
		})
	}
}

func TestDecodeDomainPageFallbackShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		raw        rawResponse
		wantIDs    []string
		wantCursor string
		wantErr    string
	}{
		{
			name: "object fallback with snake cursor",
			raw: rawResponse{
				Data: []byte(`{"domains":[{"id":"dom_1","domainAddress":"example.com"}],"next_cursor":"cursor-2"}`),
			},
			wantIDs:    []string{"dom_1"},
			wantCursor: "cursor-2",
		},
		{
			name: "raw response cursor wins over object cursor",
			raw: rawResponse{
				Data:       []byte(`{"domains":[{"id":"dom_1","domainAddress":"example.com"}],"nextCursor":"payload-cursor"}`),
				NextCursor: "envelope-cursor",
			},
			wantIDs:    []string{"dom_1"},
			wantCursor: "envelope-cursor",
		},
		{
			name: "empty object page with cursor",
			raw: rawResponse{
				Data: []byte(`{"domains":[],"pageToken":"cursor-3"}`),
			},
			wantCursor: "cursor-3",
		},
		{
			name: "object fallback requires domains field",
			raw: rawResponse{
				Data: []byte(`{"nextCursor":"cursor-4"}`),
			},
			wantErr: "missing domains",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page, err := decodeDomainPage(tt.raw)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCursor, page.NextCursor)
			require.Len(t, page.Domains, len(tt.wantIDs))
			for i, id := range tt.wantIDs {
				assert.Equal(t, id, page.Domains[i].ID)
			}
		})
	}
}

func TestUnwrapRawResponseExtractsNestedCursor(t *testing.T) {
	t.Parallel()

	raw, err := unwrapRawResponse([]byte(`{
		"request_id":"req",
		"success":true,
		"data":[],
		"pagination":{"next_cursor":"cursor-2"}
	}`))

	require.NoError(t, err)
	assert.JSONEq(t, `[]`, string(raw.Data))
	assert.Equal(t, "cursor-2", raw.NextCursor)
}
