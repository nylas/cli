// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Smoke tests for Nylas Air.
 *
 * These tests verify that the application loads correctly
 * and all major UI elements are present and functional.
 */

test.describe('Smoke Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Collect console errors for debugging
    page.on('pageerror', (error) => {
      console.error('Page error:', error.message);
    });
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
  });

  test('home page loads without JavaScript errors', async ({ page }) => {
    const errors = [];
    page.on('pageerror', (error) => errors.push(error.message));

    // Wait for DOM to be fully loaded
    await page.waitForLoadState('domcontentloaded');
    // Small delay for async initialization
    await page.waitForTimeout(500);

    // Check for critical JavaScript errors (filter out expected ones)
    const criticalErrors = errors.filter((e) => {
      // Ignore expected errors in test mode
      if (e.includes('Failed to load resource')) return false;
      if (e.includes('404')) return false;
      return true;
    });

    expect(criticalErrors).toHaveLength(0);
  });

  test('main navigation is present', async ({ page }) => {

    // Main nav bar
    const nav = page.locator(selectors.nav.main);
    await expect(nav).toBeVisible();

    // Logo
    await expect(page.locator(selectors.nav.logo)).toBeVisible();
    await expect(nav.getByText('Nylas Air')).toBeVisible();

    // Navigation tabs
    await expect(page.locator(selectors.nav.tabEmail)).toBeVisible();
    await expect(page.locator(selectors.nav.tabCalendar)).toBeVisible();
    await expect(page.locator(selectors.nav.tabContacts)).toBeVisible();

    // Search trigger
    await expect(page.locator(selectors.nav.searchTrigger)).toBeVisible();

    // Settings button
    await expect(page.locator(selectors.nav.settingsBtn)).toBeVisible();
  });

  test('current account email is fully visible', async ({ page }) => {
    const currentEmail = page.locator(selectors.nav.currentAccountEmail);
    await expect(currentEmail).toBeVisible();
    await expect(currentEmail).toContainText('@');

    const currentFits = await currentEmail.evaluate(
      (el) => el.scrollWidth <= el.clientWidth + 1
    );
    expect(currentFits).toBeTruthy();

    await page.locator(selectors.nav.accountSwitcher).click();

    const dropdownEmail = page.locator(selectors.nav.accountDropdownEmail).first();
    await expect(dropdownEmail).toBeVisible();

    const dropdownFits = await dropdownEmail.evaluate(
      (el) => el.scrollWidth <= el.clientWidth + 1
    );
    expect(dropdownFits).toBeTruthy();
  });

  test('email view is the default active view', async ({ page }) => {
    // Email view should be visible and active
    const emailView = page.locator(selectors.views.email);
    await expect(emailView).toBeVisible();
    await expect(emailView).toHaveClass(/active/);

    // Other views should not be active
    const calendarView = page.locator(selectors.views.calendar);
    await expect(calendarView).not.toHaveClass(/active/);

    const contactsView = page.locator(selectors.views.contacts);
    await expect(contactsView).not.toHaveClass(/active/);
  });

  test('email view contains core components', async ({ page }) => {
    // Wait for email view
    await expect(page.locator(selectors.email.view)).toBeVisible();

    // Folder sidebar
    const sidebar = page.locator(selectors.email.folderSidebar);
    await expect(sidebar).toBeVisible();

    // Compose button
    const composeBtn = page.locator(selectors.email.composeBtn);
    await expect(composeBtn).toBeVisible();
    await expect(composeBtn).toHaveText(/Compose/);

    // Email list container
    await expect(
      page.locator(selectors.email.emailListContainer)
    ).toBeVisible();

    // Preview pane
    await expect(page.locator(selectors.email.preview)).toBeVisible();
  });

  test('folder list is present in sidebar', async ({ page }) => {
    // Folder list
    const folderList = page.locator(selectors.email.folderList);
    await expect(folderList).toBeVisible();

    // Wait for skeleton loaders to disappear or folders to load
    // (may show skeleton if loading, or folders if loaded)
    await page.waitForTimeout(1000); // Allow initial load
  });

  test('filter tabs are present in email list', async ({ page }) => {
    // Filter tabs container
    const filterTabs = page.locator(selectors.email.filterTabs);
    await expect(filterTabs).toBeVisible();

    // Individual filter tabs
    await expect(page.locator('.filter-tab').filter({ hasText: 'All' })).toBeVisible();
    await expect(page.locator('.filter-tab').filter({ hasText: 'VIP' })).toBeVisible();
    await expect(page.locator('.filter-tab').filter({ hasText: 'Unread' })).toBeVisible();

    // "All" tab should be active by default
    const allTab = page.locator('.filter-tab').filter({ hasText: 'All' });
    await expect(allTab).toHaveClass(/active/);
  });

  test('preview pane shows empty state initially', async ({ page }) => {
    // Preview pane should have empty state
    const emptyState = page.locator(selectors.email.preview).locator('.empty-state');
    await expect(emptyState).toBeVisible();

    // Empty state content
    await expect(emptyState.locator('.empty-title')).toHaveText('Select an email');
  });

  test('toast container exists for notifications', async ({ page }) => {
    // Toast container should exist (hidden when empty)
    const toastContainer = page.locator(selectors.toast.container);
    await expect(toastContainer).toBeAttached();
  });

  test('accessibility: skip link is present', async ({ page }) => {
    // Skip link should exist
    const skipLink = page.locator(selectors.general.skipLink);
    await expect(skipLink).toBeAttached();
    await expect(skipLink).toHaveAttribute('href', '#main-content');
  });

  test('accessibility: live region for announcements exists', async ({ page }) => {
    // Live region for screen reader announcements
    const liveRegion = page.locator(selectors.general.liveRegion);
    await expect(liveRegion).toBeAttached();
    await expect(liveRegion).toHaveAttribute('role', 'status');
    await expect(liveRegion).toHaveAttribute('aria-live', 'polite');
  });

  test('page has proper document title', async ({ page }) => {
    await expect(page).toHaveTitle('Nylas Air');
  });

  test('page has proper viewport meta tag', async ({ page }) => {
    const viewport = page.locator('meta[name="viewport"]');
    await expect(viewport).toHaveAttribute(
      'content',
      expect.stringContaining('width=device-width')
    );
  });
});
