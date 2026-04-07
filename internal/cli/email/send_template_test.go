package email

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestValidateHostedTemplateSendOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    hostedTemplateSendOptions
		subject string
		body    string
		wantErr bool
	}{
		{
			name:    "no hosted template flags",
			opts:    hostedTemplateSendOptions{},
			subject: "Hello",
			body:    "World",
		},
		{
			name:    "template with inline content is rejected",
			opts:    hostedTemplateSendOptions{TemplateID: "tpl-123"},
			subject: "Hello",
			wantErr: true,
		},
		{
			name:    "render only requires template id",
			opts:    hostedTemplateSendOptions{RenderOnly: true},
			wantErr: true,
		},
		{
			name:    "template data requires template id",
			opts:    hostedTemplateSendOptions{TemplateData: `{"name":"Ada"}`},
			wantErr: true,
		},
		{
			name:    "grant scope without template id is rejected",
			opts:    hostedTemplateSendOptions{TemplateScope: string(domain.ScopeGrant)},
			wantErr: true,
		},
		{
			name: "valid hosted template request",
			opts: hostedTemplateSendOptions{
				TemplateID:    "tpl-123",
				TemplateScope: string(domain.ScopeApplication),
				TemplateData:  `{"user":{"name":"Ada"}}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHostedTemplateSendOptions(tt.opts, tt.subject, tt.body)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateHostedTemplateSendOptions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExtractRenderedEmailContent(t *testing.T) {
	tests := []struct {
		name        string
		result      domain.TemplateRenderResult
		wantSubject string
		wantBody    string
		wantErr     bool
	}{
		{
			name: "subject and body",
			result: domain.TemplateRenderResult{
				"subject": "Hello Ada",
				"body":    "<p>Hello Ada</p>",
			},
			wantSubject: "Hello Ada",
			wantBody:    "<p>Hello Ada</p>",
		},
		{
			name: "html fallback",
			result: domain.TemplateRenderResult{
				"subject": "Hello Ada",
				"html":    "<p>Hello Ada</p>",
			},
			wantSubject: "Hello Ada",
			wantBody:    "<p>Hello Ada</p>",
		},
		{
			name: "missing subject",
			result: domain.TemplateRenderResult{
				"body": "<p>Hello Ada</p>",
			},
			wantErr: true,
		},
		{
			name: "missing body",
			result: domain.TemplateRenderResult{
				"subject": "Hello Ada",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subject, body, err := extractRenderedEmailContent(tt.result)
			if (err != nil) != tt.wantErr {
				t.Fatalf("extractRenderedEmailContent() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if subject != tt.wantSubject {
				t.Fatalf("subject = %q, want %q", subject, tt.wantSubject)
			}
			if body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}

func TestRenderHostedTemplateForSend(t *testing.T) {
	t.Run("defaults to app scope", func(t *testing.T) {
		mockClient := &nylas.MockClient{
			RenderRemoteTemplateFunc: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
				if scope != domain.ScopeApplication {
					t.Fatalf("scope = %q, want %q", scope, domain.ScopeApplication)
				}
				if grantID != "" {
					t.Fatalf("grantID = %q, want empty", grantID)
				}
				if templateID != "tpl-app" {
					t.Fatalf("templateID = %q, want %q", templateID, "tpl-app")
				}
				return domain.TemplateRenderResult{
					"subject": "Hello Ada",
					"body":    "<p>Hello Ada</p>",
				}, nil
			},
		}

		rendered, err := renderHostedTemplateForSend(context.Background(), mockClient, "", hostedTemplateSendOptions{
			TemplateID:   "tpl-app",
			TemplateData: `{"user":{"name":"Ada"}}`,
			Strict:       true,
		})
		if err != nil {
			t.Fatalf("renderHostedTemplateForSend() error = %v", err)
		}
		if rendered == nil {
			t.Fatal("renderHostedTemplateForSend() returned nil result")
			return
		}
		if rendered.Subject != "Hello Ada" {
			t.Fatalf("subject = %q, want %q", rendered.Subject, "Hello Ada")
		}
		if rendered.Body != "<p>Hello Ada</p>" {
			t.Fatalf("body = %q, want %q", rendered.Body, "<p>Hello Ada</p>")
		}
	})

	t.Run("grant scope falls back to send grant", func(t *testing.T) {
		mockClient := &nylas.MockClient{
			RenderRemoteTemplateFunc: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
				if scope != domain.ScopeGrant {
					t.Fatalf("scope = %q, want %q", scope, domain.ScopeGrant)
				}
				if grantID != "grant-send" {
					t.Fatalf("grantID = %q, want %q", grantID, "grant-send")
				}
				return domain.TemplateRenderResult{
					"subject": "Grant Hello",
					"body":    "<p>Grant Hello</p>",
				}, nil
			},
		}

		rendered, err := renderHostedTemplateForSend(context.Background(), mockClient, "grant-send", hostedTemplateSendOptions{
			TemplateID:    "tpl-grant",
			TemplateScope: string(domain.ScopeGrant),
			Strict:        true,
		})
		if err != nil {
			t.Fatalf("renderHostedTemplateForSend() error = %v", err)
		}
		if rendered == nil {
			t.Fatal("renderHostedTemplateForSend() returned nil result")
			return
		}
		if rendered.GrantID != "grant-send" {
			t.Fatalf("GrantID = %q, want %q", rendered.GrantID, "grant-send")
		}
	})

	t.Run("grant scope resolves template grant email", func(t *testing.T) {
		configDir := filepath.Join(t.TempDir(), "nylas")
		t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))
		t.Setenv("HOME", t.TempDir())
		t.Setenv("NYLAS_DISABLE_KEYRING", "true")
		t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
		t.Setenv("NYLAS_API_KEY", "")
		t.Setenv("NYLAS_GRANT_ID", "")

		store, err := keyring.NewEncryptedFileStore(configDir)
		if err != nil {
			t.Fatalf("NewEncryptedFileStore() error = %v", err)
		}
		grantStore := keyring.NewGrantStore(store)
		if err := grantStore.SaveGrant(domain.GrantInfo{
			ID:    "grant-email-id",
			Email: "lookup@example.com",
		}); err != nil {
			t.Fatalf("SaveGrant() error = %v", err)
		}

		mockClient := &nylas.MockClient{
			RenderRemoteTemplateFunc: func(ctx context.Context, scope domain.RemoteScope, grantID, templateID string, req *domain.TemplateRenderRequest) (domain.TemplateRenderResult, error) {
				if scope != domain.ScopeGrant {
					t.Fatalf("scope = %q, want %q", scope, domain.ScopeGrant)
				}
				if grantID != "grant-email-id" {
					t.Fatalf("grantID = %q, want %q", grantID, "grant-email-id")
				}
				return domain.TemplateRenderResult{
					"subject": "Grant Hello",
					"body":    "<p>Grant Hello</p>",
				}, nil
			},
		}

		rendered, err := renderHostedTemplateForSend(context.Background(), mockClient, "", hostedTemplateSendOptions{
			TemplateID:      "tpl-grant",
			TemplateScope:   string(domain.ScopeGrant),
			TemplateGrantID: "lookup@example.com",
			Strict:          true,
		})
		if err != nil {
			t.Fatalf("renderHostedTemplateForSend() error = %v", err)
		}
		if rendered == nil {
			t.Fatal("renderHostedTemplateForSend() returned nil result")
			return
		}
		if rendered.GrantID != "grant-email-id" {
			t.Fatalf("GrantID = %q, want %q", rendered.GrantID, "grant-email-id")
		}
	})
}

func TestHostedTemplateSendNeedsGrant(t *testing.T) {
	tests := []struct {
		name    string
		opts    hostedTemplateSendOptions
		want    bool
		wantErr bool
	}{
		{
			name: "non render send needs grant",
			opts: hostedTemplateSendOptions{TemplateID: "tpl-app"},
			want: true,
		},
		{
			name: "render only app scope does not need grant",
			opts: hostedTemplateSendOptions{TemplateID: "tpl-app", RenderOnly: true},
			want: false,
		},
		{
			name: "render only grant scope with explicit template grant does not need send grant",
			opts: hostedTemplateSendOptions{
				TemplateID:      "tpl-grant",
				TemplateScope:   string(domain.ScopeGrant),
				TemplateGrantID: "grant@example.com",
				RenderOnly:      true,
			},
			want: false,
		},
		{
			name: "render only grant scope without template grant needs send grant",
			opts: hostedTemplateSendOptions{
				TemplateID:    "tpl-grant",
				TemplateScope: string(domain.ScopeGrant),
				RenderOnly:    true,
			},
			want: true,
		},
		{
			name: "invalid scope returns error",
			opts: hostedTemplateSendOptions{
				TemplateID:    "tpl-invalid",
				TemplateScope: "invalid",
				RenderOnly:    true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hostedTemplateSendNeedsGrant(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("hostedTemplateSendNeedsGrant() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("hostedTemplateSendNeedsGrant() = %v, want %v", got, tt.want)
			}
		})
	}
}
