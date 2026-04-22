.PHONY: build test-unit test-race test-integration test-integration-fast test-cli-regressions test-integration-agent test-cleanup test-coverage test-air test-air-integration test-e2e test-e2e-air test-e2e-ui test-playwright test-playwright-air test-playwright-ui test-playwright-interactive test-playwright-headed clean clean-cache install fmt vet lint vuln deps security check-context ci ci-full help

# Disable parallel Make execution - prevents Go build cache corruption on btrfs (CachyOS)
.NOTPARALLEL:

# ============================================================================
# Environment Configuration
# ============================================================================
# Load .env file if it exists (for NYLAS_API_KEY, NYLAS_GRANT_ID, etc.)
# Uses simple KEY=value format, no 'export' prefix needed
-include .env
export

# Strip quotes from env vars (Make doesn't handle quoted values like shell does)
NYLAS_API_KEY := $(patsubst "%",%,$(patsubst '%',%,$(NYLAS_API_KEY)))
NYLAS_GRANT_ID := $(patsubst "%",%,$(patsubst '%',%,$(NYLAS_GRANT_ID)))
NYLAS_CLIENT_ID := $(patsubst "%",%,$(patsubst '%',%,$(NYLAS_CLIENT_ID)))
NYLAS_AGENT_DOMAIN := $(patsubst "%",%,$(patsubst '%',%,$(NYLAS_AGENT_DOMAIN)))

# Rate limit defaults (can be overridden in .env)
NYLAS_TEST_RATE_LIMIT_RPS ?= 1.0
NYLAS_TEST_RATE_LIMIT_BURST ?= 3

# ============================================================================
# Tool Versions (use @latest for automatic updates)
# ============================================================================
GOVULNCHECK_VERSION := latest

# ============================================================================
# Build Configuration
# ============================================================================
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X github.com/nylas/cli/internal/version.Version=$(VERSION) -X github.com/nylas/cli/internal/version.Commit=$(COMMIT) -X github.com/nylas/cli/internal/version.BuildDate=$(BUILD_DATE)"

# ============================================================================
# Build Targets
# ============================================================================
build:
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/nylas ./cmd/nylas

# ============================================================================
# Code Quality Targets
# ============================================================================
fmt:
	@echo "=== Formatting Code ==="
	go fmt ./...
	@echo "✓ Code formatted"

vet:
	@echo "=== Running go vet ==="
	go vet ./...
	@echo "✓ Go vet passed"

lint:
	@echo "=== Running golangci-lint ==="
	golangci-lint run --timeout=5m
	@echo "✓ Linting passed"

vuln:
	@echo "=== Checking for vulnerabilities ==="
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "Installing govulncheck $(GOVULNCHECK_VERSION)..."; \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
	}
	govulncheck ./...
	@echo "✓ No vulnerabilities found"

# ============================================================================
# Test Targets
# ============================================================================
test-unit:
	@echo "=== Running Unit Tests ==="
	@go clean -testcache
	go test ./... -short -v
	@echo "✓ Unit tests passed"

test-race:
	@echo "=== Running Race Detector Tests ==="
	@go clean -testcache
	go test ./... -short -race
	@echo "✓ Race detector tests passed"

test-coverage:
	@echo "=== Running Tests with Coverage ==="
	@go clean -testcache
	go test ./... -cover -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

# ============================================================================
# Air Web UI Tests
# ============================================================================
test-air:
	@echo "=== Running Nylas Air Tests ==="
	@go clean -testcache
	go test ./internal/air/... -v
	@echo "✓ All Air tests passed"

# Nylas Air integration tests (requires Google account as default)
# Skips automatically if no Google account is configured as default
# Rate limiting: 1 RPS with burst of 3 to stay well under Nylas API limits
# -p 1: Run test packages sequentially to prevent rate limit issues
test-air-integration:
	@echo "=== Running Nylas Air Integration Tests ==="
	@echo "Note: Requires a Google account configured as default"
	@echo ""
	@go clean -testcache
	NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
	NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
	go test -tags=integration ./internal/air/... -v -timeout 5m -p 1
	@echo "✓ All Air integration tests passed"


