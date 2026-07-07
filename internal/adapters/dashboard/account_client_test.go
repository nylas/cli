package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAccountClientTestServer(t *testing.T, handler func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, body map[string]any)) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Helper()

		rawBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var body map[string]any
		if len(rawBody) > 0 {
			require.NoError(t, json.Unmarshal(rawBody, &body))
		}

		assert.Equal(t, version.UserAgent(), r.Header.Get("User-Agent"))
		handler(t, w, r, rawBody, body)
	}))
}

func writeDashboardEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	writeDashboardEnvelopeWithCursor(t, w, data, "")
}

func writeDashboardEnvelopeWithCursor(t *testing.T, w http.ResponseWriter, data any, nextCursor string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]any{
		"request_id": "req-123",
		"success":    true,
		"data":       data,
	}
	if nextCursor != "" {
		resp["nextCursor"] = nextCursor
	}
	require.NoError(t, json.NewEncoder(w).Encode(resp))
}

func TestAccountClientPublicEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, body map[string]any)
		run     func(t *testing.T, client *AccountClient)
	}{
		{
			name: "register",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/register", r.URL.Path)
				assert.Equal(t, "user@example.com", body["email"])
				assert.Equal(t, "secret", body["password"])
				assert.Equal(t, true, body["privacyPolicyAccepted"])
				assert.NotEmpty(t, r.Header.Get("DPoP"))

				writeDashboardEnvelope(t, w, map[string]any{
					"verificationChannel": "email",
					"expiresAt":           "2026-04-20T12:00:00Z",
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.Register(context.Background(), "user@example.com", "secret", true)
				require.NoError(t, err)
				assert.Equal(t, "email", resp.VerificationChannel)
			},
		},
		{
			name: "verify email code",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/verify-email-code", r.URL.Path)
				assert.Equal(t, "user@example.com", body["email"])
				assert.Equal(t, "123456", body["code"])
				assert.Equal(t, "us", body["region"])

				writeDashboardEnvelope(t, w, map[string]any{
					"userToken": "user-token",
					"orgToken":  "org-token",
					"user": map[string]any{
						"publicId": "user-1",
					},
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.VerifyEmailCode(context.Background(), "user@example.com", "123456", "us")
				require.NoError(t, err)
				assert.Equal(t, "user-token", resp.UserToken)
			},
		},
		{
			name: "resend verification code",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/resend-verification-code", r.URL.Path)
				assert.Equal(t, "user@example.com", body["email"])

				writeDashboardEnvelope(t, w, map[string]any{})
			},
			run: func(t *testing.T, client *AccountClient) {
				require.NoError(t, client.ResendVerificationCode(context.Background(), "user@example.com"))
			},
		},
		{
			name: "login MFA completion",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/login/mfa", r.URL.Path)
				assert.Equal(t, "user-1", body["userPublicId"])
				assert.Equal(t, "654321", body["code"])
				assert.Equal(t, "org-1", body["orgPublicId"])

				writeDashboardEnvelope(t, w, map[string]any{
					"userToken": "user-token",
					"orgToken":  "org-token",
					"user": map[string]any{
						"publicId": "user-1",
					},
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.LoginMFA(context.Background(), "user-1", "654321", "org-1")
				require.NoError(t, err)
				assert.Equal(t, "org-token", resp.OrgToken)
			},
		},
		{
			name: "refresh",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, _ map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/refresh", r.URL.Path)
				assert.Empty(t, rawBody)
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))

				writeDashboardEnvelope(t, w, map[string]any{
					"userToken": "user-token-new",
					"orgToken":  "org-token-new",
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.Refresh(context.Background(), "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "user-token-new", resp.UserToken)
				assert.Equal(t, "org-token-new", resp.OrgToken)
			},
		},
		{
			name: "logout",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, _ map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/logout", r.URL.Path)
				assert.Empty(t, rawBody)
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))

				writeDashboardEnvelope(t, w, map[string]any{})
			},
			run: func(t *testing.T, client *AccountClient) {
				require.NoError(t, client.Logout(context.Background(), "user-token", "org-token"))
			},
		},
		{
			name: "sso start register",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/sso/start", r.URL.Path)
				assert.Equal(t, "google_SSO", body["loginType"])
				assert.Equal(t, "register", body["mode"])
				assert.Equal(t, true, body["privacyPolicyAccepted"])

				writeDashboardEnvelope(t, w, map[string]any{
					"flowId":                  "flow-1",
					"verificationUri":         "https://example.com/device",
					"verificationUriComplete": "https://example.com/device?code=abc",
					"userCode":                "ABCDEF",
					"expiresIn":               300,
					"interval":                5,
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.SSOStart(context.Background(), domain.SSOLoginTypeGoogle, "register", true, "")
				require.NoError(t, err)
				assert.Equal(t, "flow-1", resp.FlowID)
			},
		},
		{
			name: "start SAML SSO flow sends email for home-realm discovery",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/auth/cli/sso/start", r.URL.Path)
				assert.Equal(t, "saml_SSO", body["loginType"])
				assert.Equal(t, "login", body["mode"])
				assert.Equal(t, "user@acme.com", body["email"])
				_, hasPrivacy := body["privacyPolicyAccepted"]
				assert.False(t, hasPrivacy)

				writeDashboardEnvelope(t, w, map[string]any{
					"flowId":                  "flow-saml",
					"verificationUri":         "https://accounts.example.com/pages/cli-saml",
					"verificationUriComplete": "https://accounts.example.com/pages/cli-saml?code=ABCD2345",
					"userCode":                "ABCD2345",
					"expiresIn":               600,
					"interval":                5,
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.SSOStart(context.Background(), domain.SSOLoginTypeSAML, "login", false, "user@acme.com")
				require.NoError(t, err)
				assert.Equal(t, "flow-saml", resp.FlowID)
				assert.Equal(t, "ABCD2345", resp.UserCode)
			},
		},
		{
			name: "get current session",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, rawBody []byte, _ map[string]any) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/sessions/current", r.URL.Path)
				assert.Empty(t, rawBody)
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))

				writeDashboardEnvelope(t, w, map[string]any{
					"user": map[string]any{
						"publicId": "user-1",
					},
					"currentOrg": "org-1",
					"relations": []map[string]any{
						{"orgPublicId": "org-1", "orgName": "Acme"},
					},
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.GetCurrentSession(context.Background(), "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "org-1", resp.CurrentOrg)
				require.Len(t, resp.Relations, 1)
			},
		},
		{
			name: "switch org",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/sessions/switch-org", r.URL.Path)
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
				assert.Equal(t, "org-2", body["orgPublicId"])

				writeDashboardEnvelope(t, w, map[string]any{
					"orgToken":     "org-token-new",
					"orgSessionId": "session-2",
					"org": map[string]any{
						"publicId": "org-2",
						"name":     "Beta",
					},
				})
			},
			run: func(t *testing.T, client *AccountClient) {
				resp, err := client.SwitchOrg(context.Background(), "org-2", "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "org-token-new", resp.OrgToken)
				assert.Equal(t, "Beta", resp.Org.Name)
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

func TestAccountClientLoginVariants(t *testing.T) {
	t.Parallel()

	t.Run("success response", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
			assert.Equal(t, "/auth/cli/login", r.URL.Path)
			assert.Equal(t, "user@example.com", body["email"])
			assert.Equal(t, "secret", body["password"])
			assert.Equal(t, "org-1", body["orgPublicId"])

			writeDashboardEnvelope(t, w, map[string]any{
				"userToken": "user-token",
				"orgToken":  "org-token",
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1"},
				},
			})
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		auth, mfa, err := client.Login(context.Background(), "user@example.com", "secret", "org-1")
		require.NoError(t, err)
		assert.NotNil(t, auth)
		assert.Nil(t, mfa)
		assert.Equal(t, "user-token", auth.UserToken)
	})

	t.Run("mfa required response", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
			writeDashboardEnvelope(t, w, map[string]any{
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1"},
				},
				"totpFactor": map[string]any{
					"factorSid": "factor-1",
				},
			})
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		auth, mfa, err := client.Login(context.Background(), "user@example.com", "secret", "")
		require.NoError(t, err)
		assert.Nil(t, auth)
		require.NotNil(t, mfa)
		assert.Equal(t, "factor-1", mfa.TOTPFactor.FactorSID)
	})

	t.Run("unexpected payload returns login failed", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
			writeDashboardEnvelope(t, w, map[string]any{"status": "unknown"})
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		auth, mfa, err := client.Login(context.Background(), "user@example.com", "secret", "")
		require.Error(t, err)
		assert.Nil(t, auth)
		assert.Nil(t, mfa)
		assert.ErrorIs(t, err, domain.ErrDashboardLoginFailed)
	})

	t.Run("transport and API errors are wrapped", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
			w.WriteHeader(http.StatusUnauthorized)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			}))
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		auth, mfa, err := client.Login(context.Background(), "user@example.com", "secret", "")
		require.Error(t, err)
		assert.Nil(t, auth)
		assert.Nil(t, mfa)
		assert.ErrorIs(t, err, domain.ErrDashboardLoginFailed)
		assert.Contains(t, err.Error(), "INVALID_CREDENTIALS")
	})
}

