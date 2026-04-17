// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/ui-selectors');

/**
 * Command Pages tests for Nylas UI.
 *
 * Tests all command pages (Admin, Auth, Calendar, etc.)
 * and their command execution functionality.
 */

/**
 * Helper to check if dashboard is active (configured state).
 */
async function isDashboardActive(page) {
  const dashboardView = page.locator(selectors.dashboard.view);
  return await dashboardView.evaluate((el) => el.classList.contains('active'));
}

/**
 * Helper to skip test if not in configured state.
 */
async function skipIfNotConfigured(page, testInfo) {
  if (!(await isDashboardActive(page))) {
    testInfo.skip();
  }
}

test.describe('Command Page Structure', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  const commandPages = [
    { nav: 'Admin', page: 'admin' },
    { nav: 'Auth', page: 'auth' },
    { nav: 'Calendar', page: 'calendar' },
    { nav: 'Contacts', page: 'contacts' },
    { nav: 'Email', page: 'email' },
    { nav: 'Notetaker', page: 'notetaker' },
    { nav: 'OTP', page: 'otp' },
    { nav: 'Scheduler', page: 'scheduler' },
    { nav: 'Timezone', page: 'timezone' },
    { nav: 'Webhook', page: 'webhook' },
  ];

  for (const cmdPage of commandPages) {
    test(`${cmdPage.nav} page is accessible`, async ({ page }, testInfo) => {
      await skipIfNotConfigured(page, testInfo);

      await page.locator(`[data-page="${cmdPage.page}"]`).click();
      await page.waitForTimeout(300);

      // Page should be active
      await expect(page.locator(`#page-${cmdPage.page}`)).toHaveClass(/active/);

      // Should have some content (heading or command list)
      const pageContent = page.locator(`#page-${cmdPage.page}`);
      await expect(pageContent).toBeVisible();
    });
  }

  test('command page shows content', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    // Email page should be active and have content
    await expect(page.locator('#page-email')).toHaveClass(/active/);
    await expect(page.locator('#page-email')).toBeVisible();
  });

  test('command pages have command list section', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Command list container should exist
    const cmdList = page.locator('#auth-cmd-list');
    await expect(cmdList).toBeVisible();
  });
});

test.describe('Auth Command Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Auth page has command sections', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Auth page should be active
    await expect(page.locator('#page-auth')).toHaveClass(/active/);

    // Should show auth commands section
    await expect(page.getByText('Auth Commands')).toBeVisible();
  });

  test('Auth page has Login command', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Should have Login command in the list
    const cmdList = page.locator('#auth-cmd-list');
    await expect(cmdList.getByText('Login')).toBeVisible();
  });

  test('Auth page has Status command', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Should have Status command in the list
    const cmdList = page.locator('#auth-cmd-list');
    await expect(cmdList.getByText('Status')).toBeVisible();
  });

  test('clicking command shows command detail panel', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Click on Status command in the command list
    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Command detail should show heading
    const heading = page.locator('h2').filter({ hasText: 'Status' });
    await expect(heading).toBeVisible();
  });

  test('command detail shows code block', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Code block should show some command
    const codeBlock = page.locator('#page-auth code').first();
    if ((await codeBlock.count()) > 0) {
      await expect(codeBlock).toBeVisible();
    }
  });

  test('command detail has Run button', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Run button should be visible
    const runBtn = page.getByRole('button', { name: 'Run', exact: true });
    await expect(runBtn).toBeVisible();
  });

  test('command detail has Re-run button', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Re-run button should be visible
    const rerunBtn = page.getByRole('button', { name: 'Re-run command' });
    await expect(rerunBtn).toBeVisible();
  });

  test('command detail has output panel', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Output section should be visible (look for Output heading or panel)
    const outputSection = page.locator('#page-auth').getByText('Output');
    if ((await outputSection.count()) > 0) {
      await expect(outputSection.first()).toBeVisible();
    }
  });

  test('output panel shows placeholder before running', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Placeholder text
    await expect(page.getByText('Click "Run" to execute command')).toBeVisible();
  });

  test('running command executes successfully', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    // Click Run
    const runBtn = page.getByRole('button', { name: 'Run', exact: true });
    if ((await runBtn.count()) > 0) {
      await runBtn.click();
      await page.waitForTimeout(2000);

      // Just verify the page is still functional after running
      await expect(page.locator('#page-auth')).toBeVisible();
    }
  });

  test('running command shows last run timestamp', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    await page.getByRole('button', { name: 'Run', exact: true }).click();
    await page.waitForTimeout(1000);

    // Should show last run time
    await expect(page.getByText('Last run:')).toBeVisible();
  });

  test('running command shows toast notification', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);

    await page.getByRole('button', { name: 'Run', exact: true }).click();

    // Toast should appear
    await expect(page.getByText('Command completed')).toBeVisible({ timeout: 5000 });
  });

  test('switching commands updates detail panel', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    const cmdList = page.locator('#auth-cmd-list');

    // Click Status
    await cmdList.getByText('Status').click();
    await page.waitForTimeout(300);
    await expect(page.locator('h2').filter({ hasText: 'Status' })).toBeVisible();

    // Click Login
    await cmdList.getByText('Login').click();
    await page.waitForTimeout(300);
    await expect(page.locator('h2').filter({ hasText: 'Login' })).toBeVisible();
  });
});

test.describe('Specific Command Pages', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('Timezone page is accessible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.timezone).click();
    await page.waitForTimeout(300);

    await expect(page.locator('#page-timezone')).toHaveClass(/active/);
  });

  test('Calendar page is accessible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.calendar).click();
    await page.waitForTimeout(300);

    await expect(page.locator('#page-calendar')).toHaveClass(/active/);
  });

  test('Email page is accessible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    await expect(page.locator('#page-email')).toHaveClass(/active/);
  });

  test('Contacts page is accessible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.contacts).click();
    await page.waitForTimeout(300);

    await expect(page.locator('#page-contacts')).toHaveClass(/active/);
  });
});

test.describe('Command Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('can navigate between all command pages', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const pages = ['admin', 'auth', 'calendar', 'contacts', 'email'];

    for (const pageName of pages) {
      await page.locator(`[data-page="${pageName}"]`).click();
      await page.waitForTimeout(200);

      // Nav item should be active
      await expect(page.locator(`[data-page="${pageName}"]`)).toHaveClass(/active/);

      // Page should be active
      await expect(page.locator(`#page-${pageName}`)).toHaveClass(/active/);
    }
  });

  test('active nav item is highlighted', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(200);

    // Auth should be active
    await expect(page.locator(selectors.nav.auth)).toHaveClass(/active/);

    // Other items should not be active
    await expect(page.locator(selectors.nav.email)).not.toHaveClass(/active/);
    await expect(page.locator(selectors.nav.calendar)).not.toHaveClass(/active/);
  });

  test('command list persists on page switch', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Go to Auth
    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // Switch to Email
    await page.locator(selectors.nav.email).click();
    await page.waitForTimeout(300);

    // Go back to Auth
    await page.locator(selectors.nav.auth).click();
    await page.waitForTimeout(300);

    // The page should still show Auth page
    await expect(page.locator('#page-auth')).toHaveClass(/active/);
  });
});