# ============================================================================
# Integration Tests
# ============================================================================
# Integration tests (requires NYLAS_API_KEY and NYLAS_GRANT_ID env vars)
# Uses 10 minute timeout to prevent hanging on slow LLM calls
# Output saved to test-integration.txt
# Set NYLAS_TEST_SKIP_AGENT=true to skip TestCLI_Agent* when those already ran
# NYLAS_DISABLE_KEYRING=true prevents keychain popup and skips tests that need local grant store
# Rate limiting: 1 RPS with burst of 3 to stay well under Nylas API limits
# -p 1: Run test packages sequentially to prevent rate limit issues
test-integration:
	@go clean -testcache
	@bash -o pipefail -c '\
		skip_agent=$$(printf "%s" "$${NYLAS_TEST_SKIP_AGENT:-}" | tr "[:upper:]" "[:lower:]"); \
		if [ "$$skip_agent" = "1" ] || [ "$$skip_agent" = "true" ]; then \
			NYLAS_DISABLE_KEYRING=true \
			NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
			NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
			NYLAS_TEST_BINARY=$(CURDIR)/bin/nylas \
			go test -tags=integration -v -timeout 10m -p 1 -skip TestCLI_Agent ./... 2>&1 | tee test-integration.txt; \
		else \
			NYLAS_DISABLE_KEYRING=true \
			NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
			NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
			NYLAS_TEST_BINARY=$(CURDIR)/bin/nylas \
			go test -tags=integration -v -timeout 10m -p 1 ./... 2>&1 | tee test-integration.txt; \
		fi \
	'

# Integration tests excluding slow LLM-dependent tests (for when Ollama is slow/unavailable)
# Runs: Admin, Timezone, AIConfig, CalendarAI (Basic, Adapt, Analyze working hours)
# Rate limiting: 1 RPS with burst of 3 to stay well under Nylas API limits
# -p 1: Run test packages sequentially to prevent rate limit issues
test-integration-fast:
	@go clean -testcache
	@bash -o pipefail -c '\
		NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
		NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
		NYLAS_TEST_BINARY=$(CURDIR)/bin/nylas \
		go test ./internal/cli/integration/... -tags=integration -v -timeout 2m -p 1 \
			-run "TestCLI_Admin|TestCLI_Timezone|TestCLI_AIConfig|TestCLI_AIProvider|TestCLI_CalendarAI_Basic|TestCLI_CalendarAI_Adapt|TestCLI_CalendarAI_Analyze_Respects|TestCLI_CalendarAI_Analyze_Default|TestCLI_CalendarAI_Analyze_Disabled|TestCLI_CalendarAI_Analyze_Focus|TestCLI_CalendarAI_Analyze_With" \
	'

# Focused CLI regression checks for command removals and agent behavior.
# This makes ci-full explicitly verify the removed inbound surface and auth-level provider rejection.
test-cli-regressions: build
	@echo "=== Running CLI Regression Checks ==="
	@go clean -testcache
	go test ./internal/cli/agent -v
	NYLAS_DISABLE_KEYRING=true \
	NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
	NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
	NYLAS_TEST_BINARY=$(CURDIR)/bin/nylas \
	go test ./internal/cli/integration/... -tags=integration -v -timeout 10m -p 1 \
		-run 'TestCLI_(InboundRemoved|InboxAliasRemoved|HelpOmitsInbound|AuthLoginRejectsInboxProvider|ConnectorSurfaces_HideInboxProvider|AdminConnectorsCreate_RejectsInboxProvider|AdminConnectorsShow_HidesInboxProvider)$$'
	@echo "✓ CLI regression checks passed"

