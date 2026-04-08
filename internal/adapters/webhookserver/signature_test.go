package webhookserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifySignature(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"type":"message.created"}`)
	binaryPayload := []byte{0x00, 0x01, 0x02, 0xff}
	secret := "test-secret"
	signature := ComputeSignature(payload, secret)
	binarySignature := ComputeSignature(binaryPayload, secret)

	tests := []struct {
		name      string
		payload   []byte
		signature string
		secret    string
		want      bool
	}{
		{name: "plain hex signature", payload: payload, signature: signature, secret: secret, want: true},
		{name: "sha256 prefix", payload: payload, signature: "sha256=" + signature, secret: secret, want: true},
		{name: "uppercase prefix and whitespace", payload: payload, signature: " SHA256=" + signature + " ", secret: secret, want: true},
		{name: "invalid signature", payload: payload, signature: "invalid", secret: secret, want: false},
		{name: "wrong secret", payload: payload, signature: signature, secret: "wrong-secret", want: false},
		{name: "empty secret rejected", payload: payload, signature: ComputeSignature(payload, ""), secret: "", want: false},
		{name: "empty payload", payload: []byte{}, signature: ComputeSignature([]byte{}, secret), secret: secret, want: true},
		{name: "binary payload", payload: binaryPayload, signature: binarySignature, secret: secret, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, VerifySignature(tt.payload, tt.signature, tt.secret))
		})
	}
}

func TestNormalizeSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim lowercase prefix", input: " sha256=ABCDEF ", want: "abcdef"},
		{name: "trim uppercase prefix", input: "SHA256=ABCDEF", want: "abcdef"},
		{name: "plain hex", input: "ABCDEF", want: "abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, NormalizeSignature(tt.input))
		})
	}
}
