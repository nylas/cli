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
		serverResponse map[string]interface{}
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:   "returns contacts without params",
			params: nil,
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
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
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
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
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
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
		serverResponse map[string]interface{}
	}{
		{
			name: "includes limit param",
			params: &domain.ContactQueryParams{
				Limit: 50,
			},
			wantQueryKeys: []string{"limit"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes page token",
			params: &domain.ContactQueryParams{
				PageToken: "next-page-token",
			},
			wantQueryKeys: []string{"page_token"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes phone number filter",
			params: &domain.ContactQueryParams{
				PhoneNumber: "+1-555-0100",
			},
			wantQueryKeys: []string{"phone_number"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes source filter",
			params: &domain.ContactQueryParams{
				Source: "address_book",
			},
			wantQueryKeys: []string{"source"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes group filter",
			params: &domain.ContactQueryParams{
				Group: "group-123",
			},
			wantQueryKeys: []string{"group"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes recurse flag",
			params: &domain.ContactQueryParams{
				Recurse: true,
			},
			wantQueryKeys: []string{"recurse"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
		},
		{
			name: "includes profile picture flag",
			params: &domain.ContactQueryParams{
				ProfilePicture: true,
			},
			wantQueryKeys: []string{"profile_picture"},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
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
			response := map[string]interface{}{
				"data": []map[string]interface{}{
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
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:      "returns contact successfully",
			contactID: "contact-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
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
			serverResponse: map[string]interface{}{
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

			response := map[string]interface{}{
				"data": map[string]interface{}{
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

			response := map[string]interface{}{
				"data": map[string]interface{}{
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

func TestHTTPClient_CreateContact(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.CreateContactRequest
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
	}{
		{
			name: "creates contact with basic info",
			request: &domain.CreateContactRequest{
				GivenName: "John",
				Surname:   "Doe",
			},
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":         "new-contact-123",
					"grant_id":   "grant-123",
					"given_name": "John",
					"surname":    "Doe",
				},
			},
			statusCode: http.StatusCreated,
			wantErr:    false,
		},
		{
			name: "creates contact with full info",
			request: &domain.CreateContactRequest{
				GivenName:   "Jane",
				MiddleName:  "Marie",
				Surname:     "Smith",
				Suffix:      "PhD",
				Nickname:    "Janie",
				Birthday:    "1985-05-20",
				CompanyName: "Tech Corp",
				JobTitle:    "CTO",
				ManagerName: "CEO Person",
				Notes:       "VIP contact",
				Emails: []domain.ContactEmail{
					{Email: "jane@work.com", Type: "work"},
					{Email: "jane@personal.com", Type: "personal"},
				},
				PhoneNumbers: []domain.ContactPhone{
					{Number: "+1-555-0100", Type: "mobile"},
				},
			},
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":         "new-contact-456",
					"grant_id":   "grant-123",
					"given_name": "Jane",
					"surname":    "Smith",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/contacts", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			contact, err := client.CreateContact(ctx, "grant-123", tt.request)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, contact.ID)
		})
	}
}

func TestHTTPClient_UpdateContact(t *testing.T) {
	t.Run("updates contact fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/v3/grants/grant-123/contacts/contact-456", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":         "contact-456",
					"grant_id":   "grant-123",
					"given_name": "Updated",
					"surname":    "Name",
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
		givenName := "Updated"
		surname := "Name"
		req := &domain.UpdateContactRequest{
			GivenName: &givenName,
			Surname:   &surname,
		}
		contact, err := client.UpdateContact(ctx, "grant-123", "contact-456", req)

		require.NoError(t, err)
		assert.Equal(t, "contact-456", contact.ID)
	})
}

func TestHTTPClient_DeleteContact(t *testing.T) {
	tests := []struct {
		name       string
		contactID  string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes with 200",
			contactID:  "contact-123",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes with 204",
			contactID:  "contact-456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			contactID:  "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/grant-123/contacts/" + tt.contactID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]string{"message": "not found"},
					})
				}
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.DeleteContact(ctx, "grant-123", tt.contactID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
