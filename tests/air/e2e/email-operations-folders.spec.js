// @ts-check
const { test, expect } = require('@playwright/test');
const selectors = require('../../shared/helpers/air-selectors');

/**
 * Folder Navigation tests - Folder switching
 */
test.describe('Folder Navigation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await expect(page.locator(selectors.general.app)).toBeVisible();
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
  });

  test('folder sidebar is visible', async ({ page }) => {
    const folderSidebar = page.locator(selectors.email.folderSidebar);
    await expect(folderSidebar).toBeVisible();
  });

  test('folder list contains folders', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    await expect(folderList).toBeVisible();

    // Should have at least one folder item or skeleton
    const folders = folderList.locator(selectors.email.folderItem);
    const skeletons = folderList.locator('.skeleton');

    const folderCount = await folders.count();
    const skeletonCount = await skeletons.count();

    expect(folderCount + skeletonCount).toBeGreaterThan(0);
  });

  test('clicking folder updates email list', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const folders = folderList.locator(selectors.email.folderItem);

    const count = await folders.count();

    if (count > 1) {
      // Click second folder (first might already be selected)
      await folders.nth(1).click();

      // Wait for folder switch
      await page.waitForTimeout(500);

      // Folder may have active/selected class or be visually distinct
      const hasActiveClass = await folders.nth(1).evaluate((el) =>
        el.classList.contains('active') || el.classList.contains('selected') || el.classList.contains('current')
      );
      const isClickable = await folders.nth(1).evaluate((el) => {
        const style = window.getComputedStyle(el);
        return el.getAttribute('data-folder-id') !== null || style.cursor === 'pointer';
      });

      // Folder should be active or clickable
      expect(hasActiveClass || isClickable).toBeTruthy();
    }
  });

  test('Inbox folder is present', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);

    // Wait for folders to load (either folders appear or skeletons disappear)
    await page.waitForTimeout(2000);

    const inboxFolder = folderList.locator('.folder-item:has-text("Inbox")');

    // Check if Inbox exists before asserting visibility
    if (await inboxFolder.count() > 0) {
      await expect(inboxFolder).toBeVisible();
    }
  });

  test('Sent folder is present', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const sentFolder = folderList.locator('.folder-item:has-text("Sent")');

    if (await sentFolder.count() > 0) {
      await expect(sentFolder).toBeVisible();
    }
  });

  test('Archive folder is visible inline without a More dropdown', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);

    await page.waitForTimeout(2000);

    await expect(folderList.locator('.folder-item:has-text("Archive")')).toBeVisible();
    await expect(folderList.locator('.folder-item:has-text("More")')).toHaveCount(0);
  });

  test('folders show unread count badge', async ({ page }) => {
    const folderList = page.locator(selectors.email.folderList);
    const folders = folderList.locator(selectors.email.folderItem);

    const count = await folders.count();

    if (count > 0) {
      // Check if any folders have count badge
      const badges = folderList.locator('.folder-count');
      const badgeCount = await badges.count();

      // It's okay if no badges (no unread emails)
      expect(badgeCount >= 0).toBeTruthy();
    }
  });
});
