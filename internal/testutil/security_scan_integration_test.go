//go:build integration

package testutil

import (
	"strings"
	"testing"
)

func TestSecurityScanIntegrationFailsClosedForSensitiveJSON(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, "credentials.json", "{}\n")
	runGit(t, repoDir, "add", "credentials.json")

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Sensitive tracked files found") {
		t.Fatalf("security scan output %q does not mention tracked sensitive files", output)
	}
}