func TestAccountClientLoginMFAWrapsUnderlyingError(t *testing.T) {
	t.Parallel()

	server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
		w.WriteHeader(http.StatusUnauthorized)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_TOTP",
				"message": "Invalid MFA code",
			},
		}))
	})
	defer server.Close()

	client := &AccountClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
		dpop:       &mockDPoP{proof: "test-proof"},
	}

	resp, err := client.LoginMFA(context.Background(), "user-1", "654321", "org-1")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, domain.ErrDashboardLoginFailed)
	assert.Contains(t, err.Error(), "INVALID_TOTP")
}

func TestAccountClientSSOPollVariants(t *testing.T) {
	t.Parallel()

	t.Run("complete response populates auth", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, r *http.Request, _ []byte, body map[string]any) {
			assert.Equal(t, "/auth/cli/sso/poll", r.URL.Path)
			assert.Equal(t, "flow-1", body["flowId"])
			assert.Equal(t, "org-1", body["orgPublicId"])

			writeDashboardEnvelope(t, w, map[string]any{
				"status":    "complete",
				"userToken": "user-token",
				"orgToken":  "org-token",
				"user": map[string]any{
					"publicId": "user-1",
				},
			})
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		resp, err := client.SSOPoll(context.Background(), "flow-1", "org-1")
		require.NoError(t, err)
		require.NotNil(t, resp.Auth)
		assert.Equal(t, "user-token", resp.Auth.UserToken)
	})

	t.Run("mfa response populates MFA payload", func(t *testing.T) {
		t.Parallel()

		server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
			writeDashboardEnvelope(t, w, map[string]any{
				"status": "mfa_required",
				"user": map[string]any{
					"publicId": "user-1",
				},
				"organizations": []map[string]any{
					{"publicId": "org-1"},
				},
				"totpFactor": map[string]any{
					"factorSid": "factor-1",
				},
			})
		})
		defer server.Close()

		client := &AccountClient{
			baseURL:    server.URL,
			httpClient: server.Client(),
			dpop:       &mockDPoP{proof: "test-proof"},
		}

		resp, err := client.SSOPoll(context.Background(), "flow-1", "")
		require.NoError(t, err)
		require.NotNil(t, resp.MFA)
		assert.Equal(t, "factor-1", resp.MFA.TOTPFactor.FactorSID)
	})
}

func TestAccountClientRefreshPropagatesUnderlyingError(t *testing.T) {
	t.Parallel()

	server := newAccountClientTestServer(t, func(t *testing.T, w http.ResponseWriter, _ *http.Request, _ []byte, _ map[string]any) {
		w.WriteHeader(http.StatusUnauthorized)
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_SESSION",
				"message": "Invalid or expired session",
			},
		}))
	})
	defer server.Close()

	client := &AccountClient{
		baseURL:    server.URL,
		httpClient: server.Client(),
		dpop:       &mockDPoP{proof: "test-proof"},
	}

	resp, err := client.Refresh(context.Background(), "user-token", "org-token")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.True(t, errors.Is(err, domain.ErrDashboardSessionExpired))
	assert.Contains(t, err.Error(), "failed to refresh session")
}

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
