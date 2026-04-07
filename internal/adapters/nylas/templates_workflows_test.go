package nylas

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func newTemplatesWorkflowTestClient(t *testing.T, handler http.HandlerFunc) (*HTTPClient, func()) {
	t.Helper()

	server := httptest.NewServer(handler)
	client := NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("", "", "test-api-key")

	return client, server.Close
}

func TestListRemoteTemplates(t *testing.T) {
	t.Run("application_scope", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/v3/templates" {
				t.Fatalf("path = %s, want /v3/templates", r.URL.Path)
			}
			if got := r.URL.Query().Get("limit"); got != "25" {
				t.Fatalf("limit = %q, want 25", got)
			}
			if got := r.URL.Query().Get("page_token"); got != "cursor-123" {
				t.Fatalf("page_token = %q, want cursor-123", got)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [{
					"id": "tpl-1",
					"engine": "mustache",
					"name": "Welcome",
					"subject": "Welcome {{user.name}}",
					"body": "<p>Hello {{user.name}}</p>",
					"created_at": 1700000000,
					"updated_at": 1700000100,
					"object": "template"
				}],
				"next_cursor": "cursor-456",
				"request_id": "req-1"
			}`))
		})
		defer cleanup()

		resp, err := client.ListRemoteTemplates(context.Background(), domain.ScopeApplication, "", &domain.CursorListParams{
			Limit:     25,
			PageToken: "cursor-123",
		})
		if err != nil {
			t.Fatalf("ListRemoteTemplates() error = %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("len(resp.Data) = %d, want 1", len(resp.Data))
		}
		if resp.Data[0].ID != "tpl-1" {
			t.Fatalf("template id = %q, want tpl-1", resp.Data[0].ID)
		}
		if resp.NextCursor != "cursor-456" {
			t.Fatalf("next_cursor = %q, want cursor-456", resp.NextCursor)
		}
	})

	t.Run("grant_scope_escapes_grant_id", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/grants/nyla@example.com/templates" {
				t.Fatalf("path = %s, want grant-scoped path", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[]}`))
		})
		defer cleanup()

		if _, err := client.ListRemoteTemplates(context.Background(), domain.ScopeGrant, "nyla@example.com", nil); err != nil {
			t.Fatalf("ListRemoteTemplates() error = %v", err)
		}
	})
}

func TestGetRemoteTemplate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/templates/tpl-123" {
				t.Fatalf("path = %s, want /v3/templates/tpl-123", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"tpl-123","engine":"mustache","name":"Welcome","subject":"Hi","body":"<p>Hello</p>"}}`))
		})
		defer cleanup()

		template, err := client.GetRemoteTemplate(context.Background(), domain.ScopeApplication, "", "tpl-123")
		if err != nil {
			t.Fatalf("GetRemoteTemplate() error = %v", err)
		}
		if template.ID != "tpl-123" {
			t.Fatalf("template.ID = %q, want tpl-123", template.ID)
		}
	})

	t.Run("not_found", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"error":{"message":"not found"}}`, http.StatusNotFound)
		})
		defer cleanup()

		_, err := client.GetRemoteTemplate(context.Background(), domain.ScopeApplication, "", "missing")
		if !errors.Is(err, domain.ErrTemplateNotFound) {
			t.Fatalf("GetRemoteTemplate() error = %v, want ErrTemplateNotFound", err)
		}
	})
}

