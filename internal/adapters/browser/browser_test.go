package browser

import (
	"errors"
	"testing"
)

func TestNewDefaultBrowser(t *testing.T) {
	browser := NewDefaultBrowser()
	if browser == nil {
		t.Error("NewDefaultBrowser() returned nil")
	}
}

func TestDefaultBrowser_Open(t *testing.T) {
	// Skip to avoid opening actual browser during automated test runs
	// This test is for manual verification only
	t.Skip("skipping browser open test - use manual testing to verify browser functionality")

	browser := NewDefaultBrowser()

	// We can't actually test opening a URL without a browser
	// Just test that it doesn't panic and returns a result
	err := browser.Open("https://example.com")
	// Error is OK (no browser available), we just want to ensure it doesn't panic
	_ = err
}

func TestNewMockBrowser(t *testing.T) {
	mock := NewMockBrowser()
	if mock == nil {
		t.Error("NewMockBrowser() returned nil")
		return
	}
	if mock.OpenCalled {
		t.Error("OpenCalled should be false initially")
	}
	if mock.LastURL != "" {
		t.Error("LastURL should be empty initially")
	}
}

func TestMockBrowser_Open(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		openFunc func(url string) error
		wantErr  bool
	}{
		{
			name:     "successful open",
			url:      "https://example.com",
			openFunc: nil,
			wantErr:  false,
		},
		{
			name: "open with error",
			url:  "https://example.com",
			openFunc: func(url string) error {
				return errors.New("failed to open")
			},
			wantErr: true,
		},
		{
			name: "open with custom func",
			url:  "https://custom.com",
			openFunc: func(url string) error {
				if url != "https://custom.com" {
					return errors.New("wrong URL")
				}
				return nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockBrowser()
			mock.OpenFunc = tt.openFunc

			err := mock.Open(tt.url)

			if !mock.OpenCalled {
				t.Error("OpenCalled should be true")
			}
			if mock.LastURL != tt.url {
				t.Errorf("LastURL = %q, want %q", mock.LastURL, tt.url)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Open() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockBrowser_RecordsMultipleCalls(t *testing.T) {
	mock := NewMockBrowser()

	_ = mock.Open("https://first.com")
	if mock.LastURL != "https://first.com" {
		t.Errorf("First URL = %q, want %q", mock.LastURL, "https://first.com")
	}

	_ = mock.Open("https://second.com")
	if mock.LastURL != "https://second.com" {
		t.Errorf("Second URL = %q, want %q", mock.LastURL, "https://second.com")
	}
}
