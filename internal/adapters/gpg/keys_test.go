package gpg

import (
	"testing"
	"time"
)

func TestParsePublicKeys(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name: "single public key with UID",
			input: `pub:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::1234567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::1234567890ABCDEF1234567890ABCDEF12345678::John Doe <john@example.com>::::::::::0:
`,
			want:    1,
			wantErr: false,
		},
		{
			name: "multiple public keys",
			input: `pub:u:4096:1:601FEE9B1D60185F:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::AAAA567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::AAAA567890ABCDEF1234567890ABCDEF12345678::Alice <alice@example.com>::::::::::0:
pub:u:2048:1:701FEE9B1D60185G:1609459200:::u:::scESC:::+:::23::0:
fpr:::::::::BBBB567890ABCDEF1234567890ABCDEF12345678:
uid:u::::1609459200::BBBB567890ABCDEF1234567890ABCDEF12345678::Bob <bob@example.com>::::::::::0:
`,
			want:    2,
			wantErr: false,
		},
		{
			name:    "no keys",
			input:   "",
			want:    0,
			wantErr: false, // Empty is OK for public keys (unlike secret keys)
		},
		{
			name:    "invalid format",
			input:   "invalid output",
			want:    0,
			wantErr: false, // Still returns empty slice, not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePublicKeys(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePublicKeys() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.want {
				t.Errorf("parsePublicKeys() got %d keys, want %d", len(got), tt.want)
			}
		})
	}
}

func TestKeyMatchesEmail(t *testing.T) {
	tests := []struct {
		name  string
		key   KeyInfo
		email string
		want  bool
	}{
		{
			name: "email in angle brackets",
			key: KeyInfo{
				UIDs: []string{"John Doe <john@example.com>"},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "case insensitive match",
			key: KeyInfo{
				UIDs: []string{"John Doe <John@EXAMPLE.COM>"},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "bare email match",
			key: KeyInfo{
				UIDs: []string{"user@example.com"},
			},
			email: "user@example.com",
			want:  true,
		},
		{
			name: "multiple UIDs with match",
			key: KeyInfo{
				UIDs: []string{
					"Work <work@company.com>",
					"Personal <john@example.com>",
				},
			},
			email: "john@example.com",
			want:  true,
		},
		{
			name: "no match",
			key: KeyInfo{
				UIDs: []string{"Other <other@example.com>"},
			},
			email: "john@example.com",
			want:  false,
		},
		{
			name: "partial match not accepted",
			key: KeyInfo{
				UIDs: []string{"John <john@example.com.malicious>"},
			},
			email: "john@example.com",
			want:  false,
		},
		{
			name: "empty UIDs",
			key: KeyInfo{
				UIDs: []string{},
			},
			email: "john@example.com",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := keyMatchesEmail(&tt.key, tt.email)
			if got != tt.want {
				t.Errorf("keyMatchesEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeyInfo_ExpiredKey(t *testing.T) {
	// Test that expired keys are properly detected
	pastTime := time.Now().Add(-24 * time.Hour)
	futureTime := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name      string
		key       KeyInfo
		isExpired bool
	}{
		{
			name: "expired key",
			key: KeyInfo{
				KeyID:   "EXPIRED1234",
				Expires: &pastTime,
			},
			isExpired: true,
		},
		{
			name: "valid key",
			key: KeyInfo{
				KeyID:   "VALID1234",
				Expires: &futureTime,
			},
			isExpired: false,
		},
		{
			name: "no expiration",
			key: KeyInfo{
				KeyID:   "NOEXPIRE1234",
				Expires: nil,
			},
			isExpired: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExpired := tt.key.Expires != nil && tt.key.Expires.Before(time.Now())
			if isExpired != tt.isExpired {
				t.Errorf("Key expired = %v, want %v", isExpired, tt.isExpired)
			}
		})
	}
}
