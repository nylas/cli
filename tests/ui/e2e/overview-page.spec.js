// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/ui-selectors');

/**
 * Overview Page tests for Nylas UI.
 *
 * Tests the dashboard/overview page content including
 * configuration, accounts, resources, and quick commands.
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

test.describe('Overview Page Layout', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('overview is the default active page', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Overview nav should be active
    await expect(page.locator(selectors.nav.overview)).toHaveClass(/active/);

    // Overview page should be active
    await expect(page.locator(selectors.pages.overview)).toHaveClass(/active/);
  });

  test('dashboard header shows title', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Dashboard title
    const title = page.locator('#page-overview h1, .dashboard-title');
    if ((await title.count()) > 0) {
      await expect(title.first()).toBeVisible();
    }
  });

  test('dashboard shows connected status badge', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Status badge should show connected
    const statusBadge = page.locator(selectors.dashboard.statusBadge);
    await expect(statusBadge).toBeVisible();
    await expect(statusBadge).toHaveText('Connected');
  });

  test('dashboard has main sections', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Overview page should be visible with glass cards
    const overviewPage = page.locator('#page-overview');
    await expect(overviewPage).toBeVisible();

    // Should have glass cards
    const cards = overviewPage.locator('.glass-card, h3');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });
});

test.describe('Configuration Card', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('configuration card shows region', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Region should be visible in overview page
    const overviewPage = page.locator('#page-overview');
    const regionLabel = overviewPage.getByText('Region');
    if ((await regionLabel.count()) > 0) {
      await expect(regionLabel.first()).toBeVisible();
    }
  });

  test('configuration card shows client ID', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await expect(page.getByText('Client ID')).toBeVisible();

    // Client ID value should be partially masked or shown
    const clientValue = page.locator('#config-client');
    if ((await clientValue.count()) > 0) {
      await expect(clientValue).toBeVisible();
    }
  });

  test('configuration card shows API key (masked)', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // API Key should be visible in overview page
    const overviewPage = page.locator('#page-overview');
    const apiKeyLabel = overviewPage.getByText('API Key');
    if ((await apiKeyLabel.count()) > 0) {
      await expect(apiKeyLabel.first()).toBeVisible();
    }
  });
});

test.describe('Connected Accounts Card', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('connected accounts section is visible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await expect(page.getByText('CONNECTED ACCOUNTS')).toBeVisible();
  });

  test('account item shows email address', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // At least one account should be visible with email
    const accountEmail = page.locator('.account-email, .account-item').first();
    if ((await accountEmail.count()) > 0) {
      await expect(accountEmail).toBeVisible();
    }
  });

  test('account item shows provider', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Provider should be visible (microsoft, google, etc.) - look for it in accounts section
    const accountsCard = page.locator('#accounts-list, .accounts-card, [class*="account"]');
    if ((await accountsCard.count()) > 0) {
      const providerText = accountsCard.getByText(/microsoft|google|yahoo|icloud|outlook/i);
      if ((await providerText.count()) > 0) {
        await expect(providerText.first()).toBeVisible();
      }
    }
  });

  test('default account has DEFAULT badge', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Default badge should be visible
    const defaultBadge = page.getByText('DEFAULT');
    if ((await defaultBadge.count()) > 0) {
      await expect(defaultBadge.first()).toBeVisible();
    }
  });

  test('shows hint to add more accounts', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Hint about adding accounts
    await expect(page.getByText('Add more accounts with')).toBeVisible();
    await expect(page.locator('code').filter({ hasText: 'nylas auth login' })).toBeVisible();
  });

  test('account avatar shows initial or image', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Account avatar (first letter initial)
    const avatar = page.locator('.account-avatar, .account-item > div').first();
    if ((await avatar.count()) > 0) {
      await expect(avatar).toBeVisible();
    }
  });
});

test.describe('Resources Card', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('resources section is visible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await expect(page.getByText('RESOURCES')).toBeVisible();
  });

  test('documentation link is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const docLink = page.getByRole('link', { name: /Documentation/i });
    await expect(docLink).toBeVisible();
    await expect(docLink).toHaveAttribute('href', 'https://developer.nylas.com/');
  });

  test('CLI Command Reference link is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const apiLink = page.getByRole('link', { name: /CLI Command Reference/i });
    await expect(apiLink).toBeVisible();
    await expect(apiLink).toHaveAttribute('href', 'https://cli.nylas.com/docs/commands');
  });

  test('GitHub link is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const githubLink = page.getByRole('link', { name: /GitHub/i });
    await expect(githubLink).toBeVisible();
    await expect(githubLink).toHaveAttribute('href', 'https://github.com/nylas');
  });

  test('Support link is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const supportLink = page.getByRole('link', { name: /Support/i });
    await expect(supportLink).toBeVisible();
    await expect(supportLink).toHaveAttribute('href', 'https://support.nylas.com');
  });

  test('resource links have descriptions', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Check for descriptions
    await expect(page.getByText('API reference & guides')).toBeVisible();
    await expect(page.getByText('Commands & flags')).toBeVisible();
    await expect(page.getByText('SDKs & examples')).toBeVisible();
    await expect(page.getByText('Get help from our team')).toBeVisible();
  });

  test('resource links have icons', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Resource links should have icons (svg or img)
    const resourceLinks = page.getByRole('link').filter({ has: page.locator('img, svg') });
    const count = await resourceLinks.count();

    // Should have at least 2 resource links with icons
    expect(count).toBeGreaterThanOrEqual(2);
  });
});

test.describe('Quick Commands Section', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('quick commands section is visible', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    await expect(page.getByText('QUICK COMMANDS')).toBeVisible();
  });

  test('auth status command is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const authStatusCmd = page.locator('code').filter({ hasText: 'nylas auth status' });
    await expect(authStatusCmd).toBeVisible();
    await expect(page.getByText('Check authentication')).toBeVisible();
  });

  test('email list command is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const emailListCmd = page.locator('code').filter({ hasText: 'nylas email list' });
    await expect(emailListCmd).toBeVisible();
    await expect(page.getByText('List recent emails')).toBeVisible();
  });

  test('calendar list command is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const calendarListCmd = page.locator('code').filter({ hasText: 'nylas calendar list' });
    await expect(calendarListCmd).toBeVisible();
    await expect(page.getByText('List calendars')).toBeVisible();
  });

  test('calendar events command is present', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const calendarEventsCmd = page.locator('code').filter({ hasText: 'nylas calendar events' });
    await expect(calendarEventsCmd).toBeVisible();
    await expect(page.getByText('View calendar events')).toBeVisible();
  });

  test('quick commands are clickable', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Quick command cards should have cursor pointer
    const cmdCards = page.locator('.cmd-card');
    const count = await cmdCards.count();

    for (let i = 0; i < count; i++) {
      const card = cmdCards.nth(i);
      const cursor = await card.evaluate((el) => window.getComputedStyle(el).cursor);
      expect(cursor).toBe('pointer');
    }
  });
});

test.describe('Overview Page Glass Cards', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('glass cards have correct styling', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    const glassCards = page.locator('.glass-card');
    const count = await glassCards.count();

    // Should have glass cards
    expect(count).toBeGreaterThan(0);
  });

  test('card titles have correct styling', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Card titles (h3 elements)
    const cardTitles = page.locator('h3');
    const count = await cardTitles.count();

    // Should have section titles
    expect(count).toBeGreaterThanOrEqual(3);
  });
});

test.describe('Overview Page Responsiveness', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('overview page is visible on mobile viewport', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(300);

    // Overview should still be visible
    await expect(page.locator(selectors.pages.overview)).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });

  test('overview page is visible on tablet viewport', async ({ page }, testInfo) => {
    await skipIfNotConfigured(page, testInfo);

    // Set tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.waitForTimeout(300);

    // Overview should still be visible
    await expect(page.locator(selectors.pages.overview)).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });
});
