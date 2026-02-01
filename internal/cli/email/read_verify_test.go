package email

import (
	"strings"
	"testing"
)

func TestExtractFullContentType(t *testing.T) {
	tests := []struct {
		name    string
		rawMIME string
		want    string
	}{
		{
			name: "simple content-type",
			rawMIME: `Content-Type: text/plain; charset=utf-8

Body here`,
			want: "text/plain; charset=utf-8",
		},
		{
			name: "multipart with continuation",
			rawMIME: `From: test@example.com
Content-Type: multipart/signed; protocol="application/pgp-signature";
	micalg=pgp-sha256; boundary="=_signed_123"

Body here`,
			want: `multipart/signed; protocol="application/pgp-signature"; micalg=pgp-sha256; boundary="=_signed_123"`,
		},
		{
			name:    "content-type with CRLF",
			rawMIME: "Content-Type: text/html\r\n\r\nBody",
			want:    "text/html",
		},
		{
			name:    "no content-type",
			rawMIME: "From: test@example.com\n\nBody",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFullContentType(tt.rawMIME)
			if got != tt.want {
				t.Errorf("extractFullContentType() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFindHeaderEnd(t *testing.T) {
	tests := []struct {
		name    string
		rawMIME string
		want    int
	}{
		{
			name:    "LF line endings",
			rawMIME: "Header: value\n\nBody",
			want:    15, // Position after \n\n
		},
		{
			name:    "CRLF line endings",
			rawMIME: "Header: value\r\n\r\nBody",
			want:    17, // Position after \r\n\r\n
		},
		{
			name:    "no blank line",
			rawMIME: "Header: value",
			want:    -1,
		},
		{
			name:    "multiple headers",
			rawMIME: "Header1: value1\nHeader2: value2\n\nBody",
			want:    33, // Position after \n\n
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findHeaderEnd(tt.rawMIME)
			if got != tt.want {
				t.Errorf("findHeaderEnd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractSignedContent(t *testing.T) {
	tests := []struct {
		name      string
		rawMIME   string
		boundary  string
		wantStart string // Check that content starts with this
		wantEnd   string // Check that content ends with this
		wantErr   bool
	}{
		{
			name: "simple signed content",
			rawMIME: `Content-Type: multipart/signed; boundary="=_signed_123"

--=_signed_123
Content-Type: text/plain; charset=utf-8

Hello World
--=_signed_123
Content-Type: application/pgp-signature

-----BEGIN PGP SIGNATURE-----
...
-----END PGP SIGNATURE-----
--=_signed_123--`,
			boundary:  "=_signed_123",
			wantStart: "Content-Type: text/plain",
			wantEnd:   "Hello World",
			wantErr:   false,
		},
		{
			name:     "missing first boundary",
			rawMIME:  "No boundary here",
			boundary: "=_signed_123",
			wantErr:  true,
		},
		{
			name: "missing second boundary",
			rawMIME: `--=_signed_123
Content-Type: text/plain

Hello`,
			boundary: "=_signed_123",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractSignedContent(tt.rawMIME, tt.boundary)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractSignedContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				content := string(got)
				if !strings.HasPrefix(content, tt.wantStart) {
					t.Errorf("Content doesn't start with %q, got: %q", tt.wantStart, content[:min(50, len(content))])
				}
				if !strings.HasSuffix(content, tt.wantEnd) {
					t.Errorf("Content doesn't end with %q, got: %q", tt.wantEnd, content[max(0, len(content)-50):])
				}
				// Verify CRLF line endings
				if strings.Contains(content, "\n") && !strings.Contains(content, "\r\n") {
					t.Error("Content should have CRLF line endings")
				}
			}
		})
	}
}

func TestDecodeQuotedPrintable(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "plain text",
			input: []byte("Hello World"),
			want:  "Hello World",
		},
		{
			name:  "encoded newline",
			input: []byte("Line1=0ALine2"),
			want:  "Line1\nLine2",
		},
		{
			name:  "encoded equals",
			input: []byte("a=3Db"),
			want:  "a=b",
		},
		{
			name:  "soft line break",
			input: []byte("Hello =\nWorld"),
			want:  "Hello World",
		},
		{
			name:  "soft line break CRLF",
			input: []byte("Hello =\r\nWorld"),
			want:  "Hello World",
		},
		{
			name:  "PGP signature pattern",
			input: []byte("-----BEGIN PGP SIGNATURE-----=0A=0AiQJJ=0A-----END PGP SIGNATURE-----"),
			want:  "-----BEGIN PGP SIGNATURE-----\n\niQJJ\n-----END PGP SIGNATURE-----",
		},
		{
			name:  "mixed encoding",
			input: []byte("Test=0Awith=3Dequals=0Aand newlines"),
			want:  "Test\nwith=equals\nand newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeQuotedPrintable(tt.input)
			if string(got) != tt.want {
				t.Errorf("decodeQuotedPrintable() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestIsHexPair(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0A", true},
		{"FF", true},
		{"3D", true},
		{"0a", true},
		{"ff", true},
		{"A", false},   // too short
		{"AAA", false}, // too long
		{"GG", false},  // invalid hex
		{"0Z", false},  // invalid hex
		{"", false},    // empty
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isHexPair(tt.input)
			if got != tt.want {
				t.Errorf("isHexPair(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestHexToByte(t *testing.T) {
	tests := []struct {
		input string
		want  byte
	}{
		{"00", 0x00},
		{"0A", 0x0A},
		{"0a", 0x0a},
		{"FF", 0xFF},
		{"ff", 0xff},
		{"3D", 0x3D},
		{"20", 0x20},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := hexToByte(tt.input)
			if got != tt.want {
				t.Errorf("hexToByte(%q) = %02X, want %02X", tt.input, got, tt.want)
			}
		})
	}
}