# Agent integration checks require explicit credentials plus an agent domain so the lifecycle suites do not self-skip.
test-integration-agent: build
	@echo "=== Running Agent Integration Checks ==="
	@: "$${NYLAS_API_KEY:?NYLAS_API_KEY is required for agent integration tests}"
	@: "$${NYLAS_GRANT_ID:?NYLAS_GRANT_ID is required for agent integration tests}"
	@: "$${NYLAS_AGENT_DOMAIN:?NYLAS_AGENT_DOMAIN is required for agent integration tests}"
	@go clean -testcache
	NYLAS_DISABLE_KEYRING=true \
	NYLAS_TEST_RATE_LIMIT_RPS=$(NYLAS_TEST_RATE_LIMIT_RPS) \
	NYLAS_TEST_RATE_LIMIT_BURST=$(NYLAS_TEST_RATE_LIMIT_BURST) \
	NYLAS_TEST_BINARY=$(CURDIR)/bin/nylas \
	go test ./internal/cli/integration/... -tags=integration -v -timeout 10m -p 1 \
		-run 'TestCLI_Agent.*$$'
	@echo "✓ Agent integration checks passed"

# Clean up test resources (virtual calendars, test grants, test events, test emails, etc.)
test-cleanup:
	@echo "=== Cleaning up test resources ==="
	@echo ""
	@echo "0. Killing any leftover test processes and freeing ports..."
	@-pkill -f "nylas.*webhook.*server" 2>/dev/null || true
	@-pkill -f "nylas.*ui" 2>/dev/null || true
	@-pkill -f "cloudflared.*tunnel" 2>/dev/null || true
	@-pkill -f "cloudflared" 2>/dev/null || true
	@-lsof -ti :3099 | xargs kill -9 2>/dev/null || true
	@-lsof -ti :8080 | xargs kill -9 2>/dev/null || true
	@-lsof -ti :9000 | xargs kill -9 2>/dev/null || true
	@echo "  ✓ Processes cleaned up"
	@echo ""
	@echo "1. Cleaning test emails (messages and drafts)..."
	@./bin/nylas email list --limit 100 --id 2>/dev/null | \
		grep -E "(Test|Integration|Draft|AI|Metadata)" -A1 | \
		grep "ID:" | \
		awk '{print $$2}' | \
		while read msg_id; do \
			if [ ! -z "$$msg_id" ]; then \
				echo "  Deleting test message: $$msg_id"; \
				./bin/nylas email delete $$msg_id --force 2>/dev/null && \
				echo "    ✓ Deleted message $$msg_id" || echo "    ⚠ Could not delete $$msg_id"; \
			fi \
		done
	@echo ""
	@echo "2. Cleaning test events from calendars..."
	@./bin/nylas calendar events list --limit 100 2>/dev/null | \
		awk '/AI Test|Test Meeting|Integration Test|test-event/ { \
			getline; getline; getline; getline; \
			if ($$0 ~ /ID:/) { split($$0, arr, " "); print arr[2] } \
		}' | \
		while read event_id; do \
			if [ ! -z "$$event_id" ]; then \
				echo "  Deleting test event: $$event_id"; \
				./bin/nylas calendar events delete $$event_id --force 2>/dev/null && \
				echo "    ✓ Deleted event $$event_id" || echo "    ⚠ Could not delete $$event_id"; \
			fi \
		done
	@echo ""
	@echo "3. Cleaning test virtual calendar grants..."
	@./bin/nylas admin grants list | grep -E "^(test-|integration-)" | awk '{print $$2}' | while read grant_id; do \
		if [ ! -z "$$grant_id" ] && [ "$$grant_id" != "ID" ]; then \
			echo "  Deleting test grant: $$grant_id"; \
			curl -s -X DELETE "https://api.us.nylas.com/v3/grants/$$grant_id" \
				-H "Authorization: Bearer $$NYLAS_API_KEY" > /dev/null && \
			echo "    ✓ Deleted grant $$grant_id" || echo "    ✗ Failed to delete $$grant_id"; \
		fi \
	done
	@echo ""
	@echo "✓ Test cleanup complete"

