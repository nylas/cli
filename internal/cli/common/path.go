package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateExecutablePath validates that an executable path is safe to use.
// It checks that the path exists, is executable, and doesn't contain suspicious patterns.
func ValidateExecutablePath(path string) error {
	if path == "" {
		return fmt.Errorf("executable path is empty")
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains '..' which may indicate path traversal: %s", path)
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if file exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("executable not found: %s", absPath)
		}
		return fmt.Errorf("failed to stat executable: %w", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("path is not a regular file: %s", absPath)
	}

	// Check if executable (on Unix systems)
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable: %s", absPath)
	}

	return nil
}

// FindExecutableInPath finds an executable in the system PATH.
// Returns the full path or an error if not found.
func FindExecutableInPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("executable %q not found in PATH", name)
	}

	// Validate the found path
	if err := ValidateExecutablePath(path); err != nil {
		return "", fmt.Errorf("found executable failed validation: %w", err)
	}

	return path, nil
}

// SafeCommand creates a validated exec.Cmd for an external command.
// It validates the executable path and returns an error if unsafe.
func SafeCommand(name string, args ...string) (*exec.Cmd, error) {
	// Find executable in PATH
	execPath, err := FindExecutableInPath(name)
	if err != nil {
		return nil, err
	}

	// Create command with validated path
	//nolint:gosec // G204: execPath is validated by FindExecutableInPath before use
	cmd := exec.Command(execPath, args...)
	return cmd, nil
}
