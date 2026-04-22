package setup

import "testing"

func TestRunNonInteractive_ReconfiguresExistingAPIKey(t *testing.T) {
	originalVerify := verifyAPIKeyFn
	originalResolve := resolveAPIKeyApplicationFn
	originalEnsureCallback := ensureSetupCallbackURIFn
	originalActivate := activateAPIKeyFn
	originalGetStatus := getSetupStatusFn
	originalStepGrantSync := stepGrantSyncFn
	originalPrintComplete := printCompleteFn
	t.Cleanup(func() {
		verifyAPIKeyFn = originalVerify
		resolveAPIKeyApplicationFn = originalResolve
		ensureSetupCallbackURIFn = originalEnsureCallback
		activateAPIKeyFn = originalActivate
		getSetupStatusFn = originalGetStatus
		stepGrantSyncFn = originalStepGrantSync
		printCompleteFn = originalPrintComplete
	})

	verifyCalls := 0
	verifyAPIKeyFn = func(apiKey, region string) error {
		verifyCalls++
		if apiKey != "nyl_new_key" {
			t.Fatalf("expected api key %q, got %q", "nyl_new_key", apiKey)
		}
		if region != "eu" {
			t.Fatalf("expected region %q, got %q", "eu", region)
		}
		return nil
	}

	resolveCalls := 0
	resolveAPIKeyApplicationFn = func(apiKey, region, explicitClientID string, interactive bool) (*APIKeyApplication, error) {
		resolveCalls++
		if interactive {
			t.Fatal("expected non-interactive application resolution")
		}
		if explicitClientID != "client-123" {
			t.Fatalf("expected client ID %q, got %q", "client-123", explicitClientID)
		}
		return &APIKeyApplication{ClientID: "client-123", OrgID: "org-456"}, nil
	}

	callbackCalls := 0
	ensureSetupCallbackURIFn = func(apiKey, clientID, region string) error {
		callbackCalls++
		if clientID != "client-123" {
			t.Fatalf("expected callback setup client ID %q, got %q", "client-123", clientID)
		}
		return nil
	}

	activateCalls := 0
	activateAPIKeyFn = func(apiKey, clientID, region, orgID string) error {
		activateCalls++
		if apiKey != "nyl_new_key" {
			t.Fatalf("expected activated api key %q, got %q", "nyl_new_key", apiKey)
		}
		if clientID != "client-123" {
			t.Fatalf("expected activated client ID %q, got %q", "client-123", clientID)
		}
		if region != "eu" {
			t.Fatalf("expected activated region %q, got %q", "eu", region)
		}
		if orgID != "org-456" {
			t.Fatalf("expected activated org ID %q, got %q", "org-456", orgID)
		}
		return nil
	}

	getStatusCalls := 0
	getSetupStatusFn = func() SetupStatus {
		getStatusCalls++
		return SetupStatus{HasAPIKey: true}
	}

	grantSyncCalls := 0
	stepGrantSyncFn = func(status *SetupStatus) {
		grantSyncCalls++
		if status == nil || !status.HasAPIKey {
			t.Fatal("expected refreshed status with API key configured")
		}
	}

	printCompleteCalls := 0
	printCompleteFn = func() {
		printCompleteCalls++
	}

	err := runNonInteractive(wizardOpts{
		apiKey:   "nyl_new_key",
		clientID: "client-123",
		region:   "eu",
	}, SetupStatus{HasAPIKey: true})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if verifyCalls != 1 {
		t.Fatalf("expected verify to run once, got %d", verifyCalls)
	}
	if resolveCalls != 1 {
		t.Fatalf("expected application resolution once, got %d", resolveCalls)
	}
	if callbackCalls != 1 {
		t.Fatalf("expected callback setup once, got %d", callbackCalls)
	}
	if activateCalls != 1 {
		t.Fatalf("expected activation once, got %d", activateCalls)
	}
	if getStatusCalls != 1 {
		t.Fatalf("expected status refresh once, got %d", getStatusCalls)
	}
	if grantSyncCalls != 1 {
		t.Fatalf("expected grant sync once, got %d", grantSyncCalls)
	}
	if printCompleteCalls != 1 {
		t.Fatalf("expected completion message once, got %d", printCompleteCalls)
	}
}
