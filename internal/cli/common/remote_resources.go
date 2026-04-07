package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
)

// ResolveGrantIdentifier resolves a grant identifier, accepting either a grant ID or an email.
func ResolveGrantIdentifier(identifier string) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", nil
	}
	if !containsAt(identifier) {
		return identifier, nil
	}

	secretStore, err := openSecretStore()
	if err != nil {
		return "", err
	}

	grantStore := keyring.NewGrantStore(secretStore)
	grant, err := grantStore.GetGrantByEmail(identifier)
	if err != nil {
		if errors.Is(err, domain.ErrGrantNotFound) {
			return "", fmt.Errorf("no grant found for email: %s", identifier)
		}
		return "", wrapSecretStoreError(err)
	}

	return grant.ID, nil
}

// ResolveScopeGrantID resolves the grant ID when a command targets grant-scoped resources.
func ResolveScopeGrantID(scope domain.RemoteScope, grantID string) (string, error) {
	if scope != domain.ScopeGrant {
		return "", nil
	}
	if grantID == "" {
		return GetGrantID(nil)
	}
	return ResolveGrantIdentifier(grantID)
}

// LoadJSONFile decodes a JSON file into target.
func LoadJSONFile(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return nil
}

// ReadJSONStringMap parses inline JSON or JSON from a file into a map.
func ReadJSONStringMap(value, file string) (map[string]any, error) {
	if value != "" && file != "" {
		return nil, NewUserError("only one of --data or --data-file may be used", "Choose either inline JSON or a JSON file")
	}

	var data []byte
	switch {
	case file != "":
		fileData, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", file, err)
		}
		data = fileData
	case value != "":
		data = []byte(value)
	default:
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, NewUserError("invalid JSON data", "Provide a valid JSON object with --data or --data-file")
	}
	if result == nil {
		result = map[string]any{}
	}
	return result, nil
}

// ReadStringOrFile returns a string from an inline flag or file.
func ReadStringOrFile(name, value, file string, required bool) (string, error) {
	if value != "" && file != "" {
		return "", NewUserError(
			fmt.Sprintf("only one of --%s or --%s-file may be used", name, name),
			fmt.Sprintf("Choose either --%s or --%s-file", name, name),
		)
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", file, err)
		}
		return string(data), nil
	}

	if required && value == "" {
		return "", ValidateRequiredFlag("--"+name, value)
	}

	return value, nil
}
