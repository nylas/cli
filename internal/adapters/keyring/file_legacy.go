package keyring

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// deriveLegacyKey derives the pre-v2 machine-specific fallback key so older
// encrypted files can still be read and then re-encrypted with a
// passphrase-derived key (one-shot migration).
//
// The key is a SHA-256 hash of concatenated host metadata.  It is
// intentionally low-entropy compared to a user-supplied passphrase and exists
// only to allow migration from legacy installations.
func deriveLegacyKey() ([]byte, error) {
	var identifiers []byte

	hostname, _ := os.Hostname()
	identifiers = append(identifiers, []byte(hostname)...)

	identifiers = append(identifiers, []byte(os.Getenv("USER"))...)
	identifiers = append(identifiers, []byte(os.Getenv("USERNAME"))...) // Windows

	home, _ := os.UserHomeDir()
	identifiers = append(identifiers, []byte(home)...)

	identifiers = append(identifiers, []byte(getMachineID())...)

	// Static salt to prevent rainbow table attacks against this specific construction.
	identifiers = append(identifiers, []byte("nylas-cli-v1-secret-store")...)

	hash := sha256.Sum256(identifiers)
	return hash[:], nil
}

// getMachineID attempts to read a platform-specific machine identifier.
// Returns an empty string when no identifier is available; callers handle this
// gracefully by concatenating an empty contribution.
//
// On macOS, both candidate paths typically require elevated privileges and
// won't exist on a stock install — most macOS users will fall through with
// an empty machine ID. That's intentional: this helper feeds the legacy
// machine-derived migration key, which now exists only to decrypt files
// written by older versions and re-encrypt them under the user-supplied
// passphrase. Empty contribution means the legacy key is weaker, but the
// migration path requires NYLAS_FILE_STORE_PASSPHRASE to be set anyway.
func getMachineID() string {
	switch runtime.GOOS {
	case "linux":
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			return string(data)
		}
		if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			return string(data)
		}
	case "darwin":
		// Both files are typically root-owned on modern macOS.
		if data, err := os.ReadFile("/var/db/SystemKey"); err == nil {
			return string(data)
		}
		if data, err := os.ReadFile("/Library/Preferences/SystemConfiguration/com.apple.smb.server.plist"); err == nil {
			return string(data)
		}
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData != "" {
			guidPath := filepath.Join(programData, "Microsoft", "Crypto", "RSA", "MachineKeys", ".GUID")
			cleanPath := filepath.Clean(guidPath)
			if strings.HasPrefix(cleanPath, filepath.Clean(programData)) {
				if data, err := os.ReadFile(cleanPath); err == nil {
					return string(data)
				}
			}
		}
		if systemRoot := os.Getenv("SYSTEMROOT"); systemRoot != "" {
			return systemRoot
		}
	}
	return ""
}
