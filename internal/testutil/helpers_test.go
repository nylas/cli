package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestTempConfig(t *testing.T) {
	content := "test: value"
	path := TempConfig(t, content)

	// #nosec G304 -- reading test file created by test helper
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp config: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content = %q, want %q", string(data), content)
	}
}

func TestTempFile(t *testing.T) {
	content := "test content"
	path := TempFile(t, "test.txt", content)

	// #nosec G304 -- reading test file created by test helper
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != content {
		t.Errorf("Content = %q, want %q", string(data), content)
	}
}

func TestAssertEqual(t *testing.T) {
	// Test successful assertion (should not fail)
	AssertEqual(t, "hello", "hello", "strings should be equal")
	AssertEqual(t, 42, 42, "numbers should be equal")
	AssertEqual(t, true, true, "booleans should be equal")
}

func TestAssertContains(t *testing.T) {
	AssertContains(t, "hello world", "world", "should contain substring")
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{"empty substring", "hello", "", true},
		{"contains", "hello world", "world", true},
		{"does not contain", "hello", "xyz", false},
		{"exact match", "test", "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	t.Run("writes status and JSON body", func(t *testing.T) {
		data := map[string]string{"id": "123", "name": "test"}

		rec := httptest.NewRecorder()
		WriteJSONResponse(t, rec, http.StatusOK, data)

		resp := rec.Result()
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		var got map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if got["id"] != "123" {
			t.Errorf("id = %q, want %q", got["id"], "123")
		}
		if got["name"] != "test" {
			t.Errorf("name = %q, want %q", got["name"], "test")
		}
	})

	t.Run("writes custom status codes", func(t *testing.T) {
		rec := httptest.NewRecorder()
		WriteJSONResponse(t, rec, http.StatusCreated, map[string]bool{"ok": true})

		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
		}
	})

	t.Run("writes array data", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		rec := httptest.NewRecorder()
		WriteJSONResponse(t, rec, http.StatusOK, items)

		var got []string
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})

	t.Run("integrates with httptest server", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			WriteJSONResponse(t, w, http.StatusOK, map[string]string{"status": "ok"})
		}))
		defer server.Close()

		resp, err := http.Get(server.URL) //nolint:gosec // test URL
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var result map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			t.Fatalf("decode failed: %v", err)
		}
		if result["status"] != "ok" {
			t.Errorf("status = %q, want %q", result["status"], "ok")
		}
	})

	t.Run("returns 500 on encode failure", func(t *testing.T) {
		// json.Marshal cannot encode channels — use this to trigger failure.
		unencodable := make(chan int)

		// Use a fake *testing.T so we can observe the Errorf call without
		// failing the real test.
		fakeT := &testing.T{}

		rec := httptest.NewRecorder()
		WriteJSONResponse(fakeT, rec, http.StatusOK, unencodable)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		if !fakeT.Failed() {
			t.Error("expected fakeT to be marked as failed")
		}
	})
}
