//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_GetContacts(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.ContactQueryParams
		serverResponse map[string]any
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:   "returns contacts without params",
			params: nil,
			serverResponse: map[string]any{
				"data": []map[string]any{
					{
						"id":           "contact-1",
						"grant_id":     "grant-123",
						"given_name":   "John",
						"surname":      "Doe",
						"company_name": "Acme Corp",
						"emails":       []map[string]string{{"email": "john@example.com", "type": "work"}},
					},
					{
						"id":         "contact-2",
						"grant_id":   "grant-123",
						"given_name": "Jane",
						"surname":    "Smith",
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name: "filters by email",
			params: &domain.ContactQueryParams{
				Email: "john@example.com",
			},
			serverResponse: map[string]any{
				"data": []map[string]any{
					{"id": "contact-1", "given_name": "John"},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  1,
			wantErr:    false,
		},
		{
			name:   "returns empty list",
			params: nil,
			serverResponse: map[string]any{
				"data": []any{},
			},
			statusCode: http.StatusOK,
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/v3/grants/grant-123/contacts")

				if tt.params != nil && tt.params.Email != "" {
					assert.Equal(t, tt.params.Email, r.URL.Query().Get("email"))
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			contacts, err := client.GetContacts(ctx, "grant-123", tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, contacts, tt.wantCount)
		})
	}
}

func TestHTTPClient_GetContactsWithCursor(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.ContactQueryParams
		wantQueryKeys  []string
		serverResponse map[string]any
	}{
		{
			name: "includes limit param",
			params: &domain.ContactQueryParams{
				Limit: 50,
			},
			wantQueryKeys: []string{"limit"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes page token",
			params: &domain.ContactQueryParams{
				PageToken: "next-page-token",
			},
			wantQueryKeys: []string{"page_token"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes phone number filter",
			params: &domain.ContactQueryParams{
				PhoneNumber: "+1-555-0100",
			},
			wantQueryKeys: []string{"phone_number"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes source filter",
			params: &domain.ContactQueryParams{
				Source: "address_book",
			},
			wantQueryKeys: []string{"source"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes group filter",
			params: &domain.ContactQueryParams{
				Group: "group-123",
			},
			wantQueryKeys: []string{"group"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes recurse flag",
			params: &domain.ContactQueryParams{
				Recurse: true,
			},
			wantQueryKeys: []string{"recurse"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
		{
			name: "includes profile picture flag",
			params: &domain.ContactQueryParams{
				ProfilePicture: true,
			},
			wantQueryKeys: []string{"profile_picture"},
			serverResponse: map[string]any{
				"data": []any{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for _, key := range tt.wantQueryKeys {
					assert.NotEmpty(t, r.URL.Query().Get(key), "Missing query param: %s", key)
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, _ = client.GetContactsWithCursor(ctx, "grant-123", tt.params)
		})
	}

	t.Run("returns pagination info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{"id": "contact-1", "given_name": "Alice"},
				},
				"next_cursor": "eyJsYXN0X2lkIjoiY29udGFjdC0xIn0=",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		result, err := client.GetContactsWithCursor(ctx, "grant-123", nil)

		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, "eyJsYXN0X2lkIjoiY29udGFjdC0xIn0=", result.Pagination.NextCursor)
		assert.True(t, result.Pagination.HasMore)
	})
}

func TestHTTPClient_GetContact(t *testing.T) {
	tests := []struct {
		name           string
		contactID      string
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:      "returns contact successfully",
			contactID: "contact-123",
			serverResponse: map[string]any{
				"data": map[string]any{
					"id":           "contact-123",
					"grant_id":     "grant-123",
					"given_name":   "John",
					"middle_name":  "William",
					"surname":      "Doe",
					"suffix":       "Jr.",
					"nickname":     "Johnny",
					"birthday":     "1990-01-15",
					"company_name": "Acme Corp",
					"job_title":    "Engineer",
					"manager_name": "Jane Manager",
					"notes":        "Important contact",
					"picture_url":  "https://example.com/photo.jpg",
					"source":       "address_book",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "returns error for not found",
			contactID: "nonexistent",
			serverResponse: map[string]any{
				"error": map[string]string{"message": "contact not found"},
			},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				expectedPath := "/v3/grants/grant-123/contacts/" + tt.contactID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			contact, err := client.GetContact(ctx, "grant-123", tt.contactID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.contactID, contact.ID)
		})
	}
}

func TestHTTPClient_GetContactWithPicture(t *testing.T) {
	t.Run("includes profile_picture query param", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "true", r.URL.Query().Get("profile_picture"))

			response := map[string]any{
				"data": map[string]any{
					"id":         "contact-123",
					"given_name": "John",
					"picture":    "base64encodedpicturedata",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		contact, err := client.GetContactWithPicture(ctx, "grant-123", "contact-123", true)

		require.NoError(t, err)
		assert.Equal(t, "base64encodedpicturedata", contact.Picture)
	})

	t.Run("excludes profile_picture when false", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Empty(t, r.URL.Query().Get("profile_picture"))

			response := map[string]any{
				"data": map[string]any{
					"id":         "contact-123",
					"given_name": "John",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		_, err := client.GetContactWithPicture(ctx, "grant-123", "contact-123", false)

		require.NoError(t, err)
	})
}
