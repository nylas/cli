package keyring

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// zeroBytes overwrites b with zeros so a key derived from a user
// passphrase doesn't linger in heap memory after use. Go's GC retains
// allocations until they're collected; for a long-running `nylas air` /
// `nylas chat` process the derived AES key would otherwise survive in
// RAM for the lifetime of the process.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// encryptWithKey encrypts plaintext with AES-256-GCM using the given key.
// The returned bytes are base64-encoded and include the nonce prepended to
// the ciphertext.
func encryptWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decryptWithKey decrypts base64-encoded AES-256-GCM ciphertext using the
// given key.  The nonce is read from the first gcm.NonceSize() bytes of the
// decoded ciphertext.
func decryptWithKey(key, data []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
