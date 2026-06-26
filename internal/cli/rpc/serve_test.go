package rpc

import (
	"testing"
	"time"
)

func TestPollInterval(t *testing.T) {
	const def = 30 * time.Second

	tests := []struct {
		name string
		env  map[string]string
		want time.Duration
	}{
		{name: "unset falls back to default", env: nil, want: def},
		{name: "empty falls back to default", env: map[string]string{"K": ""}, want: def},
		{name: "valid duration is used", env: map[string]string{"K": "2s"}, want: 2 * time.Second},
		{name: "invalid duration falls back", env: map[string]string{"K": "nope"}, want: def},
		{name: "zero falls back", env: map[string]string{"K": "0s"}, want: def},
		{name: "negative falls back", env: map[string]string{"K": "-5s"}, want: def},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string { return tt.env[key] }
			if got := pollInterval(getenv, "K", def); got != tt.want {
				t.Fatalf("pollInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}