func TestCreateAndUpdateRemoteTemplate(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
				t.Fatalf("content-type = %q, want json", ct)
			}

			var req domain.CreateRemoteTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.Name != "Booking confirmed message" {
				t.Fatalf("req.Name = %q, want Booking confirmed message", req.Name)
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"tpl-new","engine":"mustache","name":"Booking confirmed message","subject":"Confirmed","body":"<p>Hello</p>"}}`))
		})
		defer cleanup()

		template, err := client.CreateRemoteTemplate(context.Background(), domain.ScopeApplication, "", &domain.CreateRemoteTemplateRequest{
			Name:    "Booking confirmed message",
			Subject: "Confirmed",
			Body:    "<p>Hello</p>",
			Engine:  "mustache",
		})
		if err != nil {
			t.Fatalf("CreateRemoteTemplate() error = %v", err)
		}
		if template.ID != "tpl-new" {
			t.Fatalf("template.ID = %q, want tpl-new", template.ID)
		}
	})

	t.Run("update", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Fatalf("method = %s, want PUT", r.Method)
			}
			if r.URL.Path != "/v3/grants/grant-123/templates/tpl-123" {
				t.Fatalf("path = %s, want grant-scoped update path", r.URL.Path)
			}

			var req domain.UpdateRemoteTemplateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.Name == nil || *req.Name != "Updated" {
				t.Fatalf("updated name not sent correctly")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"id":"tpl-123","engine":"mustache","name":"Updated","subject":"Confirmed","body":"<p>Hello</p>"}}`))
		})
		defer cleanup()

		name := "Updated"
		template, err := client.UpdateRemoteTemplate(context.Background(), domain.ScopeGrant, "grant-123", "tpl-123", &domain.UpdateRemoteTemplateRequest{
			Name: &name,
		})
		if err != nil {
			t.Fatalf("UpdateRemoteTemplate() error = %v", err)
		}
		if template.Name != "Updated" {
			t.Fatalf("template.Name = %q, want Updated", template.Name)
		}
	})
}

func TestDeleteAndRenderRemoteTemplate(t *testing.T) {
	t.Run("delete", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				t.Fatalf("method = %s, want DELETE", r.Method)
			}
			if r.URL.Path != "/v3/templates/tpl-123" {
				t.Fatalf("path = %s, want /v3/templates/tpl-123", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		defer cleanup()

		if err := client.DeleteRemoteTemplate(context.Background(), domain.ScopeApplication, "", "tpl-123"); err != nil {
			t.Fatalf("DeleteRemoteTemplate() error = %v", err)
		}
	})

	t.Run("render_template", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/templates/tpl-123/render" {
				t.Fatalf("path = %s, want render path", r.URL.Path)
			}

			var req domain.TemplateRenderRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req.Variables["user"] == nil {
				t.Fatalf("variables missing from render request")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"subject":"Welcome Nylas","body":"<p>Hello Nylas</p>"}}`))
		})
		defer cleanup()

		result, err := client.RenderRemoteTemplate(context.Background(), domain.ScopeApplication, "", "tpl-123", &domain.TemplateRenderRequest{
			Variables: map[string]any{"user": map[string]any{"name": "Nylas"}},
		})
		if err != nil {
			t.Fatalf("RenderRemoteTemplate() error = %v", err)
		}
		if result["subject"] != "Welcome Nylas" {
			t.Fatalf("result[subject] = %v, want Welcome Nylas", result["subject"])
		}
	})

	t.Run("render_html", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/grants/grant-123/templates/render" {
				t.Fatalf("path = %s, want grant-scoped html render path", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"html":"<p>Hello Nylas</p>"}}`))
		})
		defer cleanup()

		result, err := client.RenderRemoteTemplateHTML(context.Background(), domain.ScopeGrant, "grant-123", &domain.TemplateRenderHTMLRequest{
			Body:   "<p>Hello {{user.name}}</p>",
			Engine: "mustache",
		})
		if err != nil {
			t.Fatalf("RenderRemoteTemplateHTML() error = %v", err)
		}
		if result["html"] != "<p>Hello Nylas</p>" {
			t.Fatalf("result[html] = %v, want rendered html", result["html"])
		}
	})
}