# ============================================================================
# Playwright E2E Tests (Air + UI Web Interfaces)
# ============================================================================
# E2E tests using Playwright for:
# - Nylas Air: Modern web email client (http://localhost:7365)
# - Nylas UI: Web-based CLI admin interface (http://localhost:7363)
# Requires: npm (in tests/ directory)

# Run all E2E tests (Air + UI)
test-e2e: test-playwright

# Run only Air (web email client) tests
test-e2e-air: test-playwright-air

# Run only UI (CLI admin interface) tests
test-e2e-ui: test-playwright-ui

test-playwright:
	@echo "=== Running All Playwright E2E Tests (Air + UI) ==="
	@command -v npm >/dev/null 2>&1 || { \
		echo "ERROR: npm not installed"; \
		echo "Install Node.js and npm first"; \
		exit 1; \
	}
	@echo "Building latest binary..."
	@$(MAKE) --no-print-directory build
	@echo ""
	@echo "Installing Playwright dependencies..."
	@cd tests && npm install
	@echo ""
	@echo "Running E2E tests..."
	@cd tests && npx playwright test
	@echo ""
	@echo "✓ Playwright E2E tests complete!"
	@echo "  Report: tests/playwright-report/index.html"

test-playwright-air:
	@echo "=== Running Playwright Air (Web Email Client) Tests ==="
	@command -v npm >/dev/null 2>&1 || { \
		echo "ERROR: npm not installed"; \
		exit 1; \
	}
	@$(MAKE) --no-print-directory build
	@cd tests && npm install
	@cd tests && npx playwright test --project=air-chromium
	@echo "✓ Air E2E tests complete!"

test-playwright-ui:
	@echo "=== Running Playwright UI (CLI Admin Interface) Tests ==="
	@command -v npm >/dev/null 2>&1 || { \
		echo "ERROR: npm not installed"; \
		exit 1; \
	}
	@$(MAKE) --no-print-directory build
	@cd tests && npm install
	@cd tests && npx playwright test --project=ui-chromium
	@echo "✓ UI E2E tests complete!"

test-playwright-interactive:
	@echo "=== Running Playwright E2E Tests (Interactive Mode) ==="
	@$(MAKE) --no-print-directory build
	@cd tests && npm install
	@cd tests && npx playwright test --ui

test-playwright-headed:
	@echo "=== Running Playwright E2E Tests (Headed Browser) ==="
	@$(MAKE) --no-print-directory build
	@cd tests && npm install
	@cd tests && npx playwright test --headed

# ============================================================================
# Security Targets
# ============================================================================
security:
	@echo "=== Security Scan ==="
	@echo "Checking for hardcoded API keys..."
	@grep -rE "nyk_v0[a-zA-Z0-9_]{20,}" --include="*.go" . | grep -v "_test.go" && echo "WARNING: Possible API key found!" || echo "✓ No API keys found"
	@echo ""
	@echo "Checking for credential patterns..."
	@grep -rE "(api_key|password|secret)\s*=\s*\"[^\"]+\"" --include="*.go" . | grep -v "_test.go" | grep -v "mock.go" && echo "WARNING: Possible credentials found!" || echo "✓ No hardcoded credentials"
	@echo ""
	@echo "Checking for full credential logging..."
	@grep -rE "fmt\.(Print|Fprint|Sprint).*[Aa]pi[Kk]ey[^:\[]" --include="*.go" . | grep -v "token.go" | grep -v "doctor.go" && echo "WARNING: Possible credential logging!" || echo "✓ No credential logging"
	@echo ""
	@echo "Checking staged files..."
	@git diff --cached --name-only | grep -E "\.(env|key|pem|json)$$" && echo "WARNING: Sensitive file staged!" || echo "✓ No sensitive files staged"
	@echo ""
	@echo "=== Security scan complete ==="

# ============================================================================
# CI Targets
# ============================================================================
# Primary CI target - Go tools already use all CPU cores internally
ci: fmt vet lint test-unit test-race security vuln build
	@echo ""
	@echo "================================="
	@echo "✓ All CI checks passed!"
	@echo "================================="

