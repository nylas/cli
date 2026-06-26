package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeAuditService struct {
	ports.AuditStore

	getConfig  func() (*domain.AuditConfig, error)
	saveConfig func(*domain.AuditConfig) error
	list       func(context.Context, int) ([]domain.AuditEntry, error)
	query      func(context.Context, *domain.AuditQueryOptions) ([]domain.AuditEntry, error)
	summary    func(context.Context, int) (*domain.AuditSummary, error)
	clear      func(context.Context) error
	path       func() string
	stats      func() (int, int64, *domain.AuditEntry, error)
	cleanup    func(context.Context) error
}

func (f *fakeAuditService) GetConfig() (*domain.AuditConfig, error) {
	if f.getConfig == nil {
		return nil, errors.New("unexpected GetConfig")
	}
	return f.getConfig()
}

func (f *fakeAuditService) SaveConfig(cfg *domain.AuditConfig) error {
	if f.saveConfig == nil {
		return errors.New("unexpected SaveConfig")
	}
	return f.saveConfig(cfg)
}

func (f *fakeAuditService) List(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
	if f.list == nil {
		return nil, errors.New("unexpected List")
	}
	return f.list(ctx, limit)
}

func (f *fakeAuditService) Query(ctx context.Context, opts *domain.AuditQueryOptions) ([]domain.AuditEntry, error) {
	if f.query == nil {
		return nil, errors.New("unexpected Query")
	}
	return f.query(ctx, opts)
}

func (f *fakeAuditService) Summary(ctx context.Context, days int) (*domain.AuditSummary, error) {
	if f.summary == nil {
		return nil, errors.New("unexpected Summary")
	}
	return f.summary(ctx, days)
}

func (f *fakeAuditService) Clear(ctx context.Context) error {
	if f.clear == nil {
		return errors.New("unexpected Clear")
	}
	return f.clear(ctx)
}

func (f *fakeAuditService) Path() string {
	if f.path == nil {
		return ""
	}
	return f.path()
}

func (f *fakeAuditService) Stats() (int, int64, *domain.AuditEntry, error) {
	if f.stats == nil {
		return 0, 0, nil, errors.New("unexpected Stats")
	}
	return f.stats()
}

func (f *fakeAuditService) Cleanup(ctx context.Context) error {
	if f.cleanup == nil {
		return errors.New("unexpected Cleanup")
	}
	return f.cleanup(ctx)
}

type auditRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func TestRegisterAuditHandlers(t *testing.T) {
	now := time.Date(2026, 6, 25, 12, 30, 0, 0, time.UTC)
	entry := domain.AuditEntry{
		ID:        "audit-1",
		Timestamp: now,
		Command:   "email list",
		GrantID:   "grant-1",
		Status:    domain.AuditStatusSuccess,
	}

	tests := []struct {
		name   string
		method string
		params string
		svc    *fakeAuditService
		assert func(*testing.T, auditRPCResponse)
	}{
		{
			name:   "audit.list returns entries",
			method: "audit.list",
			params: `{"limit":2}`,
			svc: &fakeAuditService{
				list: func(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
					if limit != 2 {
						t.Fatalf("limit = %d, want 2", limit)
					}
					return []domain.AuditEntry{entry}, nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					Entries []domain.AuditEntry `json:"entries"`
				}
				unmarshalAuditResult(t, resp, &result)
				if len(result.Entries) != 1 || result.Entries[0].ID != "audit-1" {
					t.Fatalf("entries = %#v, want audit-1", result.Entries)
				}
			},
		},
		{
			name:   "audit.query forwards filters and returns entries",
			method: "audit.query",
			params: `{"limit":5,"since":"2026-06-01T00:00:00Z","until":"2026-06-25T00:00:00Z","command":"email list","status":"success","grant_id":"grant-1","request_id":"req-1","invoker":"ada","invoker_source":"terminal"}`,
			svc: &fakeAuditService{
				query: func(ctx context.Context, opts *domain.AuditQueryOptions) ([]domain.AuditEntry, error) {
					if opts.Limit != 5 || opts.Command != "email list" || opts.Status != "success" || opts.GrantID != "grant-1" || opts.RequestID != "req-1" || opts.Invoker != "ada" || opts.InvokerSource != "terminal" {
						t.Fatalf("opts = %#v, want decoded query filters", opts)
					}
					if opts.Since.IsZero() || opts.Until.IsZero() {
						t.Fatalf("opts times = %s %s, want decoded since/until", opts.Since, opts.Until)
					}
					return []domain.AuditEntry{entry}, nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					Entries []domain.AuditEntry `json:"entries"`
				}
				unmarshalAuditResult(t, resp, &result)
				if len(result.Entries) != 1 || result.Entries[0].Command != "email list" {
					t.Fatalf("entries = %#v, want email list", result.Entries)
				}
			},
		},
		{
			name:   "audit.summary returns summary",
			method: "audit.summary",
			params: `{"days":14}`,
			svc: &fakeAuditService{
				summary: func(ctx context.Context, days int) (*domain.AuditSummary, error) {
					if days != 14 {
						t.Fatalf("days = %d, want 14", days)
					}
					return &domain.AuditSummary{
						StartDate:       now.AddDate(0, 0, -14),
						EndDate:         now,
						Days:            14,
						TotalCommands:   3,
						SuccessCount:    2,
						ErrorCount:      1,
						SuccessPercent:  66.67,
						CommandCounts:   map[string]int{"email list": 3},
						AccountCounts:   map[string]int{"grant-1": 3},
						InvokerCounts:   map[string]int{"ada": 3},
						TotalAPICalls:   4,
						AvgResponseTime: time.Second,
						APIErrorRate:    25,
					}, nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result domain.AuditSummary
				unmarshalAuditResult(t, resp, &result)
				if result.Days != 14 || result.TotalCommands != 3 {
					t.Fatalf("summary = %#v, want 14 days and 3 commands", result)
				}
			},
		},
		{
			name:   "audit.stats returns file stats",
			method: "audit.stats",
			params: `{}`,
			svc: &fakeAuditService{
				stats: func() (int, int64, *domain.AuditEntry, error) {
					return 3, 4096, &entry, nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					FileCount      int                `json:"file_count"`
					TotalSizeBytes int64              `json:"total_size_bytes"`
					OldestEntry    *domain.AuditEntry `json:"oldest_entry"`
				}
				unmarshalAuditResult(t, resp, &result)
				if result.FileCount != 3 || result.TotalSizeBytes != 4096 || result.OldestEntry == nil || result.OldestEntry.ID != "audit-1" {
					t.Fatalf("stats = %#v, want file_count 3, total_size_bytes 4096, oldest audit-1", result)
				}
			},
		},
		{
			name:   "audit.config.read returns config",
			method: "audit.config.read",
			params: `{}`,
			svc: &fakeAuditService{
				getConfig: func() (*domain.AuditConfig, error) {
					return &domain.AuditConfig{
						Enabled:       true,
						Initialized:   true,
						Path:          "/tmp/audit",
						RetentionDays: 30,
						MaxSizeMB:     100,
						Format:        "jsonl",
						LogAPIDetails: true,
						LogRequestID:  true,
						RotateDaily:   true,
						CompressOld:   false,
					}, nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result domain.AuditConfig
				unmarshalAuditResult(t, resp, &result)
				if !result.Enabled || result.Path != "/tmp/audit" || result.RetentionDays != 30 {
					t.Fatalf("config = %#v, want enabled /tmp/audit retention 30", result)
				}
			},
		},
		{
			name:   "audit.config.save saves config",
			method: "audit.config.save",
			params: `{"enabled":true,"initialized":true,"path":"/tmp/audit","retention_days":45,"max_size_mb":200,"format":"jsonl","log_api_details":true,"log_request_id":true,"rotate_daily":true,"compress_old":true}`,
			svc: &fakeAuditService{
				saveConfig: func(cfg *domain.AuditConfig) error {
					if cfg == nil || !cfg.Enabled || cfg.Path != "/tmp/audit" || cfg.RetentionDays != 45 || cfg.MaxSizeMB != 200 || !cfg.CompressOld {
						t.Fatalf("cfg = %#v, want decoded audit config", cfg)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					OK bool `json:"ok"`
				}
				unmarshalAuditResult(t, resp, &result)
				if !result.OK {
					t.Fatal("ok = false, want true")
				}
			},
		},
		{
			name:   "audit.path returns path",
			method: "audit.path",
			params: `{}`,
			svc: &fakeAuditService{
				path: func() string {
					return "/tmp/audit"
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					Path string `json:"path"`
				}
				unmarshalAuditResult(t, resp, &result)
				if result.Path != "/tmp/audit" {
					t.Fatalf("path = %q, want /tmp/audit", result.Path)
				}
			},
		},
		{
			name:   "audit.clear clears logs",
			method: "audit.clear",
			params: `{}`,
			svc: &fakeAuditService{
				clear: func(ctx context.Context) error {
					return nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					Cleared bool `json:"cleared"`
				}
				unmarshalAuditResult(t, resp, &result)
				if !result.Cleared {
					t.Fatal("cleared = false, want true")
				}
			},
		},
		{
			name:   "audit.cleanup returns ok",
			method: "audit.cleanup",
			params: `{}`,
			svc: &fakeAuditService{
				cleanup: func(ctx context.Context) error {
					return nil
				},
			},
			assert: func(t *testing.T, resp auditRPCResponse) {
				requireNoAuditRPCError(t, resp)

				var result struct {
					OK bool `json:"ok"`
				}
				unmarshalAuditResult(t, resp, &result)
				if !result.OK {
					t.Fatal("ok = false, want true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterAuditHandlers(d, tt.svc)

			resp := dispatchAuditRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func TestRegisterAuditHandlers_ClientErrorMapsToInternalError(t *testing.T) {
	clientErr := errors.New("audit unavailable")
	d := NewDispatcher()
	var loggedErr error
	d.LogError = func(err error) {
		loggedErr = err
	}
	RegisterAuditHandlers(d, &fakeAuditService{
		list: func(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
			return nil, clientErr
		},
	})

	resp := dispatchAuditRequest(t, d, "audit.list", `{}`)
	requireAuditRPCErrorCode(t, resp, InternalError)
	if !errors.Is(loggedErr, clientErr) {
		t.Fatalf("logged error = %v, want wrapped %v", loggedErr, clientErr)
	}
}

func dispatchAuditRequest(t *testing.T, d *Dispatcher, method, params string) auditRPCResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp auditRPCResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}

func requireNoAuditRPCError(t *testing.T, resp auditRPCResponse) {
	t.Helper()

	if resp.Error != nil {
		t.Fatalf("Error = %#v, want nil", resp.Error)
	}
}

func requireAuditRPCErrorCode(t *testing.T, resp auditRPCResponse, want int) {
	t.Helper()

	if resp.Error == nil {
		t.Fatal("Error = nil, want RPC error")
	}
	if resp.Error.Code != want {
		t.Fatalf("Error.Code = %d, want %d", resp.Error.Code, want)
	}
}

func unmarshalAuditResult(t *testing.T, resp auditRPCResponse, dest any) {
	t.Helper()

	if err := json.Unmarshal(resp.Result, dest); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
}
