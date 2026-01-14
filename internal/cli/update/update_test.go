package update

import (
	"runtime"
	"testing"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.3.5", "v0.3.5"},
		{"v0.3.5", "v0.3.5"},
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{" v0.3.5 ", "v0.3.5"},
		{" 0.3.5 ", "v0.3.5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeVersion(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v0.3.5", "0.3.5"},
		{"0.3.5", "0.3.5"},
		{"v1.0.0", "1.0.0"},
		{" v0.3.5 ", "0.3.5"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersion(tt.input)
			if got != tt.expected {
				t.Errorf("parseVersion(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsUpdateAvailable(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"update available", "0.3.4", "0.3.5", true},
		{"same version", "0.3.5", "0.3.5", false},
		{"newer than latest", "0.3.6", "0.3.5", false},
		{"dev version", "dev", "0.3.5", true},
		{"empty version", "", "0.3.5", true},
		{"major update", "0.3.5", "1.0.0", true},
		{"minor update", "0.3.5", "0.4.0", true},
		{"with v prefix current", "v0.3.4", "0.3.5", true},
		{"with v prefix latest", "0.3.4", "v0.3.5", true},
		{"both with v prefix", "v0.3.4", "v0.3.5", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUpdateAvailable(tt.current, tt.latest)
			if got != tt.expected {
				t.Errorf("isUpdateAvailable(%q, %q) = %v, want %v", tt.current, tt.latest, got, tt.expected)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected int
	}{
		{"older", "0.3.4", "0.3.5", -1},
		{"same", "0.3.5", "0.3.5", 0},
		{"newer", "0.3.6", "0.3.5", 1},
		{"dev", "dev", "0.3.5", -1},
		{"empty", "", "0.3.5", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareVersions(tt.current, tt.latest)
			if got != tt.expected {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.expected)
			}
		})
	}
}

func TestGetAssetName(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"0.3.5", "nylas_0.3.5_" + testGOOS() + "_" + testGOARCH() + testExt()},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := getAssetName(tt.version)
			if got != tt.expected {
				t.Errorf("getAssetName(%q) = %q, want %q", tt.version, got, tt.expected)
			}
		})
	}
}

// Helper functions for platform-specific expected values
func testGOOS() string {
	return runtime.GOOS
}

func testGOARCH() string {
	return runtime.GOARCH
}

func testExt() string {
	if testGOOS() == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}

func TestFindAsset(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "nylas_0.3.5_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin"},
			{Name: "nylas_0.3.5_linux_amd64.tar.gz", BrowserDownloadURL: "https://example.com/linux"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
		},
	}

	tests := []struct {
		name      string
		assetName string
		wantNil   bool
		wantURL   string
	}{
		{"find darwin", "nylas_0.3.5_darwin_arm64.tar.gz", false, "https://example.com/darwin"},
		{"find linux", "nylas_0.3.5_linux_amd64.tar.gz", false, "https://example.com/linux"},
		{"not found", "nylas_0.3.5_windows_amd64.zip", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findAsset(release, tt.assetName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("findAsset(%q) = %v, want nil", tt.assetName, got)
				}
			} else {
				if got == nil {
					t.Errorf("findAsset(%q) = nil, want non-nil", tt.assetName)
				} else if got.BrowserDownloadURL != tt.wantURL {
					t.Errorf("findAsset(%q).BrowserDownloadURL = %q, want %q", tt.assetName, got.BrowserDownloadURL, tt.wantURL)
				}
			}
		})
	}
}

func TestFindChecksumAsset(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "nylas_0.3.5_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/darwin"},
			{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums"},
		},
	}

	got := findChecksumAsset(release)
	if got == nil {
		t.Error("findChecksumAsset() = nil, want non-nil")
	} else if got.Name != "checksums.txt" {
		t.Errorf("findChecksumAsset().Name = %q, want %q", got.Name, "checksums.txt")
	}

	// Test without checksums
	releaseNoChecksum := &Release{
		Assets: []Asset{
			{Name: "nylas_0.3.5_darwin_arm64.tar.gz"},
		},
	}

	got = findChecksumAsset(releaseNoChecksum)
	if got != nil {
		t.Errorf("findChecksumAsset() = %v, want nil", got)
	}
}
