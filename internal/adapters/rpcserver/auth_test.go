package rpcserver

import (
	"errors"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

type fakeSecretStore struct {
	secrets map[string]string
	getErr  error
	setErr  error
	setKey  string
	setVal  string
}

func newFakeSecretStore() *fakeSecretStore {
	return &fakeSecretStore{secrets: make(map[string]string)}
}

func (f *fakeSecretStore) Set(key, value string) error {
	if f.setErr != nil {
		return f.setErr
	}
	f.setKey = key
	f.setVal = value
	f.secrets[key] = value
	return nil
}

func (f *fakeSecretStore) Get(key string) (string, error) {
	if f.getErr != nil {
		if errors.Is(f.getErr, domain.ErrSecretNotFound) && f.secrets[key] != "" {
			return f.secrets[key], nil
		}
		return "", f.getErr
	}
	return f.secrets[key], nil
}

func (f *fakeSecretStore) Delete(key string) error {
	delete(f.secrets, key)
	return nil
}

func (f *fakeSecretStore) IsAvailable() bool { return true }

func (f *fakeSecretStore) Name() string { return "fake" }

func TestGenerateToken(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken() second error = %v", err)
	}

	if first == "" {
		t.Fatal("GenerateToken() returned empty token")
	}
	if first == second {
		t.Fatal("GenerateToken() returned the same token twice")
	}
	if len(first) != 43 {
		t.Fatalf("GenerateToken() length = %d, want 43", len(first))
	}
	if strings.ContainsAny(first, "+/=") {
		t.Fatalf("GenerateToken() = %q, want URL-safe token without padding", first)
	}
}

func TestResolveToken(t *testing.T) {
	storeErr := errors.New("store unavailable")

	tests := []struct {
		name     string
		store    *fakeSecretStore
		getenv   func(string) string
		want     string
		wantErr  error
		wantSet  bool
		afterGet bool
	}{
		{
			name:  "env token wins",
			store: &fakeSecretStore{getErr: storeErr},
			getenv: func(key string) string {
				if key == EnvWSToken {
					return "env-token"
				}
				return ""
			},
			want: "env-token",
		},
		{
			name:  "store token returned",
			store: &fakeSecretStore{secrets: map[string]string{KeyRPCSessionToken: "stored-token"}},
			getenv: func(string) string {
				return ""
			},
			want: "stored-token",
		},
		{
			name:     "empty store generates and persists",
			store:    newFakeSecretStore(),
			getenv:   func(string) string { return "" },
			wantSet:  true,
			afterGet: true,
		},
		{
			name:     "missing store token generates and persists",
			store:    &fakeSecretStore{secrets: make(map[string]string), getErr: domain.ErrSecretNotFound},
			getenv:   func(string) string { return "" },
			wantSet:  true,
			afterGet: true,
		},
		{
			name:  "store error propagates",
			store: &fakeSecretStore{getErr: storeErr},
			getenv: func(string) string {
				return ""
			},
			wantErr: storeErr,
		},
		{
			name:    "store set error propagates",
			store:   &fakeSecretStore{secrets: make(map[string]string), setErr: storeErr},
			getenv:  func(string) string { return "" },
			wantErr: storeErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveToken(tt.store, tt.getenv)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ResolveToken() error = %v, want wrapping %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveToken() error = %v", err)
			}
			if tt.want != "" && got != tt.want {
				t.Fatalf("ResolveToken() = %q, want %q", got, tt.want)
			}
			if tt.wantSet && (tt.store.setKey != KeyRPCSessionToken || tt.store.setVal == "") {
				t.Fatalf("ResolveToken() persisted key/value = %q/%q, want key %q and non-empty value", tt.store.setKey, tt.store.setVal, KeyRPCSessionToken)
			}
			if tt.afterGet {
				stored, err := tt.store.Get(KeyRPCSessionToken)
				if err != nil {
					t.Fatalf("store.Get() error = %v", err)
				}
				if stored != got {
					t.Fatalf("stored token = %q, want generated token %q", stored, got)
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		provided string
		want     bool
	}{
		{name: "match", expected: "abc123", provided: "abc123", want: true},
		{name: "equal length mismatch exercises subtle compare path", expected: "abc123", provided: "abc124", want: false},
		{name: "different length mismatch", expected: "abc123", provided: "abc1234", want: false},
		{name: "empty expected", expected: "", provided: "abc123", want: false},
		{name: "empty provided", expected: "abc123", provided: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateToken(tt.expected, tt.provided); got != tt.want {
				t.Fatalf("ValidateToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateOrigin(t *testing.T) {
	allowed := []string{"http://localhost:3000", "https://app.example.com"}
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "allow-list hit", origin: "http://localhost:3000", want: true},
		{name: "allow-list miss", origin: "http://localhost:3001", want: false},
		{name: "empty origin allowed", origin: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateOrigin(tt.origin, allowed); got != tt.want {
				t.Fatalf("ValidateOrigin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLoopback(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		want    bool
		wantErr bool
	}{
		{name: "ipv4 loopback", addr: "127.0.0.1:8080", want: true},
		{name: "ipv6 loopback", addr: "[::1]:8080", want: true},
		{name: "localhost", addr: "localhost:0", want: true},
		{name: "unspecified ipv4", addr: "0.0.0.0:8080", want: false},
		{name: "public ip", addr: "8.8.8.8:8080", want: false},
		{name: "garbage", addr: "garbage", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsLoopback(tt.addr)
			if tt.wantErr {
				if err == nil {
					t.Fatal("IsLoopback() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("IsLoopback() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("IsLoopback() = %v, want %v", got, tt.want)
			}
		})
	}
}