# Run full CI pipeline including integration tests and cleanup (requires env vars)
# This is the COMPLETE validation - runs everything and cleans up after
# Output saved to ci-full.txt for review
ci-full:
	@echo "================================="
	@echo "Running Full CI Pipeline..."
	@echo "================================="
	@: > ci-full.txt
	@bash -o pipefail -c '\
		set -eu; \
		cleanup_needed=0; \
		run_cleanup() { \
			if [ "$$cleanup_needed" -eq 1 ]; then \
				echo ""; \
				echo "================================="; \
				echo "Cleaning up test resources..."; \
				echo "================================="; \
				$(MAKE) --no-print-directory test-cleanup || true; \
				cleanup_needed=0; \
			fi; \
		}; \
		trap '"'"'status=$$?; run_cleanup; trap - EXIT; exit $$status'"'"' EXIT; \
		exec > >(tee -a ci-full.txt) 2>&1; \
		$(MAKE) --no-print-directory ci; \
		echo ""; \
		echo "================================="; \
		echo "Running CLI Regression Checks..."; \
		echo "================================="; \
		$(MAKE) --no-print-directory test-cli-regressions; \
		echo ""; \
		echo "================================="; \
		echo "Running Agent Integration Checks..."; \
		echo "================================="; \
		: "$${NYLAS_API_KEY:?NYLAS_API_KEY is required for agent integration tests}"; \
		: "$${NYLAS_GRANT_ID:?NYLAS_GRANT_ID is required for agent integration tests}"; \
		: "$${NYLAS_AGENT_DOMAIN:?NYLAS_AGENT_DOMAIN is required for agent integration tests}"; \
		cleanup_needed=1; \
		$(MAKE) --no-print-directory test-integration-agent; \
		echo ""; \
		echo "================================="; \
		echo "Running Integration Tests..."; \
		echo "================================="; \
		NYLAS_TEST_SKIP_AGENT=true $(MAKE) --no-print-directory test-integration; \
		$(MAKE) --no-print-directory test-air-integration; \
		run_cleanup; \
		echo ""; \
		echo "================================="; \
		echo "✓ Full CI pipeline completed!"; \
		echo "  - All quality checks passed"; \
		echo "  - All tests passed"; \
		echo "  - Test resources cleaned up"; \
		echo "================================="; \
	'
	@echo ""
	@echo "Results saved to ci-full.txt"

