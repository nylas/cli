//go:build integration

package integration

import "testing"

func TestUseLocalAuthForIntegration(t *testing.T) {
	t.Setenv("NYLAS_TEST_USE_LOCAL_AUTH", "")
	if useLocalAuthForIntegration() {
		t.Fatal("expected local auth fallback to be disabled by default")
	}

	t.Setenv("NYLAS_TEST_USE_LOCAL_AUTH", "true")
	if !useLocalAuthForIntegration() {
		t.Fatal("expected local auth fallback to enable with true")
	}

	t.Setenv("NYLAS_TEST_USE_LOCAL_AUTH", "1")
	if !useLocalAuthForIntegration() {
		t.Fatal("expected local auth fallback to enable with 1")
	}
}
