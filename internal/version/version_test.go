package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestUserAgent(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "dev version",
			version: "dev",
			want:    "nylas-cli/dev",
		},
		{
			name:    "release version",
			version: "1.0.0",
			want:    "nylas-cli/1.0.0",
		},
		{
			name:    "prerelease version",
			version: "2.0.0-beta.1",
			want:    "nylas-cli/2.0.0-beta.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			original := Version
			t.Cleanup(func() { Version = original })

			Version = tt.version
			got := UserAgent()

			// Check version prefix
			if !strings.HasPrefix(got, tt.want) {
				t.Errorf("UserAgent() = %q, want prefix %q", got, tt.want)
			}

			// Check platform info is included
			expectedPlatform := "(" + runtime.GOOS + "/" + runtime.GOARCH + ")"
			if !strings.HasSuffix(got, expectedPlatform) {
				t.Errorf("UserAgent() = %q, want suffix %q", got, expectedPlatform)
			}
		})
	}
}

func TestUserAgent_Format(t *testing.T) {
	got := UserAgent()

	// Format should be: nylas-cli/VERSION (OS/ARCH)
	parts := strings.Split(got, " ")
	if len(parts) != 2 {
		t.Errorf("UserAgent() = %q, want format 'nylas-cli/VERSION (OS/ARCH)'", got)
	}

	// First part should start with nylas-cli/
	if !strings.HasPrefix(parts[0], "nylas-cli/") {
		t.Errorf("UserAgent() first part = %q, want prefix 'nylas-cli/'", parts[0])
	}

	// Second part should be (OS/ARCH)
	if !strings.HasPrefix(parts[1], "(") || !strings.HasSuffix(parts[1], ")") {
		t.Errorf("UserAgent() platform part = %q, want format '(OS/ARCH)'", parts[1])
	}
}