func TestWorkflowOperations(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/workflows" {
				t.Fatalf("path = %s, want /v3/workflows", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"data": [{
					"id": "wf-1",
					"grant_id": "grant-123",
					"is_enabled": true,
					"name": "Booking Confirmation",
					"trigger_event": "booking.created",
					"delay": 1,
					"template_id": "tpl-123",
					"date_created": 1700000000
				}],
				"next_cursor": "cursor-123"
			}`))
		})
		defer cleanup()

		resp, err := client.ListWorkflows(context.Background(), domain.ScopeApplication, "", &domain.CursorListParams{Limit: 10})
		if err != nil {
			t.Fatalf("ListWorkflows() error = %v", err)
		}
		if len(resp.Data) != 1 {
			t.Fatalf("len(resp.Data) = %d, want 1", len(resp.Data))
		}
		if resp.Data[0].TemplateID != "tpl-123" {
			t.Fatalf("template_id = %q, want tpl-123", resp.Data[0].TemplateID)
		}
	})

	t.Run("create_update_get_delete", func(t *testing.T) {
		step := 0
		client, cleanup := newTemplatesWorkflowTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch step {
			case 0:
				if r.Method != http.MethodPost || r.URL.Path != "/v3/grants/grant-123/workflows" {
					t.Fatalf("unexpected create request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"data":{"id":"wf-new","is_enabled":true,"name":"Booking Confirmation","trigger_event":"booking.created","delay":1,"template_id":"tpl-123","date_created":1700000000}}`))
			case 1:
				if r.Method != http.MethodPut || r.URL.Path != "/v3/grants/grant-123/workflows/wf-new" {
					t.Fatalf("unexpected update request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"data":{"id":"wf-new","is_enabled":false,"name":"Updated Workflow","trigger_event":"booking.created","delay":5,"template_id":"tpl-123","date_created":1700000000}}`))
			case 2:
				if r.Method != http.MethodGet || r.URL.Path != "/v3/grants/grant-123/workflows/wf-new" {
					t.Fatalf("unexpected get request: %s %s", r.Method, r.URL.Path)
				}
				_, _ = w.Write([]byte(`{"data":{"id":"wf-new","is_enabled":false,"name":"Updated Workflow","trigger_event":"booking.created","delay":5,"template_id":"tpl-123","date_created":1700000000}}`))
			case 3:
				if r.Method != http.MethodDelete || r.URL.Path != "/v3/grants/grant-123/workflows/wf-new" {
					t.Fatalf("unexpected delete request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				t.Fatalf("unexpected request step %d", step)
			}
			step++
		})
		defer cleanup()

		enabled := true
		created, err := client.CreateWorkflow(context.Background(), domain.ScopeGrant, "grant-123", &domain.CreateRemoteWorkflowRequest{
			Name:         "Booking Confirmation",
			TriggerEvent: "booking.created",
			TemplateID:   "tpl-123",
			Delay:        1,
			IsEnabled:    &enabled,
		})
		if err != nil {
			t.Fatalf("CreateWorkflow() error = %v", err)
		}
		if created.ID != "wf-new" {
			t.Fatalf("created.ID = %q, want wf-new", created.ID)
		}

		updatedName := "Updated Workflow"
		updatedDelay := 5
		disabled := false
		updated, err := client.UpdateWorkflow(context.Background(), domain.ScopeGrant, "grant-123", "wf-new", &domain.UpdateRemoteWorkflowRequest{
			Name:      &updatedName,
			Delay:     &updatedDelay,
			IsEnabled: &disabled,
		})
		if err != nil {
			t.Fatalf("UpdateWorkflow() error = %v", err)
		}
		if updated.Name != "Updated Workflow" {
			t.Fatalf("updated.Name = %q, want Updated Workflow", updated.Name)
		}

		fetched, err := client.GetWorkflow(context.Background(), domain.ScopeGrant, "grant-123", "wf-new")
		if err != nil {
			t.Fatalf("GetWorkflow() error = %v", err)
		}
		if fetched.Delay != 5 {
			t.Fatalf("fetched.Delay = %d, want 5", fetched.Delay)
		}

		if err := client.DeleteWorkflow(context.Background(), domain.ScopeGrant, "grant-123", "wf-new"); err != nil {
			t.Fatalf("DeleteWorkflow() error = %v", err)
		}
	})
}
