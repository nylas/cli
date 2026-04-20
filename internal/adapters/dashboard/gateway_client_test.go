package dashboard

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayClientOperations(t *testing.T) {
	tests := []struct {
		name    string
		run     func(t *testing.T, client *GatewayClient)
		handler func(t *testing.T, w http.ResponseWriter, r *http.Request, body map[string]any)
	}{
		{
			name: "list applications",
			handler: func(t *testing.T, w http.ResponseWriter, r *http.Request, body map[string]any) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "Bearer user-token", r.Header.Get("Authorization"))
				assert.Equal(t, "org-token", r.Header.Get("X-Nylas-Org"))
				assert.NotEmpty(t, r.Header.Get("DPoP"))

				assert.Contains(t, body["query"].(string), "applications(filter: $filter)")
				variables := body["variables"].(map[string]any)
				filter := variables["filter"].(map[string]any)
				assert.Equal(t, "org-1", filter["orgPublicId"])

				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"applications": map[string]any{
							"applications": []map[string]any{
								{
									"applicationId":  "app-1",
									"organizationId": "org-1",
									"region":         "us",
									"environment":    "sandbox",
									"branding": map[string]any{
										"name":        "App One",
										"description": "Primary",
									},
								},
							},
						},
					},
				}))
			},
			run: func(t *testing.T, client *GatewayClient) {
				apps, err := client.ListApplications(context.Background(), "org-1", "us", "user-token", "org-token")
				require.NoError(t, err)
				require.Len(t, apps, 1)
				assert.Equal(t, "app-1", apps[0].ApplicationID)
				require.NotNil(t, apps[0].Branding)
				assert.Equal(t, "App One", apps[0].Branding.Name)
			},
		},
		{
			name: "create application",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request, body map[string]any) {
				assert.Contains(t, body["query"].(string), "createApplication(orgPublicId: $orgPublicId, options: $options)")
				variables := body["variables"].(map[string]any)
				assert.Equal(t, "org-1", variables["orgPublicId"])
				options := variables["options"].(map[string]any)
				assert.Equal(t, "eu", options["region"])
				branding := options["branding"].(map[string]any)
				assert.Equal(t, "Created App", branding["name"])

				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"createApplication": map[string]any{
							"applicationId":  "app-new",
							"clientSecret":   "secret",
							"organizationId": "org-1",
							"region":         "eu",
							"environment":    "production",
							"branding": map[string]any{
								"name": "Created App",
							},
						},
					},
				}))
			},
			run: func(t *testing.T, client *GatewayClient) {
				app, err := client.CreateApplication(context.Background(), "org-1", "eu", "Created App", "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "app-new", app.ApplicationID)
				assert.Equal(t, "secret", app.ClientSecret)
			},
		},
		{
			name: "list api keys",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request, body map[string]any) {
				assert.Contains(t, body["query"].(string), "apiKeys(appId: $appId)")
				variables := body["variables"].(map[string]any)
				assert.Equal(t, "app-1", variables["appId"])

				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"apiKeys": []map[string]any{
							{
								"id":          "key-1",
								"name":        "CI",
								"status":      "active",
								"permissions": []string{"send"},
								"expiresAt":   10.0,
								"createdAt":   5.0,
							},
						},
					},
				}))
			},
			run: func(t *testing.T, client *GatewayClient) {
				keys, err := client.ListAPIKeys(context.Background(), "app-1", "us", "user-token", "org-token")
				require.NoError(t, err)
				require.Len(t, keys, 1)
				assert.Equal(t, "key-1", keys[0].ID)
			},
		},
		{
			name: "create api key includes expiresIn when set",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request, body map[string]any) {
				assert.Contains(t, body["query"].(string), "createApiKey(appId: $appId, options: $options)")
				variables := body["variables"].(map[string]any)
				assert.Equal(t, "app-1", variables["appId"])
				options := variables["options"].(map[string]any)
				assert.Equal(t, "Nightly", options["name"])
				assert.Equal(t, float64(30), options["expiresIn"])

				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"createApiKey": map[string]any{
							"id":          "key-2",
							"name":        "Nightly",
							"apiKey":      "secret-key",
							"status":      "active",
							"permissions": []string{"send"},
							"expiresAt":   123.0,
							"createdAt":   100.0,
						},
					},
				}))
			},
			run: func(t *testing.T, client *GatewayClient) {
				key, err := client.CreateAPIKey(context.Background(), "app-1", "us", "Nightly", 30, "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "secret-key", key.APIKey)
			},
		},
		{
			name: "create api key omits expiresIn when zero",
			handler: func(t *testing.T, w http.ResponseWriter, _ *http.Request, body map[string]any) {
				variables := body["variables"].(map[string]any)
				options := variables["options"].(map[string]any)
				assert.Equal(t, "Default", options["name"])
				_, ok := options["expiresIn"]
				assert.False(t, ok)

				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"createApiKey": map[string]any{
							"id":          "key-3",
							"name":        "Default",
							"apiKey":      "key-default",
							"status":      "active",
							"permissions": []string{"send"},
							"expiresAt":   0.0,
							"createdAt":   100.0,
						},
					},
				}))
			},
			run: func(t *testing.T, client *GatewayClient) {
				key, err := client.CreateAPIKey(context.Background(), "app-1", "us", "Default", 0, "user-token", "org-token")
				require.NoError(t, err)
				assert.Equal(t, "key-default", key.APIKey)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				tt.handler(t, w, r, body)
			}))
			defer server.Close()

			origGatewayURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL")
			t.Cleanup(func() { setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_URL", origGatewayURL) })
			require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_URL", server.URL))

			client := NewGatewayClient(&mockDPoP{proof: "test-proof"})
			tt.run(t, client)
		})
	}
}

