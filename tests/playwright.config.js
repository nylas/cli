// @ts-check
const { defineConfig, devices } = require('@playwright/test');

/**
 * Playwright configuration for Nylas E2E tests.
 *
 * Supports three test targets:
 * - Air: Modern web email client (http://localhost:7365)
 * - UI: Web-based CLI admin interface (http://localhost:7363)
 * - Chat: AI chat interface (http://localhost:7367)
 *
 * @see https://playwright.dev/docs/test-configuration
 */

// Environment variables
const isCI = !!process.env.CI;
const airPort = parseInt(process.env.AIR_PORT || '7365', 10);
const uiPort = parseInt(process.env.UI_PORT || '7363', 10);
const chatPort = parseInt(process.env.CHAT_PORT || '7367', 10);

module.exports = defineConfig({
  // Run tests in parallel within files
  fullyParallel: true,

  // Fail the build on CI if accidentally left test.only
  forbidOnly: isCI,

  // Retry on CI only
  retries: isCI ? 2 : 0,

  // Single worker on CI for stability, parallel locally
  workers: isCI ? 1 : undefined,

  // Reporter configuration
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results/results.json' }],
  ],

  // Global timeout for each test
  timeout: 30000,

  // Expect timeout
  expect: {
    timeout: 5000,
  },

  // Output directory for test artifacts
  outputDir: 'test-results/',

  // Projects (test configurations)
  projects: [
    // =========================================================================
    // Nylas Air (Modern Web Email Client)
    // =========================================================================
    {
      name: 'air-chromium',
      testDir: './air/e2e',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: `http://localhost:${airPort}`,
        viewport: { width: 1280, height: 720 },
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'on-first-retry',
        actionTimeout: 10000,
        navigationTimeout: 30000,
      },
    },

    // =========================================================================
    // Nylas UI (Web-based CLI Admin Interface)
    // =========================================================================
    {
      name: 'ui-chromium',
      testDir: './ui/e2e',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: `http://localhost:${uiPort}`,
        viewport: { width: 1280, height: 720 },
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'on-first-retry',
        actionTimeout: 10000,
        navigationTimeout: 30000,
      },
    },

    // =========================================================================
    // Nylas Chat (AI Chat Interface)
    // =========================================================================
    {
      name: 'chat-chromium',
      testDir: './chat/e2e',
      use: {
        ...devices['Desktop Chrome'],
        baseURL: `http://localhost:${chatPort}`,
        viewport: { width: 1280, height: 720 },
        trace: 'on-first-retry',
        screenshot: 'only-on-failure',
        video: 'on-first-retry',
        actionTimeout: 10000,
        navigationTimeout: 30000,
      },
    },
  ],

  // Web server configurations
  webServer: [
    // Nylas Air server (port 7365)
    {
      command: 'cd .. && go run cmd/nylas/main.go air --no-browser --port ' + airPort,
      port: airPort,
      timeout: 60000,
      reuseExistingServer: !isCI,
      env: {
        AIR_TEST_MODE: 'true',
      },
    },
    // Nylas UI server (port 7363)
    {
      command: 'cd .. && go run cmd/nylas/main.go ui --no-browser --port ' + uiPort,
      port: uiPort,
      timeout: 60000,
      reuseExistingServer: !isCI,
    },
    // Nylas Chat server (port 7367)
    {
      command: 'cd .. && go run cmd/nylas/main.go chat --no-browser --port ' + chatPort,
      port: chatPort,
      timeout: 60000,
      reuseExistingServer: !isCI,
    },
  ],
});