# ============================================================================
# Utility Targets
# ============================================================================
# Check context size for Claude Code
check-context:
	@echo "📊 Context Size Report"
	@echo "======================"
	@echo ""
	@echo "Auto-loaded files:"
	@ls -lh CLAUDE.md $$(ls .claude/rules/*.md 2>/dev/null | grep -v '.local.md') docs/DEVELOPMENT.md docs/security/overview.md 2>/dev/null | awk '{print $$5, $$9}'
	@echo ""
	@echo "On-demand files (excluded from auto-load):"
	@ls -lh docs/COMMANDS.md docs/commands/mcp.md docs/commands/ai.md docs/ARCHITECTURE.md 2>/dev/null | awk '{print $$5, $$9}'
	@echo ""
	@TOTAL=$$(ls -l CLAUDE.md $$(ls .claude/rules/*.md 2>/dev/null | grep -v '.local.md') docs/DEVELOPMENT.md docs/security/overview.md 2>/dev/null | awk '{sum+=$$5} END {print int(sum/1024)}'); \
	ONDEMAND=$$(ls -l docs/COMMANDS.md docs/commands/mcp.md docs/commands/ai.md docs/ARCHITECTURE.md 2>/dev/null | awk '{sum+=$$5} END {print int(sum/1024)}'); \
	echo "Auto-loaded context: $${TOTAL}KB (~$$((TOTAL / 4)) tokens)"; \
	echo "On-demand available: $${ONDEMAND}KB"; \
	echo ""; \
	if [ $$TOTAL -gt 50 ]; then \
		echo "⚠️  Context exceeds 50KB budget (currently $${TOTAL}KB)"; \
	else \
		echo "✅ Context within 50KB budget ($${TOTAL}KB)"; \
	fi

clean:
	@echo "=== Cleaning build artifacts ==="
	rm -rf bin/
	rm -f coverage.out coverage.html ci-full.txt test-integration.txt *.test
	@echo "✓ Cleanup complete"

clean-cache:
	@echo "=== Cleaning Go caches (use when cache is corrupted) ==="
	go clean -cache -modcache -testcache
	go mod download
	@echo "✓ Go caches cleaned and modules re-downloaded"

install: build
	@echo "=== Installing binary to GOPATH/bin ==="
	cp bin/nylas $(GOPATH)/bin/nylas
	@echo "✓ Installed to $(GOPATH)/bin/nylas"

deps:
	@echo "=== Updating dependencies ==="
	go mod tidy
	go mod download
	@echo "✓ Dependencies updated"

# Run a specific package's tests
# Usage: make test-pkg PKG=email
test-pkg:
	@echo "=== Testing package: $(PKG) ==="
	go test ./internal/cli/$(PKG)/... -v

# Quick build and run
run: build
	./bin/nylas $(ARGS)

# ============================================================================
# Help
# ============================================================================
help:
	@echo "=========================================="
	@echo "Nylas CLI - Makefile Help"
	@echo "=========================================="
	@echo ""
	@echo "🚀 PRIMARY COMMAND (Does Everything):"
	@echo "  ci-full                    - Complete CI pipeline:"
	@echo "                               • All code quality checks"
	@echo "                               • All unit & integration tests"
	@echo "                               • Automatic cleanup"
	@echo "                               • Output saved to ci-full.txt"
	@echo ""
	@echo "BUILD:"
	@echo "  build                      - Build the CLI binary"
	@echo "  install                    - Install binary to GOPATH/bin"
	@echo "  clean                      - Remove build artifacts"
	@echo ""
	@echo "CODE QUALITY:"
	@echo "  fmt                        - Format code with go fmt"
	@echo "  vet                        - Run go vet analysis"
	@echo "  lint                       - Run golangci-lint (5m timeout)"
	@echo "  vuln                       - Check for vulnerabilities"
	@echo "  security                   - Scan for hardcoded credentials"
	@echo ""
	@echo "TESTING:"
	@echo "  test-unit                  - Run unit tests (-short)"
	@echo "  test-race                  - Run tests with race detector"
	@echo "  test-coverage              - Generate coverage report"
	@echo "  test-air                   - Run Air web UI tests"
	@echo ""
	@echo "INTEGRATION TESTS (auto-loads .env file):"
	@echo "  test-integration           - Run all integration tests"
	@echo "                               (rate limited: 1 RPS, sequential)"
	@echo "  test-integration-fast      - Run fast tests (skip LLM)"
	@echo "                               (rate limited: 1 RPS, sequential)"
	@echo "  test-air-integration       - Run Air integration tests"
	@echo "                               (rate limited: 1 RPS, sequential)"
	@echo "  test-cleanup               - Clean up test resources"
	@echo ""
	@echo "  Required .env variables:"
	@echo "    NYLAS_API_KEY            - Your Nylas API key"
	@echo "    NYLAS_GRANT_ID           - Your grant ID"
	@echo ""
	@echo "CI (Granular):"
	@echo "  ci                         - All quality checks (no integration)"
	@echo "                               (fmt, vet, lint, test-unit,"
	@echo "                                test-race, security, vuln, build)"
	@echo ""
	@echo "UTILITIES:"
	@echo "  deps                       - Update dependencies"
	@echo "  check-context              - Check Claude Code context size"
	@echo "  help                       - Show this help"
	@echo ""
	@echo "=========================================="
	@echo "Recommended workflows:"
	@echo "  make ci-full               - Complete validation (use this!)"
	@echo "  make ci                    - Quick pre-commit checks"
	@echo "  make test-coverage         - Check coverage locally"
	@echo "=========================================="