func TestGatewayClientHandlesGraphQLErrorsAndRedirects(t *testing.T) {
	t.Run("GraphQL error is surfaced", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]any{
					{
						"message": "top-level",
						"extensions": map[string]any{
							"message": "more specific",
						},
					},
				},
			}))
		}))
		defer server.Close()

		origGatewayURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL")
		t.Cleanup(func() { setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_URL", origGatewayURL) })
		require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_URL", server.URL))

		client := NewGatewayClient(&mockDPoP{proof: "test-proof"})
		apps, err := client.ListApplications(context.Background(), "org-1", "us", "user-token", "org-token")
		require.Error(t, err)
		assert.Nil(t, apps)
		assert.Contains(t, err.Error(), "failed to list applications")
		assert.Contains(t, err.Error(), "GraphQL error: more specific")
	})

	t.Run("GraphQL invalid session preserves structured auth error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]any{
					{
						"message": "session expired",
						"extensions": map[string]any{
							"code":    "INVALID_SESSION",
							"message": "Invalid or expired session",
						},
					},
				},
			}))
		}))
		defer server.Close()

		origGatewayURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL")
		t.Cleanup(func() { setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_URL", origGatewayURL) })
		require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_URL", server.URL))

		client := NewGatewayClient(&mockDPoP{proof: "test-proof"})
		apps, err := client.ListApplications(context.Background(), "org-1", "us", "user-token", "org-token")
		require.Error(t, err)
		assert.Nil(t, apps)
		assert.ErrorIs(t, err, domain.ErrDashboardSessionExpired)
		assert.Contains(t, err.Error(), "INVALID_SESSION")
	})

	t.Run("HTTP 401 GraphQL invalid session preserves structured auth error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]any{
					{
						"message": "INVALID_SESSION",
						"extensions": map[string]any{
							"code": "UNAUTHENTICATED",
						},
					},
				},
			}))
		}))
		defer server.Close()

		origGatewayURL := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL")
		t.Cleanup(func() { setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_URL", origGatewayURL) })
		require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_URL", server.URL))

		client := NewGatewayClient(&mockDPoP{proof: "test-proof"})
		apps, err := client.ListApplications(context.Background(), "org-1", "us", "user-token", "org-token")
		require.Error(t, err)
		assert.Nil(t, apps)
		assert.ErrorIs(t, err, domain.ErrDashboardSessionExpired)
		assert.Contains(t, err.Error(), "INVALID_SESSION")
		assert.Contains(t, err.Error(), "Invalid or expired session")
	})

	t.Run("redirect is not followed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", "https://redirected.example.com")
			w.WriteHeader(http.StatusFound)
		}))
		defer server.Close()

		client := NewGatewayClient(&mockDPoP{proof: "test-proof"})
		_, err := client.doGraphQL(context.Background(), server.URL, "query { ok }", map[string]any{}, "user-token", "org-token")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server redirected to https://redirected.example.com")
	})
}

func TestGatewayURLPrecedence(t *testing.T) {
	origGlobal := os.Getenv("NYLAS_DASHBOARD_GATEWAY_URL")
	origUS := os.Getenv("NYLAS_DASHBOARD_GATEWAY_US_URL")
	origEU := os.Getenv("NYLAS_DASHBOARD_GATEWAY_EU_URL")
	t.Cleanup(func() {
		setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_URL", origGlobal)
		setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_US_URL", origUS)
		setEnvOrUnsetLocal("NYLAS_DASHBOARD_GATEWAY_EU_URL", origEU)
	})

	require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_URL", "https://global.example.com/graphql"))
	require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_US_URL", "https://us.example.com/graphql"))
	require.NoError(t, os.Setenv("NYLAS_DASHBOARD_GATEWAY_EU_URL", "https://eu.example.com/graphql"))

	assert.Equal(t, "https://us.example.com/graphql", gatewayURL("us"))
	assert.Equal(t, "https://eu.example.com/graphql", gatewayURL("eu"))

	require.NoError(t, os.Unsetenv("NYLAS_DASHBOARD_GATEWAY_US_URL"))
	require.NoError(t, os.Unsetenv("NYLAS_DASHBOARD_GATEWAY_EU_URL"))

	assert.Equal(t, "https://global.example.com/graphql", gatewayURL("us"))
	assert.Equal(t, "https://global.example.com/graphql", gatewayURL("eu"))
}

func TestFormatGraphQLError(t *testing.T) {
	assert.Equal(t, "specific", formatGraphQLError(graphQLError{
		Message: "generic",
		Extensions: &graphQLExtensions{
			Message: "specific",
		},
	}))
	assert.Equal(t, "generic", formatGraphQLError(graphQLError{Message: "generic"}))
}

func setEnvOrUnsetLocal(key, value string) {
	if value == "" {
		_ = os.Unsetenv(key)
		return
	}
	_ = os.Setenv(key, value)
}
