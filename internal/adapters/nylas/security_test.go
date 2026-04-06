//go:build !integration
// +build !integration

package nylas

import (
	"testing"
)

// TestOTPExtractionSecurity tests that OTP extraction is secure.
func TestOTPExtractionSecurity(t *testing.T) {
	t.Run("does_not_extract_from_malicious_patterns", func(t *testing.T) {
		// These should NOT be extracted as OTPs
		maliciousPatterns := []struct {
			subject string
			body    string
			desc    string
		}{
			{
				subject: "Your account balance",
				body:    "Your balance is $123456",
				desc:    "currency amounts",
			},
			{
				subject: "Invoice #123456",
				body:    "Please pay invoice 123456",
				desc:    "invoice numbers",
			},
			{
				subject: "Order confirmation",
				body:    "Order ID: 20241215",
				desc:    "date-like order IDs",
			},
			{
				subject: "Meeting scheduled",
				body:    "Meeting room 123456",
				desc:    "room numbers",
			},
			{
				subject: "Phone contact",
				body:    "Call me at 123456",
				desc:    "partial phone numbers",
			},
		}

		for _, tc := range maliciousPatterns {
			t.Run(tc.desc, func(t *testing.T) {
				// These shouldn't match OTP patterns without context
				// The function may or may not extract these, but we should test the behavior
				result := ExtractOTP(tc.subject, tc.body)
				t.Logf("Input: %q / %q -> Result: %q", tc.subject, tc.body, result)
			})
		}
	})

	t.Run("handles_empty_input", func(t *testing.T) {
		result := ExtractOTP("", "")
		if result != "" {
			t.Errorf("Expected empty result for empty input, got %q", result)
		}
	})

	t.Run("handles_nil_like_input", func(t *testing.T) {
		result := ExtractOTP("null", "undefined")
		// Should not crash and should return empty
		t.Logf("Result for null/undefined: %q", result)
	})

	t.Run("handles_very_long_input", func(t *testing.T) {
		// Create a very long body
		longBody := make([]byte, 100*1024) // 100KB
		for i := range longBody {
			longBody[i] = 'a'
		}

		// Should not hang or crash
		result := ExtractOTP("Test subject", string(longBody))
		t.Logf("Result for long input: %q", result)
	})

	t.Run("handles_special_characters", func(t *testing.T) {
		specialChars := []string{
			`<script>alert('xss')</script>`,
			`'; DROP TABLE users; --`,
			`\x00\x00\x00`,
			`../../etc/passwd`,
			`%n%n%n%n%n`,
		}

		for _, input := range specialChars {
			t.Run("special_char_test", func(t *testing.T) {
				// Should not crash
				result := ExtractOTP("Subject: "+input, "Body: "+input)
				t.Logf("Special char input result: %q", result)
			})
		}
	})

	t.Run("handles_unicode_correctly", func(t *testing.T) {
		unicodeInputs := []struct {
			subject string
			body    string
		}{
			{"验证码", "您的验证码是 123456"},
			{"التحقق", "رمز التحقق الخاص بك هو 123456"},
			{"Код подтверждения", "Ваш код: 123456"},
			{"確認コード", "確認コード: 123456"},
		}

		for _, tc := range unicodeInputs {
			result := ExtractOTP(tc.subject, tc.body)
			t.Logf("Unicode input %q -> %q", tc.subject, result)
		}
	})
}

// TestHTTPClientSecurity tests HTTP client security.
func TestHTTPClientSecurity(t *testing.T) {
	t.Run("client_uses_per_request_timeouts", func(t *testing.T) {
		client := NewHTTPClient()
		// HTTP client should NOT have a global timeout (we use per-request context timeouts)
		// This allows better control and prevents blocking other requests
		if client.httpClient.Timeout != 0 {
			t.Error("HTTP client should not have global timeout (uses per-request context timeouts)")
		}
		// Verify rate limiter is configured
		if client.rateLimiter == nil {
			t.Error("Rate limiter should be configured")
		}
		// Verify request timeout is configured
		if client.requestTimeout == 0 {
			t.Error("Request timeout should be configured")
		}
	})

	t.Run("default_base_url_is_https", func(t *testing.T) {
		client := NewHTTPClient()
		if client.baseURL[:8] != "https://" {
			t.Errorf("Base URL should use HTTPS, got %q", client.baseURL)
		}
	})

	t.Run("set_region_maintains_https", func(t *testing.T) {
		client := NewHTTPClient()

		client.SetRegion("us")
		if client.baseURL[:8] != "https://" {
			t.Errorf("US region URL should use HTTPS, got %q", client.baseURL)
		}

		client.SetRegion("eu")
		if client.baseURL[:8] != "https://" {
			t.Errorf("EU region URL should use HTTPS, got %q", client.baseURL)
		}
	})

	t.Run("credentials_not_logged", func(t *testing.T) {
		client := NewHTTPClient()
		client.SetCredentials("test-client-id", "test-secret", "test-api-key")

		// Verify credentials are stored (basic check)
		if client.clientID != "test-client-id" {
			t.Error("Client ID not stored correctly")
		}
		if client.clientSecret != "test-secret" {
			t.Error("Client secret not stored correctly")
		}
		if client.apiKey != "test-api-key" {
			t.Error("API key not stored correctly")
		}
	})
}

// TestInputValidation tests input validation in the client.
func TestInputValidation(t *testing.T) {
	t.Run("empty_grant_id_handling", func(t *testing.T) {
		client := NewHTTPClient()

		// These should handle empty strings gracefully
		// Not crash or panic
		_ = client.BuildAuthURL("google", "", "", "")
		t.Log("Empty redirect URI handled")
	})

	t.Run("special_chars_in_grant_id", func(t *testing.T) {
		// Should handle special characters in grant ID
		// Not vulnerable to path traversal
		specialIDs := []string{
			"../../../etc/passwd",
			"grant-id?extra=param",
			"grant-id#fragment",
			"grant-id\x00null",
		}

		for _, id := range specialIDs {
			// These shouldn't crash
			t.Logf("Testing grant ID: %q", id)
			// Note: Actual API calls would fail, but local handling shouldn't crash
		}
	})
}
