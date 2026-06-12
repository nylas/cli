package calendar

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetLocalTimeZone(t *testing.T) {
	tz := getLocalTimeZone()

	if tz == "" {
		t.Error("Expected non-empty timezone, got empty string")
	}

	// Verify it's a valid timezone
	_, err := time.LoadLocation(tz)
	if err != nil {
		t.Errorf("getLocalTimeZone() returned invalid timezone %q: %v", tz, err)
	}
}

func TestValidateTimeZone(t *testing.T) {
	tests := []struct {
		name    string
		tz      string
		wantErr bool
	}{
		{
			name:    "valid IANA timezone",
			tz:      "America/New_York",
			wantErr: false,
		},
		{
			name:    "valid UTC",
			tz:      "UTC",
			wantErr: false,
		},
		{
			name:    "valid Europe timezone",
			tz:      "Europe/London",
			wantErr: false,
		},
		{
			name:    "valid Asia timezone",
			tz:      "Asia/Tokyo",
			wantErr: false,
		},
		{
			name:    "invalid timezone",
			tz:      "Invalid/Timezone",
			wantErr: true,
		},
		{
			name:    "timezone abbreviation (not IANA)",
			tz:      "PST",
			wantErr: true,
		},
		{
			name:    "empty timezone",
			tz:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeZone(tt.tz)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateTimeZone(%q) error = %v, wantErr %v", tt.tz, err, tt.wantErr)
			}
		})
	}
}

func TestGetSystemTimeZone(t *testing.T) {
	tz := getSystemTimeZone()

	// Should return a valid IANA timezone or UTC
	if tz == "" {
		t.Error("Expected non-empty timezone")
	}

	// Verify it's loadable
	_, err := time.LoadLocation(tz)
	if err != nil {
		t.Errorf("getSystemTimeZone() returned invalid timezone %q: %v", tz, err)
	}
}

func TestGetSystemTimeZone_RespectsTZEnv(t *testing.T) {
	// Arizona does not observe DST and is not reachable via the UTC-offset
	// heuristic, so this only passes if TZ is read directly.
	t.Setenv("TZ", "America/Phoenix")

	if got := getSystemTimeZone(); got != "America/Phoenix" {
		t.Errorf("getSystemTimeZone() = %q, want %q", got, "America/Phoenix")
	}
}

func TestGetSystemTimeZone_IgnoresInvalidTZEnv(t *testing.T) {
	t.Setenv("TZ", "Not/AZone")

	tz := getSystemTimeZone()
	if tz == "Not/AZone" {
		t.Error("getSystemTimeZone() returned the invalid TZ value verbatim")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		t.Errorf("getSystemTimeZone() returned invalid timezone %q: %v", tz, err)
	}
}

func TestZoneFromZoneinfoPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "linux zoneinfo path",
			path: "/usr/share/zoneinfo/America/New_York",
			want: "America/New_York",
		},
		{
			name: "macOS zoneinfo path",
			path: "/var/db/timezone/zoneinfo/Asia/Tokyo",
			want: "Asia/Tokyo",
		},
		{
			name: "single-segment zone",
			path: "/usr/share/zoneinfo/UTC",
			want: "UTC",
		},
		{
			name: "no zoneinfo marker",
			path: "/etc/localtime",
			want: "",
		},
		{
			name: "invalid zone after marker",
			path: "/usr/share/zoneinfo/Not/AZone",
			want: "",
		},
		{
			name: "empty path",
			path: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zoneFromZoneinfoPath(tt.path); got != tt.want {
				t.Errorf("zoneFromZoneinfoPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestZoneFromLocaltimeSymlink(t *testing.T) {
	t.Run("resolves zone from symlink target", func(t *testing.T) {
		dir := t.TempDir()
		zoneDir := filepath.Join(dir, "zoneinfo", "America")
		if err := os.MkdirAll(zoneDir, 0o750); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(zoneDir, "Phoenix")
		if err := os.WriteFile(target, []byte("TZif"), 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(dir, "localtime")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}

		if got := zoneFromLocaltimeSymlink(link); got != "America/Phoenix" {
			t.Errorf("zoneFromLocaltimeSymlink() = %q, want %q", got, "America/Phoenix")
		}
	})

	t.Run("missing path returns empty", func(t *testing.T) {
		if got := zoneFromLocaltimeSymlink(filepath.Join(t.TempDir(), "missing")); got != "" {
			t.Errorf("zoneFromLocaltimeSymlink() = %q, want empty", got)
		}
	})
}
