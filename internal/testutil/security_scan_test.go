package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecurityScanFailsForHardcodedCredentials(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, "leak.go", `package main

var api_key = "not-a-real-secret"
`)

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Possible credentials found") {
		t.Fatalf("security scan output %q does not mention credentials", output)
	}
}

func TestSecurityScanRedactsMatchedSecretValues(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	secret := "nyk_v0abcdefghijklmnopqrstuvwx"
	writeFile(t, repoDir, "leak.go", `package main

var apiKey = "`+secret+`"
`)

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if strings.Contains(output, secret) {
		t.Fatalf("security scan leaked matched secret value in output:\n%s", output)
	}
	if !strings.Contains(output, "leak.go:3") {
		t.Fatalf("security scan output %q does not include sanitized file location", output)
	}
}

func TestSecurityScanScansNonGoCredentialFiles(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	secret := "demo-client-secret-12345"
	writeFile(t, repoDir, "settings.yaml", "client_secret: "+secret+"\n")

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Possible credentials found") {
		t.Fatalf("security scan output %q does not mention credentials", output)
	}
	if strings.Contains(output, secret) {
		t.Fatalf("security scan leaked matched secret value in output:\n%s", output)
	}
	if !strings.Contains(output, "settings.yaml:1") {
		t.Fatalf("security scan output %q does not include sanitized file location", output)
	}
}

func TestSecurityScanScansTokenCredentialNames(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	secret := "tokenvalue1234567890"
	writeFile(t, repoDir, "app.go", `package main

var access_token = "`+secret+`"
`)

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Possible credentials found") {
		t.Fatalf("security scan output %q does not mention credentials", output)
	}
	if strings.Contains(output, secret) {
		t.Fatalf("security scan leaked matched secret value in output:\n%s", output)
	}
	if !strings.Contains(output, "app.go:3") {
		t.Fatalf("security scan output %q does not include sanitized file location", output)
	}
}

func TestSecurityScanDetectsFormattedAPIKeyLogging(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, "logging.go", `package main

import "fmt"

func main() {
	apiKey := loadAPIKey()
	_ = fmt.Sprintf("loaded key %s", apiKey)
}

func loadAPIKey() string {
	return "from-keyring"
}
`)

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Possible credential logging") {
		t.Fatalf("security scan output %q does not mention credential logging", output)
	}
	if !strings.Contains(output, "logging.go:7") {
		t.Fatalf("security scan output %q does not include sanitized file location", output)
	}
	if strings.Contains(output, "loaded key") || strings.Contains(output, "apiKey") {
		t.Fatalf("security scan leaked matched logging line in output:\n%s", output)
	}
}

func TestSecurityScanFailsForTrackedSensitiveFiles(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, ".env.production", "NYLAS_API_KEY=not-a-real-key\n")
	runGit(t, repoDir, "add", ".env.production")

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Sensitive tracked files found") {
		t.Fatalf("security scan output %q does not mention tracked sensitive files", output)
	}
}

func TestSecurityScanFailsForTrackedSensitiveJSONFiles(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, "service-account.json", "{}\n")
	runGit(t, repoDir, "add", "service-account.json")

	output, err := runSecurityScan(t, repoDir)
	if err == nil {
		t.Fatalf("security scan passed, want failure. output:\n%s", output)
	}
	if !strings.Contains(output, "Sensitive tracked files found") {
		t.Fatalf("security scan output %q does not mention tracked sensitive files", output)
	}
}

func TestSecurityScanAllowsTestFixtures(t *testing.T) {
	repoDir := initSecurityScanRepo(t)
	writeFile(t, repoDir, "fixture_test.go", `package main

var api_key = "not-a-real-secret"
`)
	writeFile(t, repoDir, "testdata/settings.yaml", "client_secret: fixture-secret-value\n")

	output, err := runSecurityScan(t, repoDir)
	if err != nil {
		t.Fatalf("security scan failed, want success. output:\n%s", output)
	}
}

func initSecurityScanRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	writeFile(t, repoDir, "main.go", "package main\n")
	runGit(t, repoDir, "add", "main.go")
	return repoDir
}

func runSecurityScan(t *testing.T, repoDir string) (string, error) {
	t.Helper()

	cmd := exec.Command("sh", securityScanScriptPath(t), repoDir)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func securityScanScriptPath(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd returned error: %v", err)
	}

	for {
		candidate := filepath.Join(dir, "scripts", "security-scan.sh")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find scripts/security-scan.sh from %s", dir)
		}
		dir = parent
	}
}

func writeFile(t *testing.T, repoDir, name, contents string) {
	t.Helper()

	path := filepath.Join(repoDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		t.Fatalf("os.MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}
}

func runGit(t *testing.T, repoDir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(),
		"HOME="+t.TempDir(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
