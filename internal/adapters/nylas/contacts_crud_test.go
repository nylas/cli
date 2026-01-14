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

func TestHTTPClient_CreateContact(t *testing.T) {
	tests := []struct {
		name           string
		request        *domain.CreateContactRequest
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
	}{
		{
			name: "creates contact with basic info",
			request: &domain.CreateContactRequest{
				GivenName: "John",
				Surname:   "Doe",
			},
			serverResponse: map[string]any{
				"data": map[string]any{
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
			serverResponse: map[string]any{
				"data": map[string]any{
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

			response := map[string]any{
				"data": map[string]any{
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
					_ = json.NewEncoder(w).Encode(map[string]any{
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
