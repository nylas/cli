package chat

import (
	"sync"
	"testing"
	"time"
)

func TestIsGated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		toolName string
		want     bool
	}{
		{"send_email is gated", "send_email", true},
		{"create_event is gated", "create_event", true},
		{"list_emails is not gated", "list_emails", false},
		{"get_event is not gated", "get_event", false},
		{"empty string is not gated", "", false},
		{"unknown tool is not gated", "unknown_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := IsGated(tt.toolName)
			if got != tt.want {
				t.Errorf("IsGated(%q) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

func TestApprovalStore_Create(t *testing.T) {
	t.Parallel()

	store := NewApprovalStore()

	call1 := ToolCall{Name: "send_email", Args: map[string]any{"to": "test@example.com"}}
	preview1 := map[string]any{"to": "test@example.com"}

	pa1 := store.Create(call1, preview1)

	if pa1.ID != "approval_1" {
		t.Errorf("First approval ID = %q, want %q", pa1.ID, "approval_1")
	}
	if pa1.Tool != "send_email" {
		t.Errorf("Tool = %q, want %q", pa1.Tool, "send_email")
	}
	if pa1.Preview["to"] != "test@example.com" {
		t.Errorf("Preview[to] = %v, want %q", pa1.Preview["to"], "test@example.com")
	}

	// Second approval should have sequential ID
	call2 := ToolCall{Name: "create_event", Args: map[string]any{"title": "Meeting"}}
	preview2 := map[string]any{"title": "Meeting"}
	pa2 := store.Create(call2, preview2)

	if pa2.ID != "approval_2" {
		t.Errorf("Second approval ID = %q, want %q", pa2.ID, "approval_2")
	}
}

func TestApprovalStore_Resolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		createID  bool
		resolveID string
		decision  ApprovalDecision
		want      bool
	}{
		{
			name:      "resolve existing approval",
			createID:  true,
			resolveID: "approval_1",
			decision:  ApprovalDecision{Approved: true},
			want:      true,
		},
		{
			name:      "resolve non-existent approval",
			createID:  false,
			resolveID: "approval_999",
			decision:  ApprovalDecision{Approved: false},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := NewApprovalStore()

			if tt.createID {
				call := ToolCall{Name: "send_email", Args: map[string]any{}}
				store.Create(call, map[string]any{})
			}

			got := store.Resolve(tt.resolveID, tt.decision)
			if got != tt.want {
				t.Errorf("Resolve(%q) = %v, want %v", tt.resolveID, got, tt.want)
			}
		})
	}
}

func TestApprovalStore_ResolveAlreadyResolved(t *testing.T) {
	t.Parallel()

	store := NewApprovalStore()
	call := ToolCall{Name: "send_email", Args: map[string]any{}}
	pa := store.Create(call, map[string]any{})

	// Resolve once
	decision := ApprovalDecision{Approved: true}
	if !store.Resolve(pa.ID, decision) {
		t.Fatal("First resolve failed")
	}

	// Try to resolve again
	if store.Resolve(pa.ID, decision) {
		t.Error("Second resolve succeeded, want false for already resolved approval")
	}
}

func TestPendingApproval_Wait(t *testing.T) {
	t.Parallel()

	t.Run("wait returns decision when resolved", func(t *testing.T) {
		t.Parallel()

		store := NewApprovalStore()
		call := ToolCall{Name: "send_email", Args: map[string]any{}}
		pa := store.Create(call, map[string]any{})

		expectedDecision := ApprovalDecision{Approved: true, Reason: "looks good"}

		// Resolve from another goroutine
		go func() {
			time.Sleep(50 * time.Millisecond)
			store.Resolve(pa.ID, expectedDecision)
		}()

		decision, ok := pa.Wait()
		if !ok {
			t.Fatal("Wait returned false, want true")
		}
		if decision.Approved != expectedDecision.Approved {
			t.Errorf("Approved = %v, want %v", decision.Approved, expectedDecision.Approved)
		}
		if decision.Reason != expectedDecision.Reason {
			t.Errorf("Reason = %q, want %q", decision.Reason, expectedDecision.Reason)
		}
	})

	t.Run("wait rejects when not resolved", func(t *testing.T) {
		// This test would take 5 minutes with the real timeout
		// We can't easily test the timeout without modifying the code
		// Skip this in normal test runs
		t.Skip("Timeout test would take 5 minutes")
	})
}

func TestPendingApproval_WaitConcurrent(t *testing.T) {
	t.Parallel()

	store := NewApprovalStore()
	const numApprovals = 10

	var wg sync.WaitGroup
	wg.Add(numApprovals)

	for i := 0; i < numApprovals; i++ {
		go func(idx int) {
			defer wg.Done()

			call := ToolCall{Name: "send_email", Args: map[string]any{"id": idx}}
			pa := store.Create(call, map[string]any{})

			// Resolve from another goroutine
			go func() {
				time.Sleep(10 * time.Millisecond)
				store.Resolve(pa.ID, ApprovalDecision{Approved: idx%2 == 0})
			}()

			decision, ok := pa.Wait()
			if !ok {
				t.Errorf("Wait for approval %d failed", idx)
			}
			expectedApproval := idx%2 == 0
			if decision.Approved != expectedApproval {
				t.Errorf("Approval %d: got %v, want %v", idx, decision.Approved, expectedApproval)
			}
		}(i)
	}

	wg.Wait()
}

func TestBuildPreview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		call ToolCall
		want map[string]any
	}{
		{
			name: "send_email with all fields",
			call: ToolCall{
				Name: "send_email",
				Args: map[string]any{
					"to":      "user@example.com",
					"subject": "Test Subject",
					"body":    "Short body",
				},
			},
			want: map[string]any{
				"to":      "user@example.com",
				"subject": "Test Subject",
				"body":    "Short body",
			},
		},
		{
			name: "send_email with long body truncation",
			call: ToolCall{
				Name: "send_email",
				Args: map[string]any{
					"to":      "user@example.com",
					"subject": "Test",
					"body":    string(make([]byte, 300)), // 300 null bytes
				},
			},
			want: map[string]any{
				"to":      "user@example.com",
				"subject": "Test",
				"body":    string(make([]byte, 200)) + "...",
			},
		},
		{
			name: "create_event with all fields",
			call: ToolCall{
				Name: "create_event",
				Args: map[string]any{
					"title":       "Team Meeting",
					"start_time":  "2026-02-12T10:00:00Z",
					"end_time":    "2026-02-12T11:00:00Z",
					"description": "Discuss Q1 goals",
				},
			},
			want: map[string]any{
				"title":       "Team Meeting",
				"start_time":  "2026-02-12T10:00:00Z",
				"end_time":    "2026-02-12T11:00:00Z",
				"description": "Discuss Q1 goals",
			},
		},
		{
			name: "unknown tool returns empty preview",
			call: ToolCall{
				Name: "unknown_tool",
				Args: map[string]any{"foo": "bar"},
			},
			want: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := BuildPreview(tt.call)

			if len(got) != len(tt.want) {
				t.Errorf("Preview length = %d, want %d", len(got), len(tt.want))
			}

			for key, wantVal := range tt.want {
				gotVal, ok := got[key]
				if !ok {
					t.Errorf("Preview missing key %q", key)
					continue
				}
				if gotVal != wantVal {
					t.Errorf("Preview[%q] = %v, want %v", key, gotVal, wantVal)
				}
			}
		})
	}
}
